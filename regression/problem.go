package regression

import "runtime"
import "sync"
import "math"
import "sync/atomic"

// Callback is invoked on each improvement. It receives the candidate program
// (or, from MinimizeRounds, the candidate with its size filled in), its raw
// sum-of-squared-error sos, and the token count size of the entire composed
// program so far (K of the full expression, not the candidate alone). beta and
// rho are the iteration and round indices, seed0/seed1 the seed components.
type Callback func(prog []byte, sos float64, size uint, beta, rho uint, seed0, seed1 uint64)

type Problem [][2]float64

// MinimizeOptions configures the complexity (size) penalty applied to the
// compared loss: total = sos + Lambda * size(composed). Lambda == 0 reproduces
// the penalty-free behavior exactly. Size is the token count K of the composed
// program P accumulated so far (the identity program x has size 1); it is used
// so the penalty is charged on the entire composed expression being optimized,
// never on the current round's candidate alone.
type MinimizeOptions struct {
	Lambda float64
	Size   uint
}

// composedSize returns the token count K of the composed program Join(c, P)
// where the accumulated program P has size pSize. Replacing c's variable leaf
// with P grows the size by len(c)-1, so K = len(c)+pSize-1; a constant
// candidate (no variable leaf) is not grown into P by Join, so its composed
// size is just pSize.
func composedSize(c Program, pSize uint) uint {
	for i := range c {
		if c[i] == EXX {
			return uint(len(c)) + pSize - 1
		}
	}
	return pSize
}

// hasVar reports whether the program contains a variable (x) leaf.
func hasVar(c Program) bool {
	for i := range c {
		if c[i] == EXX {
			return true
		}
	}
	return false
}

func MinimizeCommon(prob Problem, length, iters uint, sos *float64, best *Program, mut *sync.Mutex, cb Callback, seed [2]uint64, bestScore *uint64, opts MinimizeOptions) {

	mut.Lock()
	atomic.StoreUint64(bestScore, math.Float64bits(*sos+opts.Lambda*float64(opts.Size)))
	mut.Unlock()
	r := NewRandomizer(seed)

outer:
	for i := uint(0); i < iters; i++ {
		candidate := r.New(length)

		// The penalty is charged on the full composed expression Join(c, P):
		// composed size is composedSize(candidate, opts.Size) and the candidate's
		// total is sos(c) + Lambda*that size. A candidate improves only if its
		// total is below the current best total sos(P)+Lambda*len(P), so prune
		// once its running sos exceeds that best total minus the candidate's
		// penalty. A non-positive threshold means no candidate can win, so skip
		// it without evaluating any points (also speeds up large-Lambda runs).
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
			delta := tested.Evaluate(prob[j][0]) - prob[j][1]
			sq := delta * delta
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

func Minimize(prob Problem, length, iters uint, sos *float64, mut *sync.Mutex, cb Callback, seed [2]uint64, opts MinimizeOptions) (p Program) {
	var best Program
	var bestScore uint64
	MinimizeCommon(prob, length, iters, sos, &best, mut, cb, seed, &bestScore, opts)
	mut.Lock()
	p = best
	mut.Unlock()
	return
}

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

// MinimizeThreaded runs MinimizeCommon across runtime.GOMAXPROCS(0) workers
// sharing the global best (program, score) under a mutex. Each candidate is
// pruned against an atomic mirror of the best score, so workers track the
// freshest global threshold at the cost of one lock-free atomic load per
// candidate. The authoritative score check and program update still happen
// under the mutex, preserving program-score consistency. Trade-off: the atomic
// read avoids per-candidate mutex contention (most candidates prune early) while
// keeping the threshold far fresher than the previous cached-on-improvement value.
// It returns found=true when any candidate beat the incoming sos threshold, so
// callers can distinguish a productive round from a stalled one.
func MinimizeThreaded(prob Problem, length, iters uint, sos float64, cb Callback, seed [2]uint64, opts MinimizeOptions) (p Program, newsos float64, found bool) {

	procs := runtime.GOMAXPROCS(0)
	var mut sync.Mutex
	var wg sync.WaitGroup
	var best Program
	var bestScore uint64

	for i := 0; i < procs; i++ {
		it := distIter(iters, procs, i)
		wg.Add(1)
		go func(prob Problem, length, it uint, sos *float64, best *Program, mut *sync.Mutex, cb Callback, bestScore *uint64, opts MinimizeOptions) {
			MinimizeCommon(prob, length, it, sos, best, mut, cb, [2]uint64{seed[0], uint64(i) + seed[1]}, bestScore, opts)
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

// nextStall tracks the count of consecutive rounds without improvement and
// reports whether the search should stop. An improvement resets the counter; a
// stalled round increments it; the search stops once the counter reaches
// stagnate consecutive stalled rounds. A single stalled round never stops the
// search unless stagnate is 0 or 1.
func nextStall(found bool, stall, stagnate uint) (newStall uint, stop bool) {
	if found {
		return 0, false
	}
	stall++
	if stall >= stagnate {
		return stall, true
	}
	return stall, false
}

// MinimizeRounds runs up to rounds residual-fitting rounds. Each round searches
// (via MinimizeThreaded) for a program that improves on the accumulated best
// score sos. A round that finds an improvement resets the stagnation counter;
// a round without improvement increments it. The search stops early once
// stagnate consecutive rounds produce no improvement, so a stalled search does
// not burn through the whole round budget.
func MinimizeRounds(prob Problem, length, iters, rounds, stagnate uint, sos float64, cb Callback, seed [2]uint64, opts MinimizeOptions) (p Program, newsos float64) {
	prob = append(Problem(nil), prob...)
	newsos = sos
	p = []byte{0}
	stall := uint(0)
	for r := uint(0); r < rounds; r++ {
		var prog Program
		var found bool
		roundOpts := opts
		roundOpts.Size = uint(len(p))
		prog, _, found = MinimizeThreaded(prob, length, iters, sos, func(prog []byte, sos float64, size uint, beta, rho uint, seed0, seed1 uint64) {
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
			prob[i][0] = evaluated.Evaluate(prob[i][0])
			sum := prob[i][0] - prob[i][1]
			sq := sum * sum
			sos += sq
		}
		p = Join(prog, p)
		// The "solved" early-exit uses the full total, never raw sos: with the
		// penalty the composed best is sos + Lambda*len(p), so a perfect fit
		// (sos==0) only stops when the penalty is also zero.
		if sos+opts.Lambda*float64(len(p)) <= 0 {
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
