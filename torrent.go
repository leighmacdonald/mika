package main

import (
	"fmt"
	"github.com/garyburd/redigo/redis"
	"log"
)

type Torrent struct {
	TorrentID       uint64           `redis:"torrent_id"`
	Seeders         int16            `redis:"seeders"`
	Leechers        int16            `redis:"leechers"`
	Snatches        int16            `redis:"snatches"`
	Announces       uint64           `redis:"announces"`
	Uploaded        uint64           `redis:"uploaded"`
	Downloaded      uint64           `redis:"downloaded"`
	TorrentKey      string           `redis:"-"`
	TorrentPeersKey string           `redis:"-"`
	Peers           map[string]*Peer `redis:"-"`
}

func (torrent *Torrent) Update(announce *AnnounceRequest) {
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
	)
}

// Fetch an existing peers data if it exists, other wise generate a
// new peer with default data values. The data is parsed into a Peer
// struct and returned.
func (torrent *Torrent) GetPeer(r redis.Conn, peer_id string) (*Peer, error) {
	mika.RLock()
	peer, cached := torrent.Peers[peer_id]
	mika.RUnlock()
	if !cached {
		peer_reply, err := r.Do("HGETALL", fmt.Sprintf("t:t:%d:%s", torrent.TorrentID, peer_id))
		if err != nil {
			log.Println("Error executing peer fetch query: ", err)
			return nil, err
		}
		peer, err = makePeer(peer_reply, torrent.TorrentID, peer_id)
		if err != nil {
			return nil, err
		}
		mika.Lock()
		torrent.Peers[peer_id] = peer
		mika.Unlock()
		Debug("New peer in memory:", peer_id)
	}
	return peer, nil
}

// Add a peer to a torrents active peer_id list
func (torrent *Torrent) AddPeer(r redis.Conn, peer *Peer) bool {
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

	r.Send("SREM", fmt.Sprintf("t:t:%d:p", torrent.TorrentID), peer.PeerID)

	r.Send("HSET", peer.KeyPeer, "active", 0)

	return true
}

// Get an array of peers for a supplied torrent_id
func (torrent *Torrent) GetPeers(r redis.Conn, max_peers int) []*Peer {
	peers_reply, err := r.Do("SMEMBERS", torrent.TorrentPeersKey)
	if err != nil || peers_reply == nil {
		log.Println("Error fetching peers_resply", err)
		return nil
	}
	peer_ids, err := redis.Strings(peers_reply, nil)
	if err != nil {
		log.Println("Error parsing peers_resply", err)
		return nil
	}

	known_peers := len(peer_ids)
	if known_peers > max_peers {
		known_peers = max_peers
	}

	// TODO check in-memory peers.

	for _, peer_id := range peer_ids[0:known_peers] {
		r.Send("HGETALL", fmt.Sprintf("t:t:%d:%s", torrent.TorrentID, peer_id))
	}
	r.Flush()
	peers := make([]*Peer, known_peers)

	for i := 1; i <= known_peers; i++ {
		peer_reply, err := r.Receive()
		if err != nil {
			log.Println(err)
		} else {
			peer, err := makePeer(peer_reply, torrent.TorrentID, peer_ids[i-1])
			if err != nil {
				log.Println("Error trying to make new peer", err)
			} else {
				peers = append(peers, peer)
			}
		}
	}
	return peers
}
