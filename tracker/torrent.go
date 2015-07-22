package tracker

import (
	"fmt"
	"git.totdev.in/totv/mika/util"
	log "github.com/Sirupsen/logrus"
	"github.com/garyburd/redigo/redis"
	"math"
	"strings"
	"sync"
)

type Torrent struct {
	sync.RWMutex
	Name            string  `redis:"name" json:"name"`
	TorrentID       uint64  `redis:"torrent_id" json:"torrent_id"`
	InfoHash        string  `redis:"info_hash" json:"info_hash"`
	Seeders         int     `redis:"seeders" json:"seeders"`
	Leechers        int     `redis:"leechers" json:"leechers"`
	Snatches        int16   `redis:"snatches" json:"snatches"`
	Announces       uint64  `redis:"announces" json:"announces"`
	Uploaded        uint64  `redis:"uploaded" json:"uploaded"`
	Downloaded      uint64  `redis:"downloaded" json:"downloaded"`
	TorrentKey      string  `redis:"-" json:"-"`
	TorrentPeersKey string  `redis:"-" json:"-"`
	Enabled         bool    `redis:"enabled" json:"enabled"`
	Reason          string  `redis:"reason" json:"reason"`
	Peers           []*Peer `redis:"-" json:"peers"`
	MultiUp         float64 `redis:"multi_up" json:"multi_up"`
	MultiDn         float64 `redis:"multi_dn" json:"multi_dn"`
}

type TorrentStats struct {
	TorrentID uint64 `json:"torrent_id"`
	InfoHash  string `json:"info_hash"`
	Seeders   int    `json:"seeders"`
	Leechers  int    `json:"leechers"`
	Completed int    `json:"completed"`
}

// NewTorrent allocates and returns a new Torrent instance pointer with all
// the minimum value required to operated in place
func NewTorrent(info_hash string, name string, torrent_id uint64) *Torrent {
	torrent := &Torrent{
		Name:            name,
		InfoHash:        strings.ToLower(info_hash),
		TorrentKey:      fmt.Sprintf("t:t:%s", info_hash),
		TorrentPeersKey: fmt.Sprintf("t:tpeers:%s", info_hash),
		Enabled:         true,
		Peers:           []*Peer{},
		TorrentID:       torrent_id,
		MultiUp:         1.0,
		MultiDn:         1.0,
	}
	return torrent
}

// MergeDB will pull the torrent details from redis and overwrite the currently
// stored valued in the Torrent instance. This should only be called when initializing
// a torrent for the first time in the applications lifetime, such as during initialization.
func (torrent *Torrent) MergeDB(r redis.Conn) error {
	torrent_reply, err := r.Do("HGETALL", torrent.TorrentKey)
	if err != nil {
		log.WithFields(log.Fields{
			"key": torrent.TorrentKey,
			"err": err.Error(),
			"fn":  "MergeDB",
		}).Error("Failed to get torrent from redis")
		return err
	}

	values, err := redis.Values(torrent_reply, nil)
	if err != nil {
		log.WithFields(log.Fields{
			"key": torrent.TorrentKey,
			"err": err.Error(),
			"fn":  "MergeDB",
		}).Error("Failed to parse torrent reply")
		return err
	}

	err = redis.ScanStruct(values, torrent)
	if err != nil {
		log.WithFields(log.Fields{
			"key": torrent.TorrentKey,
			"err": err.Error(),
			"fn":  "MergeDB",
		}).Error("Torrent scanstruct failure")
		return err
	}

	if torrent.TorrentID == 0 {
		log.WithFields(log.Fields{
			"key": torrent.TorrentKey,
			"err": err.Error(),
			"fn":  "MergeDB",
		}).Error("Trying to fetch info hash without valid key")
		r.Do("DEL", fmt.Sprintf("t:t:%s", torrent.InfoHash))
	}
	return nil
}

// Update handles updating the stored values according to a users announce request
func (torrent *Torrent) Update(announce *AnnounceRequest) {
	s, l := torrent.PeerCounts()
	torrent.Lock()
	torrent.Announces++
	torrent.Seeders = s
	torrent.Leechers = l
	if announce.Event == COMPLETED {
		torrent.Snatches++
		log.WithFields(log.Fields{
			"fn":        "Update",
			"name":      torrent.Name,
			"info_hash": torrent.InfoHash,
			"snatches":  torrent.Snatches,
		}).Info("Snatch registered")
	}
	torrent.Unlock()
}

// Sync writes the torrent data to redis for permanent storage
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

// findPeer locates a peer currently in the torrents swarm and returns
// the peer pointer if found.
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

