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
	SpeedUP           float64 `redis:"speed_up"`
	SpeedDN           float64 `redis:"speed_dj"`
	Uploaded          uint64  `redis:"uploaded"`
	Downloaded        uint64  `redis:"downloaded"`
	Corrupt           uint64  `redis:"corrupt"`
	IP                string  `redis:"ip"`
	Port              uint64  `redis:"port"`
	Left              uint64  `redis:"left"`
	Announces         uint64  `redis:"announces"`
	TotalTime         uint32  `redis:"total_time"`
	AnnounceLast      int32   `redis:"last_announce"`
	AnnounceFirst     int32   `redis:"first_announce"`
	New               bool    `redis:"new"`
	PeerID            string  `redis:"peer_id"`
	Active            bool    `redis:"active"`
	UserID            uint64  `redis:"user_id"`
	TorrentID         uint64  `redis:"torrent_id"`
	KeyPeer           string  `redis:"-"`
	KeyUserActive     string  `redis:"-"`
	KeyUserIncomplete string  `redis:"-"`
	KeyUserComplete   string  `redis:"-"`
	KeyUserHNR        string  `redis:"-"`
}

// Update the stored values with the data from an announce
func (peer *Peer) Update(announce *AnnounceRequest) {
	cur_time := unixtime()
	peer.PeerID = announce.PeerID
	peer.Announces++
	// Change to int or byte?
	peer.Uploaded += announce.Uploaded
	peer.Downloaded += announce.Downloaded
	peer.IP = announce.IPv4.String()
	peer.Corrupt += announce.Corrupt
	peer.Left = announce.Left
	peer.SpeedUP = estSpeed(peer.AnnounceLast, cur_time, announce.Uploaded)
	peer.SpeedDN = estSpeed(peer.AnnounceLast, cur_time, announce.Downloaded)
	if announce.Event == STOPPED {
		peer.Active = false
	} else {
		peer.Active = true
	}
	// Must be active to have a real time delta
	if peer.Active && peer.AnnounceLast > 0 {
		time_diff := uint64(unixtime() - peer.AnnounceLast)
		// Ignore long periods of inactivity
		if time_diff < (uint64(config.AnnInterval) * 4) {
			peer.TotalTime += uint32(time_diff)
		}
	}
}

func (peer *Peer) SetUserID(user_id uint64) {
	peer.UserID = user_id
	peer.KeyUserActive = fmt.Sprintf("t:u:%d:active", user_id)
	peer.KeyUserIncomplete = fmt.Sprintf("t:u:%d:incomplete", user_id)
	peer.KeyUserComplete = fmt.Sprintf("t:u:%d:complete", user_id)
	peer.KeyUserHNR = fmt.Sprintf("t:u:%d:hnr", user_id)
}

func (peer *Peer) Sync(r redis.Conn) {
	r.Send(
		"HMSET", peer.KeyPeer,
		"ip", peer.IP,
		"port", peer.Port,
		"left", peer.Left,
		"first_announce", peer.AnnounceFirst,
		"last_announce", peer.AnnounceLast,
		"total_time", peer.TotalTime,
		"speed_up", peer.SpeedUP,
		"speed_dn", peer.SpeedDN,
		"active", peer.Active,
		"uploaded", peer.Uploaded,
		"downloaded", peer.Downloaded,
		"corrupt", peer.Corrupt,
		"user_id", peer.UserID, // Shouldn't need to be here
		"peer_id", peer.PeerID, // Shouldn't need to be here
		"torrent_id", peer.TorrentID, // Shouldn't need to be here
	)

}

func (peer *Peer) IsSeeder() bool {
	return peer.Left > 0
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

// Generate a new instance of a peer from the redis reply if data is contained
// within, otherwise just return a default value peer
func makePeer(redis_reply interface{}, torrent_id uint64, peer_id string) (Peer, error) {
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
		TorrentID:     torrent_id,
		KeyPeer:       fmt.Sprintf("t:t:%d:%s", torrent_id, peer_id),
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
			peer.New = false
		}
	}
	return peer, nil
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
