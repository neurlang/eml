package regression

import "runtime"
import "sync"
import "math"

type Callback func(prog []byte, sos float64, beta, rho uint)

type Problem [][2]float64

func Minimize(prob Problem, length, iters uint, sos *float64, mut *sync.Mutex, cb Callback) (p Program) {

	mut.Lock()
	var sossos = *sos
	mut.Unlock()
outer:
	for i := uint(0); i < iters; i++ {
		candidate := New(length)

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

func MinimizeThreaded(prob Problem, length, iters uint, sos float64, cb Callback) (p Program, newsos float64) {

	procs := runtime.GOMAXPROCS(0)
	iter := iters / uint(procs)
	var mut sync.Mutex
	var wg sync.WaitGroup

	for i := 0; i < procs; i++ {
		wg.Add(1)
		go func(prob Problem, length, iters uint, sos *float64, mut *sync.Mutex, cb Callback) {
			prog := Minimize(prob, length, iter, sos, mut, cb)
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

func MinimizeRounds(prob Problem, length, iters, rounds uint, sos float64, cb Callback) (p Program, newsos float64) {
	p = []byte{0}
	for r := uint(0); r < rounds; r++ {
		var prog Program
		prog, newsos = MinimizeThreaded(prob, length, iters, sos, func(prog []byte, sos float64, beta, rho uint) {
			cb(prog, sos, beta, r)
		})

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
