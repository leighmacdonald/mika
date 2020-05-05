package model

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

// InfoHash is a unique 20byte identifier for a torrent
type InfoHash [20]byte

// InfoHashFromString returns a binary infohash from the info string
func InfoHashFromString(s string) InfoHash {
	var buf [20]byte
	copy(buf[:], s)
	return buf
}

// Bytes returns the raw bytes of the info_hash. This is primarily useful for inserting to SQL stores since
// they have trouble with the sized variant
func (ih InfoHash) Bytes() []byte {
	return ih[:]
}

// String implements fmt.Stringer, returning the base16 encoded PeerID.
func (ih *InfoHash) String() string {
	return fmt.Sprintf("%x", ih[:])
}

// RawString returns a 20-byte string of the raw bytes of the ID.
func (ih *InfoHash) RawString() string {
	return string(ih.Bytes())
}

// Torrent is the core struct for our torrent being tracked
type Torrent struct {
	sync.RWMutex
	ReleaseName    string   `db:"release_name" redis:"release_name" json:"release_name"`
	InfoHash       InfoHash `db:"info_hash" redis:"info_hash" json:"info_hash"`
	TotalCompleted int16    `db:"total_completed" redis:"total_completed" json:"total_completed"`
	// This is stored as MB to reduce storage costs
	TotalUploaded uint32 `db:"total_uploaded" redis:"total_uploaded" json:"total_uploaded"`
	// This is stored as MB to reduce storage costs
	TotalDownloaded uint32 `db:"total_downloaded" redis:"total_downloaded" json:"total_downloaded"`
	IsDeleted       bool   `db:"is_deleted" redis:"is_deleted" json:"is_deleted"`
	// When you have a message to pass to a client set enabled = false and set the reason message.
	// If IsDeleted is true, then nothing will be returned to the client
	IsEnabled bool `db:"is_enabled" redis:"is_enabled" json:"is_enabled"`
	// Reason when set will return a message to the torrent client
	Reason string `db:"reason" redis:"reason" json:"reason"`
	// Upload multiplier added to the users totals
	MultiUp float64 `db:"multi_up" redis:"multi_up" json:"multi_up"`
	// Download multiplier added to the users totals
	// 0 denotes freeleech status
	MultiDn   float64   `db:"multi_dn"  redis:"multi_dn" json:"multi_dn"`
	CreatedOn time.Time `db:"created_on" redis:"created_on" json:"created_on"`
	UpdatedOn time.Time `db:"updated_on" redis:"updated_on" json:"updated_on"`
}

// TorrentStats is used to relay info stats for a torrent around. It contains rolled up stats
// from peer info as well as the normal torrent stats.
type TorrentStats struct {
	TorrentID uint64 `json:"torrent_id"`
	InfoHash  string `json:"info_hash"`
	Seeders   int    `json:"seeders"`
	Leechers  int    `json:"leechers"`
	Snatches  int    `json:"snatches"`
}

// NewTorrent allocates and returns a new Torrent instance pointer with all
// the minimum value required to operated in place
func NewTorrent(ih InfoHash, name string) *Torrent {
	torrent := &Torrent{
		ReleaseName: name,
		InfoHash:    ih,
		IsDeleted:   false,
		IsEnabled:   true,
		MultiUp:     1.0,
		MultiDn:     1.0,
		CreatedOn:   time.Now().UTC(),
		UpdatedOn:   time.Now().UTC(),
	}
	return torrent
}

// WhiteListClient defines a whitelisted bittorrent client allowed to participate
// in swarms. This is not a foolproof solution as its fairly trivial for a motivated
// attacker to fake this.
type WhiteListClient struct {
	ClientID     uint16    `json:"client_id"`
	ClientPrefix string    `json:"client_prefix"`
	ClientName   string    `json:"client_name"`
	CreatedOn    time.Time `json:"created_on"`
}

// Match returns true if the client matches this prefix
func (wl WhiteListClient) Match(client string) bool {
	return strings.HasPrefix(client, wl.ClientPrefix)
}
