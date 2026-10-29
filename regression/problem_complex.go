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

func MinimizeCommonComplex(prob ProblemComplex, length, iters uint, sos *float64, best *Program, mut *sync.Mutex, cb Callback, seed [2]uint64, bestScore *uint64, opts MinimizeOptions) {

	mut.Lock()
	atomic.StoreUint64(bestScore, math.Float64bits(*sos+opts.Lambda*float64(opts.Size)))
	mut.Unlock()
	r := NewRandomizer(seed)

outer:
	for i := uint(0); i < iters; i++ {
		candidate := r.New(length)

		composed := composedSize(candidate, opts.Size)
		cPen := opts.Lambda * float64(composed)
		var sossos = math.Float64frombits(atomic.LoadUint64(bestScore))
		threshold := sossos - cPen
		if threshold <= 0 {
			continue outer
		}

		var sum float64

		for j := range prob {
			var tested = candidate
			delta := tested.EvaluateComplex(prob[j][0]) - prob[j][1]
			sq := real(delta)*real(delta) + imag(delta)*imag(delta)
			sum += sq
			if sum > threshold {
				continue outer
			}
			if math.IsNaN(sum) {
				continue outer
			}
		}

		total := sum + cPen
		mut.Lock()
		baseSize := opts.Size
		if *best != nil {
			baseSize = composedSize(*best, opts.Size)
		}
		if *sos+opts.Lambda*float64(baseSize) > total {
			atomic.StoreUint64(bestScore, math.Float64bits(total))
			*best = candidate
			if cb != nil {
				cb(*best, sum, composed, i, 0, seed[0], seed[1])
			}
			*sos = sum
		}
		mut.Unlock()
	}
}

func MinimizeComplex(prob ProblemComplex, length, iters uint, sos *float64, mut *sync.Mutex, cb Callback, seed [2]uint64, opts MinimizeOptions) (p Program) {
	var best Program
	var bestScore uint64
	MinimizeCommonComplex(prob, length, iters, sos, &best, mut, cb, seed, &bestScore, opts)
	mut.Lock()
	p = best
	mut.Unlock()
	return
}

// MinimizeThreadedComplex is the complex-domain analogue of MinimizeThreaded:
// workers prune each candidate against an atomic mirror of the global best
// score (one lock-free load per candidate) while the authoritative score check
// and program update stay under the mutex, keeping the returned program and
// score synchronized. It returns found=true when any candidate beat the
// incoming sos threshold.
func MinimizeThreadedComplex(prob ProblemComplex, length, iters uint, sos float64, cb Callback, seed [2]uint64, opts MinimizeOptions) (p Program, newsos float64, found bool) {

	procs := runtime.GOMAXPROCS(0)
	var mut sync.Mutex
	var wg sync.WaitGroup
	var best Program
	var bestScore uint64

	for i := 0; i < procs; i++ {
		it := distIter(iters, procs, i)
		wg.Add(1)
		go func(prob ProblemComplex, length, it uint, sos *float64, best *Program, mut *sync.Mutex, cb Callback, bestScore *uint64, opts MinimizeOptions) {
			MinimizeCommonComplex(prob, length, it, sos, best, mut, cb, [2]uint64{seed[0], uint64(i) + seed[1]}, bestScore, opts)
			wg.Done()
		}(prob, length, it, &sos, &best, &mut, cb, &bestScore, opts)
	}

	wg.Wait()

	mut.Lock()
	if best != nil {
		p = best
		newsos = sos
		found = true
	}
	mut.Unlock()

	return
}

// MinimizeRoundsComplex is the complex-domain analogue of MinimizeRounds: it
// runs up to rounds residual-fitting rounds and stops early once stagnate
// consecutive rounds produce no improvement.
func MinimizeRoundsComplex(prob ProblemComplex, length, iters, rounds, stagnate uint, sos float64, cb Callback, seed [2]uint64, opts MinimizeOptions) (p Program, newsos float64) {
	prob = append(ProblemComplex(nil), prob...)
	newsos = sos
	p = []byte{0}
	stall := uint(0)
	for r := uint(0); r < rounds; r++ {
		var prog Program
		var found bool
		roundOpts := opts
		roundOpts.Size = uint(len(p))
		prog, _, found = MinimizeThreadedComplex(prob, length, iters, sos, func(prog []byte, sos float64, size uint, beta, rho uint, seed0, seed1 uint64) {
			if cb != nil {
				cb(prog, sos, size, beta, r, seed0, seed1)
			}
		}, seed, roundOpts)

		var stop bool
		stall, stop = nextStall(found, stall, stagnate)
		if stop {
			break
		}
		if !found {
			continue
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
		if sos+opts.Lambda*float64(len(p)) <= 0 {
			break
		}
		newsos = sos
	}
	return
}
