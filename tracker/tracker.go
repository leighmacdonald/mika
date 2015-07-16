// Package tracker provides the majority of the core bittorrent tracker functionality
package tracker

import (
	//	"git.totdev.in/totv/mika/conf"
	"git.totdev.in/totv/mika/db"
	log "github.com/Sirupsen/logrus"
	"github.com/garyburd/redigo/redis"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	// The global tracker instance
	Mika *Tracker

	// Channels used to sync models to redis
	//	SyncUserC    = make(chan *User, 100)
	//	SyncTorrentC = make(chan *Torrent, 500)
	SyncEntityC = make(chan db.DBEntity, 500)

	key_leechers = "t:i:leechers"
	key_seeders  = "t:i:seeders"
	key_snatches = "t:i:snatches"
)

// Main track struct holding all the known models
type Tracker struct {
	// Torrents and torrent lock
	TorrentsMutex *sync.RWMutex
	Torrents      map[string]*Torrent

	// Users and User lock
	UsersMutex *sync.RWMutex
	Users      map[uint64]*User

	// Whitelist and whitelist lock
	WhitelistMutex *sync.RWMutex
	Whitelist      []string

	stopChan chan bool
}

// NewTracker created a new allocated Tracker instance to use. Initialize() should
// before attempting to serve any requests.
func NewTracker() *Tracker {
	// Alloc tracker
	tracker := &Tracker{
		Torrents:       make(map[string]*Torrent),
		Users:          make(map[uint64]*User),
		TorrentsMutex:  new(sync.RWMutex),
		UsersMutex:     new(sync.RWMutex),
		Whitelist:      []string{},
		WhitelistMutex: new(sync.RWMutex),
		stopChan:       make(chan bool),
	}

	return tracker
}

// Initialize loads the models into memory from redis.
func (tracker *Tracker) Initialize() error {
	log.WithFields(log.Fields{
		"peer_id": "Initialize",
	}).Info("Initializing models in memory.")
	r := db.Pool.Get()
	tracker.initWhitelist(r)
	tracker.initUsers(r)
	tracker.initTorrents(r)
	r.Close()
	return nil
}

func (tracker *Tracker) Shutdown() {
	tracker.stopChan <- true
}

// GetTorrentByID will fetch torrent data from the database and return a Torrent struct.
// If the torrent doesn't exist in the database a new skeleton Torrent
// instance will be returned.
func (tracker *Tracker) GetTorrentByID(r redis.Conn, torrent_id uint64, make_new bool) *Torrent {
	tracker.TorrentsMutex.RLock()
	defer tracker.TorrentsMutex.RUnlock()
	for _, torrent := range tracker.Torrents {
		if torrent.TorrentID == torrent_id {
			return torrent
		}
	}
	return nil
}

// FindTorrentByInfoHash will fetch the torrent_id that corresponds to the info_hash supplied
// as a GET value. If the info_hash doesn't return an id we consider the torrent
// either soft-deleted or non-existent
func (tracker *Tracker) FindTorrentByInfoHash(info_hash string) *Torrent {
	info_hash = strings.ToLower(info_hash)
	tracker.TorrentsMutex.RLock()
	torrent, _ := tracker.Torrents[info_hash]
	tracker.TorrentsMutex.RUnlock()
	return torrent
}

// AddTorrent adds a new torrent to the tracker, allowing users to make requests
// using its info_hash
func (tracker *Tracker) AddTorrent(torrent *Torrent) {
	tracker.TorrentsMutex.Lock()
	tracker.Torrents[torrent.InfoHash] = torrent
	tracker.TorrentsMutex.Unlock()
	log.WithFields(log.Fields{
		"info_hash": torrent.InfoHash,
		"fn":        "AddTorrent",
	}).Debug("Registered new torrent in tracker")
}

