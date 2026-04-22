package regression

import (
	"testing"
)

func FuzzNew(f *testing.F) {
	f.Add(uint(1))
	f.Add(uint(5))
	f.Add(uint(10))
	f.Add(uint(20))

	f.Fuzz(func(t *testing.T, length uint) {
		p := New(length)

		exxCount := 0
		for _, b := range p {
			if b == EXX {
				exxCount++
			}
		}

		if exxCount != 1 {
			t.Errorf("New(%d) produced program with %d EXX bytes, expected exactly 1. Program: %v", length, exxCount, p)
		}

		var q = make(Program, len(p))
		copy(q, p)
		c := ClearVars(q)
		if c < 1 {
			t.Errorf("ClearVars returned %d, expected at least 1. Program: %v", c, p)
		}

		for _, b := range q {
			if b == EXX {
				t.Errorf("EXX found after ClearVars. Program: %v", q)
				break
			}
		}

		var evaluated = p
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Evaluate panicked on program %v: %v", p, r)
			}
		}()
		evaluated.Evaluate(1.0)
	})
}
