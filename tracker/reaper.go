package tracker

import (
	"git.totdev.in/totv/mika/db"
	"git.totdev.in/totv/mika/util"
	log "github.com/Sirupsen/logrus"
	"github.com/garyburd/redigo/redis"
	"strings"
)

// Will mark a torrent peer as inactive and remove them
// from the torrents active peer_id set
func (tracker *Tracker) ReapPeer(info_hash, peer_id string) {
	r := db.Pool.Get()
	defer r.Close()
	if r.Err() != nil {
		util.CaptureMessage(r.Err().Error())
		log.WithFields(log.Fields{
			"peer_id":   peer_id[0:6],
			"info_hash": info_hash[0:6],
			"fn":        "ReapPeer",
		}).Errorln("Failed to get redis conn:", r.Err().Error())
		return
	}

	log.WithFields(log.Fields{"peer_id": peer_id[0:6], "info_hash": info_hash}).Info("Reaping peer")

	torrent := tracker.FindTorrentByInfoHash(info_hash)
	if torrent == nil {
		log.WithFields(log.Fields{
			"peer_id":   peer_id[0:6],
			"info_hash": info_hash,
			"fn":        "ReapPeer",
		}).Errorln("Failed to fetch torrent while reaping")
		return
	}

	// Fetch before we set active to 0
	peer := torrent.findPeer(peer_id)
	if peer == nil {
		log.WithFields(log.Fields{
			"peer_id":   peer_id[0:6],
			"info_hash": info_hash,
			"fn":        "ReapPeer",
		}).Errorln("Failed to fetch peer while reaping")
		return
	}
	torrent.DelPeer(r, peer)

	r.Send("SREM", peer.User.KeyActive, peer.PeerID)

	r.Flush()
	v, err := r.Receive()
	if err != nil {
		log.WithFields(log.Fields{
			"peer_id":   peer_id[0:6],
			"info_hash": info_hash,
			"fn":        "ReapPeer",
		}).Errorln("Tried to remove non-existant peer: ", err.Error())
	} else if v == "1" {
		log.WithFields(log.Fields{
			"peer_id":   peer_id[0:6],
			"info_hash": info_hash,
			"fn":        "ReapPeer",
		}).Info("Reaped peer successfully")
	}
}

// peerStalker is a goroutine that will watch for peer key expiry events and
// act on them, removing them from the active peer lists
func (tracker *Tracker) peerStalker() {
	r := db.Pool.Get()
	defer r.Close()
	if r.Err() != nil {
		util.CaptureMessage(r.Err().Error())
		log.WithFields(log.Fields{
			"fn": "peerStalker",
		}).Debug("Reaper cannot connect to redis: ", r.Err().Error())
		return
	}

	psc := redis.PubSubConn{Conn: r}
	psc.Subscribe("__keyevent@0__:expired")
	for {
		switch v := psc.Receive().(type) {

		case redis.Message:
			log.WithFields(log.Fields{
				"key": string(v.Data),
				"fn":  "peerStalker",
			}).Debug("Got key expire event")
			p := strings.SplitN(string(v.Data[:]), ":", 5)
			if len(p) >= 4 {
				tracker.ReapPeer(p[2], p[3])
			}

		case redis.Subscription:
			log.WithFields(log.Fields{
				"channel": v.Channel,
				"fn":      "peerStalker",
			}).Info("Subscribed successfully to key expiry channel")

		case error:
			log.WithFields(log.Fields{
				"fn": "peerStalker",
			}).Error(v.Error())
		}
	}
}
