package postgres

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v4"
	"github.com/leighmacdonald/mika/config"
	"github.com/leighmacdonald/mika/store"
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
	db.Exec(context.Background(), schema)
	t.Cleanup(func() {
		clearDB(db)
	})
}

func TestMain(m *testing.M) {
	config.Read("mika_testing_postgres")
	if viper.GetString(string(config.GeneralRunMode)) != "test" {
		log.Info("Skipping database tests, not running in testing mode")
		os.Exit(0)
		return
	}
	exitCode := m.Run()
	os.Exit(exitCode)
}
