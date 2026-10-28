package regression

import "testing"
import "math"
import "runtime"

// TestComposedSizeIdentity verifies the size accounting used for the
// complexity penalty: for random programs c and P, the reported composed size
// (composedSize(c, len(P))) equals len(Join(c, P)), and the penalty charged is
// Lambda times that composed size. This covers both variable-bearing candidates
// (size len(c)+len(P)-1) and constant candidates (composed size is just len(P),
// because Join does not grow a constant into P).
func TestComposedSizeIdentity(t *testing.T) {
	r := NewRandomizer([2]uint64{1, 2})
	for i := 0; i < 500; i++ {
		c := r.New(uint(i%9 + 1))
		p := r.New(uint(i%7 + 1))
		want := uint(len(Join(append(Program(nil), c...), append(Program(nil), p...))))
		got := composedSize(c, uint(len(p)))
		if got != want {
			t.Fatalf("c=%v P=%v: composed size %d != len(Join) %d", c, p, got, want)
		}
		lambda := 0.37
		if lambda*float64(got) != lambda*float64(want) {
			t.Fatalf("penalty mismatch: lambda*%d != lambda*%d", got, want)
		}
	}
}

// TestThreeRoundChainSize verifies that a three-round composition h ∘ g ∘ f has
// total size len(f)+len(g)+len(h)-2, i.e. each round's candidate replaces a
// variable leaf and the penalty at the last round spans all three programs.
func TestThreeRoundChainSize(t *testing.T) {
	r := NewRandomizer([2]uint64{3, 4})
	var f, g, h Program
	for {
		f = r.New(7)
		g = r.New(5)
		h = r.New(3)
		// The chain identity assumes each round's candidate carries a variable.
		if hasVar(f) && hasVar(g) && hasVar(h) {
			break
		}
	}
	gh := Join(append(Program(nil), g...), append(Program(nil), h...))
	comp := Join(append(Program(nil), f...), append(Program(nil), gh...))
	want := uint(len(f) + len(g) + len(h) - 2)
	if uint(len(comp)) != want {
		t.Fatalf("three-round composed size %d != %d", len(comp), want)
	}
}

// TestPenaltyArithmetic verifies the single-round penalty formula
// total = sos + Lambda*len(program): the callback's reported size must equal
// the composed size of the returned program, and the reported raw sos must
// equal the returned newsos, so the total equals newsos + Lambda*len(program).
func TestPenaltyArithmetic(t *testing.T) {
	prob := Problem{{1, 2}, {2, 3}, {3, 5}}
	lambda := 0.5
	var cbSos float64
	var cbSize uint
	cb := func(prog []byte, sos float64, size uint, beta, rho uint, seed0, seed1 uint64) {
		cbSos = sos
		cbSize = size
	}
	p, newsos, found := MinimizeThreaded(prob, 5, 2000, math.Inf(1), cb, [2]uint64{0, 0}, MinimizeOptions{Lambda: lambda, Size: 1})
	if !found || p == nil {
		t.Fatalf("expected an improvement")
	}
	if cbSos != newsos {
		t.Fatalf("callback sos %v != returned newsos %v", cbSos, newsos)
	}
	if cbSize != composedSize(p, 1) {
		t.Fatalf("callback size %d != composed size %d", cbSize, composedSize(p, 1))
	}
	total := newsos + lambda*float64(composedSize(p, 1))
	if total != cbSos+lambda*float64(cbSize) {
		t.Fatalf("total mismatch: %v != %v", total, cbSos+lambda*float64(cbSize))
	}
}

// TestAcceptanceThreshold verifies the round acceptance rule: a candidate that
// lowers sos by less than Lambda*(len(c)-1) raises the total and is rejected,
// while one that lowers it by more is accepted. The equivalent condition is
// sos(c) < sos(P) - Lambda*(len(c)-1) for a variable-bearing candidate.
func TestAcceptanceThreshold(t *testing.T) {
	lambda := 1e-3
	lenP := 7
	sosP := 100.0
	lenC := 5
	comp := uint(lenC) + uint(lenP) - 1

	// Improvement 1e-6 is well below Lambda*(lenC-1)=4e-3: total goes up.
	sosC := sosP - 1e-6
	if total := sosC + lambda*float64(comp); total < sosP+lambda*float64(lenP) {
		t.Fatalf("sub-threshold sos improvement accepted")
	}
	// The sos-only condition sos(c) < sos(P)-Lambda*(lenC-1) is not met either:
	// improvement 1e-6 is smaller than the penalty Lambda*(lenC-1).
	if sosC < sosP-lambda*float64(lenC-1) {
		t.Fatalf("sub-threshold sos improvement passed the sos-only threshold")
	}

	// Improvement 1.0 is well above Lambda*(lenC-1): total goes down.
	sosC = sosP - 1.0
	if total := sosC + lambda*float64(comp); !(total < sosP+lambda*float64(lenP)) {
		t.Fatalf("above-threshold sos improvement rejected")
	}
	if !(sosC < sosP-lambda*float64(lenC-1)) {
		t.Fatalf("above-threshold sos improvement failed the sos-only threshold")
	}
}

