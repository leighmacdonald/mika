package main

import (
	"fmt"
	"github.com/garyburd/redigo/redis"
	"log"
	"strconv"
	"strings"
	"sync"
)

type Torrent struct {
	Queued
	sync.RWMutex
	TorrentID       uint64  `redis:"torrent_id" json:"torrent_id"`
	InfoHash        string  `redis:"info_hash" json:"info_hash"`
	Seeders         int16   `redis:"seeders" json:"seeders"`
	Leechers        int16   `redis:"leechers" json:"leechers"`
	Snatches        int16   `redis:"snatches" json:"snatches"`
	Announces       uint64  `redis:"announces" json:"announces" `
	Uploaded        uint64  `redis:"uploaded" json:"uploaded"`
	Downloaded      uint64  `redis:"downloaded" json:"downloaded"`
	TorrentKey      string  `redis:"-" json:"key_torrent"`
	TorrentPeersKey string  `redis:"-" json:"key_peers"`
	Enabled         bool    `redis:"enabled" json:"enabled"`
	Reason          string  `redis:"reason" json:"reason"`
	Peers           []*Peer `redis:"-" json:"peers"`
}

func (torrent *Torrent) Update(announce *AnnounceRequest) {
	torrent.Lock()
	defer torrent.Unlock()
	torrent.Announces++
	torrent.Uploaded += announce.Uploaded
	torrent.Downloaded += announce.Downloaded
}

func (torrent *Torrent) Sync(r redis.Conn) {
	r.Send(
		"HMSET", torrent.TorrentKey,
		"torrent_id", torrent.TorrentID,
		"seeders", torrent.Seeders,
		"leechers", torrent.Leechers,
		"snatches", torrent.Snatches,
		"announces", torrent.Announces,
		"uploaded", torrent.Uploaded,
		"downloaded", torrent.Downloaded,
		"info_hash", torrent.InfoHash,
		"reason", torrent.Reason,
		"enabled", torrent.Enabled,
	)
}

func (torrent *Torrent) findPeer(peer_id string) *Peer {
	torrent.RLock()
	defer torrent.RUnlock()
	for _, peer := range torrent.Peers {
		if peer.PeerID == peer_id {
			return peer
		}
	}
	return nil
}

func (torrent *Torrent) Delete(reason string) {
	torrent.Enabled = false
	torrent.Reason = reason
}

func (torrent *Torrent) DelReason() string {
	if torrent.Reason == "" {
		return "Torrent deleted"
	} else {
		return torrent.Reason
	}
}

// Fetch an existing peers data if it exists, other wise generate a
// new peer with default data values. The data is parsed into a Peer
// struct and returned.
func (torrent *Torrent) GetPeer(r redis.Conn, peer_id string) (*Peer, error) {
	peer := torrent.findPeer(peer_id)
	if peer == nil {
		peer_reply, err := r.Do("HGETALL", fmt.Sprintf("t:t:%d:%s", torrent.TorrentID, peer_id))
		if err != nil {
			log.Println("Error executing peer fetch query: ", err)
			return nil, err
		}
		peer, err = makePeer(peer_reply, torrent.TorrentID, peer_id)
		if err != nil {
			return nil, err
		}

		torrent.Lock()
		torrent.Peers = append(torrent.Peers, peer)
		torrent.Unlock()
	}
	return peer, nil
}

// Add a peer to a torrents active peer_id list
func (torrent *Torrent) AddPeer(r redis.Conn, peer *Peer) bool {
	torrent.Lock()
	torrent.Peers = append(torrent.Peers, peer)
	torrent.Unlock()

	v, err := r.Do("SADD", fmt.Sprintf("t:t:%d:p", torrent.TorrentID), peer.PeerID)
	if err != nil {
		log.Println("Error executing peer fetch query: ", err)
		return false
	}
	if v == "0" {
		log.Println("Tried to add peer to set with existing element")
	}
	return true
}

// Remove a peer from a torrents active peer_id list
func (torrent *Torrent) DelPeer(r redis.Conn, peer *Peer) bool {
	for i, tor_peer := range torrent.Peers {
		if tor_peer == peer {
			if len(torrent.Peers) == 1 {
				torrent.Peers = nil
			} else {
				torrent.Peers = append(torrent.Peers[:i], torrent.Peers[i+1:]...)
			}
			break
		}
	}

	r.Send("SREM", fmt.Sprintf("t:t:%d:p", torrent.TorrentID), peer.PeerID)
	r.Send("HSET", peer.KeyPeer, "active", 0)

	return true
}

func (torrent *Torrent) HasPeer(peer *Peer) bool {
	for _, p := range torrent.Peers {
		if peer == p {
			return true
		}
	}
	return false
}

func (torrent *Torrent) PeerCounts() (int16, int16) {
	s, l := 0, 0
	for _, p := range torrent.Peers {
		if p.IsSeeder() {
			s++
		} else {
			l++
		}
	}
	return int16(s), int16(l)
}

// Get an array of peers for the torrent
func (torrent *Torrent) GetPeers(r redis.Conn, max_peers int) []*Peer {
	return torrent.Peers[0:UMin(uint64(len(torrent.Peers)), uint64(max_peers))]
}

func initTorrents(r redis.Conn) {
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
		torrent_id, err := strconv.ParseUint(pcs[2], 10, 64)
		if err != nil {
			// Other key type probably
			continue
		}
		user := fetchTorrent(r, torrent_id)
		if user != nil {
			mika.Torrents[torrent_id] = user
			torrents++
		}
	}

	log.Println(fmt.Sprintf("Loaded %d torrents into memory", torrents))
}

func fetchTorrent(r redis.Conn, torrent_id uint64) *Torrent {
	// Make new struct to use for cache
	torrent := &Torrent{
		TorrentID: torrent_id,
		InfoHash:  "",
		Enabled:   true,
		Peers:     []*Peer{},
	}

	torrent_reply, err := r.Do("HGETALL", fmt.Sprintf("t:t:%d", torrent_id))
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

	// Reset counts since we cant guarantee the accuracy after restart
	// TODO allow reloading of peer/seed counts if a maximum time has not elapsed
	// since the last startup.
	torrent.Seeders = 0
	torrent.Leechers = 0

	// Make these once and save the results in mem
	torrent.TorrentKey = fmt.Sprintf("t:t:%d", torrent_id)
	torrent.TorrentPeersKey = fmt.Sprintf("t:t:%d:p", torrent_id)

	return torrent
}
