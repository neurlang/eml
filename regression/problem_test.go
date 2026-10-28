package regression

import "testing"
import "math"
import "runtime"

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
