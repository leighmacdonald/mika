package main

import (
	//"github.com/influxdb/influxdb/client"
	"log"
	"time"
	//"net/url"
	"sync"
)

const (
	EV_ANNOUNCE         = iota
	EV_ANNOUNCE_FAIL    = iota
	EV_SCRAPE           = iota
	EV_SCRAPE_FAIL      = iota
	EV_INVALID_PASSKEY  = iota
	EV_INVALID_INFOHASH = iota
	EV_INVALID_CLIENT   = iota
)

type StatsCounter struct {
	sync.RWMutex
	channel         chan int
	Requests        uint64
	RequestsFail    uint64
	Announce        uint64
	AnnounceFail    uint64
	Scrape          uint64
	ScrapeFail      uint64
	InvalidPasskey  uint64
	InvalidInfohash uint64
	InvalidClient   uint64

	//influxDB *client.Client
}

func NewStatCounter(c chan int) *StatsCounter {
	//	u, err := url.Parse(config.InfluxDSN)
	//	if err != nil {
	//		log.Fatal(err)
	//	}
	//
	//	conf := client.Config{
	//		URL:      *u,
	//		Username: config.InfluxUser,
	//		Password: config.InfluxPass,
	//	}
	//
	//	con, err := client.NewClient(conf)
	//	if err != nil {
	//		log.Fatal(err)
	//	}
	//
	//	dur, ver, err := con.Ping()
	//	if err != nil {
	//		log.Fatal(err)
	//	}
	//
	//	log.Printf("InfluxDB Happy as a Hippo! %v, %s", dur, ver)

	counter := &StatsCounter{
		channel:         c,
		Requests:        0,
		RequestsFail:    0,
		Announce:        0,
		AnnounceFail:    0,
		Scrape:          0,
		ScrapeFail:      0,
		InvalidPasskey:  0,
		InvalidInfohash: 0,
		InvalidClient:   0,
		//influxDB:        con,
	}
	return counter
}

func (stats *StatsCounter) counter() {
	for {
		v := <-stats.channel
		stats.Lock()
		switch v {
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

func (stats *StatsCounter) statPrinter() {
	for {
		time.Sleep(60 * time.Second)
		stats.RLock()
		req_sec := stats.Requests / 60
		log.Printf("Ann: %d/%d Scr: %d/%d InvPK: %d InvIH: %d InvCL: %d Req/s: %d",
			stats.Announce, stats.AnnounceFail, stats.Scrape, stats.ScrapeFail,
			stats.InvalidPasskey, stats.InvalidInfohash, stats.InvalidClient, req_sec)
		stats.RUnlock()
		stats.Lock()
		stats.Requests = 0
		stats.Unlock()

	}
}