func (tracker *Tracker) DelTorrent(torrent *Torrent) bool {
	if torrent.Enabled {
		torrent.Lock()
		torrent.Enabled = false
		torrent.Unlock()
		log.WithFields(log.Fields{
			"fn":        "DelTorrent",
			"info_hash": torrent.InfoHash,
		}).Info("Deleted torrent successfully")
		SyncEntityC <- torrent
		return true
	} else {
		return false
	}
}

// DeleteUser will completely remove the user from the trackers memory. This is different
// than disabling users which should still be known to the system.
//
// TODO Make sure other references are dropped so GC can take over.
func (tracker *Tracker) DelUser(user *User) {
	// user.Cleanup()

	tracker.UsersMutex.Lock()
	delete(tracker.Users, user.UserID)
	tracker.UsersMutex.Unlock()

	log.WithFields(log.Fields{
		"fn":      "tracker.DelUser",
		"user_id": user.UserID,
	}).Info("Deleted user successfully")
}

// GetUserByID fetches a user from the backend database. Id auto_create is set
// it will also make a new user if an existing one was not found.
func (tracker *Tracker) FindUserByID(user_id uint64) *User {
	tracker.UsersMutex.RLock()
	user, _ := tracker.Users[user_id]
	tracker.UsersMutex.RUnlock()
	return user
}

// AddUser adds a new user into the known user list of the tracker
func (tracker *Tracker) AddUser(user *User) {
	tracker.UsersMutex.Lock()
	tracker.Users[user.UserID] = user
	tracker.UsersMutex.Unlock()
	log.WithFields(log.Fields{
		"user_id":   user.UserID,
		"user_name": user.Username,
		"passkey":   user.Passkey,
		"fn":        "AddUser",
	}).Info("Added new user to tracker")
}

// initWhitelist will fetch the client whitelist from redis and load it into memory
func (tracker *Tracker) initWhitelist(r redis.Conn) {
	tracker.WhitelistMutex.Lock()
	defer tracker.WhitelistMutex.Unlock()
	tracker.Whitelist = []string{}
	a, err := r.Do("HKEYS", "t:whitelist")

	if err != nil {
		log.WithFields(log.Fields{
			"fn": "initWhitelist",
		}).Error("Failed to fetch whitelist", err)
		return
	}
	tracker.Whitelist, err = redis.Strings(a, nil)
	log.WithFields(log.Fields{
		"total_clients": len(tracker.Whitelist),
		"fn":            "initWhitelist",
	}).Info("Loaded whitelist clients")
}

// initTorrents will fetch the torrents stored in redis and load them into active memory as models
func (tracker *Tracker) initTorrents(r redis.Conn) {
	torrent_keys_reply, err := r.Do("KEYS", "t:t:*")
	if err != nil {
		log.WithFields(log.Fields{
			"err": err.Error(),
			"fn":  "initTorrents",
		}).Error("Failed to get torrent from redis")
		return
	}
	torrent_keys, err := redis.Strings(torrent_keys_reply, nil)
	if err != nil {
		log.WithFields(log.Fields{
			"err": err.Error(),
			"fn":  "initTorrents",
		}).Error("Failed to parse torrent keys reply")
		return
	}
	torrents := 0

	for _, torrent_key := range torrent_keys {
		pcs := strings.SplitN(torrent_key, ":", 3)
		// Skip malformed keys and peer suffixed keys
		if len(pcs) != 3 || len(pcs[2]) != 40 {
			continue
		}
		torrent := NewTorrent(pcs[2], "", 0)
		// torrent := t.FindTorrentByInfoHash(pcs[2])
		if torrent != nil {
			torrent.MergeDB(r)
			tracker.AddTorrent(torrent)
			torrents++
		} else {
			// Drop keys we don't have valid ids for
			log.WithFields(log.Fields{
				"fn":  "initTorrents",
				"key": torrent_key,
			}).Warn("Unknown key, deleting..")
			r.Do("DEL", torrent_key)
		}
	}
	log.WithFields(log.Fields{
		"count": torrents,
		"fn":    "initTorrents",
	}).Info("Loaded torrents into memory")
}

