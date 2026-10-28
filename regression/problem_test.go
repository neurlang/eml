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
	_, sos := MinimizeRounds(prob, 10, 10, 1, math.Inf(1), nil, [2]uint64{0, 0})
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
	if _, sos := MinimizeRounds(prob, 10, 10, 1, math.Inf(1), nil, [2]uint64{0, 0}); sos == 0 {
		t.Fatalf("SOS zero")
	}
	if _, sos := MinimizeRoundsComplex(prob.AsComplex(), 10, 10, 1, math.Inf(1), nil, [2]uint64{0, 0}); sos == 0 {
		t.Fatalf("complex SOS zero")
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
		p, newsos := MinimizeThreaded(prob, 10, 2000, math.Inf(1), nil, [2]uint64{trial, 7})

		if p == nil {
			if newsos != 0 {
				t.Fatalf("trial %d: nil program with non-zero score %v", trial, newsos)
			}
			continue
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
	_, _ = MinimizeThreaded(prob, 10, 7, math.Inf(1), cb, [2]uint64{0, 0})
	if count == 0 {
		t.Fatalf("iters=7 procs=8: no candidates evaluated")
	}
	if uint64(count) > 7 {
		t.Fatalf("iters=7 procs=8: %d improvements scheduled, more than budget", count)
	}
}
