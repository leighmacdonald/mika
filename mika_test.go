package main

import (
	"testing"
)

func TestUMin(t *testing.T) {
	a := uint64(1)
	b := uint64(2)
	v := UMin(a, b)
	if v != a {
		t.Error("Invalid min value")
	}
}

func TestUMax(t *testing.T) {
	a := uint64(1)
	b := uint64(2)
	v := UMax(a, b)
	if v != b {
		t.Error("Invalid min value")
	}
}
