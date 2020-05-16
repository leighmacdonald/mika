package memory

import (
	"github.com/leighmacdonald/mika/store"
	"testing"
)

func TestMemoryTorrentStore(t *testing.T) {
	store.TestTorrentStore(t, NewTorrentStore())
}

func TestMemoryPeerStore(t *testing.T) {
	store.TestPeerStore(t, NewPeerStore(), NewTorrentStore(), NewUserStore())
}

func TestMemoryUserStore(t *testing.T) {
	store.TestUserStore(t, NewUserStore())
}
