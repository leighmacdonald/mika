package main

import (
	"fmt"
	"github.com/garyburd/redigo/redis"
	"log"
	"strconv"
	"strings"
	"sync"
)

type Tracker struct {
	sync.RWMutex
	Torrents map[string]*Torrent
	Users    map[uint64]*User
}

// Load the models into memory from redis
func (t *Tracker) Initialize() error {
	log.Println("Initializing models in memory...")
	r := pool.Get()
	defer r.Close()

	t.initWhitelist(r)
	t.initUsers(r)
	t.initTorrents(r)

	return nil
}

// Fetch a torrents data from the database and return a Torrent struct.
// If the torrent doesn't exist in the database a new skeleton Torrent
// instance will be returned.
func (t *Tracker) GetTorrentByID(r redis.Conn, torrent_id uint64, make_new bool) *Torrent {
	mika.RLock()
	defer mika.RUnlock()
	for _, torrent := range t.Torrents {
		if torrent.TorrentID == torrent_id {
			return torrent
		}
	}
	return nil
}

// Fetch the stores torrent_id that corresponds to the info_hash supplied
// as a GET value. If the info_hash doesn't return an id we consider the torrent
// either soft-deleted or non-existent
func (t *Tracker) GetTorrentByInfoHash(r redis.Conn, info_hash string, make_new bool) *Torrent {
	mika.RLock()
	defer mika.RUnlock()
	torrent, exists := t.Torrents[info_hash]
	if exists {
		return torrent
	}
	if make_new {
		torrent = t.FetchTorrent(r, info_hash)
		if torrent == nil {
			return nil
		}
		mika.Lock()
		t.Torrents[info_hash] = torrent
		mika.Unlock()
		Debug("Added new torrent to in-memory cache:", info_hash)
	}
	return nil
}

func (t *Tracker) FetchTorrent(r redis.Conn, info_hash string) *Torrent {
	// Make new struct to use for cache
	torrent := &Torrent{
		InfoHash: info_hash,
		Name:     "",
		Enabled:  true,
		Peers:    []*Peer{},
	}

	exists_reply, err := r.Do("EXISTS", fmt.Sprintf("t:t:%s", info_hash))
	exists, err := redis.Bool(exists_reply, err)
	if err != nil {
		exists = false
	}
	if exists {
		torrent_reply, err := r.Do("HGETALL", fmt.Sprintf("t:t:%s", info_hash))
		if err != nil {
			log.Println("Failed to get torrent from redis", err)
			return nil
		}

		values, err := redis.Values(torrent_reply, nil)
		if err != nil {
			log.Println("Failed to parse torrent reply: ", err)
			return nil
		}

		err = redis.ScanStruct(values, torrent)
		if err != nil {
			log.Println("Torrent scanstruct failure", err)
			return nil
		}

		if torrent.TorrentID == 0 {
			Debug("Trying to fetch info hash without valid key:", info_hash)
			r.Do("DEL", fmt.Sprintf("t:t:%s", torrent.InfoHash))
			return nil
		}
	}
	// Reset counts since we cant guarantee the accuracy after restart
	// TODO allow reloading of peer/seed counts if a maximum time has not elapsed
	// since the last startup.
	torrent.Seeders = 0
	torrent.Leechers = 0

	// Make these once and save the results in mem
	torrent.TorrentKey = fmt.Sprintf("t:t:%s", info_hash)
	torrent.TorrentPeersKey = fmt.Sprintf("t:t:%s:p", info_hash)

	return torrent
}

func (t *Tracker) initWhitelist(r redis.Conn) {
	whitelist = []string{}
	a, err := r.Do("HKEYS", "t:whitelist")

	if err != nil {
		log.Println(err)
		return
	}
	whitelist, err = redis.Strings(a, nil)
	log.Println(fmt.Sprintf("Loaded %d whitelist clients", len(whitelist)))
}

func (t *Tracker) initTorrents(r redis.Conn) {
	torrent_keys_reply, err := r.Do("KEYS", "t:t:*")
	if err != nil {
		log.Println("Failed to get torrent from redis", err)
		return
	}
	torrent_keys, err := redis.Strings(torrent_keys_reply, nil)
	if err != nil {
		log.Println("Failed to parse torrent keys reply: ", err)
		return
	}
	torrents := 0
	mika.Lock()
	defer mika.Unlock()
	for _, torrent_key := range torrent_keys {
		pcs := strings.SplitN(torrent_key, ":", 3)
		if len(pcs) != 3 {
			continue
		}
		torrent := t.FetchTorrent(r, pcs[2])
		if torrent != nil {
			mika.Torrents[torrent.InfoHash] = torrent
			torrents++
		} else {
			// Drop keys we don't have valid id's'for
			r.Do("DEL", torrent_key)
		}
	}

	log.Println(fmt.Sprintf("Loaded %d torrents into memory", torrents))
}

// Load all the users into memory
func (t *Tracker) initUsers(r redis.Conn) {
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
