package main

import "fmt"
import "flag"
import "strconv"
import "math"
import "time"
import "math/rand"
import "encoding/json"
import "github.com/neurlang/eml/regression"

type floatSlice []float64

func (f *floatSlice) String() string {
	return fmt.Sprint(*f)
}

func (f *floatSlice) Set(value string) error {
	v, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return err
	}
	*f = append(*f, v)
	return nil
}

type callback_data struct {
	Percent      uint
	ErrorSqSum   float64
	FunctionZero float64
	Iteration    uint
	Round        uint
	Array        []int
	Data         []byte
	Function     string
}

func emptySpace(space int) string {
	emptySpace := ""
	for i := 0; i < space; i++ {
		emptySpace += " "
	}
	return emptySpace
}
func progressBar(progress, width int) string {
	progressBar := ""
	for i := 0; i < progress; i++ {
		progressBar += "="
	}
	return progressBar
}
func progressbar(sos float64, pos, max uint64, name string) {
	const progressBarWidth = 40
	if max > 0 {
		progress := int(pos * progressBarWidth / max)
		percent := int(pos * 100 / max)
		fmt.Printf("\r%f [%s%s] %d%% %s;\x1b[K",
			sos, progressBar(progress, progressBarWidth),
			emptySpace(progressBarWidth-progress), percent, name)
	}
}

func callback_progress(prog []byte, sos float64, beta, rho uint, percent uint) {
	debugged := regression.Program(prog)
	progressbar(sos, uint64(percent), 100, debugged.Debug())
}
func callback_json(prog []byte, sos float64, beta, rho uint, percent uint) {
	debugged := regression.Program(prog)
	evaluated := regression.Program(prog)
	var arr []int
	for i := range prog {
		arr = append(arr, int(prog[i]))
	}
	var d = callback_data{
		Array:        arr,
		Data:         prog,
		Function:     debugged.Debug(),
		ErrorSqSum:   sos,
		FunctionZero: evaluated.Evaluate(0),
		Iteration:    beta,
		Round:        rho,
		Percent:      percent,
	}
	data, err := json.Marshal(&d)
	if err != nil {
		data, err = json.Marshal(err.Error())
		if err != nil {
			println("{\"Error\":" + string(data) + "}")
			return
		}
		return
	}
	println(string(data))
}

func main() {
	var xs, ys floatSlice
	var a, r, b, s int
	var js bool
	flag.Var(&xs, "x", "function inputs")
	flag.Var(&ys, "y", "function outputs")
	flag.IntVar(&a, "a", 1, "alpha (function size per round)")
	flag.IntVar(&r, "r", 1, "rho (rounds)")
	flag.IntVar(&b, "b", 1, "beta (brute forcing iterations)")
	flag.IntVar(&s, "s", 0, "seed (random seed)")
	flag.BoolVar(&js, "json", false, "json (dump format)")
	flag.Parse()

	if s == 0 {
		rand.Seed(time.Now().UnixNano())
	} else {
		rand.Seed(int64(s))
	}

	for len(xs) < len(ys) {
		xs = append(xs, 0)
	}
	for len(ys) < len(xs) {
		ys = append(ys, 0)
	}

	var problem regression.Problem
	var cb regression.Callback
	if js {
		cb = func(prog []byte, sos float64, beta, rho uint) {
			callback_json(prog, sos, beta, rho, (100*(beta+uint(b)*rho))/(uint(b)*uint(r+1)))
		}
	} else {
		cb = func(prog []byte, sos float64, beta, rho uint) {
			callback_progress(prog, sos, beta, rho, (100*(beta+uint(b)*rho))/(uint(b)*uint(r+1)))
		}
	}

	for i := range xs {
		problem = append(problem, [2]float64{xs[i], ys[i]})
	}

	prog, sos := regression.MinimizeRounds(problem, uint(a), uint(b), uint(r), math.Inf(1), cb)
	cb(prog, sos, uint(b), uint(r))
	if !js {
		debugged := regression.Program(prog)
		evaluated := regression.Program(prog)
		fmt.Println("\r")
		fmt.Println("--- RESULTS ---")
		fmt.Println("Final formula:", debugged.Debug())
		fmt.Println("Error Sum Of Squares:", sos)
		fmt.Println("Formula eval at 0:", evaluated.Evaluate(0))
		for i := range xs {
			evaluated2 := regression.Program(prog)
			fmt.Printf("Formula eval at %f: %f\n", xs[i], evaluated2.Evaluate(xs[i]))
		}
	}
}
