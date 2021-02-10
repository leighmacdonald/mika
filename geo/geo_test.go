package geo

import (
	"github.com/leighmacdonald/mika/config"
	"github.com/leighmacdonald/mika/util"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"math"
	"net"
	"os"
	"path/filepath"
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

func isSkipped() bool {
	return config.GeoDB.APIKey == "" || !util.Exists(filepath.Join(config.GeoDB.Path, geoDatabaseLocationFile))
}
func TestGetLocation(t *testing.T) {
	if isSkipped() {
		t.SkipNow()
	}
	db, err := New(config.GeoDB.Path)
	require.NoError(t, err, "Failed to open database")
	defer func() { db.Close() }()
	ip4 := db.GetLocation(net.ParseIP("45.136.241.10"))
	require.Equal(t, uint32(16509), ip4.ASN)
	require.Equal(t, "AMAZON-02", ip4.AS)
	require.Equal(t, "US", ip4.ISOCode)
	if math.Round(ip4.LatLong.Latitude) != 46 || math.Round(ip4.LatLong.Longitude) != -123 {
		t.Errorf("Invalid coord value: %f", ip4.LatLong)
	}
	ip6 := db.GetLocation(net.ParseIP("2001:4860:4860::6464")) // Sprint owned IP6
	if math.Round(ip6.LatLong.Latitude) != 37 || math.Round(ip6.LatLong.Longitude) != -122 {
		t.Errorf("Invalid coord value: %f", ip6.LatLong)
	}
	require.Equal(t, "US", ip6.ISOCode)

}

func TestDistance(t *testing.T) {
	if isSkipped() {
		t.SkipNow()
	}
	fp := util.FindFile(config.GeoDB.Path)
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
	db, _ := New(config.GeoDB.Path)
	defer func() { db.Close() }()
	a := LatLong{38.000000, -97.000000}
	b := LatLong{37.000000, -98.000000}
	for n := 0; n < t.N; n++ {
		_ = db.distance(a, b)
	}
}

func TestDownloadDB(t *testing.T) {
	key := config.GeoDB.APIKey
	if key == "" {
		t.SkipNow()
	}
	p := util.FindFile(config.GeoDB.Path)
	for _, fp := range []string{geoDatabaseLocationFile, geoDatabaseASNFile4, geoDatabaseASNFile6} {
		fullPath := filepath.Join(p, fp)
		if util.Exists(fullPath) {
			file, err := os.Stat(fullPath)
			if err != nil {
				log.Fatalf("failed to stat file: %s", err)
			}
			if time.Since(file.ModTime()).Hours() >= 6 {
				if err := os.Remove(fullPath); err != nil {
					t.Fatalf("Could not remove geo file %s: %s", fullPath, err)
				}
			} else {
				t.Skipf("Skipping download test, file age too new")
			}
		}
	}

	err2 := DownloadDB(p, key)
	require.NoError(t, err2)
	_, err3 := New(p)
	require.NoError(t, err3, "failed to verify downloaded mmdb")
}

func TestMain(m *testing.M) {
	_ = config.Read("mika_testing")
	os.Exit(m.Run())
}
