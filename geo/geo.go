// Package geo provides the Lat/Long distance calculation functionality and GeoIP
// lookup functionality used to determine the best peers to use.
package geo

import (
	"fmt"
	"github.com/leighmacdonald/mika/util"
	"github.com/oschwald/maxminddb-golang"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"math"
	"net"
	"net/http"
	"os"
	"strings"
)

const (
	pi             = math.Pi
	twoPi          = math.Pi * 2.0
	maxLoopCount   = 20
	eps            = 1.0e-23
	kilometer      = 2
	geoDownloadURL = "https://download.maxmind.com/app/geoip_download?edition_id=GeoLite2-City&license_key=%s&suffix=tar.gz"
)

// City provides the country and lat/long
type City struct {
	Country  Country `maxminddb:"country"`
	Location LatLong `maxminddb:"location"`
}

// Country is the ISO country code
type Country struct {
	ISOCode string `maxminddb:"iso_code"`
}

// LatLong provides a container and some helper functions for location data
type LatLong struct {
	Latitude  float64 `maxminddb:"latitude"`
	Longitude float64 `maxminddb:"longitude"`
}

// String returns a comma separated lat long pair string
func (ll LatLong) String() string {
	return fmt.Sprintf("%f,%f", ll.Latitude, ll.Longitude)
}

// LatLongFromString will return a LatLong from a string formatted like N,-N
func LatLongFromString(s string) LatLong {
	p := strings.Split(s, ",")
	if len(p) != 2 {
		log.Warnf("Received invalid lat long string: %s", s)
		return LatLong{0, 0}
	}
	return LatLong{
		util.StringToFloat64(p[0], 0),
		util.StringToFloat64(p[1], 0),
	}
}

type ellipsoid struct {
	Ellipse        ellipse
	DistanceUnits  int
	DistanceFactor float64
}

type ellipse struct {
	Equatorial    float64
	InvFlattening float64
}

// distance computes the distances between two LatLong pairings
func (db *DB) distance(llA LatLong, llB LatLong) float64 {
	return math.Floor(db.ellipsoid.to(llA.Latitude, llA.Longitude, llB.Latitude, llB.Longitude))
}

// DownloadDB will fetch a new geoip database from maxmind and install it, uncompressed,
// into the configured geodb_path config file path usually defined in the configuration
// files.
func DownloadDB(outputPath string, apiKey string) error {
	if apiKey == "" {
		return errors.New("invalid maxmind api key")
	}
	url := fmt.Sprintf(geoDownloadURL, apiKey)
	resp, err := http.Get(url)
	if err != nil {
		return errors.Wrap(err, "Failed to downloaded geoip db")
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Error("Failed to close response body for geodb download")
		}
	}()
	s, err := os.OpenFile(outputPath, os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	return extractTarGz(resp.Body, s)
}

// deg2rad converts degrees to radians
func deg2rad(d float64) float64 {
	return d * pi / 180.0
}

// to will return the distance to another lat/long pairing.
func (ellipsoid ellipsoid) to(lat1, lon1, lat2, lon2 float64) (distance float64) {
	distance, _ = ellipsoid.calculateBearing(
		deg2rad(lat1), deg2rad(lon1),
		deg2rad(lat2), deg2rad(lon2),
	)
	distance /= ellipsoid.DistanceFactor
	return distance
}

// calculateBearing will take 2 lat/long pairs and compute the distance and bearing of the
// values.
func (ellipsoid ellipsoid) calculateBearing(lat1, lon1, lat2, lon2 float64) (distance, bearing float64) {
	a := ellipsoid.Ellipse.Equatorial
	f := 1 / ellipsoid.Ellipse.InvFlattening

	if lon1 < 0 {
		lon1 += twoPi
	}
	if lon2 < 0 {
		lon2 += twoPi
	}

	r := 1.0 - f
	cLat1 := math.Cos(lat1)
	if cLat1 == 0 {
		log.Warningln("Division by Zero in geo.go.")
		return 0.0, 0.0
	}
	cLat2 := math.Cos(lat2)
	if cLat2 == 0 {
		log.Warningln("Division by Zero in geo.go.")
		return 0.0, 0.0
	}
	tu1 := r * math.Sin(lat1) / cLat1
	tu2 := r * math.Sin(lat2) / cLat2
	cu1 := 1.0 / (math.Sqrt((tu1 * tu1) + 1.0))
	su1 := cu1 * tu1
	cu2 := 1.0 / (math.Sqrt((tu2 * tu2) + 1.0))
	s := cu1 * cu2
	baz := s * tu2
	faz := baz * tu1
	dLon := lon2 - lon1

	x := dLon
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
		x = (1.0-c)*x*f + dLon
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
	// baz = math.Atan2(cu1*sx, baz*cx-su1*cu2) + pi
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
		faz += twoPi
	}
	if faz >= pi {
		faz -= twoPi
	}
	return s, faz
}

// DB handles opening and querying from the maxmind Cities geo memory mapped database (.mmdb) file.
type DB struct {
	db        *maxminddb.Reader
	ellipsoid ellipsoid
}

// New opens the .mmdb file for querying and sets up the ellipsoid configuration for more accurate
// geo queries
func New(path string, verify bool) *DB {
	db, err := maxminddb.Open(path)
	if err != nil {
		log.Fatal(err)
	}
	if verify {
		if err := db.Verify(); err != nil {
			log.Fatalf("Failed to validate maxmind geodb file: %s", err)
		}
	}
	return &DB{
		db: db,
		ellipsoid: ellipsoid{
			ellipse{6378137.0, 298.257223563}, // WGS84, because why not
			kilometer,
			1000.0,
		},
	}
}

// Close close the underlying memory mapped file
func (db *DB) Close() error {
	return db.db.Close()
}

// GetLocation returns the geo location of the input IP addr
func (db *DB) GetLocation(ip net.IP) City {
	var record City
	err := db.db.Lookup(ip, &record)
	if err != nil {
		log.Fatal(err)
	}
	return record
}
