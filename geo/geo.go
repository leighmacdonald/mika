package geo

import (
	"bytes"
	"compress/gzip"
	log "github.com/Sirupsen/logrus"
	"github.com/abh/geoip"
	"io/ioutil"
	"net/http"
	"os"
)

var (
	gi *geoip.GeoIP
)

type LatLong struct {
	Lat  float32
	Long float32
}

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

func Setup(geodb_path string) bool {
	if _, err := os.Stat(geodb_path); err != nil {
		if !DownloadDB(geodb_path) {
			return false
		}
	}

	geo, err := geoip.Open(geodb_path)
	if err != nil {
		log.Println("Could not open GeoIP database")
	} else {
		log.Println("Loaded GeoIP database")
	}
	gi = geo
	return true
}

func GetCoord(ip string) LatLong {
	record := gi.GetRecord(ip)
	if record == nil {
		return LatLong{0.0, 0.0}
	}
	return LatLong{Lat: record.Latitude, Long: record.Longitude}
}

func readGzFile(filedata []byte) ([]byte, error) {
	fz, err := gzip.NewReader(bytes.NewReader(filedata))
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
