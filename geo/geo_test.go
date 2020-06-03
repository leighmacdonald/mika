package geo

import (
	"github.com/leighmacdonald/mika/config"
	"github.com/leighmacdonald/mika/util"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"math"
	"net"
	"os"
	"testing"
	"time"
)

func TestLatLong(t *testing.T) {
	ll := LatLong{-114, 58}
	var ll2 LatLong
	var ll3 LatLong
	v, _ := ll.Value()
	require.Equal(t, "POINT(-114.000000 58.000000)", v)
	require.NoError(t, ll2.Scan([]byte("POINT(58.000000 -114.000000)")))
	require.Equal(t, ll, ll2)
	require.Error(t, ll3.Scan(1))
	require.Error(t, ll3.Scan([]byte("POINT(58.000000 -114.000000 100)")))
	require.Error(t, ll3.Scan([]byte("POINT(x -114.000000)")))
	require.Error(t, ll3.Scan([]byte("POINT(58.000000 x)")))
	require.Equal(t, LatLong{0, 0}, LatLongFromString("1 2 x"))
}

func TestGetLocation(t *testing.T) {
	if config.GetString(config.GeodbAPIKey) == "" {
		t.SkipNow()
	}
	db, err := New(config.GetString(config.GeodbPath))
	require.NoError(t, err, "Failed to open database")
	defer func() { db.Close() }()
	ip4 := db.GetLocation(net.ParseIP("45.136.241.10"))
	require.Equal(t, uint32(16509), ip4.ASN)
	require.Equal(t, "AMAZON-02", ip4.AS)
	require.Equal(t, "IL", ip4.ISOCode)
	if math.Round(ip4.LatLong.Latitude) != 32 || math.Round(ip4.LatLong.Longitude) != 35 {
		t.Errorf("Invalid coord value: %f", ip4.LatLong)
	}
	ip6 := db.GetLocation(net.ParseIP("2001:4860:4860::6464")) // Sprint owned IP6
	if math.Round(ip6.LatLong.Latitude) != 37 || math.Round(ip6.LatLong.Longitude) != -122 {
		t.Errorf("Invalid coord value: %f", ip4.LatLong)
	}
	require.Equal(t, "US", ip6.ISOCode)

}

func TestDistance(t *testing.T) {
	if config.GetString(config.GeodbAPIKey) == "" {
		t.SkipNow()
	}
	fp := util.FindFile(config.GetString(config.GeodbPath))
	if !util.Exists(fp) {
		t.Skipf("Invalid geodb directory")
	}
	db, _ := New(fp)
	defer func() { db.Close() }()
	a := LatLong{38.000000, -97.000000}
	b := LatLong{37.000000, -98.000000}
	distance := db.distance(a, b)
	if distance != 141.0 {
		t.Errorf("Invalid distances: %f != %f", distance, 141.903347)
	}
}

func BenchmarkDistance(t *testing.B) {
	db, _ := New(config.GetString(config.GeodbPath))
	defer func() { db.Close() }()
	a := LatLong{38.000000, -97.000000}
	b := LatLong{37.000000, -98.000000}
	for n := 0; n < t.N; n++ {
		_ = db.distance(a, b)
	}
}

func TestDownloadDB(t *testing.T) {
	key := config.GetString(config.GeodbAPIKey)
	if key == "" {
		t.SkipNow()
	}
	p := util.FindFile(config.GetString(config.GeodbPath))
	if util.Exists(p) {
		file, err := os.Stat(p)
		if err != nil {
			log.Fatalf("failed to stat file: %s", err)
		}
		if time.Since(file.ModTime()).Hours() >= 6 {
			if err := os.Remove(p); err != nil {
				t.Fatalf("Could not remove mmdb file: %s", err)
			}
		} else {
			t.Skipf("Skipping download test, file age too new")
		}
	}

	err2 := DownloadDB(p, key)
	require.NoError(t, err2)
	_, err3 := New(p)
	require.NoError(t, err3, "failed to verify downloaded mmdb")
}

func TestMain(m *testing.M) {
	if err := config.Read("mika_testing"); err != nil {
		log.Panicf("Failed to load test config: %s", err.Error())
	}
	os.Exit(m.Run())
}
