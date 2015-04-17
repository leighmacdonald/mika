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
}

// Fetch a torrents data from the database and return a Torrent struct.
// If the torrent doesn't exist in the database a new skeleton Torrent
// instance will be returned.
func (t *Tracker) GetTorrent(r redis.Conn, torrent_id uint64) (*Torrent, error) {
	mika.RLock()
	torrent, cached := mika.Torrents[torrent_id]
	mika.RUnlock()
	if !cached || torrent == nil {
		// Make new struct to use for cache
		torrent := Torrent{
			TorrentID:  torrent_id,
			Seeders:    0,
			Leechers:   0,
			Snatches:   0,
			Announces:  0,
			Uploaded:   0,
			Downloaded: 0,
			Peers:      make(map[string]*Peer),
		}

		torrent_reply, err := r.Do("HGETALL", fmt.Sprintf("t:t:%d", torrent_id))
		if err != nil {
			return nil, err
		}

		values, err := redis.Values(torrent_reply, nil)
		if err != nil {
			log.Println("Failed to parse peer reply: ", err)
			return nil, err
		}

		err = redis.ScanStruct(values, &torrent)
		if err != nil {
			return nil, err
		}

		// Make these once and save the results in mem
		torrent.TorrentKey = fmt.Sprintf("t:t:%d", torrent_id)
		torrent.TorrentPeersKey = fmt.Sprintf("t:t:%d:p", torrent_id)

		mika.Lock()
		mika.Torrents[torrent_id] = &torrent
		mika.Unlock()
		Debug("Added new torrent to in-memory cache:", torrent_id)
		return &torrent, nil
	}

	return torrent, nil
}

// Fetch the stores torrent_id that corresponds to the info_hash supplied
// as a GET value. If the info_hash doesn't return an id we consider the torrent
// either soft-deleted or non-existent
func (t *Tracker) GetTorrentID(r redis.Conn, info_hash string) uint64 {
	torrent_id_reply, err := r.Do("GET", fmt.Sprintf("t:info_hash:%x", info_hash))
	if err != nil {
		log.Println("Failed to execute torrent_id query", err)
		return 0
	}
	if torrent_id_reply == nil {
		log.Println("Invalid info hash")
		return 0
	}
	torrent_id, tid_err := redis.Uint64(torrent_id_reply, nil)
	if tid_err != nil {
		log.Println("Failed to parse torrent_id reply", tid_err)
		return 0
	}
	return torrent_id
}
