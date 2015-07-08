package tracker

import (
	"fmt"
	"git.totdev.in/totv/mika/db"
	"git.totdev.in/totv/mika/util"
	log "github.com/Sirupsen/logrus"
	"github.com/garyburd/redigo/redis"
	"math/rand"
	"sync"
	"time"
)

// t:usertorrent:<user_id>:<torrent_id> ->
//    first_ann, last_ann, uploaded, downloaded, seed_time, speed_up_max, speed_dn_max
type User struct {
	db.DBEntity   `redis:"-" json:"-"`
	sync.RWMutex  `redis:"-" json:"-"`
	Queued        bool         `redis:"-" json:"-"`
	UserID        uint64       `redis:"user_id" json:"user_id"`
	Enabled       bool         `redis:"enabled" json:"enabled"`
	Uploaded      uint64       `redis:"uploaded" json:"uploaded"`
	Downloaded    uint64       `redis:"downloaded" json:"downloaded"`
	Snatches      uint32       `redis:"snatches" json:"snatches"`
	Passkey       string       `redis:"passkey" json:"passkey"`
	UserKey       string       `redis:"-" json:"key"`
	Username      string       `redis:"username" json:"username"`
	CanLeech      bool         `redis:"can_leech" json:"can_leech"`
	Announces     uint64       `redis:"announces" json:"announces"`
	Peers         []*Peer      `redis:"-" json:"-"`
	KeyActive     string       `redis:"-" json:"-"`
	KeyIncomplete string       `redis:"-" json:"-"`
	KeyComplete   string       `redis:"-" json:"-"`
	KeyHNR        string       `redis:"-" json:"-"`
	Scheduler     *time.Ticker `redis:"-" json:"-"`

	// Active torrent set. Does not include historical or recently reaped peers.
	torrents []*Torrent
}

// findUserID find a user_id from the supplied passkey. A return value
// of 0 denotes a non-existing or disabled user_id
func (tracker *Tracker) findUserID(passkey string) uint64 {
	for _, user := range tracker.Users {
		if user.Passkey == passkey {
			return user.UserID
		}
	}
	return 0
}

func (user *User) scheduler(ticker *time.Ticker) {
	// Randomize the scheduler start time to make sure everyone isn't updating at the exact
	// same moment.
	time.Sleep(time.Millisecond * time.Duration(rand.Intn(60)))
	for range user.Scheduler.C {
		log.WithFields(log.Fields{
			"fn":      "schedualer",
			"user_id": user.UserID,
		}).Debug("User scheduler executed")
	}
}

// MergeDB will update the user instance with the currently active data
// from the backend redis database. This will simple overwrite and existing
// data in the user instance.
func (user *User) MergeDB(r redis.Conn) error {
	user_reply, err := r.Do("HGETALL", user.UserKey)
	if err != nil {
		log.WithFields(log.Fields{
			"fn":      "MergeDB",
			"err":     err.Error(),
			"user_id": user.UserID,
		}).Error("Failed to fetch user data from db")
		return err
	}

	values, err := redis.Values(user_reply, nil)
	if err != nil {
		log.WithFields(log.Fields{
			"fn":      "MergeDB",
			"err":     err.Error(),
			"user_id": user.UserID,
		}).Error("Failed to parse user reply")
		return err
	}

	err = redis.ScanStruct(values, user)
	if err != nil {
		log.WithFields(log.Fields{
			"fn":      "MergeDB",
			"err":     err.Error(),
			"user_id": user.UserID,
		}).Error("Failed to scan redis values")
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
		Peers:         make([]*Peer, 1),
		UserKey:       fmt.Sprintf("t:u:%d", user_id),
		KeyActive:     fmt.Sprintf("t:u:active:%d", user_id),
		KeyIncomplete: fmt.Sprintf("t:u:incomplete:%d", user_id),
		KeyComplete:   fmt.Sprintf("t:u:complete:%d", user_id),
		KeyHNR:        fmt.Sprintf("t:u:hnr:%d", user_id),
		Queued:        false,
		Scheduler:     time.NewTicker(time.Minute * 15),
	}

	go user.scheduler(user.Scheduler)

	return user
}

