package model

type PeerID [20]byte

func PeerIDFromString(s string) PeerID {
	var buf [20]byte
	copy(buf[:], s)
	return buf
}

// Peer represents a single unique peer in a swarm
type Peer struct {

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
	TotalTime uint64 `redis:"total_time" json:"total_time"`

	// Last announce timestamp
	AnnounceLast int32 `redis:"last_announce" json:"last_announce"`

	// First announce timestamp
	AnnounceFirst int32 `redis:"first_announce" json:"first_announce"`

	// Peer id, reported by client. Must have white-listed prefix
	PeerId PeerID `redis:"peer_id" json:"peer_id"`

	// Coord geo.LatLong

	UserId uint64 `redis:"-" json:"-"`
}

// IsNew checks if the peer is making its first announce request
func (peer *Peer) IsNew() bool {
	return peer.AnnounceLast == 0
}
