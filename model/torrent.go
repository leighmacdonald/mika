package model

import (
	"database/sql/driver"
	"errors"
	"fmt"
	"github.com/leighmacdonald/mika/consts"
	"strings"
)

// InfoHash is a unique 20byte identifier for a torrent
type InfoHash [20]byte

// InfoHashFromString returns a binary infohash from the info string
func InfoHashFromString(s string) InfoHash {
	var buf [20]byte
	copy(buf[:], s)
	return buf
}

func (ih *InfoHash) Value() (driver.Value, error) {
	return ih.Bytes(), nil
}

// Scan implements the sql Scanner interface for conversion to our custom type
func (ih *InfoHash) Scan(v interface{}) error {
	// Should be more strictly to check this type.
	vt, ok := v.([]byte)
	if !ok {
		return errors.New("failed to convert value to infohash")
	}
	cnt := copy(ih[:], vt)
	if cnt != 20 {
		return errors.New(fmt.Sprintf("invalid data length received: %d, expected 20", cnt))
	}
	return nil
}

// Bytes returns the raw bytes of the info_hash. This is primarily useful for inserting to SQL stores since
// they have trouble with the sized variant
func (ih *InfoHash) Bytes() []byte {
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
	MultiDn float64 `db:"multi_dn"  redis:"multi_dn" json:"multi_dn"`
}

// TorrentStats is used to relay info stats for a torrent around. It contains rolled up stats
// from peer info as well as the normal torrent stats.
type TorrentStats struct {
	Seeders  int `json:"seeders"`
	Leechers int `json:"leechers"`
	Snatches int `json:"snatches"`
	Event    consts.AnnounceType
}

// NewTorrent allocates and returns a new Torrent instance pointer with all
// the minimum value required to operated in place
func NewTorrent(ih InfoHash, name string) Torrent {
	torrent := Torrent{
		ReleaseName: name,
		InfoHash:    ih,
		IsDeleted:   false,
		IsEnabled:   true,
		MultiUp:     1.0,
		MultiDn:     1.0,
	}
	return torrent
}

type Torrents []Torrent

// WhiteListClient defines a whitelisted bittorrent client allowed to participate
// in swarms. This is not a foolproof solution as its fairly trivial for a motivated
// attacker to fake this.
type WhiteListClient struct {
	ClientPrefix string `db:"client_prefix" json:"client_prefix"`
	ClientName   string `db:"client_name" json:"client_name"`
}

// Match returns true if the client matches this prefix
func (wl WhiteListClient) Match(client string) bool {
	return strings.HasPrefix(client, wl.ClientPrefix)
}
