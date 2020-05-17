package postgres

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v4"
	"github.com/leighmacdonald/mika/config"
	"github.com/leighmacdonald/mika/store"
	"github.com/leighmacdonald/mika/store/memory"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"os"
	"testing"
)

func TestTorrentDriver(t *testing.T) {
	c := config.GetStoreConfig(config.Torrent)
	db, err := pgx.Connect(context.Background(), makeDSN(c))
	if err != nil {
		t.Skipf("failed to connect to postgres torrent store: %s", err.Error())
		return
	}
	setupDB(t, db)
	store.TestTorrentStore(t, &TorrentStore{db: db, ctx: context.Background()})
}

func TestUserDriver(t *testing.T) {
	db, err := pgx.Connect(context.Background(), makeDSN(config.GetStoreConfig(config.Users)))
	if err != nil {
		t.Skipf("failed to connect to postgres user store: %s", err.Error())
		return
	}
	setupDB(t, db)
	store.TestUserStore(t, &UserStore{
		db:  db,
		ctx: context.Background(),
	})
}

func TestPeerDriver(t *testing.T) {
	db, err := pgx.Connect(context.Background(), makeDSN(config.GetStoreConfig(config.Peers)))
	if err != nil {
		t.Skipf("failed to connect to postgres user store: %s", err.Error())
		return
	}
	setupDB(t, db)
	store.TestPeerStore(t, NewPeerStore(db), memory.NewTorrentStore(), memory.NewUserStore())
}

func clearDB(db *pgx.Conn) {
	ctx := context.Background()
	for _, table := range []string{"peers", "torrent", "users", "whitelist"} {
		q := fmt.Sprintf(`drop table if exists %s cascade;`, table)
		if _, err := db.Exec(ctx, q); err != nil {
			log.Panicf("Failed to prep database: %s", err.Error())
		}
	}
}

func setupDB(t *testing.T, db *pgx.Conn) {
	clearDB(db)
	if _, err := db.Exec(context.Background(), schema); err != nil {
		log.Panicf("Failed to setupDB: %s", err)
	}
	t.Cleanup(func() {
		clearDB(db)
	})
}

func TestMain(m *testing.M) {
	if err := config.Read("mika_testing_postgres"); err != nil {
		log.Info("Skipping database tests, failed to find config: mika_testing_postgres.yaml")
		os.Exit(0)
		return
	}
	if viper.GetString(string(config.GeneralRunMode)) != "test" {
		log.Info("Skipping database tests, not running in testing mode")
		os.Exit(0)
		return
	}
	exitCode := m.Run()
	os.Exit(exitCode)
}
