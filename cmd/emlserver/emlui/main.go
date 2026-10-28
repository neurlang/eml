package main

import (
	crand "crypto/rand"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"github.com/neurlang/eml/regression"
	"math"
	mrand "math/rand"
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
	var b uint = 100000
	var r uint = 10
	var g uint = 2
	var lambda float64 = 0
	lambdaInput := document.Call("getElementById", "lambda")
	if lambdaInput.Get("value").String() != "" {
		var l float64
		if err := json.Unmarshal([]byte(lambdaInput.Get("value").String()), &l); err == nil {
			lambda = l
		}
	}
	var cb = func(prog []byte, sos float64, size uint, beta, rho uint, seed0, seed1 uint64) {
		percent := (100 * (beta + uint(b)*rho)) / (uint(b) * uint(r+1))
		debugged := regression.Program(prog)
		go input.Set("innerHTML", fmt.Sprint(percent)+"%")
		go output.Set("innerHTML", "<small>loss: "+fmt.Sprint(sos+lambda*float64(size))+"</small> "+debugged.Debug("exp", "log"))
	}
	var seed [2]uint64
	for i := range seed {
		var rnd [8]byte
		if _, err := crand.Read(rnd[:]); err != nil {
			seed[i] = uint64(binary.LittleEndian.Uint64(rnd[:]))
		} else if _, err := mrand.Read(rnd[:]); err != nil {
			seed[i] = uint64(binary.LittleEndian.Uint64(rnd[:]))
		} else {
			seed[i] = mrand.Uint64()
		}
	}
	go func() {
		var opts regression.MinimizeOptions
		opts.Lambda = lambda
		opts.Size = 1
		var prog, sos = regression.MinimizeRoundsComplex(prob.AsComplex(), a, b, r, g, math.Inf(1), cb, seed, opts)
		cb(prog, sos, uint(len(prog)), b, r, seed[0], seed[1])
	}()
	select {}
}
