package main

import (
	"log"
	"time"
)

const (
	EV_ANNOUNCE = iota
	EV_ANNOUNCE_FAIL = iota
	EV_SCRAPE = iota
	EV_SCRAPE_FAIL = iota
	EV_INVALID_PASSKEY = iota
	EV_INVALID_INFOHASH = iota
	EV_INVALID_CLIENT = iota
)

type StatsCounter struct {
	channel         chan int
	Announce        uint64
	AnnounceFail    uint64
	Scrape          uint64
	ScrapeFail      uint64
	InvalidPasskey  uint64
	InvalidInfohash uint64
	InvalidClient   uint64
}

func NewStatCounter(c chan int) *StatsCounter {
	counter := &StatsCounter{
		channel:         c,
		Announce:        0,
		AnnounceFail:    0,
		Scrape:          0,
		ScrapeFail:      0,
		InvalidPasskey:  0,
		InvalidInfohash: 0,
		InvalidClient:   0,
	}

	go counter.counter()
	go counter.statPrinter()

	return counter

}

func (stats *StatsCounter) counter() {
	for {
		v := <-stats.channel
		switch v {
			case EV_ANNOUNCE:
			stats.Announce++
			case EV_ANNOUNCE_FAIL:
			stats.AnnounceFail++
			case EV_SCRAPE:
			stats.Scrape++
			case EV_SCRAPE_FAIL:
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

func (stats *StatsCounter) statPrinter() {
	for {
		time.Sleep(60 * time.Second)
		log.Printf("Ann: %d/%d Scr: %d/%d InvPK: %d InvIH: %d InvCL: %d", stats.Announce, stats.AnnounceFail, stats.Scrape, stats.ScrapeFail, stats.InvalidPasskey, stats.InvalidInfohash, stats.InvalidClient)
	}
}
