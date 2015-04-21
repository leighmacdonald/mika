package main

import (
	"github.com/garyburd/redigo/redis"
	"log"
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
	// request a connection. This signals that you're waiting for one.
	connRequest <- true
	// block and wait for the connection.
	// It'd be possible to do it with timeouts (https://gobyexample.com/timeouts),
	// but I didn't think it was that necessary for me.
	conn = <-connResponse
	return conn
}

func returnRedisConnection(conn redis.Conn) {
	connDone <- conn
}
