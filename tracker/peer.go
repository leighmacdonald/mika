package tracker

import (
	"bytes"
	"fmt"
	"git.totdev.in/totv/mika/conf"
	"git.totdev.in/totv/mika/db"
	"git.totdev.in/totv/mika/util"
	log "github.com/Sirupsen/logrus"
	"github.com/garyburd/redigo/redis"
	"net"
	"sync"
)

type Peer struct {
	db.Queued
	sync.RWMutex
	SpeedUP        float64  `redis:"speed_up" json:"speed_up"`
	SpeedDN        float64  `redis:"speed_dn" json:"speed_dn"`
	SpeedUPMax     float64  `redis:"speed_up" json:"speed_up_max"`
	SpeedDNMax     float64  `redis:"speed_dn" json:"speed_dn_max"`
	Uploaded       uint64   `redis:"uploaded" json:"uploaded"`
	Downloaded     uint64   `redis:"downloaded" json:"downloaded"`
	UploadedLast   uint64   `redis:"-" json:"-"`
	DownloadedLast uint64   `redis:"-" json:"-"`
	Corrupt        uint64   `redis:"corrupt" json:"corrupt"`
	IP             string   `redis:"ip" json:"ip"`
	Port           uint64   `redis:"port" json:"port"`
	Left           uint64   `redis:"left" json:"left"`
	Announces      uint64   `redis:"announces" json:"announces"`
	TotalTime      uint32   `redis:"total_time" json:"total_time"`
	AnnounceLast   int32    `redis:"last_announce" json:"last_announce"`
	AnnounceFirst  int32    `redis:"first_announce" json:"first_announce"`
	New            bool     `redis:"new" json:"-"`
	PeerID         string   `redis:"peer_id" json:"peer_id"`
	Active         bool     `redis:"active"  json:"active"`
	Username       string   `redis:"username"  json:"username"`
	User           *User    `redis:"-"  json:"-"`
	Torrent        *Torrent `redis:"-" json:"-"`
	KeyPeer        string   `redis:"-" json:"-"`
	KeyTimer       string   `redis:"-" json:"-"`
}

// Update the stored values with the data from an announce
func (peer *Peer) Update(announce *AnnounceRequest) (uint64, uint64) {
	peer.Lock()
	defer peer.Unlock()
	cur_time := util.Unixtime()
	peer.PeerID = announce.PeerID
	peer.Announces++

	ul_diff := uint64(0)
	dl_diff := uint64(0)

	if announce.Event == STARTED {
		peer.Uploaded = announce.Uploaded
		peer.Downloaded = announce.Downloaded
	} else if announce.Uploaded < peer.Uploaded || announce.Downloaded < peer.Downloaded {
		peer.Uploaded = announce.Uploaded
		peer.Downloaded = announce.Downloaded
	} else {
		if announce.Uploaded != peer.Uploaded {
			ul_diff = announce.Uploaded - peer.Uploaded
			peer.Uploaded = announce.Uploaded
		}
		if announce.Downloaded != peer.Downloaded {
			dl_diff = announce.Downloaded - peer.Downloaded
			peer.Downloaded = announce.Downloaded
		}

	}
	peer.IP = announce.IPv4.String()
	peer.Port = announce.Port
	peer.Corrupt = announce.Corrupt
	peer.Left = announce.Left
	peer.SpeedUP = util.EstSpeed(peer.AnnounceLast, cur_time, ul_diff)
	peer.SpeedDN = util.EstSpeed(peer.AnnounceLast, cur_time, dl_diff)
	if peer.SpeedUP > peer.SpeedUPMax {
		peer.SpeedUPMax = peer.SpeedUP
	}
	if peer.SpeedDN > peer.SpeedDNMax {
		peer.SpeedDNMax = peer.SpeedDN
	}

	// Must be active to have a real time delta
	if peer.Active && peer.AnnounceLast > 0 {
		time_diff := uint64(util.Unixtime() - peer.AnnounceLast)
		// Ignore long periods of inactivity
		if time_diff < (uint64(conf.Config.AnnInterval) * 4) {
			peer.TotalTime += uint32(time_diff)
		}
	}
	if announce.Event == STOPPED {
		peer.Active = false
	}
	return ul_diff, dl_diff
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
		"speed_up_max", peer.SpeedUPMax,
		"speed_dn_max", peer.SpeedDNMax,
		"active", peer.Active,
		"uploaded", peer.Uploaded,
		"downloaded", peer.Downloaded,
		"corrupt", peer.Corrupt,
		"username", peer.Username,
		"user_id", peer.User.UserID, // Shouldn't need to be here
		"peer_id", peer.PeerID, // Shouldn't need to be here
		"torrent_id", peer.Torrent.TorrentID, // Shouldn't need to be here
	)
}

func (peer *Peer) IsHNR() bool {
	return peer.Downloaded > conf.Config.HNRMinBytes && peer.Left > 0 && peer.TotalTime < uint32(conf.Config.HNRThreshold)
}

func (peer *Peer) IsSeeder() bool {
	return peer.Left == 0
}

// Generate a compact peer field array containing the byte representations
// of a peers IP+Port appended to each other
func MakeCompactPeers(peers []*Peer, skip_id string) []byte {
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

// NewPeer created a new peer with the minimum attributes needed
func NewPeer(peer_id string, ip string, port uint64, torrent *Torrent, user *User) *Peer {
	peer := &Peer{
		PeerID:        peer_id,
		Active:        false,
		IP:            ip,
		Port:          port,
		AnnounceFirst: util.Unixtime(),
		AnnounceLast:  util.Unixtime(),
		User:          user,
		Torrent:       torrent,
		KeyPeer:       fmt.Sprintf("t:p:%s:%s", torrent.InfoHash, peer_id),
		KeyTimer:      fmt.Sprintf("t:ptimeout:%s:%s", torrent.InfoHash, peer_id),
	}
	return peer
}

// MergeDB will apply the data stored in the db to the peer instance
// Should generally only be called on startup since it will overwrite
// existing data stored in the active peer instance.
func (peer *Peer) MergeDB(r redis.Conn) error {
	peer_reply, err := r.Do("HGETALL", peer.KeyPeer)
	if err != nil {
		log.Println("GetPeer: Error executing peer fetch query: ", err)
		return err
	}
	values, err := redis.Values(peer_reply, nil)
	if err != nil {
		log.Println("makePeer: Failed to parse peer reply: ", err)
		return err_parse_reply
	}
	if values != nil {
		err := redis.ScanStruct(values, peer)
		if err != nil {
			log.Println("makePeer: Failed to scan peer struct: ", err)
			return err_cast_reply
		}
	}
	return nil
}
