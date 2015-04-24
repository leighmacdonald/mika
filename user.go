package main

import (
	"fmt"
	"github.com/garyburd/redigo/redis"
	"log"
	"strconv"
	"strings"
	"sync"
)

type User struct {
	Queued
	sync.RWMutex
	UserID     uint64   `redis:"user_id" json:"user_id"`
	Uploaded   uint64   `redis:"uploaded" json:"uploaded"`
	Downloaded uint64   `redis:"downloaded" json:"downloaded"`
	Corrupt    uint64   `redis:"corrupt" json:"corrupt"`
	Snatches   uint32   `redis:"snatches" json:"snatches"`
	Passkey    string   `redis:"passkey" json:"-"`
	UserKey    string   `redis:"-" json:"key"`
	CanLeech   bool     `redis:"can_leech" json:"can_leech"`
	Announces  uint64   `redis:"announces" json:"announces"`
	Peers      []**Peer `redis:"-" json:"-"`
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

// Create a new user instance, loading existing data from redis if it exists
func fetchUser(r redis.Conn, user_id uint64) *User {
	user := &User{
		UserID:     user_id,
		Announces:  0,
		Corrupt:    0,
		Uploaded:   0,
		Downloaded: 0,
		Snatches:   0,
		Passkey:    "",
		CanLeech:   true,
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
		log.Println(err)
		return nil
	}
	return user
}

// fetch a user from the backend database if
func GetUser(r redis.Conn, passkey string) *User {

	user_id := findUserID(r, passkey)
	if user_id == 0 {
		return nil
	}

	mika.RLock()
	user, exists := mika.Users[user_id]
	mika.RUnlock()

	if !exists {
		user = fetchUser(r, user_id)
		mika.Lock()
		mika.Users[user_id] = user
		mika.Unlock()
		Debug("Added new user to in-memory cache:", user_id)
		return user
	}
	return user
}

// Update user stats from announce request
func (user *User) Update(announce *AnnounceRequest, upload_diff, download_diff uint64) {
	user.Lock()
	defer user.Unlock()
	user.Uploaded += upload_diff
	user.Downloaded += download_diff
	user.Corrupt += announce.Corrupt
	user.Announces++
	if announce.Event == COMPLETED {
		user.Snatches++
	}
}

// Write our bits out to redis
func (user *User) Sync(r redis.Conn) {
	r.Send(
		"HMSET", user.UserKey,
		"user_id", user.UserID,
		"uploaded", user.Uploaded,
		"downloaded", user.Downloaded,
		"corrupt", user.Corrupt,
		"snatches", user.Snatches,
		"announces", user.Announces,
		"can_leech", user.CanLeech,
		"passkey", user.Passkey,
	)
}

func (user *User) AddPeer(peer *Peer) {
	user.Peers = append(user.Peers, &peer)
}

// Load all the users into memory
func initUsers(r redis.Conn) {
	user_keys_reply, err := r.Do("KEYS", "t:u:*")
	if err != nil {
		log.Println("Failed to get torrent from redis", err)
		return
	}
	user_keys, err := redis.Strings(user_keys_reply, nil)
	if err != nil {
		log.Println("Failed to parse peer reply: ", err)
		return
	}
	users := 0
	mika.Lock()
	defer mika.Unlock()
	for _, user_key := range user_keys {
		pcs := strings.SplitN(user_key, ":", 3)
		if len(pcs) != 3 {
			continue
		}
		user_id, err := strconv.ParseUint(pcs[2], 10, 64)
		if err != nil {
			// Other key type probably
			continue
		}
		user := fetchUser(r, user_id)
		if user != nil {
			mika.Users[user_id] = user
			users++
		}
	}

	log.Println(fmt.Sprintf("Loaded %d users into memory", users))
}
