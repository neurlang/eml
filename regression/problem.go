package regression

import "runtime"
import "sync"
import "math"
import "sync/atomic"

type Callback func(prog []byte, sos float64, beta, rho uint, seed0, seed1 uint64)

type Problem [][2]float64

func MinimizeCommon(prob Problem, length, iters uint, sos *float64, best *Program, mut *sync.Mutex, cb Callback, seed [2]uint64, bestScore *uint64) {

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
			delta := tested.Evaluate(prob[j][0]) - prob[j][1]
			sq := delta * delta
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

func Minimize(prob Problem, length, iters uint, sos *float64, mut *sync.Mutex, cb Callback, seed [2]uint64) (p Program) {
	var best Program
	var bestScore uint64
	MinimizeCommon(prob, length, iters, sos, &best, mut, cb, seed, &bestScore)
	mut.Lock()
	p = best
	mut.Unlock()
	return
}

// MinimizeThreaded runs MinimizeCommon across runtime.GOMAXPROCS(0) workers
// sharing the global best (program, score) under a mutex. Each candidate is
// pruned against an atomic mirror of the best score, so workers track the
// freshest global threshold at the cost of one lock-free atomic load per
// candidate. The authoritative score check and program update still happen
// under the mutex, preserving program-score consistency. Trade-off: the atomic
// read avoids per-candidate mutex contention (most candidates prune early) while
// keeping the threshold far fresher than the previous cached-on-improvement value.
// distIter returns the number of iterations assigned to worker i when the
// total budget iters is distributed across procs workers. It uses quotient and
// remainder distribution: every worker gets iters/procs iterations, and the
// first iters%procs workers receive one extra iteration. This schedules exactly
// iters candidate evaluations in total, never more, and gives work to every
// worker whenever iters > 0 (budgets smaller than the worker count still
// perform work).
func distIter(iters uint, procs, i int) uint {
	iter := iters / uint(procs)
	if uint(i) < iters%uint(procs) {
		iter++
	}
	return iter
}

func MinimizeThreaded(prob Problem, length, iters uint, sos float64, cb Callback, seed [2]uint64) (p Program, newsos float64) {

	procs := runtime.GOMAXPROCS(0)
	var mut sync.Mutex
	var wg sync.WaitGroup
	var best Program
	var bestScore uint64

	for i := 0; i < procs; i++ {
		it := distIter(iters, procs, i)
		wg.Add(1)
		go func(prob Problem, length, it uint, sos *float64, best *Program, mut *sync.Mutex, cb Callback, bestScore *uint64) {
			MinimizeCommon(prob, length, it, sos, best, mut, cb, [2]uint64{seed[0], uint64(i) + seed[1]}, bestScore)
			wg.Done()
		}(prob, length, it, &sos, &best, &mut, cb, &bestScore)
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

func MinimizeRounds(prob Problem, length, iters, rounds uint, sos float64, cb Callback, seed [2]uint64) (p Program, newsos float64) {
	newsos = sos
	p = []byte{0}
	for r := uint(0); r < rounds; r++ {
		var prog Program
		prog, newsos = MinimizeThreaded(prob, length, iters, sos, func(prog []byte, sos float64, beta, rho uint, seed0, seed1 uint64) {
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
			prob[i][0] = evaluated.Evaluate(prob[i][0])
			sum := prob[i][0] - prob[i][1]
			sq := sum * sum
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

func Join(p, q Program) Program {
	if len(p) == 0 || (len(p) == 1 && p[0] == EXX) {
		if len(q) == 0 || (len(q) == 1 && q[0] == EXX) {
			return Program{EXX}
		}
		return q
	} else if len(q) == 0 || (len(q) == 1 && q[0] == EXX) {
		return p
	}
	for i := range p {
		if p[i] == EXX {
			p = append(p[:i], append(q, p[i+1:]...)...)
			return p
		}
	}
	return q
}
