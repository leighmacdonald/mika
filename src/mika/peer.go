package main

import (
	"bytes"
	"fmt"
	"github.com/garyburd/redigo/redis"
	"log"
	"net"
	"strings"
)

type Peer struct {
	SpeedUP       float64 `redis:"speed_up"`
	SpeedDN       float64 `redis:"speed_dj"`
	Uploaded      uint64  `redis:"uploaded"`
	Downloaded    uint64  `redis:"downloaded"`
	Corrupt       uint64  `redis:"corrupt"`
	IP            string  `redis:"ip"`
	Port          uint64  `redis:"port"`
	Left          uint64  `redis:"left"`
	Announces     uint64  `redis:"announces"`
	TotalTime     uint32  `redis:"total_time"`
	AnnounceLast  int32   `redis:"last_announce"`
	AnnounceFirst int32   `redis:"first_announce"`
	New           bool    `redis:"new"`
	PeerID        string  `redis:"peer_id"`
	Active        bool    `redis:"active"`
	UserID        uint64  `redis:"user_id"`
}

// Generate a compact peer field array containing the byte representations
// of a peers IP+Port appended to each other
func makeCompactPeers(peers []Peer, skip_id string) []byte {
	var out_buf bytes.Buffer
	for _, peer := range peers {
		if peer.Port <= 0 {
			// FIXME Why does empty peer exist with 0 port??
			continue
		}
		if peer.PeerID == skip_id {
			continue
		}

		out_buf.Write(net.ParseIP(peer.IP).To4())
		out_buf.Write([]byte{byte(peer.Port >> 8), byte(peer.Port & 0xff)})
	}
	return out_buf.Bytes()
}

// Get an array of peers for a supplied torrent_id
func GetPeers(r redis.Conn, torrent_id uint64, max_peers int) []Peer {
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

// Generate a new instance of a peer from the redis reply if data is contained
// within, otherwise just return a default value peer
func makePeer(redis_reply interface{}) (Peer, error) {
	peer := Peer{
		Active:        false,
		Announces:     0,
		SpeedUP:       0,
		SpeedDN:       0,
		Uploaded:      0,
		Downloaded:    0,
		Left:          0,
		Corrupt:       0,
		IP:            "127.0.0.1",
		Port:          0,
		AnnounceFirst: unixtime(),
		AnnounceLast:  unixtime(),
		TotalTime:     0,
		UserID:        0,
		New:           true,
	}

	values, err := redis.Values(redis_reply, nil)
	if err != nil {
		log.Println("Failed to parse peer reply: ", err)
		return peer, err_parse_reply
	}
	if values != nil {
		err := redis.ScanStruct(values, &peer)
		if err != nil {
			log.Println("Failed to fetch peer: ", err)
			return peer, err_cast_reply
		} else {
			peer.Announces += 1
			peer.New = false
		}
	}
	return peer, nil

}

// Fetch an existing peers data if it exists, other wise generate a
// new peer with default data values. The data is parsed into a Peer
// struct and returned.
func GetPeer(r redis.Conn, torrent_id uint64, peer_id string) (Peer, error) {
	peer_reply, err := r.Do("HGETALL", fmt.Sprintf("t:t:%d:%s", torrent_id, peer_id))
	if err != nil {
		log.Println("Error executing peer fetch query: ", err)
	}
	return makePeer(peer_reply)
}

// Add a peer to a torrents active peer_id list
func AddPeer(r redis.Conn, torrent_id uint64, peer_id string) bool {
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
func DelPeer(r redis.Conn, torrent_id uint64, peer_id string) bool {
	_, err := r.Do("SREM", fmt.Sprintf("t:t:%s:p", torrent_id), peer_id)
	if err != nil {
		log.Println("Error executing peer fetch query: ", err)
		return false
	}
	// Mark inactive?
	//r.Do("DEL", fmt.Sprintf("t:t:%d:p:%s", torrent_id, peer_id))
	return true
}

// Fetch a user_id from the supplied passkey. A return value
// of 0 denotes a non-existing or disabled user_id
func GetUserID(r redis.Conn, passkey string) uint64 {
	user_id_reply, err := r.Do("GET", fmt.Sprintf("t:user:%s", passkey))
	if err != nil {
		log.Println(err)
		return 0
	}
	user_id, err_b := redis.Uint64(user_id_reply, nil)
	if err_b != nil {
		log.Println("Failed to find user", err_b)
		return 0
	}
	return user_id
}

// Checked if the clients peer_id prefix matches the client prefixes
// stored in the white lists
func IsValidClient(r redis.Conn, peer_id string) bool {
	a, err := r.Do("HKEYS", "t:whitelist")

	if err != nil {
		log.Println(err)
		return false
	}
	clients, err := redis.Strings(a, nil)
	for _, client_prefix := range clients {
		if strings.HasPrefix(peer_id, client_prefix) {
			return true
		}
	}
	log.Println("Got non-whitelisted client:", peer_id)
	return false
}
