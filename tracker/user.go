package tracker

import (
	"fmt"
	"git.totdev.in/totv/mika/db"
	"git.totdev.in/totv/mika/util"
	"github.com/garyburd/redigo/redis"
	"log"
	"sync"
)

type User struct {
	db.Queued
	sync.RWMutex
	UserID        uint64   `redis:"user_id" json:"user_id"`
	Enabled       bool     `redis:"enabled" json:"enabled"`
	Uploaded      uint64   `redis:"uploaded" json:"uploaded"`
	Downloaded    uint64   `redis:"downloaded" json:"downloaded"`
	Corrupt       uint64   `redis:"corrupt" json:"corrupt"`
	Snatches      uint32   `redis:"snatches" json:"snatches"`
	Passkey       string   `redis:"passkey" json:"passkey"`
	UserKey       string   `redis:"-" json:"key"`
	Username      string   `redis:"username" json:"username"`
	CanLeech      bool     `redis:"can_leech" json:"can_leech"`
	Announces     uint64   `redis:"announces" json:"announces"`
	Peers         []**Peer `redis:"-" json:"-"`
	KeyActive     string   `redis:"-" json:"-"`
	KeyIncomplete string   `redis:"-" json:"-"`
	KeyComplete   string   `redis:"-" json:"-"`
	KeyHNR        string   `redis:"-" json:"-"`
}

// Fetch a user_id from the supplied passkey. A return value
// of 0 denotes a non-existing or disabled user_id
func (t *Tracker) findUserID(passkey string) uint64 {
	for _, user := range t.Users {
		if user.Passkey == passkey {
			return user.UserID
		}
	}
	return 0
}

// Create a new user instance, loading existing data from redis if it exists
func fetchUser(r redis.Conn, user_id uint64) *User {
	user := &User{
		UserID:     user_id,
		Announces:  0,
		Corrupt:    0,
		Uploaded:   0,
		Enabled:    true,
		Downloaded: 0,
		Snatches:   0,
		Passkey:    "",
		Username:   "",
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
		log.Println("fetchUser: Failed to parse user reply: ", err)
		return nil
	}

	err = redis.ScanStruct(values, user)
	if err != nil {
		log.Println("fetchUser: Failed to scan redis values:", err)
		return nil
	}

	user.KeyActive = fmt.Sprintf("t:u:active:%d", user_id)
	user.KeyIncomplete = fmt.Sprintf("t:u:incomplete:%d", user_id)
	user.KeyComplete = fmt.Sprintf("t:u:complete:%d", user_id)
	user.KeyHNR = fmt.Sprintf("t:u:hnr:%d", user_id)

	return user
}

func (t *Tracker) GetUserByPasskey(r redis.Conn, passkey string) *User {
	user_id := t.findUserID(passkey)
	if user_id == 0 {
		return nil
	}
	return t.GetUserByID(r, user_id, false)
}

// fetch a user from the backend database if
func (t *Tracker) GetUserByID(r redis.Conn, user_id uint64, auto_create bool) *User {

	t.UsersMutex.RLock()
	user, exists := t.Users[user_id]
	t.UsersMutex.RUnlock()

	if !exists && auto_create {
		user = fetchUser(r, user_id)
		t.UsersMutex.Lock()
		t.Users[user_id] = user
		t.UsersMutex.Unlock()
		util.Debug("Added new user to in-memory cache:", user_id)
		return user
	}
	return user
}

// Update user stats from announce request
func (user *User) Update(announce *AnnounceRequest, upload_diff, download_diff uint64, multi_up, multi_dn float64) {
	user.Lock()
	defer user.Unlock()
	if announce.Event != STARTED {
		// Ignore downloaded value when starting
		user.Uploaded += uint64(float64(upload_diff) * multi_up)
		user.Downloaded += uint64(float64(download_diff) * multi_dn)
		user.Corrupt += announce.Corrupt
	}
	user.Announces++
	if announce.Event == COMPLETED {
		user.Snatches++
	}
}

// Write our bits out to redis
// TODO only write out what is changed
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
		"enabled", user.Enabled,
		"username", user.Username,
	)
}

func (user *User) AddPeer(peer *Peer) {
	user.Peers = append(user.Peers, &peer)
}
