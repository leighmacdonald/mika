package model

import (
	"fmt"
	"mika/geo"
	"net"
	"sync"
	"time"
)

// PeerID is the client supplied unique identifier for a peer
type PeerID [20]byte

// PeerIDFromString translates a string into a binary PeerID
func PeerIDFromString(s string) PeerID {
	var buf [20]byte
	copy(buf[:], s)
	return buf
}

// String implements fmt.Stringer, returning the base16 encoded PeerID.
func (p PeerID) String() string {
	return fmt.Sprintf("%x", p[:])
}

// RawString returns a 20-byte string of the raw bytes of the ID.
func (p PeerID) RawString() string {
	return string(p[:])
}

// Peer represents a single unique peer in a swarm
type Peer struct {
	sync.RWMutex
	UserPeerID uint32 `db:"user_peer_id" redis:"user_peer_id" json:"user_peer_id"`
	// Current speed up, bytes/sec
	SpeedUP uint32 `db:"speed_up" redis:"speed_up" json:"speed_up"`
	// Current speed dn, bytes/sec
	SpeedDN uint32 `db:"speed_dn" redis:"speed_dn" json:"speed_dn"`
	// Max recorded up speed, bytes/sec
	SpeedUPMax uint32 `db:"speed_up_max"  redis:"speed_up_max" json:"speed_up_max"`
	// Max recorded dn speed, bytes/sec
	SpeedDNMax uint32 `db:"speed_dn_max" redis:"speed_dn_max" json:"speed_dn_max"`
	// Total amount uploaded as reported by client
	Uploaded uint32 `db:"total_uploaded" redis:"total_uploaded" json:"total_uploaded"`
	// Total amount downloaded as reported by client
	Downloaded uint32 `db:"total_downloaded" redis:"total_downloaded" json:"total_downloaded"`
	// Clients reported bytes left of the download
	Left uint32 `db:"total_left" redis:"total_left" json:"total_left"`
	// Total number of announces the peer has made
	Announces uint32 `db:"total_announces" redis:"total_announces" json:"total_announces"`
	// Total active swarm participation time
	TotalTime uint32 `db:"total_time" redis:"total_time" json:"total_time"`
	// Clients IPv4 Address detected automatically, does not use client supplied value
	IP net.IP `db:"addr_ip" redis:"addr_ip" json:"addr_ip"`
	// Clients reported port
	Port uint16 `db:"addr_port" redis:"addr_port" json:"addr_port"`
	// Last announce timestamp
	AnnounceLast time.Time `redis:"last_announce" json:"last_announce"`
	// First announce timestamp
	AnnounceFirst time.Time `redis:"first_announce" json:"first_announce"`
	// Peer id, reported by client. Must have white-listed prefix
	PeerID   PeerID      `db:"peer_id" redis:"peer_id" json:"peer_id"`
	Location geo.LatLong `db:"location" redis:"location" json:"location"`
	UserID   uint32      `db:"user_id" redis:"user_id" json:"user_id"`
	// TODO Do we actually care about these times? Announce times likely enough
	CreatedOn time.Time `db:"created_on" redis:"created_on" json:"created_on"`
	UpdatedOn time.Time `db:"updated_on" redis:"updated_on" json:"updated_on"`

	User *User
}

// IsNew checks if the peer is making its first announce request
func (peer *Peer) IsNew() bool {
	return peer.Announces == 0
}

type Swarm []*Peer

// Remove removes a peer from a slice
func (peers Swarm) Remove(p *Peer) []*Peer {
	for i := len(peers) - 1; i >= 0; i-- {
		if peers[i] == p {
			return append(peers[:i], peers[i+1:]...)
		}
	}
	return peers
}

// Counts returns the sums for seeders and leechers in the swarm
// TODO cache this somewhere and only update on state change
func (peers Swarm) Counts() (seeders uint, leechers uint) {
	for _, p := range peers {
		if p.Left == 0 {
			seeders++
		} else {
			leechers++
		}
	}
	return
}

func NewPeer(userId uint32, peerId PeerID, ip net.IP, port uint16) *Peer {
	return &Peer{
		RWMutex:       sync.RWMutex{},
		UserPeerID:    0,
		SpeedUP:       0,
		SpeedDN:       0,
		SpeedUPMax:    0,
		SpeedDNMax:    0,
		Uploaded:      0,
		Downloaded:    0,
		Left:          0,
		Announces:     0,
		TotalTime:     0,
		IP:            ip,
		Port:          port,
		AnnounceLast:  time.Now(),
		AnnounceFirst: time.Now(),
		PeerID:        peerId,
		Location:      geo.LatLong{Latitude: 50, Longitude: -114},
		UserID:        userId,
		CreatedOn:     time.Now(),
		UpdatedOn:     time.Now(),
		User:          nil,
	}
}
