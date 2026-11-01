package main

import "github.com/neurlang/classifier/datasets/mnist"
import "github.com/neurlang/eml/neuron"
import "math"
import "sync"

const classes = 10

func main() {
	dataslice, evalslice, _, _, err := mnist.New()
	if err != nil {
		panic(err.Error())
	}

	//dataslice = dataslice[:10000]
	//evalslice = evalslice[:100]

	opts := neuron.NeuronOpts{
		A: 10, B: 1000, R: 5, G: 100,
		Lambda: 0.01,
		Seed:   [2]uint64{42, 777},
	}
	r := neuron.NewRandomizer(opts.Seed)

	neurons := make([]neuron.Neuron, classes, classes)
	for c := range neurons {
		neurons[c] = neuron.NewRandom(mnist.ImgSize*mnist.ImgSize, 0, opts.Seed)
	}

	inputNatM := func(N, M uint) complex128 {
		return complex(float64(dataslice[N].Image[M])/(mnist.ImgSize*mnist.ImgSize*255), 0)
	}

	inputN := func(N uint) (out []complex128) {
		out = make([]complex128, mnist.ImgSize*mnist.ImgSize)
		for i := range out {
			out[i] = inputNatM(N, uint(i))
		}
		return
	}
	evalNatM := func(N, M uint) complex128 {
		return complex(float64(evalslice[N].Image[M])/(mnist.ImgSize*mnist.ImgSize*255), 0)
	}

	evalN := func(N uint) (out []complex128) {
		out = make([]complex128, mnist.ImgSize*mnist.ImgSize)
		for i := range out {
			out[i] = evalNatM(N, uint(i))
		}
		return
	}
	// forward returns the softmax probabilities over the 10 classes for sample N
	forwardN := func(set bool, N uint) (probs []float64) {
		var x []complex128
		if set {
			x = inputN(N)
		} else {
			x = evalN(N)
		}
		probs = make([]float64, classes)
		var max float64 = math.Inf(-1)
		for c := range probs {
			v := real(neurons[c].EvaluateComplex(x, -1))
			probs[c] = v
			if v > max {
				max = v
			}
		}
		var sum float64
		for c := range probs {
			probs[c] = math.Exp(probs[c] - max)
			sum += probs[c]
		}
		for c := range probs {
			probs[c] /= sum
		}
		return
	}
	success := func(set bool) (l float64) {
		var wg sync.WaitGroup
		var mut sync.Mutex
		var length int
		if set {
			length = len(dataslice)
		} else {
			length = len(evalslice)
		}
		for i := 0; i < length; i++ {
			wg.Add(1)
			go func() {
				var lbl byte
				if set {
					lbl = dataslice[i].Label
				} else {
					lbl = evalslice[i].Label
				}
				probs := forwardN(set, uint(i))
				var max float64 = math.Inf(-1)
				for c := range probs {
					if c == int(lbl) {
						continue
					}
					if probs[c] > max {
						max = probs[c]
					}
				}
				if probs[lbl] > max {
					mut.Lock()
					l++
					mut.Unlock()
				}
				wg.Done()
			}()
		}
		wg.Wait()
		l /= float64(length)
		return
	}
	// crossEntropy averages -log(p_true) over a given set
	crossEntropy := func(set bool) (l float64) {
		var wg sync.WaitGroup
		var mut sync.Mutex
		var length int
		if set {
			length = len(dataslice)
		} else {
			length = len(evalslice)
		}
		for i := 0; i < length; i++ {
			wg.Add(1)
			go func() {
				var lbl byte
				if set {
					lbl = dataslice[i].Label
				} else {
					lbl = evalslice[i].Label
				}
				probs := forwardN(set, uint(i))
				mut.Lock()
				l -= math.Log(probs[lbl])
				mut.Unlock()
				wg.Done()
			}()
		}
		wg.Wait()
		l /= float64(length)
		return
	}
	loss := func() (l float64) {
		return crossEntropy(true)
	}
	losseval := func() (l float64) {
		return crossEntropy(false)
	}
	for {
		println("loss:", loss(), "losseval:", losseval(), "avg success:", success(false))

		var toTrain = r.UintN(mnist.ImgSize * mnist.ImgSize)
		var toClass = r.UintN(10)
		var problem [][2]complex128
		for i := range dataslice {
			fw := neurons[toClass].EvaluateComplex(inputN(uint(i)), int(toTrain))
			x := inputNatM(uint(i), toTrain)
			var target complex128
			if uint(dataslice[i].Label) == uint(toClass) {
				target = complex(1, 0)
			}
			y := target - fw
			problem = append(problem, [2]complex128{x, y})
		}
		(&neurons[toClass]).Optimize(toTrain, problem, opts)
		opts.Seed[0]++
	}
}
