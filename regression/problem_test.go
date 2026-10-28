package regression

import "testing"
import "math"
import "runtime"
import "sync"

func TestZeroSOS(t *testing.T) {
	xs := []float64{1, 2, 3}
	ys := []float64{1213, 846, 6479}
	prob := Problem{}
	for i := range xs {
		prob = append(prob, [2]float64{xs[i], ys[i]})
	}
	_, sos := MinimizeRounds(prob, 10, 10, 1, 2, math.Inf(1), nil, [2]uint64{0, 0})
	if sos == 0 {
		t.Fatalf("SOS zero")
	}
}

// TestMinimizeRoundsNilCallback verifies that MinimizeRounds does not panic
// when passed a nil callback, and that the complex-domain variant behaves the
// same way.
func TestMinimizeRoundsNilCallback(t *testing.T) {
	xs := []float64{1, 2, 3}
	ys := []float64{1213, 846, 6479}
	prob := Problem{}
	for i := range xs {
		prob = append(prob, [2]float64{xs[i], ys[i]})
	}
	if _, sos := MinimizeRounds(prob, 10, 10, 1, 2, math.Inf(1), nil, [2]uint64{0, 0}); sos == 0 {
		t.Fatalf("SOS zero")
	}
	if _, sos := MinimizeRoundsComplex(prob.AsComplex(), 10, 10, 1, 2, math.Inf(1), nil, [2]uint64{0, 0}); sos == 0 {
		t.Fatalf("complex SOS zero")
	}
}

// TestMinimizeRoundsPreservesProblem verifies that MinimizeRounds does not
// mutate the caller-provided problem (issue 8): it works on a copy, so the
// original data points remain unchanged after the minimization runs.
func TestMinimizeRoundsPreservesProblem(t *testing.T) {
	xs := []float64{1, 2, 3}
	ys := []float64{1213, 846, 6479}
	prob := Problem{}
	for i := range xs {
		prob = append(prob, [2]float64{xs[i], ys[i]})
	}

	orig := make(Problem, len(prob))
	copy(orig, prob)

	seed := [2]uint64{0, 0}
	// Multiple rounds so a found round mutates prob if the bug is present.
	_, _ = MinimizeRounds(prob, 10, 2000, 20, 2, math.Inf(1), nil, seed)
	for i := range orig {
		if prob[i] != orig[i] {
			t.Fatalf("real: point %d mutated from %v to %v", i, orig[i], prob[i])
		}
	}

	cprob := prob.AsComplex()
	corig := make(ProblemComplex, len(cprob))
	copy(corig, cprob)
	_, _ = MinimizeRoundsComplex(cprob, 10, 2000, 20, 2, math.Inf(1), nil, seed)
	for i := range corig {
		if cprob[i] != corig[i] {
			t.Fatalf("complex: point %d mutated from %v to %v", i, corig[i], cprob[i])
		}
	}
}

