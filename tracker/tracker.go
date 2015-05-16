// Package tracker provides the majority of the core bittorrent tracker functionality
package tracker

import (
	"fmt"
	"git.totdev.in/totv/mika/conf"
	"git.totdev.in/totv/mika/db"
	"git.totdev.in/totv/mika/util"
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
	SyncUserC    = make(chan *User, 100)
	SyncPeerC    = make(chan *Peer, 1000)
	SyncTorrentC = make(chan *Torrent, 500)
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
	}

	return tracker
}

// Load the models into memory from redis
func (tracker *Tracker) Initialize() error {
	log.Println("Initialize: Initializing models in memory...")
	r := db.Pool.Get()
	defer r.Close()

	tracker.initWhitelist(r)
	tracker.initUsers(r)
	tracker.initTorrents(r)

	return nil
}

// Fetch a torrents data from the database and return a Torrent struct.
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

// Fetch the stores torrent_id that corresponds to the info_hash supplied
// as a GET value. If the info_hash doesn't return an id we consider the torrent
// either soft-deleted or non-existent
func (tracker *Tracker) FindTorrentByInfoHash(info_hash string) *Torrent {
	info_hash = strings.ToLower(info_hash)
	tracker.TorrentsMutex.RLock()
	torrent, _ := tracker.Torrents[info_hash]
	tracker.TorrentsMutex.RUnlock()
	return torrent
}

func (tracker *Tracker) AddTorrent(torrent *Torrent) {
	tracker.TorrentsMutex.Lock()
	tracker.Torrents[torrent.InfoHash] = torrent
	tracker.TorrentsMutex.Unlock()
	log.Debug("GetTorrentByInfoHash: Added new torrent to in-memory cache:", torrent.InfoHash)
}

// GetUserByID fetches a user from the backend database. Id auto_create is set
// it will also make a new user if an existing one was not found.
func (tracker *Tracker) FindUserByID(user_id uint64) *User {
	tracker.UsersMutex.RLock()
	user, _ := tracker.Users[user_id]
	tracker.UsersMutex.RUnlock()
	return user
}

func (tracker *Tracker) AddUser(user *User) {
	tracker.UsersMutex.Lock()
	tracker.Users[user.UserID] = user
	tracker.UsersMutex.Unlock()
	log.Debug("Added new user to memory:", user.UserID)
}

// initWhitelist will fetch the client whitelist from redis and load it into memory
func (tracker *Tracker) initWhitelist(r redis.Conn) {
	tracker.Whitelist = []string{}
	a, err := r.Do("HKEYS", "t:whitelist")

	if err != nil {
		log.Println("initWhitelist: Failed to fetch whitelist", err)
		return
	}
	tracker.Whitelist, err = redis.Strings(a, nil)
	log.Println(fmt.Sprintf("initWhitelist: Loaded %d whitelist clients", len(tracker.Whitelist)))
}

// initTorrents will fetch the torrents stored in redis and load them into active memory as models
func (tracker *Tracker) initTorrents(r redis.Conn) {
	torrent_keys_reply, err := r.Do("KEYS", "t:t:*")
	if err != nil {
		log.Println("initTorrents: Failed to get torrent from redis", err)
		return
	}
	torrent_keys, err := redis.Strings(torrent_keys_reply, nil)
	if err != nil {
		log.Println("initTorrents: Failed to parse torrent keys reply: ", err)
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
			log.Warn("initTorrents: Unknown key:", torrent_key)
			r.Do("DEL", torrent_key)
		}
	}

	log.Println(fmt.Sprintf("initTorrents: Loaded %d torrents into memory", torrents))
}

// initUsers pre loads all known users into memory from redis backend
func (tracker *Tracker) initUsers(r redis.Conn) {
	user_keys_reply, err := r.Do("KEYS", "t:u:*")
	if err != nil {
		log.Println("initUsers: Failed to get torrent from redis", err)
		return
	}
	user_keys, err := redis.Strings(user_keys_reply, nil)
	if err != nil {
		log.Println("initUsers: Failed to parse peer reply: ", err)
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

	log.Println(fmt.Sprintf("initUsers: Loaded %d users into memory", users))
}

// dbStatIndexer when running will periodically update the torrent sort indexes
func (tracker *Tracker) dbStatIndexer() {
	log.Println("dbStatIndexer: Background indexer started")
	r := db.Pool.Get()
	defer r.Close()

	key_leechers := "t:i:leechers"
	key_seeders := "t:i:seeders"
	key_snatches := "t:i:snatches"

	count := 0

	leecher_args := []uint64{}
	seeder_args := []uint64{}
	snatch_args := []uint64{}

	for {
		time.Sleep(time.Duration(conf.Config.IndexInterval) * time.Second)
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
	}
}

// syncWriter Handles writing out new data to the redis db in a queued manner
// Only items with the .InQueue flag set to false should be added.
// TODO channel as param
func (tracker *Tracker) syncWriter() {
	r := db.Pool.Get()
	defer r.Close()
	if r.Err() != nil {
		util.CaptureMessage(r.Err().Error())
		log.Println("SyncWriter: Failed to get redis conn:", r.Err().Error())
		return
	}
	for {
		select {
		case payload := <-db.SyncPayloadC:
			r.Do(payload.Command, payload.Args...)
		case user := <-SyncUserC:
			user.Sync(r)
			user.Lock()
			user.InQueue = false
			user.Unlock()
		case torrent := <-SyncTorrentC:
			torrent.Sync(r)
			torrent.Lock()
			torrent.InQueue = false
			torrent.Unlock()
		}
		err := r.Flush()
		if err != nil {
			log.Println("syncWriter: Failed to flush connection:", err)
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

	log.Println("IsValidClient: Got non-whitelisted client:", peer_id)
	return false
}
