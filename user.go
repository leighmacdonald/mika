package main

import (
	"fmt"
	"github.com/garyburd/redigo/redis"
	"log"
	"sync"
)

type User struct {
	Queued
	sync.RWMutex
	UserID     uint64
	Uploaded   uint64
	Downloaded uint64
	Corrupt    uint64
	Snatches   uint32
	UserKey    string
	Announces  uint64
	Peers      []**Peer
}

// Fetch a user_id from the supplied passkey. A return value
// of 0 denotes a non-existing or disabled user_id
func findUserID(r redis.Conn, passkey string) uint64 {
	user_id_reply, err := r.Do("GET", fmt.Sprintf("t:user:%s", passkey))
	if err != nil {
		CaptureMessage("user.findUserID")
		log.Println(err)
		return 0
	}
	user_id, err_b := redis.Uint64(user_id_reply, nil)
	if err_b != nil {
		log.Println("Failed to find user", err_b)
		return 0
	}
	return user_id
}

func GetUser(r redis.Conn, passkey string) *User {

	user_id := findUserID(r, passkey)
	if user_id == 0 {
		return nil
	}

	mika.RLock()
	user, exists := mika.Users[user_id]
	mika.RUnlock()

	if !exists {
		user = &User{
			UserID:     user_id,
			Announces:  0,
			Corrupt:    0,
			Uploaded:   0,
			Downloaded: 0,
			Snatches:   0,
			Peers:      make([]**Peer, 1),
			UserKey:    fmt.Sprintf("t:u:%d", user_id),
		}

		user_reply, err := r.Do("HGETALL", user.UserKey)
		if err != nil {
			return nil
		}

		values, err := redis.Values(user_reply, nil)
		if err != nil {
			log.Println("Failed to parse user reply: ", err)
			return nil
		}

		err = redis.ScanStruct(values, user)
		if err != nil {
			return nil
		}

		mika.Lock()
		mika.Users[user_id] = user
		mika.Unlock()
		Debug("Added new user to in-memory cache:", user_id)
		return user
	}
	return user
}

func (user *User) Update(announce *AnnounceRequest, upload_diff, download_diff uint64) {
	user.Lock()
	user.Uploaded += upload_diff
	user.Downloaded += download_diff
	user.Corrupt += announce.Corrupt
	if announce.Event == COMPLETED {
		user.Snatches++
	}
	user.Unlock()
}

func (user *User) Sync(r redis.Conn) {
	r.Send(
		"HMSET", user.UserKey,
		"user_id", user.UserID,
		"uploaded", user.Uploaded,
		"downloaded", user.Downloaded,
		"corrupt", user.Corrupt,
		"snatches", user.Snatches,
		"announces", user.Announces,
	)
}

func (user *User) AddPeer(peer *Peer) {
	user.Peers = append(user.Peers, &peer)
}
