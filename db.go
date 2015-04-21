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

	for {
		time.Sleep(time.Duration(config.IndexInterval) * time.Second)
		mika.RLock()
		for _, torrent := range mika.Torrents {
			r.Send("ZADD", key_leechers, torrent.Leechers, torrent.TorrentID)
			r.Send("ZADD", key_seeders, torrent.Seeders, torrent.TorrentID)
			r.Send("ZADD", key_snatches, torrent.Snatches, torrent.TorrentID)
			count++
			if count >= 50 {
				r.Flush()
				count = 0
			}
		}
		mika.RUnlock()
		if count > 0 {
			r.Flush()
		}
		count = 0
	}
}