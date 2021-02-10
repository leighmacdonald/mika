package mysql

import (
	"github.com/jmoiron/sqlx"
	"github.com/leighmacdonald/mika/config"
	"github.com/leighmacdonald/mika/store"
	"github.com/leighmacdonald/mika/util"
	"io/ioutil"
	"os"
	"testing"
)

const schemaDrop = "store/mysql/drop.sql"
const schemaCreate = "store/mysql/schema.sql"

func TestDriver(t *testing.T) {
	// multiStatements=true is required to exec the full schema at once
	db := sqlx.MustConnect(driverName, config.Store.DSN())
	store.TestStore(t, &Driver{db: db})
}

func execSchema(db *sqlx.DB, schemaPath string) {
	schema := util.FindFile(schemaPath)
	b, err := ioutil.ReadFile(schema)
	if err != nil {
		panic("Cannot read schema file")
	}
	db.MustExec(string(b))
}

func TestMain(m *testing.M) {
	config.General.RunMode = "test"
	config.Store.Type = driverName
	config.Store.User = "mika"
	config.Store.Password = "mika"
	config.Store.Database = "mika"
	config.Store.Host = "localhost"
	config.Store.Port = 3307
	config.Store.Properties = "parseTime=true&multiStatements=true"
	db, err := sqlx.Connect(driverName, config.Store.DSN())
	if err != nil {
		os.Exit(0)
	}
	execSchema(db, schemaDrop)
	execSchema(db, schemaCreate)
	defer execSchema(db, schemaDrop)
	exitCode := m.Run()
	os.Exit(exitCode)
}
