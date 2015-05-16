package tracker

import (
	"bytes"
	"fmt"
	"git.totdev.in/totv/mika/conf"
	"git.totdev.in/totv/mika/db"
	"git.totdev.in/totv/mika/util"
	"net"
	"sync"
)

type Peer struct {
	db.Queued
	sync.RWMutex
	SpeedUP       float64  `redis:"speed_up" json:"speed_up"`
	SpeedDN       float64  `redis:"speed_dn" json:"speed_dn"`
	SpeedUPMax    float64  `redis:"speed_up" json:"speed_up_max"`
	SpeedDNMax    float64  `redis:"speed_dn" json:"speed_dn_max"`
	Uploaded      uint64   `redis:"uploaded" json:"uploaded"`
	Downloaded    uint64   `redis:"downloaded" json:"downloaded"`
	IP            string   `redis:"ip" json:"ip"`
	Port          uint64   `redis:"port" json:"port"`
	Left          uint64   `redis:"left" json:"left"`
	Announces     uint64   `redis:"announces" json:"announces"`
	TotalTime     uint32   `redis:"total_time" json:"total_time"`
	AnnounceLast  int32    `redis:"last_announce" json:"last_announce"`
	AnnounceFirst int32    `redis:"first_announce" json:"first_announce"`
	PeerID        string   `redis:"peer_id" json:"peer_id"`
	Active        bool     `redis:"active"  json:"active"`
	Username      string   `redis:"username"  json:"username"`
	User          *User    `redis:"-"  json:"-"`
	Torrent       *Torrent `redis:"-" json:"-"`
	KeyPeer       string   `redis:"-" json:"-"`
	KeyTimer      string   `redis:"-" json:"-"`
}

// Update the stored values with the data from an announce
func (peer *Peer) Update(announce *AnnounceRequest) (uint64, uint64) {
	peer.Lock()
	defer peer.Unlock()
	cur_time := util.Unixtime()
	peer.PeerID = announce.PeerID
	peer.Announces++

	var ul_diff, dl_diff uint64

	// We only record the difference from the first announce
	if announce.Uploaded > peer.Uploaded {
		ul_diff = announce.Uploaded - peer.Uploaded
	}
	if announce.Downloaded > peer.Downloaded {
		dl_diff = announce.Downloaded - peer.Downloaded
	}

	ul_diff = 0
	dl_diff = 0

	peer.IP = announce.IPv4.String()
	peer.Port = announce.Port
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
		time_diff := uint64(cur_time - peer.AnnounceLast)
		// Ignore long periods of inactivity
		if time_diff < (uint64(conf.Config.AnnInterval) * 4) {
			peer.TotalTime += uint32(time_diff)
		}
	}

	if peer.AnnounceFirst == 0 {
		peer.AnnounceFirst = cur_time
	}
	peer.AnnounceLast = cur_time

	if announce.Event == STOPPED {
		peer.Active = false
		peer.AnnounceFirst = 0
	} else {
		peer.Active = true
	}

	return ul_diff, dl_diff
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
		AnnounceFirst: 0,
		AnnounceLast:  util.Unixtime(),
		User:          user,
		Torrent:       torrent,
		KeyPeer:       fmt.Sprintf("t:p:%s:%s", torrent.InfoHash, peer_id),
		KeyTimer:      fmt.Sprintf("t:ptimeout:%s:%s", torrent.InfoHash, peer_id),
	}
	return peer
}
