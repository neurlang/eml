package main

import "fmt"
import "flag"
import "strconv"
import "math"
import "encoding/binary"
import "math/rand"
import "encoding/json"
import "github.com/neurlang/eml/regression"

type complexSlice []complex128

func (f *complexSlice) String() string {
	return fmt.Sprint(*f)
}

func (f *complexSlice) Set(value string) error {
	v, err := strconv.ParseComplex(value, 64)
	if err != nil {
		return err
	}
	*f = append(*f, v)
	return nil
}

type callback_data struct {
	Percent      uint
	ErrorSqSum   JSONFloat
	FunctionZero JSONFloat
	Iteration    uint
	Round        uint
	Array        []int
	Data         []byte
	Function     string
	Seed         uint64
	Thread       uint64
}

type JSONFloat float64

func (j JSONFloat) MarshalJSON() ([]byte, error) {
	v := float64(j)
	if math.IsNaN(float64(j)) {
		return []byte("null"), nil
	}
	if math.IsInf(float64(j), 0) {
		// handle infinity, assign desired value to v
		// or say +/- indicates infinity
		s := `"+∞"`
		if math.IsInf(v, -1) {
			s = `"-∞"`
		}
		return []byte(s), nil
	}
	return json.Marshal(v) // marshal result as standard float64
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

func callback_progress(prog []byte, sos float64, beta, rho uint, exp, log string, percent uint, _, _ uint64) {
	debugged := regression.Program(prog)
	progressbar(sos, uint64(percent), 100, debugged.Debug(exp, log))
}
func callback_json(prog []byte, sos float64, beta, rho uint, exp, log string, percent uint, seed0, seed1 uint64) {

	debugged := regression.Program(prog)
	evaluated := regression.Program(prog)
	var arr []int
	for i := range prog {
		arr = append(arr, int(prog[i]))
	}
	var d = callback_data{
		Array:        arr,
		Data:         prog,
		Function:     debugged.Debug(exp, log),
		ErrorSqSum:   JSONFloat(sos),
		FunctionZero: JSONFloat(evaluated.Evaluate(0)),
		Iteration:    beta,
		Round:        rho,
		Percent:      percent,
		Seed:         seed0,
		Thread:       seed1,
	}
	data, err := json.Marshal(&d)
	if err != nil {
		data, err = json.Marshal(err.Error())
		if err != nil {
			println("{\"Error\":" + string(data) + "}")
			return
		}
		return
	} else {
		println(string(data))
	}
}

func main() {
	var xs, ys complexSlice
	var a, r, b, s, t int
	var exp, log = "exp", "log"
	var js bool
	flag.Var(&xs, "x", "function inputs")
	flag.Var(&ys, "y", "function outputs")
	flag.IntVar(&a, "a", 1, "alpha (function size per round)")
	flag.IntVar(&r, "r", 1, "rho (rounds)")
	flag.IntVar(&b, "b", 1, "beta (brute forcing iterations)")
	flag.IntVar(&s, "s", -1, "seed (unused for random seed)")
	flag.IntVar(&t, "t", 0, "thread (second seed component)")
	flag.BoolVar(&js, "json", false, "json (dump format)")
	flag.StringVar(&exp, "e", "exp", "exp (function name)")
	flag.StringVar(&log, "l", "log", "log (function name)")
	flag.Parse()

	var seed [2]uint64

	if s == -1 {
		var b [8]byte
		if _, err := rand.Read(b[:3]); err != nil {
			seed[0] = uint64(s)
		}
		seed[0] = uint64(binary.LittleEndian.Uint64(b[:]))
	} else {
		seed[0] = uint64(s)
	}
	seed[1] = uint64(t)

	for len(xs) < len(ys) {
		xs = append(xs, 0)
	}
	for len(ys) < len(xs) {
		ys = append(ys, 0)
	}

	var problem regression.ProblemComplex
	var cb regression.Callback
	if js {
		cb = func(prog []byte, sos float64, beta, rho uint, seed0, seed1 uint64) {
			callback_json(prog, sos, beta, rho, exp, log, (100*(beta+uint(b)*rho))/(uint(b)*uint(r+1)), seed0, seed1)
		}
	} else {
		cb = func(prog []byte, sos float64, beta, rho uint, seed0, seed1 uint64) {
			callback_progress(prog, sos, beta, rho, exp, log, (100*(beta+uint(b)*rho))/(uint(b)*uint(r+1)), seed0, seed1)
		}
	}

	for i := range xs {
		problem = append(problem, [2]complex128{xs[i], ys[i]})
	}

	prog, sos := regression.MinimizeRoundsComplex(problem, uint(a), uint(b), uint(r), math.Inf(1), cb, seed)
	cb(prog, sos, uint(b), uint(r), seed[0], seed[1])
	if !js {
		debugged := regression.Program(prog)
		evaluated := regression.Program(prog)
		fmt.Println("\r")
		fmt.Println("--- RESULTS ---")
		fmt.Println("Final formula:", debugged.Debug(exp, log))
		fmt.Println("Error Sum Of Squares:", sos)
		fmt.Println("Formula eval at 0:", evaluated.EvaluateComplex(0))
		for i := range xs {
			evaluated2 := regression.Program(prog)
			fmt.Printf("Formula eval at %f: %f\n", xs[i], evaluated2.EvaluateComplex(xs[i]))
		}
		for i := range xs {
			evaluated2 := regression.Program(prog)
			fmt.Printf("Formula simple at %f: %s\n", xs[i], evaluated2.DebugShortComplex(exp, log, xs[i]))
		}
	}
}
