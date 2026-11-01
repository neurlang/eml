package neuron

import "math"
import "github.com/neurlang/eml/regression"

type Neuron [][]byte

func New(num uint) Neuron {
	return make(Neuron, num)
}

func NewRandomizer(seed [2]uint64) *regression.Randomizer {
	return regression.NewRandomizer(seed)
}

func NewRandom(num, l uint, seed [2]uint64) Neuron {
	n := make(Neuron, num)
	r := regression.NewRandomizer(seed)
	for i := range n {
		n[i] = r.New(l)
	}
	return n
}

func (n *Neuron) EvaluateComplex(x []complex128, without int) (s complex128) {
	for i := range *n {
		if i == without {
			continue
		}
		p := regression.Program((*n)[i])
		s += p.EvaluateComplex(x[i])
	}
	return
}

type NeuronOpts struct {
	A, B, R, G uint
	Lambda     float64
	Seed       [2]uint64
}

func (n *Neuron) Optimize(num uint, problem regression.ProblemComplex, opts NeuronOpts) {
	var options regression.MinimizeOptions
	options.Lambda = opts.Lambda
	options.Size = 1
	prog, _ := regression.MinimizeRoundsComplex(problem, uint(opts.A), uint(opts.B), uint(opts.R), uint(opts.G), math.Inf(1), nil, opts.Seed, options)
	(*n)[num] = []byte(prog)
}
