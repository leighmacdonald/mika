package model

import (
	"database/sql/driver"
	"fmt"
	"github.com/leighmacdonald/mika/consts"
	"github.com/leighmacdonald/mika/geo"
	"github.com/leighmacdonald/mika/util"
	"github.com/pkg/errors"
	"net"
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

func (p *PeerID) Value() (driver.Value, error) {
	return p.Bytes(), nil
}

// Scan implements the sql Scanner interface for conversion to our custom type
func (p *PeerID) Scan(v interface{}) error {
	// Should be more strictly to check this type.
	vt, ok := v.([]byte)
	if !ok {
		return errors.New("failed to convert value to peer_id")
	}
	cnt := copy(p[:], vt)
	if cnt != 20 {
		return errors.New(fmt.Sprintf("invalid data length received: %d, expected 20", cnt))
	}
	return nil
}

// Bytes returns the raw bytes of the peer_id. This is primarily useful for inserting to SQL stores since
// they have trouble with the sized variant
func (p PeerID) Bytes() []byte {
	return p[:]
}

// String implements fmt.Stringer, returning the base16 encoded PeerID.
func (p PeerID) String() string {
	return fmt.Sprintf("%x", p[:])
}

// RawString returns a 20-byte string of the raw bytes of the ID.
func (p PeerID) RawString() string {
	return string(p.Bytes())
}

// Peer represents a single unique peer in a swarm
type Peer struct {
	// Total amount uploaded as reported by client
	Uploaded uint64 `db:"total_uploaded" redis:"total_uploaded" json:"total_uploaded"`
	// Total amount downloaded as reported by client
	Downloaded uint64 `db:"total_downloaded" redis:"total_downloaded" json:"total_downloaded"`
	// Clients reported bytes left of the download
	Left uint32 `db:"total_left" redis:"total_left" json:"total_left"`
	// Total active swarm participation time
	TotalTime uint32 `db:"total_time" redis:"total_time" json:"total_time"`
	// Current speed up, bytes/sec
	SpeedUP uint32 `db:"speed_up" redis:"speed_up" json:"speed_up"`
	// Current speed dn, bytes/sec
	SpeedDN uint32 `db:"speed_dn" redis:"speed_dn" json:"speed_dn"`
	// Max recorded up speed, bytes/sec
	SpeedUPMax uint32 `db:"speed_up_max"  redis:"speed_up_max" json:"speed_up_max"`
	// Max recorded dn speed, bytes/sec
	SpeedDNMax uint32 `db:"speed_dn_max" redis:"speed_dn_max" json:"speed_dn_max"`
	// Clients IPv4 Address detected automatically, does not use client supplied value
	IP net.IP `db:"addr_ip" redis:"addr_ip" json:"addr_ip"`
	// Clients reported port
	Port uint16 `db:"addr_port" redis:"addr_port" json:"addr_port"`
	// Total number of announces the peer has made
	Announces uint32 `db:"total_announces" redis:"total_announces" json:"total_announces"`
	// Last announce timestamp
	AnnounceLast time.Time `db:"announce_last" redis:"announce_last" json:"announce_last"`
	// First announce timestamp
	AnnounceFirst time.Time `db:"announce_first" redis:"announce_first" json:"announce_first"`
	// Peer id, reported by client. Must have white-listed prefix
	PeerID   PeerID      `db:"peer_id" redis:"peer_id" json:"peer_id"`
	InfoHash InfoHash    `db:"info_hash" redis:"info_hash" json:"info_hash"`
	Location geo.LatLong `db:"location" redis:"location" json:"location"`
	UserID   uint32      `db:"user_id" redis:"user_id" json:"user_id"`
	// TODO Do we actually care about these times? Announce times likely enough
	//CreatedOn time.Time `db:"created_on" redis:"created_on" json:"created_on"`
	//UpdatedOn time.Time `db:"updated_on" redis:"updated_on" json:"updated_on"`

	User *User
}

// Expired checks if the peer last lost contact with us
// TODO remove hard coded expiration time
func (peer *Peer) Expired() bool {
	return time.Now().Sub(peer.AnnounceLast).Seconds() > 300
}

// IsNew checks if the peer is making its first announce request
func (peer *Peer) IsNew() bool {
	return peer.Announces == 0
}

// Valid returns true if the peer data meets the minimum requirements to participate in swarms
func (peer *Peer) Valid() bool {
	return peer.UserID > 0 && peer.Port >= 1024 && util.IsPrivateIP(peer.IP)
}

// Swarm is a set of users participating in a torrent
type Swarm []Peer

// Remove removes a peer from a slice
func (peers Swarm) Remove(p PeerID) []Peer {
	for i := len(peers) - 1; i >= 0; i-- {
		if peers[i].PeerID == p {
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

// NewPeer create a new peer instance for inserting into a swarm
func NewPeer(userID uint32, peerID PeerID, ip net.IP, port uint16) Peer {
	return Peer{
		IP:            ip,
		Port:          port,
		AnnounceLast:  time.Now(),
		AnnounceFirst: time.Now(),
		PeerID:        peerID,
		Location:      geo.LatLong{Latitude: 0, Longitude: 0},
		UserID:        userID,
		User:          nil,
	}
}

type UpdateState struct {
	InfoHash InfoHash
	PeerID   PeerID
	Passkey  string
	// Total amount uploaded as reported by client
	Uploaded uint64
	// Total amount downloaded as reported by client
	Downloaded uint64
	// Clients reported bytes left of the download
	Left uint32
	// Timestamp is the time the new stats were announced
	Timestamp time.Time
	Event     consts.AnnounceType
}
