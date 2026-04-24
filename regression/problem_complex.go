package regression

import "runtime"
import "sync"
import "math"

type ProblemComplex [][2]complex128

func MinimizeComplex(prob ProblemComplex, length, iters uint, sos *float64, mut *sync.Mutex, cb Callback) (p Program) {

	mut.Lock()
	var sossos = *sos
	mut.Unlock()
outer:
	for i := uint(0); i < iters; i++ {
		candidate := New(length)

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
			p = candidate
			cb(p, sum, i, 0)
			*sos = sum
			sossos = sum
		} else {
			if sossos != *sos {
				sossos = *sos
				p = nil
			}
		}
		mut.Unlock()
	}
	return
}

func MinimizeThreadedComplex(prob ProblemComplex, length, iters uint, sos float64, cb Callback) (p Program, newsos float64) {

	procs := runtime.GOMAXPROCS(0)
	iter := iters / uint(procs)
	var mut sync.Mutex
	var wg sync.WaitGroup

	for i := 0; i < procs; i++ {
		wg.Add(1)
		go func(prob ProblemComplex, length, iters uint, sos *float64, mut *sync.Mutex, cb Callback) {
			prog := MinimizeComplex(prob, length, iter, sos, mut, cb)
			if prog != nil {
				mut.Lock()
				p = prog
				newsos = *sos
				mut.Unlock()
			}
			wg.Done()
		}(prob, length, iter, &sos, &mut, cb)
	}

	wg.Wait()

	return
}

func MinimizeRoundsComplex(prob ProblemComplex, length, iters, rounds uint, sos float64, cb Callback) (p Program, newsos float64) {
	newsos = sos
	p = []byte{0}
	for r := uint(0); r < rounds; r++ {
		var prog Program
		prog, newsos = MinimizeThreadedComplex(prob, length, iters, sos, func(prog []byte, sos float64, beta, rho uint) {
			cb(prog, sos, beta, r)
		})

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
