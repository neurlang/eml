package main

import (
	"fmt"
	"syscall/js"
	"encoding/json"
	"math"

	"github.com/neurlang/eml/regression"
)

func main() {
	document := js.Global().Get("document")
	input := document.Call("getElementById", "input")
	data := []byte(input.Get("innerHTML").String())
	input.Set("innerHTML", "")
	output := document.Call("getElementById", "output")
	var prob regression.Problem
	json.Unmarshal(data, &prob)
	var a uint = 20
	var b uint = 10
	var r uint = 10000
	var cb = func(prog []byte, sos float64, beta, rho uint) {
		percent := (100*(beta+uint(b)*rho))/(uint(b)*uint(r+1))
		debugged := regression.Program(prog)
		input.Set("innerHTML", fmt.Sprint(percent) + "%")
		output.Set("innerHTML", "<small>loss: " + fmt.Sprint(sos) + "</small> " + debugged.Debug("exp", "log"))
	}
	go func() {
		var prog, sos = regression.MinimizeRoundsComplex(prob.AsComplex(), a, b, r, math.Inf(1), cb)
		cb(prog, sos, b, r)
	} ()
	select { }
}
