package model

import (
	"fmt"
	"strings"
	"sync"
)

type InfoHash [20]byte

type Torrent struct {
	sync.RWMutex
	Name            string   `redis:"name" json:"name"`
	TorrentID       uint64   `redis:"torrent_id" json:"torrent_id"`
	InfoHash        InfoHash `redis:"info_hash" json:"info_hash"`
	Seeders         int      `redis:"seeders" json:"seeders"`
	Leechers        int      `redis:"leechers" json:"leechers"`
	Snatches        int16    `redis:"snatches" json:"snatches"`
	Announces       uint64   `redis:"announces" json:"announces"`
	Uploaded        uint64   `redis:"uploaded" json:"uploaded"`
	Downloaded      uint64   `redis:"downloaded" json:"downloaded"`
	TorrentKey      string   `redis:"-" json:"-"`
	TorrentPeersKey string   `redis:"-" json:"-"`
	Enabled         bool     `redis:"enabled" json:"enabled"`
	Reason          string   `redis:"reason" json:"reason"`
	Peers           []*Peer  `redis:"-" json:"peers"`
	MultiUp         float64  `redis:"multi_up" json:"multi_up"`
	MultiDn         float64  `redis:"multi_dn" json:"multi_dn"`
}

type TorrentStats struct {
	TorrentID uint64 `json:"torrent_id"`
	InfoHash  string `json:"info_hash"`
	Seeders   int    `json:"seeders"`
	Leechers  int    `json:"leechers"`
	Snatches  int    `json:"snatches"`
}

// NewTorrent allocates and returns a new Torrent instance pointer with all
// the minimum value required to operated in place
func NewTorrent(ih string, name string, tid uint64) *Torrent {
	torrent := &Torrent{
		Name:            name,
		InfoHash:        strings.ToLower(ih),
		TorrentKey:      fmt.Sprintf("t:t:%s", ih),
		TorrentPeersKey: fmt.Sprintf("t:tpeers:%s", ih),
		Enabled:         true,
		Peers:           []*Peer{},
		TorrentID:       tid,
		MultiUp:         1.0,
		MultiDn:         1.0,
	}
	return torrent
}
