package main

import (
	"fmt"
	"github.com/garyburd/redigo/redis"
	"log"
)

type Torrent struct {
	TorrentID  uint64 `redis:"torrent_id"`
	Seeders    int16  `redis:"seeders"`
	Leechers   int16  `redis:"leechers"`
	Snatches   int16  `redis:"snatches"`
	Announces  uint64 `redis:"announces"`
	Uploaded   uint64 `redis:"uploaded"`
	Downloaded uint64 `redis:"downloaded"`
	Peers      []*Peer `redis:"-"`
}

// Fetch an existing peers data if it exists, other wise generate a
// new peer with default data values. The data is parsed into a Peer
// struct and returned.
func (torrent *Torrent) GetPeer(r redis.Conn, peer_id string) (Peer, error) {
	peer_reply, err := r.Do("HGETALL", fmt.Sprintf("t:t:%d:%s", torrent.TorrentID, peer_id))
	if err != nil {
		log.Println("Error executing peer fetch query: ", err)
	}
	return makePeer(peer_reply)
}

// Add a peer to a torrents active peer_id list
func (torrent *Torrent) AddPeer(r redis.Conn, torrent_id uint64, peer_id string) bool {
	v, err := r.Do("SADD", fmt.Sprintf("t:t:%d:p", torrent_id), peer_id)
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
func (torrent *Torrent) DelPeer(r redis.Conn, torrent_id uint64, peer_id string) bool {
	_, err := r.Do("SREM", fmt.Sprintf("t:t:%s:p", torrent_id), peer_id)
	if err != nil {
		log.Println("Error executing peer fetch query: ", err)
		return false
	}
	// Mark inactive?
	//r.Do("DEL", fmt.Sprintf("t:t:%d:p:%s", torrent_id, peer_id))
	return true
}

// Get an array of peers for a supplied torrent_id
func (torrent *Torrent) GetPeers(r redis.Conn, torrent_id uint64, max_peers int) []Peer {
	peers_reply, err := r.Do("SMEMBERS", fmt.Sprintf("t:t:%d:p", torrent_id))
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

	for _, peer_id := range peer_ids[0:known_peers] {
		r.Send("HGETALL", fmt.Sprintf("t:t:%d:%s", torrent_id, peer_id))
	}
	r.Flush()
	peers := make([]Peer, known_peers)

	for i := 1; i <= known_peers; i++ {
		peer_reply, err := r.Receive()
		if err != nil {
			log.Println(err)
		} else {
			peer, err := makePeer(peer_reply)
			if err != nil {
				log.Println("Error trying to make new peer", err)
			} else {
				peers = append(peers, peer)
			}
		}
	}

	return peers
}
