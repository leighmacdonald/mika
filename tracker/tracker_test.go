package tracker

import (
	"fmt"
	"git.totdev.in/totv/mika/db"
	"math/rand"
	"testing"
)

func MakeTestTracker() *Tracker {
	if db.Pool == nil {
		db.Setup("localhost:6379", "")
	}
	trk := NewTracker()
	return trk
}

var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890")

func RandSeq(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func GenerateTorrent() *Torrent {
	ih := RandSeq(32)
	return NewTorrent(ih, fmt.Sprintf("Torrent.Name.%s-group", ih[0:4]), uint64(rand.Intn(100000)))
}

func TestTrackerAdd(t *testing.T) {
	torrent_a := GenerateTorrent()
	tkr := MakeTestTracker()
	tkr.AddTorrent(torrent_a)
	r := db.Pool.Get()
	defer r.Close()
	torrent_a_added := tkr.GetTorrentByID(r, torrent_a.TorrentID, false)
	if torrent_a != torrent_a_added {
		t.Error("Torrents are not equal")
	}
	not_found := tkr.GetTorrentByID(r, 99999, false)
	if not_found != nil {
		t.Error("Unexpected return value")
	}
}

func TestTrackerDel(t *testing.T) {
	torrent_a := GenerateTorrent()
	tkr := MakeTestTracker()
	tkr.AddTorrent(torrent_a)
	r := db.Pool.Get()
	defer r.Close()
	torrent_a_added := tkr.GetTorrentByID(r, torrent_a.TorrentID, false)
	if torrent_a != torrent_a_added {
		t.Error("Torrents are not equal")
	}

	not_found := tkr.GetTorrentByID(r, 99999, false)
	if not_found != nil {
		t.Error("Unexpected return value")
	}
}
