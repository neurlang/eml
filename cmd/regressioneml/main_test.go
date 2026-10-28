package main

import "testing"

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
