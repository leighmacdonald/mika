// Package stats provides functionality for recording and displaying metrics
package stats

import (
	log "github.com/Sirupsen/logrus"
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
	counterChan  = make(chan int)
	Stats        = &StatsCounter{}
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
)

type StatsCounter struct {
	sync.RWMutex
	channel         chan int
	Errors          int
	Requests        int
	RequestsFail    int
	Announce        int
	UniqueUsers     int
	AnnounceFail    int
	Scrape          int
	ScrapeFail      int
	InvalidPasskey  int
	InvalidInfohash int
	InvalidClient   int
	APIRequests     int
	APIRequestsFail int
	AnnounceUserIDS []uint64
	ScrapeUserIDS   []uint64
}

// Counter is a goroutine handling the incoming stat channel requests, sending to
// the appropriate counter instance
func (stats *StatsCounter) countReceiver() {
	for {
		v := <-stats.channel
		switch v {
		case EV_API:
			stats.APIRequests++
			stats.Requests++
		case EV_API_FAIL:
			stats.RequestsFail++
			stats.APIRequestsFail++
		case EV_ANNOUNCE:
			stats.Announce++
			stats.Requests++
		case EV_ANNOUNCE_FAIL:
			stats.RequestsFail++
			stats.AnnounceFail++
		case EV_SCRAPE:
			stats.Scrape++
			stats.Requests++
		case EV_SCRAPE_FAIL:
			stats.RequestsFail++
			stats.ScrapeFail++
		case EV_INVALID_INFOHASH:
			stats.InvalidInfohash++
		case EV_INVALID_PASSKEY:
			stats.InvalidPasskey++
		case EV_INVALID_CLIENT:
			stats.InvalidClient++
		}
	}
}

// statPrinter will periodically print out basic stat lines to standard output
func (stats *StatsCounter) statPrinter() *time.Ticker {
	ticker := time.NewTicker(60 * time.Second)
	go func() {
		for range ticker.C {
			stats.RLock()
			req_sec := stats.Announce / 60
			req_sec_api := stats.APIRequests / 60
			log.WithFields(log.Fields{
				"ann_total":   stats.Announce,
				"ann_err":     stats.AnnounceFail,
				"scr_total":   stats.Scrape,
				"scr_err":     stats.ScrapeFail,
				"inv_pk":      stats.InvalidPasskey,
				"inv_ih":      stats.InvalidInfohash,
				"inv_cl":      stats.InvalidClient,
				"req_sec_trk": req_sec,
				"req_sec_api": req_sec_api,
			}).Info("Periodic Stats")

			stats.RUnlock()
		}
	}()
	return ticker
}

func RegisterEvent(event int) {
	counterChan <- event
}

func init() {
	go Stats.countReceiver()
}
