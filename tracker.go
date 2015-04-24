package main

import (
	"fmt"
	"github.com/garyburd/redigo/redis"
	"log"
	"sync"
)

type Tracker struct {
	sync.RWMutex
	Torrents map[uint64]*Torrent
	Users    map[uint64]*User
}

// Load the models into memory from redis
func (t *Tracker) Initialize() error {
	log.Println("Initializing models in memory...")
	r := getRedisConnection()
	defer returnRedisConnection(r)

	initUsers(r)
	initTorrents(r)

	return nil
}

// Fetch a torrents data from the database and return a Torrent struct.
// If the torrent doesn't exist in the database a new skeleton Torrent
// instance will be returned.
func (t *Tracker) GetTorrentByID(r redis.Conn, torrent_id uint64, make_new bool) *Torrent {
	mika.RLock()
	torrent, cached := mika.Torrents[torrent_id]
	mika.RUnlock()
	if make_new && (!cached || torrent == nil) {
		torrent = fetchTorrent(r, torrent_id)
		if torrent != nil {
			mika.Lock()
			mika.Torrents[torrent_id] = torrent
			mika.Unlock()
			Debug("Added new torrent to in-memory cache:", torrent_id)
		} else {
			Debug("Failed to get torrent by id, no entry")
		}
	}
	return torrent
}

// Fetch the stores torrent_id that corresponds to the info_hash supplied
// as a GET value. If the info_hash doesn't return an id we consider the torrent
// either soft-deleted or non-existent
func (t *Tracker) GetTorrentByInfoHash(r redis.Conn, info_hash string, make_new bool) *Torrent {
	torrent_id_reply, err := r.Do("GET", fmt.Sprintf("t:info_hash:%x", info_hash))
	if err != nil {
		log.Println("Failed to execute torrent_id query", err)
		return nil
	}
	if torrent_id_reply == nil {
		return nil
	}
	torrent_id, tid_err := redis.Uint64(torrent_id_reply, nil)
	if tid_err != nil {
		log.Println("Failed to parse torrent_id reply", tid_err)
		return nil
	}
	return t.GetTorrentByID(r, torrent_id, make_new)
}
