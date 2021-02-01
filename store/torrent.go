package store

import (
	"database/sql/driver"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/leighmacdonald/mika/consts"
	"github.com/leighmacdonald/mika/util"
	log "github.com/sirupsen/logrus"
	"strings"
	"time"
)

// PeerHash is a merger of the infohash and peer_id, used for simpler map lookups
type PeerHash [40]byte

// NewPeerHash created a new PeerHash from the existing infohash and peer_id
func NewPeerHash(ih InfoHash, pid PeerID) PeerHash {
	var buf [40]byte
	copy(buf[0:20], ih.Bytes())
	copy(buf[20:], pid.Bytes())
	return buf
}

// InfoHash returns the first 20 bytes of the data
func (ph PeerHash) InfoHash() InfoHash {
	var buf [20]byte
	copy(buf[:], ph[0:20])
	return buf
}

// String implements fmt.Stringer, returning the base16 encoded PeerID.
func (ph PeerHash) String() string {
	return fmt.Sprintf("%x", ph[:])
}

// PeerID returns the last 20 bytes of the data
func (ph PeerHash) PeerID() PeerID {
	var buf [20]byte
	copy(buf[:], ph[20:])
	return buf
}

// InfoHash is a unique 20byte identifier for a torrent
type InfoHash [20]byte

// InfoHashFromString returns a binary infohash from the info string
func InfoHashFromString(infoHash *InfoHash, s string) error {
	if len(s) != 20 {
		return consts.ErrInvalidInfoHash
	}
	copy(infoHash[:], s)
	return nil
}

// InfoHashFromHex returns a binary infohash from a byte array
func InfoHashFromHex(infoHash *InfoHash, h string) error {
	if len(h) != 40 {
		return consts.ErrInvalidInfoHash
	}
	b, err := hex.DecodeString(h)
	if err != nil {
		return err
	}
	copy(infoHash[:], b)
	return nil
}

// PeerHashFromHex returns a binary infohash from a byte array
func PeerHashFromHex(peerHash *PeerHash, h string) error {
	b, err := hex.DecodeString(h)
	if err != nil {
		return err
	}
	copy(peerHash[:], b)
	return nil
}

// InfoHashFromBytes returns a binary infohash from a byte array
func InfoHashFromBytes(infoHash *InfoHash, b []byte) error {
	copy(infoHash[:], b)
	return nil
}

// Value implements the database.Valuer interface
func (ih *InfoHash) Value() (driver.Value, error) {
	return ih.Bytes(), nil
}

// Scan implements the sql.Scanner interface for conversion to our custom type
func (ih *InfoHash) Scan(v interface{}) error {
	// Should be more strictly to check this type.
	vt, ok := v.([]byte)
	if !ok {
		return errors.New("failed to convert value to infohash")
	}
	cnt := copy(ih[:], vt)
	if cnt != 20 {
		return fmt.Errorf("invalid data length received: %d, expected 20", cnt)
	}
	return nil
}

// Bytes returns the raw bytes of the info_hash. This is primarily useful for inserting to SQL stores since
// they have trouble with the sized variant
func (ih InfoHash) Bytes() []byte {
	return ih[:]
}

// URLEncode returns the peer id suitably  encoded for a URL
func (ih InfoHash) URLEncode() string {
	return fmt.Sprintf("%s", ih.Bytes())
}

// String implements fmt.Stringer, returning the base16 encoded PeerID.
func (ih InfoHash) String() string {
	return fmt.Sprintf("%x", ih[:])
}

// RawString returns a 20-byte string of the raw bytes of the ID.
func (ih *InfoHash) RawString() string {
	return string(ih.Bytes())
}

