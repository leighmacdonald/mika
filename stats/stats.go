// Package stats provides functionality for recording and displaying metrics
package stats

import (
	log "github.com/Sirupsen/logrus"
	"github.com/rcrowley/go-metrics"
	//	golog "log"
	"math/rand"
	"net"
	//	"os"
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
	registry metrics.Registry
)

type StatsCounter struct {
	sync.RWMutex
	channel           chan int
	Requests          metrics.Counter
	RequestsFail      metrics.Counter
	Announce          metrics.Counter
	AnnouncePerMin    metrics.GaugeFloat64
	UniqueUsers       metrics.Counter
	AnnounceFail      metrics.Counter
	Scrape            metrics.Counter
	ScrapePerMin      metrics.GaugeFloat64
	ScrapeFail        metrics.Counter
	InvalidPasskey    metrics.Counter
	InvalidInfohash   metrics.Counter
	InvalidClient     metrics.Counter
	APIRequests       metrics.Counter
	APIRequestsPerMin metrics.GaugeFloat64
	APIRequestsFail   metrics.Counter
	AnnounceUserIDS   []uint64
	ScrapeUserIDS     []uint64
}

func Setup(host, user, pass, database string, interval time.Duration) {

	registry = metrics.NewRegistry()

	if interval <= 0 {
		log.Fatal("stats.Setup: InfluxWriteBuffer must be positive integer")
	}
	if interval < 100 {
		log.Warn("InfluxWriteBuffer value should generally be above 100. Currently:", interval)
	}

	StatCounts = NewStatCounter(registry)

	addr, _ := net.ResolveTCPAddr("tcp", "localhost:2003")
	go metrics.Graphite(metrics.DefaultRegistry, 10e9, "metrics", addr)

	//	go influxdb.Influxdb(metrics.DefaultRegistry, interval, &influxdb.Config{
	//		Host:     host,
	//		Database: database,
	//		Username: user,
	//		Password: pass,
	//	})
	//go metrics.Log(registry, interval, golog.New(os.Stdout, "metrics: ", golog.Lmicroseconds))
}

func NewStatCounter(registry metrics.Registry) *StatsCounter {
	counter := &StatsCounter{
		channel:           Counter,
		Requests:          metrics.NewCounter(),
		RequestsFail:      metrics.NewCounter(),
		Announce:          metrics.NewCounter(),
		AnnouncePerMin:    metrics.NewGaugeFloat64(),
		UniqueUsers:       metrics.NewCounter(),
		AnnounceFail:      metrics.NewCounter(),
		Scrape:            metrics.NewCounter(),
		ScrapePerMin:      metrics.NewGaugeFloat64(),
		ScrapeFail:        metrics.NewCounter(),
		InvalidPasskey:    metrics.NewCounter(),
		InvalidInfohash:   metrics.NewCounter(),
		InvalidClient:     metrics.NewCounter(),
		APIRequests:       metrics.NewCounter(),
		APIRequestsPerMin: metrics.NewGaugeFloat64(),
		APIRequestsFail:   metrics.NewCounter(),
	}
	registry.Register("req_ok", counter.Requests)
	registry.Register("req_err", counter.RequestsFail)
	registry.Register("users_permin", counter.UniqueUsers)
	registry.Register("announce", counter.Announce)
	registry.Register("announce_permin", counter.AnnouncePerMin)
	registry.Register("announce_err", counter.AnnounceFail)
	registry.Register("scrape_ok", counter.Scrape)
	registry.Register("scrape_permin", counter.ScrapePerMin)
	registry.Register("scrape_err", counter.ScrapeFail)
	registry.Register("invalid_passkey", counter.InvalidPasskey)
	registry.Register("invalid_infohash", counter.InvalidInfohash)
	registry.Register("invalid_client", counter.InvalidClient)
	registry.Register("api_ok", counter.APIRequests)
	registry.Register("api_permin", counter.APIRequestsPerMin)
	registry.Register("api_err", counter.APIRequestsFail)

	metrics.RegisterRuntimeMemStats(registry)
	go metrics.CaptureRuntimeMemStats(registry, 5e9)

	go counter.Counter()
	go counter.statPrinter()

	return counter
}

func (stats *StatsCounter) Counter() {
	for {
		v := <-stats.channel
		switch v {
		case EV_API:
			stats.APIRequests.Inc(1)
			stats.APIRequestsPerMin.Update(rand.Float64())
			stats.Requests.Inc(1)
		case EV_API_FAIL:
			stats.APIRequestsFail.Inc(1)
		case EV_ANNOUNCE:
			stats.Announce.Inc(1)
			stats.AnnouncePerMin.Update(rand.Float64())
			stats.Requests.Inc(1)
		case EV_ANNOUNCE_FAIL:
			stats.AnnounceFail.Inc(1)
		case EV_SCRAPE:
			stats.Scrape.Inc(1)
			stats.ScrapePerMin.Update(rand.Float64())
			stats.Requests.Inc(1)
		case EV_SCRAPE_FAIL:
			stats.ScrapeFail.Inc(1)
		case EV_INVALID_INFOHASH:
			stats.InvalidInfohash.Inc(1)
		case EV_INVALID_PASSKEY:
			stats.InvalidPasskey.Inc(1)
		case EV_INVALID_CLIENT:
			stats.InvalidClient.Inc(1)
		}
	}
}

// statPrinter will periodically print out basic stat lines to standard output
func (stats *StatsCounter) statPrinter() *time.Ticker {
	ticker := time.NewTicker(60 * time.Second)
	go func() {
		for range ticker.C {
			stats.RLock()
			req_sec := stats.Announce.Count() / 60
			req_sec_api := stats.APIRequests.Count() / 60
			log.Printf("Ann: %d/%d Scr: %d/%d InvPK: %d InvIH: %d InvCL: %d Req/s: %d ApiReq/s: %d",
				stats.Announce.Count(), stats.AnnounceFail.Count(), stats.Scrape.Count(), stats.ScrapeFail.Count(),
				stats.InvalidPasskey.Count(), stats.InvalidInfohash.Count(), stats.InvalidClient.Count(), req_sec, req_sec_api)

			//			RecordAnnounces(stats.AnnouncePerMin)
			//			RecordScrapes(stats.ScrapePerMin)

			stats.RUnlock()

			stats.APIRequestsPerMin.Update(0)
			stats.AnnouncePerMin.Update(0)
			stats.ScrapePerMin.Update(0)

		}
	}()
	return ticker
}
