package regression

import "math"

type Program []byte

const EXX byte = 0
const ONE byte = 1
const EML byte = 2

func (p *Program) pop() byte {
	if len(*p) == 0 {
		return EXX
	}
	op := (*p)[0]
	*p = (*p)[1:]
	return op
}

func (p *Program) Evaluate(x float64) float64 {
	var op = p.pop()
	if op != EXX {
		switch op {
		case ONE:
			return 1

		case EML:
			left := p.Evaluate(x)
			right := p.Evaluate(x)
			return math.Exp(left) - math.Log(right)
		}
	}
	return x
}

func (p *Program) Debug() string {
	var op = p.pop()
	if op != EXX {
		switch op {
		case ONE:
			return "1"

		case EML:
			left := p.Debug()
			right := p.Debug()
			return "exp(" + left + ")-log(" + right + ")"
		}
	}
	return "x"
}
