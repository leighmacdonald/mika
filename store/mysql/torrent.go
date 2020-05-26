// Package mysql provides mysql/mariadb backed persistent storage
//
// NOTE this requires MySQL 8.0+ / MariaDB 10.5+ (maybe 10.4?) due to the POINT column type
package mysql

import (
	// imported for side-effects
	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"github.com/leighmacdonald/mika/config"
	"github.com/leighmacdonald/mika/consts"
	"github.com/leighmacdonald/mika/model"
	"github.com/leighmacdonald/mika/store"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

const (
	driverName = "mysql"
)

// TorrentStore implements the store.TorrentStore interface for mysql
type TorrentStore struct {
	db    *sqlx.DB
	cache *store.TorrentCache
}

// Sync batch updates the backing store with the new TorrentStats provided
func (s *TorrentStore) Sync(b map[model.InfoHash]model.TorrentStats) error {
	const q = `
		UPDATE 
		    torrent
		SET 
			total_downloaded = (total_downloaded + ?),
		    total_uploaded = (total_uploaded + ?),
		    announces = (announces + ?),
		    total_completed = (total_completed + ?)
		WHERE
			info_hash = ?
		`
	tx, err := s.db.Begin()
	if err != nil {
		return errors.Wrap(err, "Failed to being torrent Sync() tx")
	}
	stmt, err := tx.Prepare(q)
	if err != nil {
		return errors.Wrap(err, "Failed to prepare torrent Sync() tx")
	}
	for ih, stats := range b {
		_, err := stmt.Exec(
			stats.Downloaded,
			stats.Uploaded,
			stats.Announces,
			stats.Snatches,
			ih.Bytes())
		if err != nil {
			if err := tx.Rollback(); err != nil {
				log.Errorf("Failed to roll back torrent Sync() tx")
			}
			return errors.Wrap(err, "Failed to exec torrent Sync() tx")
		}
	}
	if err := tx.Commit(); err != nil {
		return errors.Wrap(err, "Failed to commit torrent Sync() tx")
	}
	return nil
}

// Conn returns the underlying database driver
func (s *TorrentStore) Conn() interface{} {
	return s.db
}

// WhiteListDelete removes a client from the global whitelist
func (s *TorrentStore) WhiteListDelete(client model.WhiteListClient) error {
	const q = `DELETE FROM whitelist WHERE client_prefix = ?`
	if _, err := s.db.Exec(q, client.ClientPrefix); err != nil {
		return errors.Wrap(err, "Failed to delete client whitelist")
	}
	return nil
}

// WhiteListAdd will insert a new client prefix into the allowed clients list
func (s *TorrentStore) WhiteListAdd(client model.WhiteListClient) error {
	const q = `INSERT INTO whitelist (client_prefix, client_name) VALUES (:client_prefix, :client_name)`
	if _, err := s.db.NamedExec(q, client); err != nil {
		return errors.Wrap(err, "Failed to insert new whitelist entry")
	}
	return nil
}

// WhiteListGetAll fetches all known whitelisted clients
func (s *TorrentStore) WhiteListGetAll() ([]model.WhiteListClient, error) {
	var wl []model.WhiteListClient
	const q = `SELECT * FROM whitelist`
	if err := s.db.Select(&wl, q); err != nil {
		return nil, errors.Wrap(err, "Failed to select client whitelists")
	}
	return wl, nil
}

// Close will close the underlying mysql database connection
func (s *TorrentStore) Close() error {
	return s.db.Close()
}

// Get returns a torrent for the hash provided
func (s *TorrentStore) Get(t *model.Torrent, hash model.InfoHash) error {
	const q = `SELECT * FROM torrent WHERE info_hash = ? AND is_deleted = false`
	if err := s.cache.Get(t, hash); err == nil {
		log.Debugf("Got cached torrent: %s", t.InfoHash.String())
		return nil
	}
	err := s.db.Get(t, q, hash.Bytes())
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			return consts.ErrInvalidInfoHash
		}
		return err
	}
	s.cache.Add(*t)
	log.Debugf("Added torrent to cache (Get()): %s", t.InfoHash.String())
	return nil
}

// Add inserts a new torrent into the backing store
func (s *TorrentStore) Add(t model.Torrent) error {
	const q = `INSERT INTO torrent (info_hash, release_name) VALUES(?, ?)`
	_, err := s.db.Exec(q, t.InfoHash.Bytes(), t.ReleaseName)
	if err != nil {
		return err
	}
	s.cache.Add(t)
	log.Debugf("Added torrent to cache (Add()): %s", t.InfoHash.String())
	return nil
}

// Delete will mark a torrent as deleted in the backing store.
// If dropRow is true, it will permanently remove the torrent from the store
func (s *TorrentStore) Delete(ih model.InfoHash, dropRow bool) error {
	var err error
	if dropRow {
		const dropQ = `DELETE FROM torrent WHERE info_hash = ?`
		_, err = s.db.Exec(dropQ, ih.Bytes())
	} else {
		const updateQ = `UPDATE torrent SET is_deleted = 1 WHERE info_hash = ?`
		_, err = s.db.NamedExec(updateQ, ih.Bytes())

	}
	if err != nil {
		return err
	}
	s.cache.Delete(ih, dropRow)
	return nil
}

type torrentDriver struct{}

// New initialize a TorrentStore implementation using the mysql backing store
func (td torrentDriver) New(cfg interface{}) (store.TorrentStore, error) {
	c, ok := cfg.(*config.StoreConfig)
	if !ok {
		return nil, consts.ErrInvalidConfig
	}
	db := sqlx.MustConnect(driverName, c.DSN())
	return &TorrentStore{
		db:    db,
		cache: store.NewTorrentCache(true),
	}, nil
}

func init() {
	store.AddTorrentDriver(driverName, torrentDriver{})
}
