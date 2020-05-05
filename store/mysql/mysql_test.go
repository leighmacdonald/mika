package mysql

import (
	"github.com/leighmacdonald/mika/config"
	"github.com/leighmacdonald/mika/store"
	"testing"
)

func TestTorrentDriver(t *testing.T) {
	config.Read("")
	s, err := store.NewTorrentStore(driverName, config.GetStoreConfig(config.Torrent))
	if err != nil {
		t.Skipf("Skipping mysql tests, could not instantiate driver: %s", err)
	}
	store.TestTorrentStore(t, s)
}
