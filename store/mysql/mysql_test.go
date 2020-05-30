package mysql

import (
	"fmt"
	"github.com/jmoiron/sqlx"
	"github.com/leighmacdonald/mika/config"
	"github.com/leighmacdonald/mika/store"
	"github.com/leighmacdonald/mika/store/memory"
	log "github.com/sirupsen/logrus"
	"os"
	"sync"
	"testing"
)

func TestTorrentDriver(t *testing.T) {
	// multiStatements=true is required to exec the full schema at once
	db := sqlx.MustConnect(driverName, config.GetStoreConfig(config.Torrent).DSN())
	setupDB(t, db)
	store.TestTorrentStore(t, &TorrentStore{db: db})
}

func TestUserDriver(t *testing.T) {
	db := sqlx.MustConnect(driverName, config.GetStoreConfig(config.Torrent).DSN())
	setupDB(t, db)
	store.TestUserStore(t, &UserStore{
		db:      db,
		users:   map[string]store.User{},
		usersMx: sync.RWMutex{},
	})
}

func TestPeerStore(t *testing.T) {
	db := sqlx.MustConnect(driverName, config.GetStoreConfig(config.Peers).DSN())
	setupDB(t, db)
	ts := memory.NewTorrentStore()
	us := memory.NewUserStore()
	store.TestPeerStore(t, &PeerStore{db: db}, ts, us)
}

func clearDB(db *sqlx.DB) {
	for _, table := range []string{"peers", "torrent", "users", "whitelist"} {
		if _, err := db.Exec(fmt.Sprintf(`drop table if exists %s cascade;`, table)); err != nil {
			log.Panicf("Failed to prep database: %s", err.Error())
		}
	}
}

func setupDB(t *testing.T, db *sqlx.DB) {
	clearDB(db)
	db.MustExec(schema)
	t.Cleanup(func() {
		clearDB(db)
	})
}

func TestMain(m *testing.M) {
	if err := config.Read("mika_testing_mysql"); err != nil {
		log.Info("Skipping database tests, failed to find config: mika_testing_mysql.yaml")
		os.Exit(0)
		return
	}
	if config.GetString(config.GeneralRunMode) != "test" {
		log.Info("Skipping database tests, not running in testing mode")
		os.Exit(0)
		return
	}
	exitCode := m.Run()
	os.Exit(exitCode)
}
