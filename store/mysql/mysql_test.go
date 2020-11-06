package mysql

import (
	"github.com/jmoiron/sqlx"
	"github.com/leighmacdonald/mika/config"
	"github.com/leighmacdonald/mika/store"
	"github.com/leighmacdonald/mika/store/memory"
	"github.com/leighmacdonald/mika/util"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"os"
	"strings"
	"testing"
)

const schemaDrop = "store/mysql/drop.sql"
const schemaCreate = "store/mysql/schema.sql"

func TestTorrentDriver(t *testing.T) {
	// multiStatements=true is required to exec the full schema at once
	db := sqlx.MustConnect(driverName, config.TorrentStore.DSN())
	store.TestTorrentStore(t, &TorrentStore{db: db})
}

func TestUserDriver(t *testing.T) {
	db := sqlx.MustConnect(driverName, config.UserStore.DSN())
	store.TestUserStore(t, &UserStore{
		db: db,
	})
}

func TestPeerStore(t *testing.T) {
	db := sqlx.MustConnect(driverName, config.PeerStore.DSN())
	ts := memory.NewTorrentStore()
	us := memory.NewUserStore()
	store.TestPeerStore(t, &PeerStore{db: db}, ts, us)
}

func execSchema(db *sqlx.DB, schemaPath string) {
	schema := util.FindFile(schemaPath)
	b, err := ioutil.ReadFile(schema)
	if err != nil {
		panic("Cannot read schema file")
	}
	for _, stmt := range strings.Split(string(b), "-- STMT") {
		if !strings.HasPrefix(stmt, "-- ") && strings.Contains(stmt, ";") {
			log.Debugf("SQL: %s", stmt)
			db.MustExec(stmt)
		}
	}
}

func TestMain(m *testing.M) {
	if err := config.Read("mika_testing_mysql"); err != nil {
		log.Info("Skipping database tests, failed to find config: mika_testing_mysql.yaml")
		os.Exit(0)
		return
	}
	if config.General.RunMode != "test" {
		log.Info("Skipping database tests, not running in testing mode")
		os.Exit(0)
		return
	}
	db := sqlx.MustConnect(driverName, config.TorrentStore.DSN())
	execSchema(db, schemaDrop)
	execSchema(db, schemaCreate)
	defer execSchema(db, schemaDrop)
	exitCode := m.Run()
	os.Exit(exitCode)
}
