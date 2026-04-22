package regression

import "testing"
import "math"

func TestZeroSOS(t *testing.T) {
	xs := []float64{1, 2, 3}
	ys := []float64{1213, 846, 6479}
	prob := Problem{}
	for i := range xs {
		prob = append(prob, [2]float64{xs[i], ys[i]})
	}
	_, sos := MinimizeRounds(prob, 10, 10, 1, math.Inf(1), nil)
	if sos == 0 {
		t.Fatalf("SOS zero")
	}
}
