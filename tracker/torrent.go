package tracker

import (
	"fmt"
	"github.com/garyburd/redigo/redis"
	"log"
	"sync"
	"git.totdev.in/totv/mika/db"
	"git.totdev.in/totv/mika/util"
)

type Torrent struct {
	db.Queued
	sync.RWMutex
	Name            string  `redis:"name" json:"name"`
	TorrentID       uint64  `redis:"torrent_id" json:"torrent_id"`
	InfoHash        string  `redis:"info_hash" json:"info_hash"`
	Seeders         int16   `redis:"seeders" json:"seeders"`
	Leechers        int16   `redis:"leechers" json:"leechers"`
	Snatches        int16   `redis:"snatches" json:"snatches"`
	Announces       uint64  `redis:"announces" json:"announces"`
	Uploaded        uint64  `redis:"uploaded" json:"uploaded"`
	Downloaded      uint64  `redis:"downloaded" json:"downloaded"`
	TorrentKey      string  `redis:"-" json:"-"`
	TorrentPeersKey string  `redis:"-" json:"-"`
	Enabled         bool    `redis:"enabled" json:"enabled"`
	Reason          string  `redis:"reason" json:"reason"`
	Peers           []*Peer `redis:"-" json:"-"`
	MultiUp         float64 `redis:"multi_up" json:"-"`
	MultiDn         float64 `redis:"multi_dn" json:"-"`
}

func (torrent *Torrent) Update(announce *AnnounceRequest, upload_diff, download_diff uint64) {
	s, l := torrent.PeerCounts()
	torrent.Lock()
	torrent.Announces++
	torrent.Uploaded += upload_diff
	torrent.Downloaded += download_diff
	torrent.Seeders = s
	torrent.Leechers = l
	if announce.Event == COMPLETED {
		torrent.Snatches++
	}
	torrent.Unlock()
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
		"name", torrent.Name,
		"multi_up", torrent.MultiUp,
		"multi_dn", torrent.MultiDn,
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
		peer_reply, err := r.Do("HGETALL", fmt.Sprintf("t:t:%s:%s", torrent.InfoHash, peer_id))
		if err != nil {
			log.Println("GetPeer: Error executing peer fetch query: ", err)
			return nil, err
		}
		peer, err = MakePeer(peer_reply, torrent.TorrentID, torrent.InfoHash, peer_id)
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
	torrent.Peers = append(torrent.Peers, peer)

	v, err := r.Do("SADD", fmt.Sprintf("t:tp:%s", torrent.InfoHash), peer.PeerID)
	if err != nil {
		log.Println("AddPeer: Error executing peer fetch query: ", err)
		return false
	}
	if v == "0" {
		log.Println("AddPeer: Tried to add peer to set with existing element")
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

	r.Send("SREM", fmt.Sprintf("t:tp:%s", torrent.InfoHash), peer.PeerID)
	r.Send("HSET", peer.KeyPeer, "active", 0)
	peer.Active = false
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
	torrent.RLock()
	defer torrent.RUnlock()
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
	return torrent.Peers[0:util.UMin(uint64(len(torrent.Peers)), uint64(max_peers))]
}
