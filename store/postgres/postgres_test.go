package postgres

import (
	"github.com/jmoiron/sqlx"
	"github.com/leighmacdonald/mika/config"
	"github.com/leighmacdonald/mika/store"
	"testing"
)

func TestTorrentDriver(t *testing.T) {
	s, err := store.NewTorrentStore(driverName, config.GetStoreConfig(config.Torrent))
	if err != nil {
		t.Skipf("Skipping postgres tests, could not instantiate driver: %s", err)
	}
	c := s.Conn().(*sqlx.DB)
	store.ClearTables(c, []string{"torrent", "whitelist"})
	store.TestTorrentStore(t, s)
}
