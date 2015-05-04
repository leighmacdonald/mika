package db

import (
	"github.com/garyburd/redigo/redis"
)

var (
	Pool *redis.Pool

	// Channels
	SyncPayloadC = make(chan Payload, 1000)
)

type DBEntity interface {
	Sync(r redis.Conn) bool
}

type Queued struct {
	InQueue bool `redis:"-" json:"-"`
}

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
