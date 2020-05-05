package memory

import (
	"github.com/leighmacdonald/mika/store"
	"testing"
)

func TestMemoryTorrentStore(t *testing.T) {
	td := torrentDriver{}
	ts, _ := td.NewTorrentStore(nil)
	store.TestTorrentStore(t, ts)
}

func TestMemoryPeerStore(t *testing.T) {
	td := torrentDriver{}
	ts, _ := td.NewTorrentStore(nil)
	pd := peerDriver{}
	ps, _ := pd.NewPeerStore(nil)
	store.TestPeerStore(t, ps, ts)
}
