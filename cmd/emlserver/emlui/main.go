package main

import (
	"fmt"
	"syscall/js"
)

func main() {
	document := js.Global().Get("document")
	output := document.Call("getElementById", "output")
	output.Set("innerHTML", "<p>Hello World</p>")
	fmt.Println("Hello World from WASM!")
}