// initUsers pre loads all known users into memory from redis backend
func (tracker *Tracker) initUsers(r redis.Conn) {
	user_keys_reply, err := r.Do("KEYS", "t:u:*")
	if err != nil {
		log.WithFields(log.Fields{
			"err": err.Error(),
			"fn":  "initUsers",
		}).Error("Failed to get torrent from redis")
		return
	}
	user_keys, err := redis.Strings(user_keys_reply, nil)
	if err != nil {
		log.WithFields(log.Fields{
			"err": err.Error(),
			"fn":  "initUsers",
		}).Error("Failed to parse peer reply")
		return
	}
	users := 0

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

		// TODO add passkey/username in addition to user_id
		user := NewUser(user_id)
		user.MergeDB(r)
		tracker.AddUser(user)

		users++
	}

	log.WithFields(log.Fields{
		"count": users,
		"fn":    "initUsers",
	}).Info("Loaded users into memory")
}

// dbStatIndexer when running will periodically update the torrent sort indexes
func (tracker *Tracker) dbStatIndexer() {
	log.WithFields(log.Fields{
		"fn": "dbStatIndexer",
	}).Info("Background indexer started")
	r := db.Pool.Get()
	defer r.Close()

	count := 0

	leecher_args := []uint64{}
	seeder_args := []uint64{}
	snatch_args := []uint64{}

	ticker := time.NewTicker(time.Second * 120)

	for {
		select {
		case <-ticker.C:
			tracker.TorrentsMutex.RLock()
			for _, torrent := range tracker.Torrents {
				tracker.TorrentsMutex.RLock()
				leecher_args = append(leecher_args, uint64(torrent.Leechers), torrent.TorrentID)
				seeder_args = append(seeder_args, uint64(torrent.Seeders), torrent.TorrentID)
				snatch_args = append(snatch_args, uint64(torrent.Snatches), torrent.TorrentID)
				tracker.TorrentsMutex.RUnlock()
				count++
			}
			tracker.TorrentsMutex.RUnlock()
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
		case <-tracker.stopChan:
			ticker.Stop()
			log.Debugln("dbStatIndexer received stop chan signal")
			return
		}
	}
}

// syncWriter Handles writing out new data to the redis db in a queued manner
// Only items with the .InQueue flag set to false should be added.
// TODO channel as param
func (tracker *Tracker) syncWriter() {
	r := db.Pool.Get()
	defer r.Close()
	if r.Err() != nil {
		log.WithFields(log.Fields{
			"fn":  "syncWriter",
			"err": r.Err().Error(),
		}).Fatal("Failed to get redis conn")
		return
	}
	for {
		select {
		case payload := <-db.SyncPayloadC:
			r.Do(payload.Command, payload.Args...)
		case entity := <-SyncEntityC:
			log.Debugln("Syncing:", entity)
			entity.Sync(r)
		case <-tracker.stopChan:
			// Make sure we flush the remaining queued entries before exiting
			r.Flush()
			return
		}
		err := r.Flush()
		if err != nil {
			log.WithFields(log.Fields{
				"fn": "syncWriter",
			}).Fatal("Failed to flush connection:", err.Error())
		} else {
			log.Debugln("Synced entity successfully")
		}
	}
}

// Checked if the clients peer_id prefix matches the client prefixes
// stored in the white lists
func (tracker *Tracker) IsValidClient(peer_id string) bool {
	for _, client_prefix := range tracker.Whitelist {
		if strings.HasPrefix(peer_id, client_prefix) {
			return true
		}
	}

	log.WithFields(log.Fields{
		"fn":      "IsValidClient",
		"peer_id": peer_id[0:6],
	}).Warn("Got non-whitelisted client")

	return false
}