// Torrent is the core struct for our torrent being tracked
type Torrent struct {
	InfoHash InfoHash `db:"info_hash" json:"info_hash"`
	Snatches uint16   `db:"total_completed" json:"total_completed"`
	// This is stored as MB to reduce storage costs
	Uploaded uint64 `db:"total_uploaded" json:"total_uploaded"`
	// This is stored as MB to reduce storage costs
	Downloaded uint64 `db:"total_downloaded" json:"total_downloaded"`
	IsDeleted  bool   `db:"is_deleted" json:"is_deleted"`
	// When you have a message to pass to a client set enabled = false and set the reason message.
	// If IsDeleted is true, then nothing will be returned to the client
	IsEnabled bool `db:"is_enabled" json:"is_enabled"`
	// Reason when set will return a message to the torrent client
	Reason string `db:"reason" json:"reason"`
	// Upload multiplier added to the users totals
	MultiUp float64 `db:"multi_up" json:"multi_up"`
	// Download multiplier added to the users totals
	// 0 denotes freeleech status
	MultiDn   float64 `db:"multi_dn" json:"multi_dn"`
	Announces uint64  `db:"announces" json:"announces"`
	Seeders   int     `db:"seeders" json:"seeders"`
	Leechers  int     `db:"leechers" json:"leechers"`

	Peers *Swarm
}

type TorrentUpdate struct {
	Keys        []string
	ReleaseName string  `json:"release_name"`
	IsDeleted   bool    `json:"is_deleted"`
	IsEnabled   bool    `json:"is_enabled"`
	Reason      string  `json:"reason"`
	MultiUp     float64 `json:"multi_up"`
	MultiDn     float64 `json:"multi_dn"`
}

// TorrentStats is used to relay info stats for a torrent around. It contains rolled up stats
// from peer info as well as the normal torrent stats.
type TorrentStats struct {
	Seeders    int    `json:"seeders"`
	Leechers   int    `json:"leechers"`
	Snatches   uint16 `json:"snatches"`
	Uploaded   uint64 `json:"uploaded"`
	Downloaded uint64 `json:"downloaded"`
	Announces  uint64 `json:"announces"`
}

// UserStats is any info we want to batch update for a user
type UserStats struct {
	Uploaded   uint64
	Downloaded uint64
	Announces  uint32
}

type AnnounceHist struct {
	Downloaded uint64
	Uploaded   uint64
	Timestamp  time.Time
}

// PeerStats is any info to batch peer updates
type PeerStats struct {
	Left   uint32
	Hist   []AnnounceHist
	Paused bool
}
type PeerSummary struct {
	TotalUp    uint64
	TotalDn    uint64
	SpeedUp    uint64
	SpeedDn    uint64
	SpeedUpMax uint64
	SpeedDnMax uint64
	LastAnn    time.Time
}

func (ps *PeerStats) Totals() PeerSummary {
	sum := PeerSummary{}
	for i, s := range ps.Hist {
		sum.TotalUp += s.Uploaded
		sum.TotalDn += s.Downloaded
		if i > 0 {
			sum.SpeedDn = util.EstSpeed(sum.LastAnn.Unix(), s.Timestamp.Unix(), s.Downloaded)
			sum.SpeedUp = util.EstSpeed(sum.LastAnn.Unix(), s.Timestamp.Unix(), s.Uploaded)
			if sum.SpeedDn > sum.SpeedDnMax {
				sum.SpeedDnMax = sum.SpeedDn
			}
			if sum.SpeedUp > sum.SpeedUpMax {
				sum.SpeedUpMax = sum.SpeedUp
			}
			log.Debugf("Dn: %s Up: %s",
				util.HumanBytesString(sum.SpeedDn), util.HumanBytesString(sum.SpeedUp))
		}
		sum.LastAnn = s.Timestamp
	}
	return sum
}

// NewTorrent allocates and returns a new Torrent instance pointer with all
// the minimum value required to operated in place
func NewTorrent(ih InfoHash) Torrent {
	torrent := Torrent{
		InfoHash:  ih,
		IsDeleted: false,
		IsEnabled: true,
		MultiUp:   1.0,
		MultiDn:   1.0,
	}
	return torrent
}

// Torrents is a basic type alias for multiple torrents
type Torrents map[InfoHash]*Torrent

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
