package main

import (
	"github.com/garyburd/redigo/redis"
	"log"
	"time"
)

// Defined a single payload to send to the backend data store (redis)
type Payload struct {
	Command string
	Args    []interface{}
}

func NewPayload(command string, args ...interface{}) Payload {
	if len(args) < 1 {
		panic("Not enough arguments to make payload")
	}
	return Payload{Command: command, Args: args}
}

//
type BulkPayload struct {
	Payloads []Payload
}

func (db *BulkPayload) AddPayload(payload ...Payload) {
	db.Payloads = append(db.Payloads, payload...)

}

var (
	pool *redis.Pool
)

// This function will periodically update the torrent sort indexes
func dbStatIndexer() {
	log.Println("dbStatIndexer: Background indexer started")
	r := pool.Get()
	defer r.Close()

	key_leechers := "t:i:leechers"
	key_seeders := "t:i:seeders"
	key_snatches := "t:i:snatches"

	count := 0

	leecher_args := []uint64{}
	seeder_args := []uint64{}
	snatch_args := []uint64{}

	for {
		time.Sleep(time.Duration(config.IndexInterval) * time.Second)
		mika.TorrentsMutex.RLock()
		for _, torrent := range mika.Torrents {
			torrent.RLock()
			leecher_args = append(leecher_args, uint64(torrent.Leechers), torrent.TorrentID)
			seeder_args = append(seeder_args, uint64(torrent.Seeders), torrent.TorrentID)
			snatch_args = append(snatch_args, uint64(torrent.Snatches), torrent.TorrentID)
			torrent.RUnlock()
			count++
		}
		mika.TorrentsMutex.RUnlock()
		if count > 0 {
			r.Send("ZADD", key_leechers, leecher_args)
			r.Send("ZADD", key_seeders, seeder_args)
			r.Send("ZADD", key_snatches, snatch_args)
			r.Flush()
			leecher_args = leecher_args[:0]
			seeder_args = seeder_args[:0]
			snatch_args = snatch_args[:0]
		}
		count = 0
	}
}

// Handle writing out new data to the redis db in a queued manner
// Only items with the .InQueue flag set to false should be added.
// TODO channel as param
func syncWriter() {
	r := pool.Get()
	defer r.Close()
	if r.Err() != nil {
		CaptureMessage(r.Err().Error())
		log.Println("SyncWriter: Failed to get redis conn:", r.Err().Error())
		return
	}
	for {
		select {
		case payload := <-sync_payload:
			Debug("Sync payload")
			r.Do(payload.Command, payload.Args...)
		case user := <-sync_user:
			Debug("sync user")
			user.Sync(r)
			user.Lock()
			user.InQueue = false
			user.Unlock()
		case torrent := <-sync_torrent:
			Debug("sync torrent")
			torrent.Sync(r)
			torrent.Lock()
			torrent.InQueue = false
			torrent.Unlock()
		case peer := <-sync_peer:
			Debug("sync peer")
			peer.Sync(r)
			peer.Lock()
			peer.InQueue = false
			peer.Unlock()
		}
		err := r.Flush()
		if err != nil {
			log.Println("syncWriter: Failed to flush connection:", err)
		}
	}
}
