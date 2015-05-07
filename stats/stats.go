package stats

import (
	"git.totdev.in/totv/mika"
	"git.totdev.in/totv/mika/conf"
	"git.totdev.in/totv/mika/util"
	"github.com/influxdb/influxdb/client"
	"github.com/labstack/echo"
	"log"
	"net/url"
	"sync"
	"time"
)

const (
	EV_ANNOUNCE         = iota
	EV_ANNOUNCE_FAIL    = iota
	EV_SCRAPE           = iota
	EV_SCRAPE_FAIL      = iota
	EV_INVALID_PASSKEY  = iota
	EV_INVALID_INFOHASH = iota
	EV_INVALID_CLIENT   = iota
	EV_API              = iota
	EV_API_FAIL         = iota
	EV_STARTUP          = iota
	EV_SHUTDOWN         = iota
)

var (
	Counter      = make(chan int)
	StatCounts   *StatsCounter
	influxDB     *client.Client
	metric_names = map[int]string{
		EV_STARTUP:          "ev_startup",
		EV_ANNOUNCE:         "ev_announce",
		EV_ANNOUNCE_FAIL:    "ev_announce_fail",
		EV_API:              "ev_api",
		EV_API_FAIL:         "ev_api_fail",
		EV_SCRAPE:           "ev_scrape",
		EV_SCRAPE_FAIL:      "ev_scrape_fail",
		EV_INVALID_PASSKEY:  "ev_invalid_pk",
		EV_INVALID_INFOHASH: "ev_invalid_info_hash",
		EV_INVALID_CLIENT:   "ev_invalid_client",
	}
	pointChan     = make(chan client.Point)
	sampleSize    = 1000
	currentSample = 0
	pts           = make([]client.Point, sampleSize)
)

type StatsCounter struct {
	sync.RWMutex
	channel         chan int
	Requests        uint64
	RequestsFail    uint64
	Announce        uint64
	UniqueUsers     uint64
	AnnounceFail    uint64
	Scrape          uint64
	ScrapeFail      uint64
	InvalidPasskey  uint64
	InvalidInfohash uint64
	InvalidClient   uint64
	APIRequests     uint64
	APIRequestsFail uint64
}

func Setup(sample_size int) {
	if sample_size <= 0 {
		log.Fatalln("stats.Setup: InfluxWriteBuffer must be positive integer")
	}
	if sample_size < 100 {
		log.Println("[WARN] InfluxWriteBuffer value should generally be above 100. Currently:", sample_size)
	}
	sampleSize = sample_size
	StatCounts = NewStatCounter()
	go backgroundWriter()
}

func NewStatCounter() *StatsCounter {
	if conf.Config.InfluxDSN == "" {
		log.Println("[WARN] Invalid influx dsn defined")
	}
	u, err := url.Parse(conf.Config.InfluxDSN)
	if err != nil {
		log.Fatal(err)
	}

	conf := client.Config{
		URL:       *u,
		Username:  conf.Config.InfluxUser,
		Password:  conf.Config.InfluxPass,
		UserAgent: mika.VersionStr(),
	}

	con, err := client.NewClient(conf)
	if err != nil {
		log.Fatal(err)
	}

	dur, ver, err := con.Ping()
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("InfluxDB Happy as a Hippo! %v, %s", dur, ver)

	counter := &StatsCounter{channel: Counter}

	influxDB = con

	go counter.Counter()
	go counter.statPrinter()

	return counter
}

func RecordAnnounce(user_id uint64) {
	p := client.Point{
		Name: "announce",
		Tags: map[string]string{
			"version": mika.VersionStr(),
		},
		Fields: map[string]interface{}{
			"user_id": user_id,
		},
		Timestamp: time.Now(),
		Precision: "s",
	}
	addPoint(p)
}

func RecordScrape(user_id uint64) {
	p := client.Point{
		Name: "scrape",
		Tags: map[string]string{
			"version": mika.VersionStr(),
		},
		Fields: map[string]interface{}{
			"user_id": user_id,
		},
		Timestamp: time.Now(),
		Precision: "s",
	}
	addPoint(p)
}

// writePoints will commit the points as a batch to the backend influxdb instance
func writePoints() {
	bps := client.BatchPoints{
		Points:          pts,
		Database:        "mika",
		RetentionPolicy: "default",
	}
	_, err := influxDB.Write(bps)
	if err != nil {
		log.Fatal("Failed to write data points, discarding:", err)
	} else {
		util.Debug("Wrote samples out successfully")
	}
}

func addPoint(pt client.Point) {
	util.Debug("Adding point:", pt)
	pointChan <- pt
}

func (stats *StatsCounter) Counter() {
	for {
		v := <-stats.channel
		stats.Lock()
		switch v {
		case EV_API:
			stats.APIRequests++
			stats.Requests++
		case EV_API_FAIL:
			stats.APIRequestsFail++
		case EV_ANNOUNCE:
			stats.Announce++
			stats.Requests++
		case EV_ANNOUNCE_FAIL:
			stats.AnnounceFail++
		case EV_SCRAPE:
			stats.Scrape++
			stats.Requests++
		case EV_SCRAPE_FAIL:
			stats.ScrapeFail++
		case EV_INVALID_INFOHASH:
			stats.InvalidInfohash++
		case EV_INVALID_PASSKEY:
			stats.InvalidPasskey++
		case EV_INVALID_CLIENT:
			stats.InvalidClient++
		}
		stats.Unlock()
	}
}

// Records api requests
func StatsMW(h echo.HandlerFunc) echo.HandlerFunc {
	return func(c *echo.Context) error {
		Counter <- EV_API
		return nil
	}
}

// statPrinter will periodically print out basic stat lines to standard output
func (stats *StatsCounter) statPrinter() *time.Ticker {
	ticker := time.NewTicker(60 * time.Second)
	go func() {
		for range ticker.C {
			time.Sleep(60 * time.Second)
			stats.RLock()
			req_sec := stats.Requests / 60
			req_sec_api := stats.APIRequests / 60
			log.Printf("Ann: %d/%d Scr: %d/%d InvPK: %d InvIH: %d InvCL: %d Req/s: %d ApiReq/s: %d",
				stats.Announce, stats.AnnounceFail, stats.Scrape, stats.ScrapeFail,
				stats.InvalidPasskey, stats.InvalidInfohash, stats.InvalidClient, req_sec, req_sec_api)
			stats.RUnlock()
			stats.Lock()
			stats.Requests = 0
			stats.APIRequests = 0
			stats.Unlock()
		}
	}()
	return ticker
}

// backgroundWriter will write out the current pts values to influxdb. We reuse the
// existing memory everytime we flush the points out
func backgroundWriter() {
	for {
		select {
		case pt := <-pointChan:
			pts[currentSample] = pt
			currentSample++
			if currentSample == sampleSize {
				writePoints()
				currentSample = 0
			}
		}
	}
}
