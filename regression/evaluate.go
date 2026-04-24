package regression

import "math/cmplx"
import "math"
import "fmt"

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
func (p *Program) EvaluateComplex(x complex128) complex128 {
	var op = p.pop()
	if op != EXX {
		switch op {
		case ONE:
			return 1

		case EML:
			left := p.EvaluateComplex(x)
			right := p.EvaluateComplex(x)
			return cmplx.Exp(left) - cmplx.Log(right)
		}
	}
	return x
}

func (p *Program) Debug(exp, log string) string {
	var op = p.pop()
	if op != EXX {
		switch op {
		case ONE:
			return "1"

		case EML:
			left := p.Debug(exp, log)
			right := p.Debug(exp, log)
			return exp + "(" + left + ")-" + log + "(" + right + ")"
		}
	}
	return "x"
}

func (p *Program) IsConst() bool {
	var op = p.pop()
	if op != EXX {
		switch op {
		case ONE:
			return true

		case EML:
			left := p.IsConst()
			right := p.IsConst()
			return left && right
		}
	}
	return false
}

func (p *Program) DebugShort(exp, log string, exx float64) string {
	var op = p.pop()
	if op != EXX {
		switch op {
		case ONE:
			return "1"

		case EML:
			var qq = *p
			var q = &qq

			var left, right string
			if p.IsConst() {
				left = fmt.Sprint(q.Evaluate(exx))
			} else {
				left = q.DebugShort(exp, log, exx)
			}
			if p.IsConst() {
				right = fmt.Sprint(q.Evaluate(exx))
			} else {
				right = q.DebugShort(exp, log, exx)
			}
			if right == "1" {
				return exp + "(" + left + ")"
			}
			if left == "1" {
				return "e-" + log + "(" + right + ")"
			}
			return exp + "(" + left + ")-" + log + "(" + right + ")"
		}
	}
	return fmt.Sprint(exx)
}
func (p *Program) DebugShortComplex(exp, log string, exx complex128) string {
	var op = p.pop()
	if op != EXX {
		switch op {
		case ONE:
			return "1"

		case EML:
			var qq = *p
			var q = &qq

			var left, right string
			if p.IsConst() {
				left = fmt.Sprint(q.EvaluateComplex(exx))
			} else {
				left = q.DebugShortComplex(exp, log, exx)
			}
			if p.IsConst() {
				right = fmt.Sprint(q.EvaluateComplex(exx))
			} else {
				right = q.DebugShortComplex(exp, log, exx)
			}
			if right == "1" {
				return exp + "(" + left + ")"
			}
			if left == "1" {
				return "e-" + log + "(" + right + ")"
			}
			return exp + "(" + left + ")-" + log + "(" + right + ")"
		}
	}
	return fmt.Sprint(exx)
}
