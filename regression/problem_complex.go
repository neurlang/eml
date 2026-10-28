package regression

import "runtime"
import "sync"
import "math"
import "sync/atomic"

type ProblemComplex [][2]complex128

func (p Problem) AsComplex() (c ProblemComplex) {
	for _, v := range p {
		c = append(c, [2]complex128{complex(v[0], 0), complex(v[1], 0)})
	}
	return
}

func MinimizeCommonComplex(prob ProblemComplex, length, iters uint, sos *float64, best *Program, mut *sync.Mutex, cb Callback, seed [2]uint64, bestScore *uint64) {

	mut.Lock()
	atomic.StoreUint64(bestScore, math.Float64bits(*sos))
	mut.Unlock()
	r := NewRandomizer(seed)

outer:
	for i := uint(0); i < iters; i++ {
		candidate := r.New(length)

		var sossos = math.Float64frombits(atomic.LoadUint64(bestScore))

		var sum float64

		for j := range prob {
			var tested = candidate
			delta := tested.EvaluateComplex(prob[j][0]) - prob[j][1]
			sq := real(delta)*real(delta) + imag(delta)*imag(delta)
			sum += sq
			if sum > sossos {
				continue outer
			}
			if math.IsNaN(sum) {
				continue outer
			}
		}

		mut.Lock()
		if *sos > sum {
			atomic.StoreUint64(bestScore, math.Float64bits(sum))
			*best = candidate
			if cb != nil {
				cb(*best, sum, i, 0, seed[0], seed[1])
			}
			*sos = sum
		}
		mut.Unlock()
	}
}

func MinimizeComplex(prob ProblemComplex, length, iters uint, sos *float64, mut *sync.Mutex, cb Callback, seed [2]uint64) (p Program) {
	var best Program
	var bestScore uint64
	MinimizeCommonComplex(prob, length, iters, sos, &best, mut, cb, seed, &bestScore)
	mut.Lock()
	p = best
	mut.Unlock()
	return
}

// MinimizeThreadedComplex is the complex-domain analogue of MinimizeThreaded:
// workers prune each candidate against an atomic mirror of the global best
// score (one lock-free load per candidate) while the authoritative score check
// and program update stay under the mutex, keeping the returned program and
// score synchronized.
func MinimizeThreadedComplex(prob ProblemComplex, length, iters uint, sos float64, cb Callback, seed [2]uint64) (p Program, newsos float64) {

	procs := runtime.GOMAXPROCS(0)
	iter := iters / uint(procs)
	var mut sync.Mutex
	var wg sync.WaitGroup
	var best Program
	var bestScore uint64

	for i := 0; i < procs; i++ {
		wg.Add(1)
		go func(prob ProblemComplex, length, iters uint, sos *float64, best *Program, mut *sync.Mutex, cb Callback, bestScore *uint64) {
			MinimizeCommonComplex(prob, length, iter, sos, best, mut, cb, [2]uint64{seed[0], uint64(i) + seed[1]}, bestScore)
			wg.Done()
		}(prob, length, iter, &sos, &best, &mut, cb, &bestScore)
	}

	wg.Wait()

	mut.Lock()
	if best != nil {
		p = best
		newsos = sos
	}
	mut.Unlock()

	return
}

func MinimizeRoundsComplex(prob ProblemComplex, length, iters, rounds uint, sos float64, cb Callback, seed [2]uint64) (p Program, newsos float64) {
	newsos = sos
	p = []byte{0}
	for r := uint(0); r < rounds; r++ {
		var prog Program
		prog, newsos = MinimizeThreadedComplex(prob, length, iters, sos, func(prog []byte, sos float64, beta, rho uint, seed0, seed1 uint64) {
			if cb != nil {
				cb(prog, sos, beta, r, seed0, seed1)
			}
		}, seed)

		if newsos >= sos {
			break
		}
		sos = 0

		for i := range prob {
			var evaluated = prog
			prob[i][0] = evaluated.EvaluateComplex(prob[i][0])
			sum := prob[i][0] - prob[i][1]
			sq := real(sum)*real(sum) + imag(sum)*imag(sum)
			sos += sq
		}
		p = Join(prog, p)
		if sos == 0 {
			break
		}
		newsos = sos
	}
	return
}
