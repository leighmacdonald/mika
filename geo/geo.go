// Package geo provides the Lat/Long distance calculation functionality and GeoIP
// lookup functionality used to determine the best peers to use.
package geo

import (
	"database/sql/driver"
	"encoding/csv"
	"fmt"
	"github.com/ip2location/ip2location-go"
	"github.com/leighmacdonald/mika/util"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"io"
	"io/ioutil"
	"math"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

const (
	pi                      = math.Pi
	twoPi                   = math.Pi * 2.0
	maxLoopCount            = 20
	eps                     = 1.0e-23
	kilometer               = 2
	geoDownloadURL          = "https://www.ip2location.com/download/?token=%s&file=%s"
	geoDatabaseASN4         = "DBASNLITE"
	geoDatabaseASN6         = "DBASNLITEIPV6"
	geoDatabaseLocation     = "DB5LITEBINIPV6"
	geoDatabaseASNFile4     = "IP2LOCATION-LITE-ASN.CSV"
	geoDatabaseASNFile6     = "IP2LOCATION-LITE-ASN.IPV6.CSV"
	geoDatabaseLocationFile = "IP2LOCATION-LITE-DB5.IPV6.BIN"
)

// Provider defines our interface for querying geo location data stores
type Provider interface {
	GetLocation(ip net.IP) Location
	Close()
}

// DummyProvider is used when we dont want to use this feature. It will always return 0, 0
type DummyProvider struct{}

// DownloadDB does nothing for the dummy provider
func (d *DummyProvider) DownloadDB(_ string, _ string) error {
	return nil
}

// Close does nothing for the dummy provider
func (d *DummyProvider) Close() {}

// GetLocation will always return 0, 0 coordinates
func (d *DummyProvider) GetLocation(_ net.IP) Location {
	return defaultLocation()
}

type LatLong struct {
	Latitude  float64
	Longitude float64
}

// Location provides a container and some helper functions for location data
type Location struct {
	ISOCode string
	LatLong LatLong
	// Autonomous system number (ASN)
	ASN uint32
	// Autonomous system (AS) name
	AS string
}

func defaultLocation() Location {
	return Location{
		ISOCode: "",
		LatLong: LatLong{Latitude: 0.0, Longitude: 0.0},
		ASN:     0,
		AS:      "",
	}
}

// Value implements the driver.Valuer interface for our custom type
func (ll *LatLong) Value() (driver.Value, error) {
	return fmt.Sprintf("POINT(%s)", ll.String()), nil
}

// Scan implements the sql.Scanner interface for conversion to our custom type
func (ll *LatLong) Scan(v interface{}) error {
	// Should be more strictly to check this type.
	llStrB, ok := v.([]byte)
	if !ok {
		return errors.New("failed to convert value to string")
	}
	llStr := string(llStrB)
	ss := strings.Split(strings.Replace(llStr, ")", "", 1), "(")
	if len(ss) != 2 {
		return errors.New("Failed to parse location")
	}
	pcs := strings.Split(ss[1], " ")
	if len(pcs) != 2 {
		return errors.New("Failed to parse location")
	}
	lon, err := strconv.ParseFloat(pcs[0], 64)
	if err != nil {
		return errors.New("Failed to parse longitude")
	}
	lat, err2 := strconv.ParseFloat(pcs[1], 64)
	if err2 != nil {
		return errors.New("Failed to parse latitude")
	}
	ll.Longitude = lon
	ll.Latitude = lat
	return nil
}

// String returns a comma separated lat long pair string
func (ll LatLong) String() string {
	return fmt.Sprintf("%f %f", ll.Latitude, ll.Longitude)
}

// LatLongFromString will return a LatLong from a string formatted like N,-N
func LatLongFromString(s string) LatLong {
	p := strings.Split(s, " ")
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
	type dlParam struct {
		dbName   string
		fileName string
	}
	dl := func(u dlParam) error {
		resp, err := http.Get(fmt.Sprintf(geoDownloadURL, apiKey, u.dbName))
		if err != nil {
			return errors.Wrap(err, "Failed to downloaded geoip db")
		}
		b, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		if err := resp.Body.Close(); err != nil {
			log.Error("Failed to close response body for geodb download")
		}

		err2 := extractZip(b, outputPath, u.fileName)

		return err2
	}
	if apiKey == "" {
		return errors.New("invalid maxmind api key")
	}
	var exitErr error
	var wg sync.WaitGroup
	for _, u := range []dlParam{
		{dbName: geoDatabaseASN4, fileName: geoDatabaseASNFile4},
		{dbName: geoDatabaseASN6, fileName: geoDatabaseASNFile6},
		{dbName: geoDatabaseLocation, fileName: geoDatabaseLocationFile},
	} {
		wg.Add(1)
		req := u
		go func() {
			if err := dl(req); err != nil {
				log.Errorf("Failed to download geo database: %s", err.Error())
			}
			wg.Done()
		}()
	}
	wg.Wait()
	return exitErr
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
	for {
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
	sync.RWMutex
	db        *ip2location.DB
	ellipsoid ellipsoid
	asn4      []asnRecord
	asn6      []asnRecord
}

type asnRecord struct {
	net *net.IPNet
	ASN uint32
	AS  string
}

// New opens the .mmdb file for querying and sets up the ellipsoid configuration for more accurate
// geo queries
func New(path string) (*DB, error) {
	db, err := ip2location.OpenDB(filepath.Join(path, geoDatabaseLocationFile))
	if err != nil {
		return nil, err
	}
	var (
		records4 []asnRecord
		records6 []asnRecord
	)
	for i, asnFileName := range []string{geoDatabaseASNFile4, geoDatabaseASNFile6} {
		asnFile, err1 := os.Open(filepath.Join(path, asnFileName))
		if err1 != nil {
			return nil, err1
		}
		reader := csv.NewReader(asnFile)
		for {
			row, err2 := reader.Read()
			if err2 == io.EOF {
				break
			}
			if err2 != nil {
				log.Fatalf("Failed to read csv row: %s", err2.Error())
			}
			_, cidr, err2 := net.ParseCIDR(row[2])
			if err2 != nil {
				continue
			}
			asNum, err := strconv.ParseUint(row[3], 10, 32)
			if err != nil {
				continue
			}
			if i == 0 {
				records4 = append(records4, asnRecord{net: cidr, ASN: uint32(asNum), AS: row[4]})
			} else {
				records6 = append(records6, asnRecord{net: cidr, ASN: uint32(asNum), AS: row[4]})
			}
		}
	}
	return &DB{
		RWMutex: sync.RWMutex{},
		db:      db,
		ellipsoid: ellipsoid{
			ellipse{6378137.0, 298.257223563}, // WGS84, because why not
			kilometer,
			1000.0,
		},
		asn4: records4,
		asn6: records6,
	}, nil
}

// Close close the underlying memory mapped file
func (db *DB) Close() {
	db.db.Close()
}

// GetLocation returns the geo location of the input IP addr
func (db *DB) GetLocation(ip net.IP) Location {
	const invalidErr = "Invalid IP address."
	res, err := db.db.Get_all(ip.String())
	if err != nil || res.Country_short == invalidErr {
		log.Errorf("Failed to get location for: %s", ip.String())
		return defaultLocation()
	}
	var asnRec asnRecord
	db.RLock()
	for _, r := range db.asn4 {
		if r.net.Contains(ip) {
			asnRec = r
			break
		}
	}
	db.RUnlock()
	return Location{
		ISOCode: res.Country_short,
		LatLong: LatLong{
			float64(res.Latitude), float64(res.Longitude),
		},
		ASN: asnRec.ASN,
		AS:  asnRec.AS,
	}
}
