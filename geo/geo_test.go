package geo_test

import (
	"git.totdev.in/totv/mika/geo"
	"testing"
)

func TestGetCoord(t *testing.T) {
	c := geo.GetCoord("12.34.56.78")
	if c.Lat != 38.000000 || c.Long != -97.000000 {
		t.Errorf("Invalid coord value: %f", c)
	}
}

func init() {
	geo.Setup("../geodb.dat")
}