func (user *User) AddHNR(r redis.Conn, torrent_id uint64) {
	r.Send("SADD", user.KeyHNR, torrent_id)
	log.WithFields(log.Fields{
		"torrent_id": torrent_id,
		"user_id":    user.UserID,
	}).Debug("Added new HnR to user")
}

// Update user stats from announce request
func (user *User) Update(announce *AnnounceRequest, peer_diff *PeerDiff, multi_up, multi_dn float64) {
	user.Lock()
	defer user.Unlock()
	if announce.Event != STARTED {
		// Apply multipliers to get value we will actually record
		upload_diff_multi := uint64(float64(peer_diff.UploadDiff) * multi_up)
		download_diff_multi := uint64(float64(peer_diff.DownloadDiff) * multi_dn)

		// Get new totals
		uploaded_new := user.Uploaded + upload_diff_multi
		downloaded_new := user.Downloaded + download_diff_multi

		if peer_diff.UploadDiff > 0 || peer_diff.DownloadDiff > 0 {
			log.WithFields(log.Fields{
				"ul_old":       util.Bytes(user.Uploaded),
				"ul_diff":      util.Bytes(peer_diff.UploadDiff),
				"ul_diff_mult": util.Bytes(upload_diff_multi),
				"ul_new":       util.Bytes(uploaded_new),
				"dl_old":       util.Bytes(user.Downloaded),
				"dl_diff":      util.Bytes(peer_diff.DownloadDiff),
				"dl_diff_mult": util.Bytes(download_diff_multi),
				"dl_new":       util.Bytes(downloaded_new),
				"user":         user.Username,
				"fn":           "Update",
			}).Info("User stat changes")
		}

		user.Uploaded = uploaded_new
		user.Downloaded = downloaded_new
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

// Join registeres a user into a
func (user *User) Join(torrent *Torrent) bool {
	for _, known_torrent := range user.torrents {
		if known_torrent == torrent {
			return false
		}
	}
	log.WithFields(log.Fields{
		"fn":        "user.Join",
		"user_id":   user.UserID,
		"info_hash": torrent.InfoHash,
	}).Debug("User joined swarm")
	user.torrents = append(user.torrents, torrent)
	return true
}

func (peer_diff *PeerDiff) Sync(r redis.Conn) {
	key := peer_diff.Key()
	r.Send("HMSET", key, "speed_up_max", peer_diff.SpeedUPMax, "speed_dn_max", peer_diff.SpeedDNMax)
	r.Send("HINCRBY", key, "downloaded", peer_diff.DownloadDiff)
	r.Send("HINCRBY", key, "uploaded", peer_diff.UploadDiff)
	r.Send("HINCRBY", key, "seed_time", peer_diff.SeedTime)
}

// AddPeer adds a peer to a users active peer list if it doesn't already exist
func (user *User) AddPeer(peer *Peer) {
	if !user.HasPeer(peer) {
		user.Peers = append(user.Peers, peer)
	}
}

// HasPeer will check for a existing peer already regisered with the user
// Ideally there should only be 1 peer/user, but allowing multiple clients
// means we need to track them.
func (user *User) HasPeer(peer *Peer) bool {
	for _, p := range user.Peers {
		if peer == p {
			return true
		}
	}
	return false
}

// CalculateBonus will return the bonus value for the parameters given.
//
// time_spend - Total time the user has spent seeding a torrent. This must account for the
//              non-active time as well.
// uploaded - Amount uploaded in total bytes for the torrent
// seeders - Current seeder count of the torrent
func CalculateBonus(time_spent uint64, uploaded uint64, seeders uint64) float64 {
	time_spent_f := float64(time_spent) / 3600.0
	if seeders == 0 {
		seeders = 1
	}
	upload := (float64(uploaded) / 1024.0 / 1024.0 / 1024.0) + 1
	return (upload + ((time_spent_f+1)*5)*(10/float64(seeders))) / 1000
}
