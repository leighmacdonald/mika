package main

import (
	"github.com/garyburd/redigo/redis"
	"log"
	"time"
)

var (
	pool *redis.Pool

	connRequest  chan bool
	connResponse chan redis.Conn
	connDone     chan redis.Conn
	connWaiting  int = 0
)

func PoolPoolManager(pool *redis.Pool) {
	var conn redis.Conn
	var err error
	for {
		select {
		case <-connRequest:
			// someone wants a connection. try to get one.
			conn = pool.Get()
			err = conn.Err()
			switch err {
			case redis.ErrPoolExhausted:
				// none left. wait for a free one.
				connWaiting++
			case nil:
				// got one. return it.
				connResponse <- conn
			default:
				// misc failure. Might want to panic or return error on channel. this may never resolve itself.
				log.Println(err, "Error getting connection")
				connWaiting++
			}
		case conn = <-connDone:
			// someone is done with a connection.
			if connWaiting > 0 {
				// someone is waiting for one. return this connection to them.
				connWaiting--
				connResponse <- conn
			} else {
				// nobody is waiting. return it to the pool.
				conn.Close()
			}
		}
	}
}

func getRedisConnection() redis.Conn {
	var conn redis.Conn
	// request a connection.
	connRequest <- true
	conn = <-connResponse
	return conn
}

func returnRedisConnection(conn redis.Conn) {
	connDone <- conn
}

// This function will periodically update the torrent sort indexes
func dbStatIndexer() {
	log.Println("Background indexer started")
	r := getRedisConnection()
	defer returnRedisConnection(r)

	key_leechers := "t:i:leechers"
	key_seeders := "t:i:seeders"
	key_snatches := "t:i:snatches"

	count := 0

	leecher_args := []uint64{}
	seeder_args := []uint64{}
	snatch_args := []uint64{}

	for {
		time.Sleep(time.Duration(config.IndexInterval) * time.Second)
		mika.RLock()
		for _, torrent := range mika.Torrents {
			leecher_args = append(leecher_args, uint64(torrent.Leechers), torrent.TorrentID)
			seeder_args = append(seeder_args, uint64(torrent.Seeders), torrent.TorrentID)
			snatch_args = append(snatch_args, uint64(torrent.Snatches), torrent.TorrentID)
			count++
		}
		mika.RUnlock()
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
