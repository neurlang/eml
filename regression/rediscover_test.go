package regression

import "testing"
import "runtime"
import "math"

// maxErr evaluates prog (as a complex program) against every point of prob and
// returns the largest squared magnitude of the residual error.
func maxErr(p Program, prob ProblemComplex) float64 {
	var worst float64
	for _, pt := range prob {
		var q = p
		out := q.EvaluateComplex(pt[0])
		d := out - pt[1]
		sq := real(d)*real(d) + imag(d)*imag(d)
		if sq > worst {
			worst = sq
		}
	}
	return worst
}

// runRoundsComplex runs MinimizeRoundsComplex at GOMAXPROCS=1 (deterministic),
// passing a copy of prob so the original points stay intact, then returns the
// final composed program and its largest squared error over the original points.
func runRoundsComplex(prob ProblemComplex, length, iters, rounds uint, seed [2]uint64) (Program, float64) {
	prev := runtime.GOMAXPROCS(1)
	defer runtime.GOMAXPROCS(prev)

	work := make(ProblemComplex, len(prob))
	copy(work, prob)

	p, _ := MinimizeRoundsComplex(work, length, iters, rounds, math.Inf(1), func([]byte, float64, uint, uint, uint64, uint64) {}, seed)
	return p, maxErr(p, prob)
}

func TestRediscoverZero(t *testing.T) {
	pts := ProblemComplex{
		{complex(1, 0), complex(0, 0)},
		{complex(2, 0), complex(0, 0)},
		{complex(-1, 0), complex(0, 0)},
		{complex(-2, 0), complex(0, 0)},
		{complex(0, 0), complex(0, 0)},
	}
	_, err := runRoundsComplex(pts, 20, 1000000, 10, [2]uint64{0, 0})
	if err > 1e-9 {
		t.Fatalf("f(x)=0 not rediscovered, largest squared error %g", err)
	}
}

func TestRediscoverNegation(t *testing.T) {
	pts := ProblemComplex{
		{complex(1, 0), complex(-1, 0)},
		{complex(2, 0), complex(-2, 0)},
		{complex(-1, 0), complex(1, 0)},
		{complex(-2, 0), complex(2, 0)},
		{complex(0, 0), complex(0, 0)},
	}
	_, err := runRoundsComplex(pts, 20, 1000000, 10, [2]uint64{0, 0})
	if err > 1e-9 {
		t.Fatalf("f(x)=-x not rediscovered, largest squared error %g", err)
	}
}

func TestRediscoverReciprocal(t *testing.T) {
	pts := ProblemComplex{
		{complex(1, 0), complex(1, 0)},
		{complex(2, 0), complex(0.5, 0)},
		{complex(0.5, 0), complex(2, 0)},
	}
	_, err := runRoundsComplex(pts, 10, 1000000, 5, [2]uint64{0, 0})
	if err > 1e-9 {
		t.Fatalf("f(x)=1/x not rediscovered, largest squared error %g", err)
	}
}
