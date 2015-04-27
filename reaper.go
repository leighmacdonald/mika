package main

import (
	"fmt"
	"github.com/garyburd/redigo/redis"
	"log"
	"strings"
)

// Will mark a torrent peer as inactive and remove them
// from the torrents active peer_id set
func ReapPeer(info_hash, peer_id string) {
	r := pool.Get()
	defer r.Close()
	if r.Err() != nil {
		CaptureMessage(r.Err().Error())
		log.Println("ReapPeer: Reaper redis conn:", r.Err().Error())
		return
	}
	Debug("Reaping peer:", info_hash, peer_id)

	torrent := mika.GetTorrentByInfoHash(r, info_hash, false)
	if torrent == nil {
		log.Println("ReapPeer:", "Failed to fetch torrent while reaping", fmt.Sprintf("%s [%s]", info_hash, peer_id))
		return
	}

	// Fetch before we set active to 0
	peer, err := torrent.GetPeer(r, peer_id)
	if err != nil {
		log.Println("ReapPeer: Failed to fetch peer while reaping", fmt.Sprintf("%s [%s]", info_hash, peer_id))
		return
	}

	torrent.DelPeer(r, peer)

	queued := 2
	r.Send("SREM", fmt.Sprintf("t:t:%s:p", info_hash), peer_id)
	r.Send("HSET", fmt.Sprintf("t:t:%s:p:%s", info_hash, peer_id), "active", 0)
	if peer.Active {
		if peer.Left > 0 {
			r.Send("HINCRBY", fmt.Sprintf("t:t:%s", info_hash), "leechers", -1)
		} else {
			r.Send("HINCRBY", fmt.Sprintf("t:t:%s", info_hash), "seeders", -1)
		}
		queued += 1
	}
	if peer.IsHNR() {
		peer.AddHNR(r, torrent.TorrentID)
	}

	r.Flush()
	v, err := r.Receive()
	queued -= 1
	if err != nil {
		log.Println("ReapPeer: Tried to remove non-existant peer: ", info_hash, peer_id)
	}
	if v == "1" {
		Debug("ReapPeer: Reaped peer successfully: ", peer_id)
	}

	// all needed i think, must match r.Send count?
	for i := 0; i < queued; i++ {
		r.Receive()
	}

}

// This is a goroutine that will watch for peer key expiry events and
// act on them, removing them from the active peer lists
func peerStalker() {
	r := pool.Get()
	defer r.Close()
	if r.Err() != nil {
		CaptureMessage(r.Err().Error())
		log.Println("peerStalker: Reaper cannot connect to redis", r.Err().Error())
		return
	}

	psc := redis.PubSubConn{Conn: r}
	psc.Subscribe("__keyevent@0__:expired")
	for {
		switch v := psc.Receive().(type) {

		case redis.Message:
			p := strings.SplitN(string(v.Data[:]), ":", 5)
			if len(p) > 4 {
				ReapPeer(p[2], p[3])
			}

		case redis.Subscription:
			Debug("peerStalker: Subscribed to channel:", v.Channel)

		case error:
			log.Println("peerStalker: Subscriber error: ", v.Error())
		}
	}
}
