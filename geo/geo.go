// Package geo provides the Lat/Long distance calculation functionality and GeoIP
// lookup functionality used to determine the best peers to use.
//
// NOTE this requires libgeoip to build, should
package geo

import (
	"bytes"
	"compress/gzip"
	"github.com/abh/geoip"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"math"
	"net/http"
	"os"
)

const (
	pi           = math.Pi
	twopi        = math.Pi * 2.0
	maxLoopCount = 20
	eps          = 1.0e-23
	kilometer    = 2
)

var (
	gi   *geoip.GeoIP
	elip Ellipsoid
)

type LatLong struct {
	Lat  float64
	Long float64
}

type Ellipsoid struct {
	Ellipse         ellipse
	Distance_units  int
	Distance_factor float64
}

type ellipse struct {
	Equatorial     float64
	Inv_flattening float64
}

// Distance computes the distances between two LatLong pairings
func (ll_a LatLong) Distance(ll_b LatLong) float64 {
	return math.Floor(elip.To(ll_a.Lat, ll_a.Long, ll_b.Lat, ll_b.Long))
}

// DownloadDB will fetch a new geoip database from maxmind and install it, uncompressed,
// into the configured geodb_path config file path usually defined in the configuration
// files.
func DownloadDB(geodb_path string) bool {
	db_url := "http://geolite.maxmind.com/download/geoip/database/GeoLiteCity.dat.gz"
	resp, err := http.Get(db_url)
	if err != nil {
		log.Errorln("Failed to downloaded geoip db:", err.Error())
		return false
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Errorln("Failed to read response body of geoip db:", err.Error())
		return false
	}

	data, err := readGzFile(body)
	if err != nil {
		log.Errorln("Failed to decode response body of geoip db:", err.Error())
		return false
	}
	err = ioutil.WriteFile(geodb_path, data, os.FileMode(0777))
	if err != nil {
		log.Errorln("Failed to write to db path:", err.Error())
		return false
	}
	log.Println("Successfully downloaded geoip db")
	return true
}

// Setup will initialized all needed data for the GeoIP lookups. This includes downloading
// a new copy of the database if none is found locally.
func Setup(geodb_path string) bool {
	elip = Ellipsoid{
		ellipse{6378137.0, 298.257223563}, // WGS84, because why not
		kilometer,
		1000.0,
	}
	if _, err := os.Stat(geodb_path); err != nil {
		if !DownloadDB(geodb_path) {
			return false
		}
	}
	geo, err := geoip.Open(geodb_path)
	if err != nil {
		log.Println("Could not open GeoIP database")
	} else {
		log.Println("Loaded GeoIP database succesfully")
	}
	gi = geo
	return true
}

// GetCoord will take a IP address and return the LatLong value associated with it from
// the maxmind geodb
func GetCoord(ip string) LatLong {
	record := gi.GetRecord(ip)
	if record == nil {
		return LatLong{0.0, 0.0}
	}
	return LatLong{Lat: float64(record.Latitude), Long: float64(record.Longitude)}
}

// readGzFile will uncompress a gzip'd byte slice and return the resulting bytes
func readGzFile(file_data []byte) ([]byte, error) {
	fz, err := gzip.NewReader(bytes.NewReader(file_data))
	if err != nil {
		return nil, err
	}
	defer fz.Close()

	s, err := ioutil.ReadAll(fz)
	if err != nil {
		return nil, err
	}
	return s, nil
}

// deg2rad converts degrees to radians
func deg2rad(d float64) (r float64) {
	return d * pi / 180.0
}

// To will return the distance to another lat/long pairing.
func (ellipsoid Ellipsoid) To(lat1, lon1, lat2, lon2 float64) (distance float64) {
	distance, _ = ellipsoid.calculateBearing(
		deg2rad(lat1), deg2rad(lon1),
		deg2rad(lat2), deg2rad(lon2),
	)
	distance /= ellipsoid.Distance_factor
	return distance
}

// calculateBearing will take 2 lat/long pairs and compute the distance and bearing of the
// values.
func (ellipsoid Ellipsoid) calculateBearing(lat1, lon1, lat2, lon2 float64) (distance, bearing float64) {
	a := ellipsoid.Ellipse.Equatorial
	f := 1 / ellipsoid.Ellipse.Inv_flattening

	if lon1 < 0 {
		lon1 += twopi
	}
	if lon2 < 0 {
		lon2 += twopi
	}

	r := 1.0 - f
	c_lat1 := math.Cos(lat1)
	if c_lat1 == 0 {
		log.Warningln("Division by Zero in ellipsoid.go.")
		return 0.0, 0.0
	}
	c_lat2 := math.Cos(lat2)
	if c_lat2 == 0 {
		log.Warningln("Division by Zero in ellipsoid.go.")
		return 0.0, 0.0
	}
	tu1 := r * math.Sin(lat1) / c_lat1
	tu2 := r * math.Sin(lat2) / c_lat2
	cu1 := 1.0 / (math.Sqrt((tu1 * tu1) + 1.0))
	su1 := cu1 * tu1
	cu2 := 1.0 / (math.Sqrt((tu2 * tu2) + 1.0))
	s := cu1 * cu2
	baz := s * tu2
	faz := baz * tu1
	d_lon := lon2 - lon1

	x := d_lon
	cnt := 0

	var c2a, c, cx, cy, cz, d, del, e, sx, sy, y float64
	// This originally was a do-while loop. Exit condition is at end of loop.
	for true {
		sx = math.Sin(x)
		cx = math.Cos(x)
		tu1 = cu2 * sx
		tu2 = baz - (su1 * cu2 * cx)

		sy = math.Sqrt(tu1*tu1 + tu2*tu2)
		cy = s*cx + faz
		y = math.Atan2(sy, cy)
		var sa float64
		if sy == 0.0 {
			sa = 1.0
		} else {
			sa = (s * sx) / sy
		}

		c2a = 1.0 - (sa * sa)
		cz = faz + faz
		if c2a > 0.0 {
			cz = ((-cz) / c2a) + cy
		}
		e = (2.0 * cz * cz) - 1.0
		c = (((((-3.0 * c2a) + 4.0) * f) + 4.0) * c2a * f) / 16.0
		d = x
		x = ((e*cy*c+cz)*sy*c + y) * sa
		x = (1.0-c)*x*f + d_lon
		del = d - x

		if math.Abs(del) <= eps {
			break
		}
		cnt++
		if cnt > maxLoopCount {
			break
		}
	}

	faz = math.Atan2(tu1, tu2)
	baz = math.Atan2(cu1*sx, (baz*cx-su1*cu2)) + pi
	x = math.Sqrt(((1.0/(r*r))-1.0)*c2a+1.0) + 1.0
	x = (x - 2.0) / x
	c = 1.0 - x
	c = ((x*x)/4.0 + 1.0) / c
	d = ((0.375 * x * x) - 1.0) * x
	x = e * cy

	s = 1.0 - e - e
	s = ((((((((sy * sy * 4.0) - 3.0) * s * cz * d / 6.0) - x) * d / 4.0) + cz) * sy * d) + y) * c * a * r

	// adjust azimuth to (0,360) or (-180,180) as specified
	if faz < -(pi) {
		faz += twopi
	}
	if faz >= pi {
		faz -= twopi
	}

	distance, bearing = s, faz
	return distance, bearing
}
