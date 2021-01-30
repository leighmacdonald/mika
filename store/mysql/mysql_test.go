package mysql

import (
	"github.com/jmoiron/sqlx"
	"github.com/leighmacdonald/mika/config"
	"github.com/leighmacdonald/mika/store"
	"github.com/leighmacdonald/mika/util"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"os"
	"testing"
)

const schemaDrop = "store/mysql/drop.sql"
const schemaCreate = "store/mysql/schema.sql"

func TestDriver(t *testing.T) {
	// multiStatements=true is required to exec the full schema at once
	db := sqlx.MustConnect(driverName, config.TorrentStore.DSN())
	store.TestStore(t, &MariaDBStore{db: db})
}

func execSchema(db *sqlx.DB, schemaPath string) {
	schema := util.FindFile(schemaPath)
	b, err := ioutil.ReadFile(schema)
	if err != nil {
		panic("Cannot read schema file")
	}
	db.MustExec(string(b))
	//for _, stmt := range strings.Split(string(b), "-- STMT") {
	//	if !strings.HasPrefix(stmt, "-- ") && strings.Contains(stmt, ";") {
	//		log.Debugf("SQL: %s", stmt)
	//		db.MustExec(string(b))
	//	}
	//}
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