// TestMinimizeThreadedBestSync verifies that MinimizeThreaded returns a
// program whose actual score equals the reported newsos, i.e. the best
// program and score stay synchronized under concurrent updates.
func TestMinimizeThreadedBestSync(t *testing.T) {
	old := runtime.GOMAXPROCS(0)
	defer runtime.GOMAXPROCS(old)
	runtime.GOMAXPROCS(8)

	xs := []float64{1, 2, 3}
	ys := []float64{1213, 846, 6479}
	prob := Problem{}
	for i := range xs {
		prob = append(prob, [2]float64{xs[i], ys[i]})
	}

	for trial := uint64(0); trial < 50; trial++ {
		p, newsos, found := MinimizeThreaded(prob, 10, 2000, math.Inf(1), nil, [2]uint64{trial, 7})

		if p == nil {
			if newsos != 0 {
				t.Fatalf("trial %d: nil program with non-zero score %v", trial, newsos)
			}
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
	}
}

// TestMinimizeThreadedIterDistribution verifies that MinimizeThreaded assigns
// the full iteration budget across workers via quotient/remainder distribution:
// every worker gets iters/procs iterations and the first iters%procs workers
// get one extra. It covers iters < procs, iters == procs, and iters % procs != 0.
func TestMinimizeThreadedIterDistribution(t *testing.T) {
	old := runtime.GOMAXPROCS(0)
	defer runtime.GOMAXPROCS(old)

	cases := []struct {
		iters uint
		procs int
	}{
		{7, 8},  // iters < procs
		{8, 8},  // iters == procs
		{10, 8}, // iters % procs != 0
		{1, 8},  // smallest nonzero budget under worker count
	}

	for _, c := range cases {
		var sum uint
		var max uint
		for i := 0; i < c.procs; i++ {
			it := distIter(c.iters, c.procs, i)
			sum += it
			if it > max {
				max = it
			}
		}
		if sum != c.iters {
			t.Fatalf("iters=%d procs=%d: scheduled %d iterations, want %d", c.iters, c.procs, sum, c.iters)
		}
		// Every worker gets work whenever iters > 0.
		if c.iters > 0 && sum == 0 {
			t.Fatalf("iters=%d procs=%d: no worker received iterations", c.iters, c.procs)
		}
		// No worker is scheduled more than ceil(iters/procs).
		ceil := (c.iters + uint(c.procs) - 1) / uint(c.procs)
		if max > ceil {
			t.Fatalf("iters=%d procs=%d: worker got %d iterations, over ceil %d", c.iters, c.procs, max, ceil)
		}
	}

	// Integration check: a budget smaller than the worker count must still
	// evaluate candidates (the historical bug scheduled zero). Count callback
	// fires as a lower bound on evaluations.
	runtime.GOMAXPROCS(8)
	xs := []float64{1, 2, 3}
	ys := []float64{1213, 846, 6479}
	prob := Problem{}
	for i := range xs {
		prob = append(prob, [2]float64{xs[i], ys[i]})
	}
	var count int64
	var mu sync.Mutex
	cb := func(prog []byte, sos float64, beta, rho uint, seed0, seed1 uint64) {
		mu.Lock()
		count++
		mu.Unlock()
	}
	_, _, _ = MinimizeThreaded(prob, 10, 7, math.Inf(1), cb, [2]uint64{0, 0})
	if count == 0 {
		t.Fatalf("iters=7 procs=8: no candidates evaluated")
	}
	if uint64(count) > 7 {
		t.Fatalf("iters=7 procs=8: %d improvements scheduled, more than budget", count)
	}
}

// TestMinimizeThreadedFoundFlag verifies that MinimizeThreaded reports whether
// the round found an improvement (issue 4): with a zero threshold no candidate
// can improve (found=false, no program), while an infinite threshold lets any
// finite candidate win (found=true, program and score reported).
func TestMinimizeThreadedFoundFlag(t *testing.T) {
	xs := []float64{1, 2, 3}
	ys := []float64{1213, 846, 6479}
	prob := Problem{}
	for i := range xs {
		prob = append(prob, [2]float64{xs[i], ys[i]})
	}
	seed := [2]uint64{0, 0}

	if p, newsos, found := MinimizeThreaded(prob, 10, 2000, 0, nil, seed); found {
		t.Fatalf("zero threshold: reported improvement %v but none is possible (p=%v newsos=%v)", found, p != nil, newsos)
	}

	p, newsos, found := MinimizeThreaded(prob, 10, 2000, math.Inf(1), nil, seed)
	if !found {
		t.Fatalf("infinite threshold: expected an improvement, got none")
	}
	if p == nil {
		t.Fatalf("infinite threshold: improvement reported without a program")
	}
	var score float64
	for j := range prob {
		var q = p
		delta := q.Evaluate(prob[j][0]) - prob[j][1]
		score += delta * delta
	}
	if score != newsos {
		t.Fatalf("reported newsos %v does not match program score %v", newsos, score)
	}
}

// TestMinimizeRoundsStallCounter verifies the stagnation counter logic used by
// MinimizeRounds (issue 4): an improvement resets the counter, a stalled round
// increments it, and the search stops only after stagnate consecutive stalled
// rounds. A single unsuccessful round never stops the search on its own.
func TestMinimizeRoundsStallCounter(t *testing.T) {
	// Immediate improvements: never accumulate, never stop.
	stall := uint(0)
	for i := uint(0); i < 10; i++ {
		var stop bool
		stall, stop = nextStall(true, stall, 2)
		if stop {
			t.Fatalf("immediate improvements must not stop the search")
		}
		if stall != 0 {
			t.Fatalf("improvement should keep counter at 0, got %d", stall)
		}
	}

	// Single stall: tolerated with stagnate=2; it increments the counter only.
	stall, stop := nextStall(false, 0, 2)
	if stop {
		t.Fatalf("a single stalled round must not stop the search")
	}
	if stall != 1 {
		t.Fatalf("single stall should increment counter to 1, got %d", stall)
	}

	// Intermittent improvements: a stall then an improvement resets the counter,
	// so the following stall is again the first of a streak.
	stall, _ = nextStall(true, stall, 2)
	if stall != 0 {
		t.Fatalf("improvement after a stall should reset counter to 0, got %d", stall)
	}
	stall, stop = nextStall(false, stall, 2)
	if stop {
		t.Fatalf("first stall of a new streak must not stop")
	}
	if stall != 1 {
		t.Fatalf("expected counter 1, got %d", stall)
	}

	// Sustained stagnation: two consecutive stalled rounds stop with stagnate=2.
	stall, stop = nextStall(false, stall, 2)
	if !stop {
		t.Fatalf("second consecutive stalled round should stop the search")
	}
	if stall != 2 {
		t.Fatalf("expected counter 2, got %d", stall)
	}

	// With stagnate=3, the same streak is not yet long enough to stop.
	stall, stop = nextStall(false, 0, 3)
	if stop {
		t.Fatalf("two stalls must not stop when stagnate=3")
	}
	stall, stop = nextStall(false, stall, 3)
	if stop {
		t.Fatalf("two stalls must not stop when stagnate=3")
	}
	stall, stop = nextStall(false, stall, 3)
	if !stop || stall != 3 {
		t.Fatalf("three stalls should stop when stagnate=3, got stop=%v stall=%d", stop, stall)
	}
}

// TestMinimizeRoundsImprovementsDoNotStall verifies that rounds which keep
// finding improvements never trip the stagnation termination (issue 4): with a
// small stagnate patience, a search that improves on every round runs through
// the whole round budget because each improvement resets the counter.
func TestMinimizeRoundsImprovementsDoNotStall(t *testing.T) {
	xs := []float64{1, 2, 3}
	ys := []float64{1213, 846, 6479}
	prob := Problem{}
	for i := range xs {
		prob = append(prob, [2]float64{xs[i], ys[i]})
	}
	seed := [2]uint64{0, 0}
	rounds := uint(20)

	for _, stagnate := range []uint{2, 20} {
		work := make(Problem, len(prob))
		copy(work, prob)
		maxRho := uint(0)
		cb := func(prog []byte, sos float64, beta, rho uint, seed0, seed1 uint64) {
			if rho > maxRho {
				maxRho = rho
			}
		}
		_, _ = MinimizeRounds(work, 10, 2000, rounds, stagnate, math.Inf(1), cb, seed)
		if maxRho != rounds-1 {
			t.Fatalf("stagnate=%d: expected all %d rounds to improve (max rho %d), got %d",
				stagnate, rounds, rounds-1, maxRho)
		}
	}
}
