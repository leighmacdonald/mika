// Package stats provides functionality for recording and displaying metrics
package stats

import (
	log "github.com/Sirupsen/logrus"
	"github.com/rcrowley/go-metrics"
	"net"
	"sync"
	"time"
)

const (
	EV_REQUEST_OK       = iota
	EV_REQUEST_ERR      = iota
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
	EV_ERROR            = iota
)

var (
	Counter      = make(chan int)
	StatCounts   *StatsCounter
	metric_names = map[int]string{
		EV_REQUEST_OK:       "req_ok",
		EV_REQUEST_ERR:      "req_err",
		EV_STARTUP:          "startup",
		EV_ANNOUNCE:         "announce",
		EV_ANNOUNCE_FAIL:    "announce_err",
		EV_API:              "api_ok",
		EV_API_FAIL:         "api_err",
		EV_SCRAPE:           "scrape_ok",
		EV_SCRAPE_FAIL:      "scrape_err",
		EV_INVALID_PASSKEY:  "invalid_passkey",
		EV_INVALID_INFOHASH: "invalid_infohash",
		EV_INVALID_CLIENT:   "invalid_client",
		EV_ERROR:            "error",
	}
	registry metrics.Registry
)

type StatsCounter struct {
	sync.RWMutex
	channel         chan int
	Errors          metrics.Counter
	Requests        metrics.Counter
	RequestsFail    metrics.Counter
	Announce        metrics.Counter
	UniqueUsers     metrics.Counter
	AnnounceFail    metrics.Counter
	Scrape          metrics.Counter
	ScrapeFail      metrics.Counter
	InvalidPasskey  metrics.Counter
	InvalidInfohash metrics.Counter
	InvalidClient   metrics.Counter
	APIRequests     metrics.Counter
	APIRequestsFail metrics.Counter
	AnnounceUserIDS []uint64
	ScrapeUserIDS   []uint64
}

func Setup(metrics_dsn string) {
	address, _ := net.ResolveTCPAddr("tcp", metrics_dsn)
	go metrics.Graphite(metrics.DefaultRegistry, 10e9, "tracker", address)
}

// NewStatCounter initializes a statcounter instance using the registiry
// passed in.
func NewStatCounter(registry metrics.Registry) *StatsCounter {
	counter := &StatsCounter{
		channel:         Counter,
		Requests:        metrics.NewCounter(),
		RequestsFail:    metrics.NewCounter(),
		Announce:        metrics.NewCounter(),
		UniqueUsers:     metrics.NewCounter(),
		AnnounceFail:    metrics.NewCounter(),
		Scrape:          metrics.NewCounter(),
		ScrapeFail:      metrics.NewCounter(),
		InvalidPasskey:  metrics.NewCounter(),
		InvalidInfohash: metrics.NewCounter(),
		InvalidClient:   metrics.NewCounter(),
		APIRequests:     metrics.NewCounter(),
		APIRequestsFail: metrics.NewCounter(),
	}
	registry.Register(metric_names[EV_REQUEST_OK], counter.Requests)
	registry.Register(metric_names[EV_REQUEST_ERR], counter.RequestsFail)
	registry.Register(metric_names[EV_ANNOUNCE], counter.Announce)
	registry.Register(metric_names[EV_ANNOUNCE_FAIL], counter.AnnounceFail)
	registry.Register(metric_names[EV_SCRAPE], counter.Scrape)
	registry.Register(metric_names[EV_SCRAPE_FAIL], counter.ScrapeFail)
	registry.Register(metric_names[EV_INVALID_PASSKEY], counter.InvalidPasskey)
	registry.Register(metric_names[EV_INVALID_INFOHASH], counter.InvalidInfohash)
	registry.Register(metric_names[EV_INVALID_CLIENT], counter.InvalidClient)
	registry.Register(metric_names[EV_API], counter.APIRequests)
	registry.Register(metric_names[EV_API_FAIL], counter.APIRequestsFail)

	metrics.RegisterRuntimeMemStats(registry)
	go metrics.CaptureRuntimeMemStats(registry, 5e9)

	go counter.Counter()
	go counter.statPrinter()

	return counter
}

// Counter is a goroutine handling the incoming stat channel requests, sending to
// the appropriate counter instance
func (stats *StatsCounter) Counter() {
	for {
		v := <-stats.channel
		switch v {
		case EV_API:
			stats.APIRequests.Inc(1)
			stats.Requests.Inc(1)
		case EV_API_FAIL:
			stats.Errors.Inc(1)
			stats.APIRequestsFail.Inc(1)
		case EV_ANNOUNCE:
			stats.Announce.Inc(1)
			stats.Requests.Inc(1)
		case EV_ANNOUNCE_FAIL:
			stats.Errors.Inc(1)
			stats.AnnounceFail.Inc(1)
		case EV_SCRAPE:
			stats.Scrape.Inc(1)
			stats.Requests.Inc(1)
		case EV_SCRAPE_FAIL:
			stats.Errors.Inc(1)
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
			log.WithFields(log.Fields{
				"ann_total":   stats.Announce.Count(),
				"ann_err":     stats.AnnounceFail.Count(),
				"scr_total":   stats.Scrape.Count(),
				"scr_err":     stats.ScrapeFail.Count(),
				"inv_pk":      stats.InvalidPasskey.Count(),
				"inv_ih":      stats.InvalidInfohash.Count(),
				"inv_cl":      stats.InvalidClient.Count(),
				"req_sec_trk": req_sec,
				"req_sec_api": req_sec_api,
			}).Info("Periodic Stats")

			stats.RUnlock()
		}
	}()
	return ticker
}

func init() {
	registry = metrics.NewRegistry()
	StatCounts = NewStatCounter(registry)
}
