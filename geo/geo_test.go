package geo

import (
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"math"
	"mika/util"
	"net"
	"os"
	"testing"
)

func TestGetCoord(t *testing.T) {
	db := New("../" + viper.GetString("geodb_path"))
	ip4 := db.GetLocation(net.ParseIP("12.34.56.78"))
	if math.Round(ip4.Location.Latitude) != 34.0 || math.Round(ip4.Location.Longitude) != -118.0 {
		t.Errorf("Invalid coord value: %f", ip4.Location)
	}
	ip6 := db.GetLocation(net.ParseIP("2600::")) // Sprint owned IP6
	if math.Round(ip6.Location.Latitude) != 38 || math.Round(ip6.Location.Longitude) != -98.0 {
		t.Errorf("Invalid coord value: %f", ip4.Location)
	}
}

func TestDistance(t *testing.T) {
	db := New("../" + viper.GetString("geodb_path"))
	a := LatLong{38.000000, -97.000000}
	b := LatLong{37.000000, -98.000000}
	distance := db.Distance(a, b)
	if distance != 141.0 {
		t.Errorf("Invalid distances: %f != %f", distance, 141.903347)
	}
}

func BenchmarkDistance(t *testing.B) {
	db := New("../" + viper.GetString("geodb_path"))
	a := LatLong{38.000000, -97.000000}
	b := LatLong{37.000000, -98.000000}
	for n := 0; n < t.N; n++ {
		_ = db.Distance(a, b)
	}
}

func TestDownloadDB(t *testing.T) {
	key := viper.GetString("geodb_api_key")
	tFile, err := ioutil.TempFile("", "prefix")
	if err != nil {
		t.Fail()
		return
	}
	defer os.Remove(tFile.Name())
	err2 := DownloadDB(tFile.Name(), key)
	assert.NoError(t, err2)
}

func init() {
	util.InitConfig("")
}
