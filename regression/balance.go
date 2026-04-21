package regression

import rand "math/rand"

// Make makes unbalanced EML program
func Make(length uint) (p Program) {
	p = make(Program, length)
	for i := range p {
		p[i] = ONE + byte(rand.Intn(2))
	}
	return
}

// Balance balances the EML program and tells the final program size.
// Past-the-end input is treated as an implicit 0 (variable) unary leaf.
func Balance(p *Program) uint {
	var parse func(i int) (size uint, next int)

	parse = func(i int) (uint, int) {
		// virtual zero leaf past end of input
		if i >= len(*p) {
			return 1, i + 1
		}

		switch (*p)[i] {
		case 0, 1:
			return 1, i + 1

		case 2:
			left, j := parse(i + 1)
			right, k := parse(j)
			return left + right + 1, k

		default:
			return 1, i + 1
		}
	}

	size, _ := parse(0)
	return size
}

// Clear vars replaces all variables X by 1 and counts ones
func ClearVars(p Program) (cnt uint) {
	for i := range p {
		if p[i] == EXX || p[i] == ONE {
			cnt++
			p[i] = ONE
		}
	}
	return
}

// ChooseVar replaces pos-th 1 with variable x
func ChooseVar(p Program, pos uint) {
	for i := range p {
		if p[i] == EXX || p[i] == ONE {
			if pos == 0 {
				p[i] = EXX
				return
			} else {
				pos--
			}
		}
	}
}

// New fully generates a Program of expected length length that will have no or one variable
func New(length uint) (p Program) {
	p = Make(length)
	var q = p
	b := Balance(&q)
	if len(p) > int(b) {
		p = p[:b]
	} else {
		for len(p) < int(b) {
			p = append(p, 0)
		}
	}
	c := ClearVars(p)
	if c > 0 {
		ChooseVar(p, uint(rand.Intn(int(c))))
	}
	return
}
