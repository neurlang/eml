package main

import (
	"crypto/rand"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"github.com/neurlang/eml/regression"
	"math"
	"syscall/js"
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
	var cb = func(prog []byte, sos float64, beta, rho uint, seed0, seed1 uint64) {
		percent := (100 * (beta + uint(b)*rho)) / (uint(b) * uint(r+1))
		debugged := regression.Program(prog)
		input.Set("innerHTML", fmt.Sprint(percent)+"%")
		output.Set("innerHTML", "<small>loss: "+fmt.Sprint(sos)+"</small> "+debugged.Debug("exp", "log"))
	}
	var rnd [8]byte
	var seed [2]uint64
	for i := range seed {
		if _, err := rand.Read(rnd[:]); err != nil {
			seed[i] = uint64(binary.LittleEndian.Uint64(rnd[:]))
		}
	}
	go func() {
		var prog, sos = regression.MinimizeRoundsComplex(prob.AsComplex(), a, b, r, math.Inf(1), cb, seed)
		cb(prog, sos, b, r, 0, 0)
	}()
	select {}
}
