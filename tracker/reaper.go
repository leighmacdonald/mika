package tracker

import (
	"fmt"
	"git.totdev.in/totv/mika/db"
	"git.totdev.in/totv/mika/util"
	log "github.com/Sirupsen/logrus"
	"github.com/garyburd/redigo/redis"
	"strings"
)

// Will mark a torrent peer as inactive and remove them
// from the torrents active peer_id set
func (t *Tracker) ReapPeer(info_hash, peer_id string) {
	r := db.Pool.Get()
	defer r.Close()
	if r.Err() != nil {
		util.CaptureMessage(r.Err().Error())
		log.Println("ReapPeer: Reaper redis conn:", r.Err().Error())
		return
	}
	log.Debug("Reaping peer:", info_hash, peer_id)

	torrent := t.FindTorrentByInfoHash(info_hash)
	if torrent == nil {
		log.Error("ReapPeer: Failed to fetch torrent while reaping", fmt.Sprintf("%s [%s]", info_hash, peer_id[0:6]))
		return
	}

	// Fetch before we set active to 0
	peer := torrent.findPeer(peer_id)
	if peer == nil {
		log.Error("ReapPeer: Failed to fetch peer while reaping ", fmt.Sprintf("%s [%s]", info_hash, peer_id[0:6]))
		return
	}
	torrent.DelPeer(r, peer)

	r.Send("SREM", peer.User.KeyActive, peer.PeerID)

	r.Flush()
	v, err := r.Receive()
	if err != nil {
		log.Error("ReapPeer: Tried to remove non-existant peer: ", info_hash, peer_id[0:6])
	}
	if v == "1" {
		log.Debug("ReapPeer: Reaped peer successfully: ", peer_id)
	}
	peer.Active = false
	SyncPeerC <- peer
}

// This is a goroutine that will watch for peer key expiry events and
// act on them, removing them from the active peer lists
func (t *Tracker) peerStalker() {
	r := db.Pool.Get()
	defer r.Close()
	if r.Err() != nil {
		util.CaptureMessage(r.Err().Error())
		log.Println("peerStalker: Reaper cannot connect to redis", r.Err().Error())
		return
	}

	psc := redis.PubSubConn{Conn: r}
	psc.Subscribe("__keyevent@0__:expired")
	for {
		switch v := psc.Receive().(type) {

		case redis.Message:
			log.Debug(string(v.Data))
			p := strings.SplitN(string(v.Data[:]), ":", 5)
			if len(p) >= 4 {
				t.ReapPeer(p[2], p[3])
			}

		case redis.Subscription:
			log.Info("peerStalker: Subscribed to channel:", v.Channel)

		case error:
			log.Error("peerStalker: Subscriber error: ", v.Error())
		}
	}
}
