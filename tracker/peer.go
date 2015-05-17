package tracker

import (
	"bytes"
	"fmt"
	"git.totdev.in/totv/mika/conf"
	"git.totdev.in/totv/mika/util"
	log "github.com/Sirupsen/logrus"
	"net"
	"sync"
)

type Peer struct {
	sync.RWMutex

	// Current speed up, bytes/sec
	SpeedUP float64 `redis:"speed_up" json:"speed_up"`

	// Current speed dn, bytes/sec
	SpeedDN float64 `redis:"speed_dn" json:"speed_dn"`

	// Max recorded up speed, bytes/sec
	SpeedUPMax float64 `redis:"speed_up" json:"speed_up_max"`

	// Max recorded dn speed, bytes/sec
	SpeedDNMax float64 `redis:"speed_dn" json:"speed_dn_max"`

	// Total amount uploaded as reported by client
	Uploaded uint64 `redis:"uploaded" json:"uploaded"`

	// Total amount downloaded as reported by client
	Downloaded uint64 `redis:"downloaded" json:"downloaded"`

	// Clients IPv4 Address detected automatically, does not use client supplied value
	IP string `redis:"ip" json:"ip"`

	// Clients reported port
	Port uint64 `redis:"port" json:"port"`

	// Clients reported bytes left of the download
	Left uint64 `redis:"left" json:"left"`

	// Total number of announces the peer has made
	Announces uint64 `redis:"announces" json:"announces"`

	// Total active swarm participation time
	TotalTime uint32 `redis:"total_time" json:"total_time"`

	// Last announce timestamp
	AnnounceLast int32 `redis:"last_announce" json:"last_announce"`

	// First announce timestamp
	AnnounceFirst int32 `redis:"first_announce" json:"first_announce"`

	// Peer id, reported by client. Must have white-listed prefix
	PeerID string `redis:"peer_id" json:"peer_id"`

	User     *User    `redis:"-"  json:"-"`
	Torrent  *Torrent `redis:"-" json:"-"`
	KeyPeer  string   `redis:"-" json:"-"`
	KeyTimer string   `redis:"-" json:"-"`
}

func (peer *Peer) IsNew() bool {
	return peer.AnnounceLast == 0
}

// Update the stored values with the data from an announce
func (peer *Peer) Update(announce *AnnounceRequest) (uint64, uint64) {
	peer.Lock()
	defer peer.Unlock()
	cur_time := util.Unixtime()
	peer.PeerID = announce.PeerID
	peer.Announces++

	var ul_diff, dl_diff uint64

	if !peer.IsNew() {
		// We only record the difference from the first announce
		if announce.Uploaded > peer.Uploaded {
			ul_diff = announce.Uploaded - peer.Uploaded
		}
		if announce.Downloaded > peer.Downloaded {
			dl_diff = announce.Downloaded - peer.Downloaded
		}
		peer.SpeedUP = util.EstSpeed(peer.AnnounceLast, cur_time, ul_diff)
		peer.SpeedDN = util.EstSpeed(peer.AnnounceLast, cur_time, dl_diff)
	}
	peer.Uploaded = announce.Uploaded
	peer.Downloaded = announce.Downloaded

	if peer.SpeedUP > peer.SpeedUPMax {
		peer.SpeedUPMax = peer.SpeedUP
	}
	if peer.SpeedDN > peer.SpeedDNMax {
		peer.SpeedDNMax = peer.SpeedDN
	}
	if ul_diff > 0 || dl_diff > 0 {
		log.WithFields(log.Fields{
			"ul_diff":      util.Bytes(ul_diff),
			"dl_diff":      util.Bytes(dl_diff),
			"user_name":    peer.User.Username,
			"peer_id":      peer.PeerID[0:8],
			"speed_up":     fmt.Sprintf("s%/s", uint64(util.Bytes(peer.SpeedUP))),
			"speed_dn":     fmt.Sprintf("s%/s", uint64(util.Bytes(peer.SpeedDN))),
			"speed_up_max": fmt.Sprintf("s%/s", uint64(util.Bytes(peer.SpeedUPMax))),
			"speed_dn_max": fmt.Sprintf("s%/s", uint64(util.Bytes(peer.SpeedDNMax))),
		}).Info("Peer stat changes")
	}

	peer.IP = announce.IPv4.String()
	peer.Port = announce.Port
	peer.Left = announce.Left

	// Must be active to have a real time delta
	if !peer.IsNew() {
		time_diff := uint64(cur_time - peer.AnnounceLast)
		// Ignore long periods of inactivity
		if time_diff < (uint64(conf.Config.AnnInterval) * 4) {
			peer.TotalTime += uint32(time_diff)
		}
	}

	if peer.IsNew() {
		peer.AnnounceFirst = cur_time
	}
	peer.AnnounceLast = cur_time

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
		IP:            ip,
		Port:          port,
		AnnounceFirst: 0,
		AnnounceLast:  0,
		User:          user,
		Torrent:       torrent,
		KeyTimer:      fmt.Sprintf("t:ptimeout:%s:%s", torrent.InfoHash, peer_id),
	}
	return peer
}
