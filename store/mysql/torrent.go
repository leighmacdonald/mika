// Package mysql provides mysql/mariadb backed persistent storage
//
// NOTE this requires MySQL 8.0+ / MariaDB 10.5+ (maybe 10.4?) due to the POINT column type
// +build mysql

package mysql

import (
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"mika/config"
	"mika/consts"
	"mika/model"
	"mika/store"
	"net/url"
)

const (
	driverName = "mysql"
)

// Shared common sql(x) config opts
type dbConfig struct {
	Type       string
	Host       string
	Port       int
	Username   string
	Password   string
	DB         string
	Properties string
}

// DSN constructs a uri for database connection strings
//
// protocol//[user]:[password]@[hosts][/database][?properties]
func (c dbConfig) DSN() string {
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

type TorrentStore struct {
	db *sqlx.DB
}

func (s *TorrentStore) Close() error {
	return s.db.Close()
}

func (s *TorrentStore) GetTorrent(hash model.InfoHash) (*model.Torrent, error) {
	const q = `SELECT * FROM torrent WHERE info_hash = ? AND is_deleted = false`
	var t *model.Torrent
	if err := s.db.Get(t, q); err != nil {
		return nil, err
	}
	return t, nil
}

func (s *TorrentStore) AddTorrent(t *model.Torrent) error {
	if t.TorrentID > 0 {
		return errors.New("Torrent ID already attached")
	}
	const q = `INSERT INTO torrent (info_hash, release_name, created_on, updated_on) VALUES( ?, ?, ?, ?)`
	res, err := s.db.NamedExec(q, t)
	if err != nil {
		return err
	}
	lastId, err := res.LastInsertId()
	if err != nil {
		return errors.New("Failed to fetch insert ID")
	}
	t.TorrentID = uint32(lastId)
	return nil
}

func (s *TorrentStore) DeleteTorrent(t *model.Torrent, dropRow bool) error {
	if dropRow {
		const dropQ = `DELETE FROM torrent WHERE torrent_id = :torrent_id`
		_, err := s.db.NamedExec(dropQ, t)
		if err != nil {
			return err
		}
	} else {
		const updateQ = `UPDATE torrent SET is_deleted = 1 WHERE torrent_id = :torrent_id`
		_, err := s.db.NamedExec(updateQ, t)
		if err != nil {
			return err
		}
	}
	return nil
}

type torrentDriver struct{}

func (td torrentDriver) NewTorrentStore(cfg interface{}) (store.TorrentStore, error) {
	c, ok := cfg.(*config.StoreConfig)
	if !ok {
		return nil, consts.ErrInvalidConfig
	}
	db := sqlx.MustConnect("mysql", c.DSN())
	return &TorrentStore{
		db: db,
	}, nil
}

func init() {
	store.AddTorrentDriver(driverName, torrentDriver{})
}
