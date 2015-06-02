package tracker_test

import (
	"git.totdev.in/totv/mika/tracker"
	"testing"
)

var (
	gb = uint64(1000000000)
)

func TestCalculateBonus(t *testing.T) {
	b1 := tracker.CalculateBonus(3600*24*365, 10*gb, 2)
	if b1 != 219.03531322574617 {
		t.Errorf("Invalid min value: %f", b1)
	}
}
