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
)

const (
	driverName = "mysql"
)

// TorrentStore implements the store.TorrentStore interface for mysql
type TorrentStore struct {
	db *sqlx.DB
}

// WhiteListDelete removes a client from the global whitelist
func (s *TorrentStore) WhiteListDelete(client model.WhiteListClient) error {
	panic("implement me")
}

// WhiteListAdd will insert a new client prefix into the allowed clients list
func (s *TorrentStore) WhiteListAdd(client model.WhiteListClient) error {
	panic("implement me")
}

// WhiteListGetAll fetches all known whitelisted clients
func (s *TorrentStore) WhiteListGetAll() ([]model.WhiteListClient, error) {
	panic("implement me")
}

// Close will close the underlying mysql database connection
func (s *TorrentStore) Close() error {
	return s.db.Close()
}

// Get returns a torrent for the hash provided
func (s *TorrentStore) Get(hash model.InfoHash) (*model.Torrent, error) {
	const q = `SELECT * FROM torrent WHERE info_hash = ? AND is_deleted = false`
	var t *model.Torrent
	if err := s.db.Get(t, q, hash.String()); err != nil {
		return nil, err
	}
	return t, nil
}

// Add inserts a new torrent into the backing store
func (s *TorrentStore) Add(t *model.Torrent) error {
	if t.TorrentID > 0 {
		return errors.New("Torrent ID already attached")
	}
	const q = `INSERT INTO torrent (info_hash, release_name, created_on, updated_on) VALUES( ?, ?, ?, ?)`
	res, err := s.db.NamedExec(q, t)
	if err != nil {
		return err
	}
	lastID, err := res.LastInsertId()
	if err != nil {
		return errors.New("Failed to fetch insert ID")
	}
	t.TorrentID = uint32(lastID)
	return nil
}

// Delete will mark a torrent as deleted in the backing store.
// If dropRow is true, it will permanently remove the torrent from the store
func (s *TorrentStore) Delete(ih model.InfoHash, dropRow bool) error {
	if dropRow {
		const dropQ = `DELETE FROM torrent WHERE info_hash = ?`
		_, err := s.db.Exec(dropQ, ih)
		if err != nil {
			return err
		}
	} else {
		const updateQ = `UPDATE torrent SET is_deleted = 1 WHERE info_hash = ?`
		_, err := s.db.NamedExec(updateQ, ih)
		if err != nil {
			return err
		}
	}
	return nil
}

type torrentDriver struct{}

// NewTorrentStore initialize a TorrentStore implementation using the mysql backing store
func (td torrentDriver) NewTorrentStore(cfg interface{}) (store.TorrentStore, error) {
	c, ok := cfg.(*config.StoreConfig)
	if !ok {
		return nil, consts.ErrInvalidConfig
	}
	db := sqlx.MustConnect(driverName, c.DSN())
	return &TorrentStore{
		db: db,
	}, nil
}

func init() {
	store.AddTorrentDriver(driverName, torrentDriver{})
}
