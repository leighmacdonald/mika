package mysql

import (
	"fmt"
	"github.com/jmoiron/sqlx"
	"github.com/leighmacdonald/mika/config"
	"github.com/leighmacdonald/mika/store"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"os"
	"testing"
)

func TestTorrentDriver(t *testing.T) {
	// multiStatements=true is required to exec the full schema at once
	dsn := fmt.Sprintf("%s?multiStatements=true", config.GetStoreConfig(config.Torrent).DSN())
	db := sqlx.MustConnect(driverName, dsn)
	setupDB(t, db)
	store.TestTorrentStore(t, &TorrentStore{db: db})
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
	config.Read("mika_testing_mysql")
	if viper.GetString(string(config.GeneralRunMode)) != "test" {
		log.Info("Skipping database tests, not running in testing mode")
		os.Exit(0)
		return
	}
	exitCode := m.Run()
	os.Exit(exitCode)
}
