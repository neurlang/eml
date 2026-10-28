package main

import "testing"
import "os"
import "io"
import "encoding/json"
import "syscall"

func TestValidateLengthsEqual(t *testing.T) {
	xs := complexSlice{1, 2}
	ys := complexSlice{3, 4}
	if err := validateLengths(xs, ys); err != nil {
		t.Fatalf("equal lengths should be accepted, got error: %v", err)
	}
}

func TestValidateLengthsShorterY(t *testing.T) {
	xs := complexSlice{1, 2}
	ys := complexSlice{3}
	if err := validateLengths(xs, ys); err == nil {
		t.Fatal("mismatched inputs (y shorter) should be rejected")
	}
}

func TestValidateLengthsLongerY(t *testing.T) {
	xs := complexSlice{1}
	ys := complexSlice{3, 4}
	if err := validateLengths(xs, ys); err == nil {
		t.Fatal("mismatched inputs (y longer) should be rejected")
	}
}

func TestCallbackJSONIncludesSizeAndLoss(t *testing.T) {
	prog := []byte{2, 0, 1}
	sos := 1.2655806e-11
	lambda := 1e-12
	size := uint(3)

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	origErr, err := syscall.Dup(2)
	if err != nil {
		t.Fatalf("dup: %v", err)
	}
	if err := syscall.Dup2(int(w.Fd()), 2); err != nil {
		t.Fatalf("dup2: %v", err)
	}
	callback_json(prog, sos, size, 100, 10, "exp", "log", 50, 123, 456, lambda)
	if err := syscall.Dup2(origErr, 2); err != nil {
		t.Fatalf("restore dup2: %v", err)
	}
	syscall.Close(origErr)
	w.Close()
	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("read: %v", err)
	}

	var d callback_data
	if err := json.Unmarshal(out, &d); err != nil {
		t.Fatalf("callback_json output should be valid JSON, got error: %v (output: %q)", err, string(out))
	}
	if d.Size != size {
		t.Fatalf("Size should be %d, got %d", size, d.Size)
	}
	expectedLoss := JSONFloat(sos + lambda*float64(size))
	if d.Loss != expectedLoss {
		t.Fatalf("Loss should be %v, got %v", expectedLoss, d.Loss)
	}
	if d.ErrorSqSum != JSONFloat(sos) {
		t.Fatalf("ErrorSqSum should be %v (unchanged), got %v", JSONFloat(sos), d.ErrorSqSum)
	}
}
