package main

import (
	"fmt"
	"github.com/garyburd/redigo/redis"
	"log"
	"strconv"
	"strings"
)

// Will mark a torrent peer as inactive and remove them
// from the torrents active peer_id set
func ReapPeer(torrent_id, peer_id string) {
	r, redis_err := RedisConn()
	if redis_err != nil {
		CaptureMessage(redis_err.Error())
		log.Println("Reaper redis conn:", redis_err.Error())
		return
	}
	defer r.Close()
	Debug("Reaping peer:", torrent_id, peer_id)

	torrent_id_uint, err := strconv.ParseUint(torrent_id, 10, 64)
	if err != nil {
		log.Println("Failed to parse torrent id into uint64", err)
		return
	}

	torrent := mika.GetTorrentByID(r, torrent_id_uint)
	if torrent == nil {
		log.Println("Failed to fetch torrent while reaping")
		return
	}

	// Fetch before we set active to 0
	peer, err := torrent.GetPeer(r, peer_id)
	if err != nil {
		log.Println("Failed to fetch peer while reaping")
		return
	}

	torrent.DelPeer(r, peer)

	queued := 2
	r.Send("SREM", fmt.Sprintf("t:t:%s:p", torrent_id), peer_id)
	r.Send("HSET", fmt.Sprintf("t:t:%s:p:%s", torrent_id, peer_id), "active", 0)
	if peer.Active {
		if peer.Left > 0 {
			r.Send("HINCRBY", fmt.Sprintf("t:t:%s", torrent_id), "leechers", -1)
		} else {
			r.Send("HINCRBY", fmt.Sprintf("t:t:%s", torrent_id), "seeders", -1)
		}
		queued += 1
	}
	if peer.TotalTime < config.HNRThreshold {
		r.Send("SADD", fmt.Sprintf("t:u:%d:hnr", peer.UserID), torrent_id)
		Debug("Added HnR:", torrent_id, peer.UserID)
	}

	r.Flush()
	v, err := r.Receive()
	queued -= 1
	if err != nil {
		log.Println("Tried to remove non-existant peer: ", torrent_id, peer_id)
	}
	if v == "1" {
		Debug("Reaped peer successfully: ", peer_id)
	}

	// all needed i think, must match r.Send count?
	for i := 0; i < queued; i++ {
		r.Receive()
	}

}

// This is a goroutine that will watch for peer key expiry events and
// act on them, removing them from the active peer lists
func peerStalker() {
	r, err := RedisConn()
	if err != nil {
		CaptureMessage(err.Error())
		log.Println("Reaper cannot connect to redis", err.Error())
		return
	}
	defer r.Close()

	psc := redis.PubSubConn{Conn: r}
	psc.Subscribe("__keyevent@0__:expired")
	for {
		switch v := psc.Receive().(type) {

		case redis.Message:
			p := strings.SplitN(string(v.Data[:]), ":", 5)
			ReapPeer(p[2], p[3])

		case redis.Subscription:
			Debug("Subscribed to channel:", v.Channel)

		case error:
			log.Println("Subscriber error: ", v.Error())
		}
	}
}
