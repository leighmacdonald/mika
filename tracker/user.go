package tracker

import (
	"fmt"
	"git.totdev.in/totv/mika/db"
	log "github.com/Sirupsen/logrus"
	"github.com/garyburd/redigo/redis"
	"sync"
)

type User struct {
	db.Queued
	sync.RWMutex
	UserID        uint64   `redis:"user_id" json:"user_id"`
	Enabled       bool     `redis:"enabled" json:"enabled"`
	Uploaded      uint64   `redis:"uploaded" json:"uploaded"`
	Downloaded    uint64   `redis:"downloaded" json:"downloaded"`
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

// findUserID find a user_id from the supplied passkey. A return value
// of 0 denotes a non-existing or disabled user_id
func (t *Tracker) findUserID(passkey string) uint64 {
	for _, user := range t.Users {
		if user.Passkey == passkey {
			return user.UserID
		}
	}
	return 0
}

func (user *User) MergeDB(r redis.Conn) error {
	user_reply, err := r.Do("HGETALL", user.UserKey)
	if err != nil {
		return err
	}

	values, err := redis.Values(user_reply, nil)
	if err != nil {
		log.Println("fetchUser: Failed to parse user reply: ", err)
		return err
	}

	err = redis.ScanStruct(values, user)
	if err != nil {
		log.Println("fetchUser: Failed to scan redis values:", err)
		return err
	}
	return nil
}

// NewUser makes a new user instance
// TODO require the username/passkey
func NewUser(user_id uint64) *User {
	user := &User{
		UserID:        user_id,
		Enabled:       true,
		Uploaded:      0,
		Downloaded:    0,
		CanLeech:      true,
		Peers:         make([]**Peer, 1),
		UserKey:       fmt.Sprintf("t:u:%d", user_id),
		KeyActive:     fmt.Sprintf("t:u:active:%d", user_id),
		KeyIncomplete: fmt.Sprintf("t:u:incomplete:%d", user_id),
		KeyComplete:   fmt.Sprintf("t:u:complete:%d", user_id),
		KeyHNR:        fmt.Sprintf("t:u:hnr:%d", user_id),
	}
	return user
}

func (user *User) AddHNR(r redis.Conn, torrent_id uint64) {
	r.Send("SADD", user.KeyHNR, torrent_id)
	log.WithFields(log.Fields{
		"torrent_id": torrent_id,
		"user_id": user.UserID,
	}).Debug("Added new HnR to user")
}

// Update user stats from announce request
func (user *User) Update(announce *AnnounceRequest, upload_diff, download_diff uint64, multi_up, multi_dn float64) {
	user.Lock()
	defer user.Unlock()
	if announce.Event != STARTED {
		// Ignore downloaded value when starting
		user.Uploaded += uint64(float64(upload_diff) * multi_up)
		user.Downloaded += uint64(float64(download_diff) * multi_dn)
	}
	user.Announces++
	if announce.Event == COMPLETED {
		user.Snatches++
	}
}

// Sync will write the pertinent date out to the redis connection. Its important to
// note that this function does not flush the changes to redis as its meant to be chained
// with other sync functions.
// TODO only write out what is changed
func (user *User) Sync(r redis.Conn) {
	r.Send(
		"HMSET", user.UserKey,
		"user_id", user.UserID,
		"uploaded", user.Uploaded,
		"downloaded", user.Downloaded,
		"snatches", user.Snatches,
		"announces", user.Announces,
		"can_leech", user.CanLeech,
		"passkey", user.Passkey,
		"enabled", user.Enabled,
		"username", user.Username,
	)
}

// AddPeer adds a peer to a users active peer list
func (user *User) AddPeer(peer *Peer) {
	user.Peers = append(user.Peers, &peer)
}

// AppendIfMissing will add a new item to a slice if the item is not currently
// a member of the slice.
func AppendIfMissing(slice []uint64, i uint64) []uint64 {
	for _, ele := range slice {
		if ele == i {
			return slice
		}
	}
	return append(slice, i)
}
