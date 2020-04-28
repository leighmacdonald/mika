package store

import (
	"fmt"
	"github.com/jmoiron/sqlx"
	log "github.com/sirupsen/logrus"
	"net/url"
)

// Shared common sql(x) config opts
type Config struct {
	Type       string
	Host       string
	Port       int
	Username   string
	Password   string
	DB         string
	Properties string
	Conn       *sqlx.DB
}

// DSN constructs a uri for database connection strings
//
// protocol//[user]:[password]@[hosts][/database][?properties]
func (c Config) DSN() string {
	props := c.Properties
	if props != "" {
		props = "?" + props
	}
	s := fmt.Sprintf("%s//%s:%s@%s:%d/%s%s",
		c.Type, c.Username, c.Password, c.Host, c.Port, c.DB, props)
	u, err := url.Parse(s)
	if err != nil {
		log.Fatalf("Failed to construct valid database DSN: %s", err.Error())
		return ""
	}
	return u.String()
}
