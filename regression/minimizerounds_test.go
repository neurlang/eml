package regression

import "testing"
import "math"

// exactExpProblem builds a Problem of exact math.Exp(x) samples, so the tree
// exp(x) == EML(EXX, ONE) fits with sos == 0.
func exactExpProblem() Problem {
	xs := []float64{1, 2, 3, 4, 5}
	prob := Problem{}
	for _, x := range xs {
		prob = append(prob, [2]float64{x, math.Exp(x)})
	}
	return prob
}

// TestMinimizeRoundsPerfectFit verifies that MinimizeRounds returns sos == 0
// (not the stale +Inf initial value) and a size-3 program when an exact fit is
// found in round 0 (issue 13): newsos must be assigned before the solved break.
func TestMinimizeRoundsPerfectFit(t *testing.T) {
	prob := exactExpProblem()
	p, sos := MinimizeRounds(prob, 20, 100000, 5, 10, math.Inf(1), nil, [2]uint64{1, 0}, MinimizeOptions{})
	if sos != 0 {
		t.Fatalf("perfect fit reported sos %v, want 0", sos)
	}
	if len(p) != 3 {
		t.Fatalf("perfect fit program size %d, want 3 (exp(x)-log(1))", len(p))
	}
}

// TestMinimizeRoundsComplexPerfectFit is the complex-domain twin of
// TestMinimizeRoundsPerfectFit.
func TestMinimizeRoundsComplexPerfectFit(t *testing.T) {
	prob := exactExpProblem().AsComplex()
	p, sos := MinimizeRoundsComplex(prob, 20, 100000, 5, 10, math.Inf(1), nil, [2]uint64{1, 0}, MinimizeOptions{})
	if sos != 0 {
		t.Fatalf("complex perfect fit reported sos %v, want 0", sos)
	}
	if len(p) != 3 {
		t.Fatalf("complex perfect fit program size %d, want 3", len(p))
	}
}

// TestMinimizeRoundsPerfectFitStopsEarly verifies that with lambda > 0 the
// search stops as soon as a round reaches sos == 0, instead of burning the
// remaining rounds: Join only grows the program, so later rounds cannot lower
// the total. Assert the callback's maximum round index is 0.
func TestMinimizeRoundsPerfectFitStopsEarly(t *testing.T) {
	prob := exactExpProblem()
	maxRound := uint(999)
	cb := func(prog []byte, sos float64, size uint, beta, rho uint, seed0, seed1 uint64) {
		if rho < maxRound {
			maxRound = rho
		}
	}
	p, sos := MinimizeRounds(prob, 20, 500000, 5, 10, math.Inf(1), cb, [2]uint64{1, 0}, MinimizeOptions{Lambda: 0.5, Size: 1})
	if sos != 0 {
		t.Fatalf("perfect fit reported sos %v, want 0", sos)
	}
	if maxRound != 0 {
		t.Fatalf("early-exit did not fire: earliest callback round %d, want 0", maxRound)
	}
	if len(p) != 3 {
		t.Fatalf("perfect fit program size %d, want 3", len(p))
	}
}

// TestMinimizeRoundsImperfectUnchanged verifies that for data with no exact fit
// (sos > 0 at every round) the search runs normally and the reported newsos
// matches the last callback improvement: the newsos-before-break reorder and the
// sos==0 exit must not change behavior for imperfect fits.
func TestMinimizeRoundsImperfectUnchanged(t *testing.T) {
	xs := []float64{1, 2, 3}
	ys := []float64{1213, 846, 6479}
	prob := Problem{}
	for i := range xs {
		prob = append(prob, [2]float64{xs[i], ys[i]})
	}
	lastSos := math.Inf(1)
	cb := func(prog []byte, sos float64, size uint, beta, rho uint, seed0, seed1 uint64) {
		lastSos = sos
	}
	p, sos := MinimizeRounds(prob, 10, 2000, 3, 2, math.Inf(1), cb, [2]uint64{0, 0}, MinimizeOptions{Lambda: 0.37, Size: 1})
	if sos <= 0 {
		t.Fatalf("imperfect fit reported sos %v, want > 0", sos)
	}
	if p == nil {
		t.Fatalf("imperfect fit returned nil program")
	}
	if lastSos == math.Inf(1) {
		t.Fatalf("callback never fired on an imperfect fit")
	}
	if sos != lastSos {
		t.Fatalf("reported sos %v != last callback sos %v", sos, lastSos)
	}
}