// Delete marks the torrent as disabled and prevent any further action
// against it from normal user requests. The torrent is not removed from
// active memory when deleted.
func (torrent *Torrent) Delete(reason string) {
	torrent.Lock()
	torrent.Enabled = false
	torrent.Reason = reason
	torrent.Unlock()
}

// DelReason returns the deletion reason if it was set for a torrent. If not
// set a default reason is returned
func (torrent *Torrent) DelReason() string {
	if torrent.Reason == "" {
		return "Torrent deleted"
	} else {
		return torrent.Reason
	}
}

// AddPeer inserts a new peer into the torrents active peer list
func (torrent *Torrent) AddPeer(r redis.Conn, peer *Peer) bool {
	if torrent.HasPeer(peer) {
		log.Warn("Tried to add duplicate peer")
		return false
	}
	torrent.Lock()
	torrent.Peers = append(torrent.Peers, peer)
	torrent.Unlock()
	peer.User.Join(torrent)
	v, err := r.Do("SADD", torrent.TorrentPeersKey, peer.PeerID)
	if err != nil {
		log.WithFields(log.Fields{
			"err": err.Error(),
			"fn":  "AddPeer",
		}).Error("Error executing peer fetch query")
		return false
	}
	if v == "0" {
		log.WithFields(log.Fields{
			"fn": "AddPeer",
		}).Warn("Tried to add peer to set with existing element")
	}
	return true
}

// DelPeer removed the peer from the torrent and marks it inactive
func (torrent *Torrent) DelPeer(r redis.Conn, peer *Peer) bool {
	torrent.RLock()
	defer torrent.RUnlock()
	for i, tor_peer := range torrent.Peers {
		if tor_peer == peer {
			if len(torrent.Peers) == 1 {
				torrent.Peers = nil
			} else {
				torrent.Peers = append(torrent.Peers[:i], torrent.Peers[i+1:]...)
				log.WithFields(log.Fields{
					"info_hash": torrent.InfoHash,
					"fn":        "DelPeer",
				}).Debug("Removed peer from torrent")
			}
			break
		}
	}

	r.Send("SREM", torrent.TorrentPeersKey, peer.PeerID)
	return true
}

// HasPeer Checks if the peer already is a member of the peer slice for the torrent
func (torrent *Torrent) HasPeer(peer *Peer) bool {
	torrent.RLock()
	defer torrent.RUnlock()
	for _, p := range torrent.Peers {
		if peer == p {
			return true
		}
	}
	return false
}

// PeerCounts counts the number of seeders and leechers the torrent currently has.
// Only active peers are counted towards the totals
func (torrent *Torrent) PeerCounts() (int, int) {
	s, l := 0, 0
	torrent.RLock()
	for _, p := range torrent.Peers {
		if p.IsSeeder() {
			s++
		} else {
			l++
		}
	}
	torrent.RUnlock()
	return s, l
}

func (torrent *Torrent) Stats() TorrentStats {
	return TorrentStats{
		torrent.TorrentID,
		torrent.InfoHash,
		torrent.Seeders,
		torrent.Leechers,
		1,
	}
}

// GetPeers returns a slice of up to max_peers from the current torrent. If the
// total peers available is less than the max peers all peers will be returned. Otherwise
// the peers are split 80/20 (leechers/seeders). If those numbers can't be met, the leecher counts are relaxed
// so that seeders can fill their spots.
func (torrent *Torrent) GetPeers(max_peers int) []*Peer {
	torrent.RLock()
	defer torrent.RUnlock()
	if len(torrent.Peers) > max_peers {
		// Calculate the initial quantities we want
		max_leechers := int(math.Ceil(float64(max_peers) * 0.2))
		max_seeders := int(max_peers) - max_leechers

		// If we don't have enough leechers too meet the requirements, add some more seeders
		_, leechers := torrent.PeerCounts()
		if max_leechers > leechers {
			max_seeders += max_leechers - leechers
			max_leechers = leechers
		}

		peers := []*Peer{}
		found_seeders, found_leechers := 0, 0
		for _, peer := range torrent.Peers {
			if peer.IsSeeder() && max_seeders > found_seeders {
				peers = append(peers, peer)
				found_seeders++
			} else if max_leechers > found_leechers {
				peers = append(peers, peer)
				found_leechers++
			}
			if len(peers) == max_peers {
				break
			}
		}
		return peers
	} else {
		return torrent.Peers[0:util.UMin(uint64(len(torrent.Peers)), uint64(max_peers))]
	}
}
