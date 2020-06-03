package store

import (
	"database/sql/driver"
	"fmt"
	"github.com/leighmacdonald/mika/consts"
	"github.com/leighmacdonald/mika/geo"
	"github.com/leighmacdonald/mika/util"
	"github.com/pkg/errors"
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

// Value implements the driver.Valuer interface
func (p *PeerID) Value() (driver.Value, error) {
	return p.Bytes(), nil
}

// Scan implements the sql.Scanner interface for conversion to our custom type
func (p *PeerID) Scan(v interface{}) error {
	// Should be more strictly to check this type.
	vt, ok := v.([]byte)
	if !ok {
		return errors.New("failed to convert value to peer_id")
	}
	cnt := copy(p[:], vt)
	if cnt != 20 {
		return fmt.Errorf("invalid data length received: %d, expected 20", cnt)
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

// URLEncode returns the peer id suitably  encoded for a URL
func (p PeerID) URLEncode() string {
	return fmt.Sprintf("%s", p.Bytes())
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
	IP   net.IP `db:"addr_ip" redis:"addr_ip" json:"addr_ip"`
	IPv6 bool   `db:"ipv6" json:"ipv6"`
	// Clients reported port
	Port uint16 `db:"addr_port" redis:"addr_port" json:"addr_port"`
	// Total number of announces the peer has made
	Announces uint32 `db:"total_announces" redis:"total_announces" json:"total_announces"`
	// Last announce timestamp
	AnnounceLast time.Time `db:"announce_last" redis:"announce_last" json:"announce_last"`
	// First announce timestamp
	AnnounceFirst time.Time `db:"announce_first" redis:"announce_first" json:"announce_first"`
	// Peer id, reported by client. Must have white-listed prefix
	PeerID      PeerID      `db:"peer_id" redis:"peer_id" json:"peer_id"`
	InfoHash    InfoHash    `db:"info_hash" redis:"info_hash" json:"info_hash"`
	Location    geo.LatLong `db:"location" redis:"location" json:"location"`
	CountryCode string      `db:"country_code" json:"country_code"`
	ASN         uint32      `db:"asn" json:"asn"`
	AS          string      `db:"as" json:"as"`
	UserID      uint32      `db:"user_id" redis:"user_id" json:"user_id"`
	// Client is the user-agent header sent
	Client string `db:"client" json:"client"`
	// TODO Do we actually care about these times? Announce times likely enough
	//CreatedOn time.Time `db:"created_on" redis:"created_on" json:"created_on"`
	//UpdatedOn time.Time `db:"updated_on" redis:"updated_on" json:"updated_on"`

	Paused bool
	User   *User
}

// Expired checks if the peer last lost contact with us
// TODO remove hard coded expiration time
func (peer *Peer) Expired() bool {
	return time.Since(peer.AnnounceLast).Seconds() > 300
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
type Swarm struct {
	Peers    map[PeerID]Peer
	Seeders  int
	Leechers int
	*sync.RWMutex
}

// NewSwarm instantiates a new swarm
func NewSwarm() Swarm {
	return Swarm{
		Peers:    make(map[PeerID]Peer),
		Seeders:  0,
		Leechers: 0,
		RWMutex:  &sync.RWMutex{},
	}
}

// Remove removes a peer from a slice
func (swarm Swarm) Remove(p PeerID) {
	swarm.Lock()
	delete(swarm.Peers, p)
	swarm.Unlock()
}

// Add inserts a new peer into the swarm
func (swarm Swarm) Add(p Peer) {
	swarm.Lock()
	swarm.Peers[p.PeerID] = p
	swarm.Unlock()
}

// UpdatePeer will update a swarm member with new stats
func (swarm Swarm) UpdatePeer(peerID PeerID, stats PeerStats) (Peer, bool) {
	swarm.Lock()
	peer, ok := swarm.Peers[peerID]
	if !ok {
		swarm.Unlock()
		return peer, false
	}
	for _, s := range stats.Hist {
		peer.Uploaded += s.Uploaded
		peer.Downloaded += s.Downloaded
		peer.AnnounceLast = s.Timestamp
	}
	peer.Announces += uint32(len(stats.Hist))
	peer.Left = stats.Left
	swarm.Peers[peerID] = peer
	swarm.Unlock()
	return peer, true
}

// ReapExpired will delete any peers from the swarm that are considered expired
func (swarm Swarm) ReapExpired(infoHash InfoHash, cache *PeerCache) {
	swarm.Lock()
	for k, peer := range swarm.Peers {
		if peer.Expired() {
			delete(swarm.Peers, k)
			cache.Delete(infoHash, peer.PeerID)
		}
	}
	swarm.Unlock()
}

// Get will copy a peer into the peer pointer passed in if it exists.
func (swarm Swarm) Get(peer *Peer, peerID PeerID) error {
	swarm.RLock()
	defer swarm.RUnlock()
	p, found := swarm.Peers[peerID]
	if !found {
		return consts.ErrInvalidPeerID
	}
	*peer = p
	return nil
}

func (swarm Swarm) Update(p Peer) error {
	swarm.RLock()
	_, found := swarm.Peers[p.PeerID]
	swarm.RUnlock()
	if !found {
		return consts.ErrInvalidPeerID
	}
	swarm.Lock()
	swarm.Peers[p.PeerID] = p
	swarm.Unlock()
	return nil
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
		Paused:        false,
	}
}

// UpdateState is used to store temporary data used for batch updates
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
	Paused    bool
}
