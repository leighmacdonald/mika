package geo_test

import (
	"git.totdev.in/totv/mika/geo"
	"testing"
)

var (
	result float64
)

func TestGetCoord(t *testing.T) {
	c := geo.GetCoord("12.34.56.78")
	if c.Lat != 38.000000 || c.Long != -97.000000 {
		t.Errorf("Invalid coord value: %f", c)
	}
}

func TestDistance(t *testing.T) {
	a := geo.LatLong{38.000000, -97.000000}
	b := geo.LatLong{37.000000, -98.000000}
	distance := a.Distance(b)
	if distance != 141.0 {
		t.Errorf("Invalid distances: %f != %f", distance, 141.903347)
	}
}
func BenchmarkDistance(t *testing.B) {
	a := geo.LatLong{38.000000, -97.000000}
	b := geo.LatLong{37.000000, -98.000000}
	var r float64
	for n := 0; n < t.N; n++ {
		r = a.Distance(b)
	}
	result = r
}

func init() {
	geo.Setup("../geodb.dat")
}
