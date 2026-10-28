package regression

import "testing"
import "runtime"
import "math"
import "fmt"
import "sync/atomic"

// benchmarkProblem returns a small, hard regression problem that keeps workers
// searching across many candidates so pruning activity stays observable.
func benchmarkProblem() Problem {
	prob := Problem{}
	xs := []float64{1, 2, 3, 4, 5, 6, 7, 8}
	ys := []float64{1213, 846, 6479, 3501, 2098, 990, 4211, 1872}
	for i := range xs {
		prob = append(prob, [2]float64{xs[i], ys[i]})
	}
	return prob
}

// BenchmarkMinimizeThreaded measures runtime and improvement (callback) events
// across worker counts. The improvement count is a proxy for how many candidates
// beat the current pruning threshold: with a fresher global-best threshold,
// fewer candidates should survive to trigger improvements relative to work done.
// Compare runs before/after a pruning change, and across worker counts, to gauge
// the synchronization-overhead versus pruning-effectiveness trade-off.
func BenchmarkMinimizeThreaded(b *testing.B) {
	procs := []int{1, 4, 8, 16, runtime.GOMAXPROCS(0)}
	for _, p := range procs {
		if p > runtime.NumCPU() {
			continue
		}
		b.Run(fmt.Sprintf("procs=%d", p), func(b *testing.B) {
			old := runtime.GOMAXPROCS(0)
			runtime.GOMAXPROCS(p)
			defer runtime.GOMAXPROCS(old)

			var improvements uint64
			for i := 0; i < b.N; i++ {
				MinimizeThreaded(benchmarkProblem(), 10, 2000, math.Inf(1), func([]byte, float64, uint, uint, uint, uint64, uint64) {
					atomic.AddUint64(&improvements, 1)
				}, [2]uint64{uint64(i), 7}, MinimizeOptions{})
			}
			b.ReportMetric(float64(improvements)/float64(b.N), "improvements/op")
		})
	}
}