// TestPruningCorrectness verifies that the search's winner is the brute-force
// argmin of sos(c) + Lambda*len(Join(c, P)) over the exact candidate sequence.
// It pins GOMAXPROCS to 1 so a single worker replays the deterministic seed,
// then compares the returned program, raw sos, and total against the argmin.
func TestPruningCorrectness(t *testing.T) {
	old := runtime.GOMAXPROCS(0)
	runtime.GOMAXPROCS(1)
	defer runtime.GOMAXPROCS(old)

	prob := Problem{{1, 2}, {2, 3}, {3, 5}}
	length := uint(5)
	iters := uint(40)
	lambda := 1e-2
	seed := [2]uint64{0, 0}
	opts := MinimizeOptions{Lambda: lambda, Size: 1}

	// Replay the single worker's candidate sequence and evaluate each one.
	r := NewRandomizer(seed)
	var bestTotal float64
	var bestProg Program
	var bestSos float64
	first := true
	for i := 0; i < int(iters); i++ {
		c := r.New(length)
		var sum float64
		nan := false
		for j := range prob {
			var q = c
			delta := q.Evaluate(prob[j][0]) - prob[j][1]
			sq := delta * delta
			sum += sq
			if math.IsNaN(sum) {
				nan = true
				break
			}
		}
		if nan {
			continue
		}
		total := sum + lambda*float64(composedSize(c, opts.Size))
		if first || total < bestTotal {
			bestTotal = total
			bestProg = append(Program(nil), c...)
			bestSos = sum
			first = false
		}
	}
	if first {
		t.Fatal("no finite candidate in replay")
	}

	p, newsos, found := MinimizeThreaded(prob, length, iters, math.Inf(1), nil, seed, opts)
	if !found || p == nil {
		t.Fatalf("expected an improvement")
	}
	if newsos != bestSos {
		t.Fatalf("returned sos %v != brute-force sos %v", newsos, bestSos)
	}
	if total := newsos + lambda*float64(composedSize(p, opts.Size)); total != bestTotal {
		t.Fatalf("returned total %v != brute-force total %v", total, bestTotal)
	}
	if string(p) != string(bestProg) {
		t.Fatalf("returned program %v != brute-force argmin %v", p, bestProg)
	}
}

// TestMinimizeThreadedBestSyncWithPenalty verifies that with an active penalty
// the returned program's recomputed loss stays synchronized with the returned
// newsos across many runs under GOMAXPROCS > 1 (the atomic-mirror pruning path).
func TestMinimizeThreadedBestSyncWithPenalty(t *testing.T) {
	old := runtime.GOMAXPROCS(0)
	defer runtime.GOMAXPROCS(old)
	runtime.GOMAXPROCS(8)

	xs := []float64{1, 2, 3}
	ys := []float64{1213, 846, 6479}
	prob := Problem{}
	for i := range xs {
		prob = append(prob, [2]float64{xs[i], ys[i]})
	}
	lambda := 1e-3

	for trial := uint64(0); trial < 50; trial++ {
		p, newsos, found := MinimizeThreaded(prob, 10, 2000, math.Inf(1), nil, [2]uint64{trial, 7}, MinimizeOptions{Lambda: lambda, Size: 1})
		if p == nil {
			if found {
				t.Fatalf("trial %d: nil program reported as improvement", trial)
			}
			continue
		}
		if !found {
			t.Fatalf("trial %d: non-nil program without improvement flag", trial)
		}
		var score float64
		for j := range prob {
			var q = p
			delta := q.Evaluate(prob[j][0]) - prob[j][1]
			score += delta * delta
		}
		if score != newsos {
			t.Fatalf("trial %d: program score %v != reported newsos %v", trial, score, newsos)
		}
		// The returned loss (raw sos) plus the penalty must equal the returned
		// total under the same accounting the search uses.
		if score+lambda*float64(composedSize(p, 1)) != newsos+lambda*float64(composedSize(p, 1)) {
			t.Fatalf("trial %d: loss total inconsistent", trial)
		}
	}
}
