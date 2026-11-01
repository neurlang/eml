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
import (
	"bytes"
	"encoding/base64"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/plotutil"
	"sort"
)

// realPoints returns some random x, y points.
func realPoints(prog *regression.Program, minx, maxx float64) plotter.XYs {
	minx *= 50.
	maxx *= 50.
	maxx = (maxx - minx) / 100.
	pts := make(plotter.XYs, 0, 101)
	var prg = make([]byte, len(*prog), len(*prog))
	copy(prg, *prog)
	for i := (0); i <= 100; i++ {
		var pt plotter.XY
		pt.X = ((float64(i) * maxx) + minx) / 50.
		pt.Y = real(prog.EvaluateComplex(complex(((float64(i)*maxx)+minx)/50., 0)))
		*prog = make([]byte, len(prg), len(prg))
		copy(*prog, prg)
		if !math.IsNaN(pt.Y) && !math.IsInf(pt.Y, 0) {
			pts = append(pts, pt)
		}
	}
	return pts
}

// imagPoints returns some random x, y points.
func imagPoints(prog *regression.Program, minx, maxx float64) plotter.XYs {
	minx *= 50.
	maxx *= 50.
	maxx = (maxx - minx) / 100.
	pts := make(plotter.XYs, 0, 101)
	var prg = make([]byte, len(*prog), len(*prog))
	copy(prg, *prog)
	for i := (0); i <= 100; i++ {
		var pt plotter.XY
		var prg = make([]byte, len(*prog), len(*prog))
		pt.X = ((float64(i) * maxx) + minx) / 50.
		pt.Y = imag(prog.EvaluateComplex(complex(((float64(i)*maxx)+minx)/50., 0)))
		*prog = make([]byte, len(prg), len(prg))
		copy(*prog, prg)
		if !math.IsNaN(pt.Y) && !math.IsInf(pt.Y, 0) {
			pts = append(pts, pt)
		}
	}
	return pts
}

func minmaxInputs(prob *regression.Problem) (minx, maxx float64) {
	minx = -1
	maxx = 1
	for i := range *prob {
		if minx > (*prob)[i][0] {
			minx = (*prob)[i][0]
		}
		if maxx < (*prob)[i][0] {
			maxx = (*prob)[i][0]
		}
	}
	return
}

// inputPoints returns some random x, y points.
func inputPoints(prob *regression.Problem) plotter.XYs {
	pts := make(plotter.XYs, len(*prob), len(*prob))
	for i := range *prob {
		pts[i].X = (*prob)[i][0]
		pts[i].Y = (*prob)[i][1]
	}
	sort.Slice(pts, func(i, j int) bool {
		return pts[i].X < pts[j].X
	})
	return pts
}

func main() {
	document := js.Global().Get("document")
	input := document.Call("getElementById", "input")
	data := []byte(input.Get("innerHTML").String())
	input.Set("innerHTML", "")
	output := document.Call("getElementById", "output")
	figure := document.Call("getElementById", "figure")
	var prob regression.Problem
	json.Unmarshal(data, &prob)
	var a uint = 20
	var b uint = 10000
	var r uint = 10
	var g uint = 1000
	var lambda float64 = 0
	lambdaInput := document.Call("getElementById", "lambda")
	if lambdaInput.Get("value").String() != "" {
		var l float64
		if err := json.Unmarshal([]byte(lambdaInput.Get("value").String()), &l); err == nil {
			lambda = l
		}
	}
	inputpts := inputPoints(&prob)
	minx, maxx := minmaxInputs(&prob)
	var cb = func(prog []byte, sos float64, size uint, beta, rho uint, seed0, seed1 uint64) {

		percent := (100 * (beta + uint(b)*rho)) / (uint(b) * uint(r+1))
		var prg = make([]byte, len(prog), len(prog))
		copy(prg, prog)
		var debugg = regression.Program(prg)
		var dbg = debugg.Debug("exp", "log")
		copy(prg, prog)

		debugged := regression.Program(prg)

		p := plot.New()
		p.Title.Text = "Regression solution"
		p.X.Label.Text = "X"
		p.Y.Label.Text = "Y"
		var imgsrc = ""
		err := plotutil.AddLinePoints(p,
			"FitReal", realPoints(&debugged, minx, maxx),
			"FitImag", imagPoints(&debugged, minx, maxx),
			"Input", inputpts)
		if err == nil {
			w, err := p.WriterTo(500, 500, "png")
			if err == nil {
				var buf bytes.Buffer

				_, err := w.WriteTo(&buf)
				if err == nil {
					result := base64.StdEncoding.EncodeToString(buf.Bytes())
					imgsrc = "data:image/png;base64," + result
				}
			}
		}
		var istr = fmt.Sprint(percent) + "%"

		var ostr = "<small>loss: " + fmt.Sprint(sos+lambda*float64(size)) + "</small> " + dbg
		input.Set("innerHTML", istr)
		output.Set("innerHTML", ostr)
		figure.Set("src", imgsrc)

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
		var prog, sos = regression.MinimizeRoundsComplex(prob.AsComplex(), a, b, r, g, math.Inf(1), nil, seed, opts)
		cb(prog, sos, uint(len(prog)), b, r, seed[0], seed[1])
	}()
	select {}
}
