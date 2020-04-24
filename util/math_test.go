package util

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

func TestEstSpeed(t *testing.T) {
	a := EstSpeed(1000, 2000, 0)
	if a != 0.0 {
		t.Errorf("A: Invalid value %f", a)
	}

	b := EstSpeed(2000, 1000, 0)
	if b != 0.0 {
		t.Errorf("B: Invalid value %f", b)
	}

	c := EstSpeed(-1, 1, 0)
	if c != 0.0 {
		t.Errorf("C: Invalid value %f", c)
	}

	d := EstSpeed(1, -1, 0)
	if d != 0.0 {
		t.Errorf("D: Invalid value %f", d)
	}

	e := EstSpeed(0, 0, 1)
	if e != 0.0 {
		t.Errorf("E: Invalid value %f", e)
	}

	ok := EstSpeed(1000, 2000, 100000000)
	if ok != 100000.0 {
		t.Errorf("E: Invalid value %f", ok)
	}
}
