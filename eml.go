package eml

import (
	"math"
)

var EML = false
var ONE = true
var EXX *bool = nil

type EmlProgram []*bool

func (p *EmlProgram) pop() *bool {
	if len(*p) == 0 {
		return nil
	}
	op := (*p)[0]
	*p = (*p)[1:]
	return op
}

func (p *EmlProgram) Evaluate(x float64) float64 {
	var op = p.pop()
	if op != nil {
		switch *op {
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

func SelfTest() {
	var euler = &EmlProgram{&EML, &ONE, &ONE}
	println(euler.Evaluate(1337))

	var negate = &EmlProgram{&EML, &EML, &ONE, &EML, &ONE, &EML, &ONE, &EML, &EML, &ONE, &ONE, &ONE, &EML, EXX, &ONE}
	println(negate.Evaluate(1))

	var reciprocal = &EmlProgram{&EML, &EML, &EML, &ONE, &EML, &ONE, &EML, &ONE, &EML, &EML, &ONE, &ONE, &ONE, EXX, &ONE}
	println(reciprocal.Evaluate(5))


}
