// Package mysql provides mysql/mariadb backed persistent storage
//
// NOTE this requires MySQL 8.0+ / MariaDB 10.5+ (maybe 10.4?) due to the POINT column type
package mysql

import (
	// imported for side-effects
	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
	"mika/config"
	"mika/consts"
	"mika/model"
	"mika/store"
)

const (
	driverName = "mysql"
)

// TorrentStore implements the store.TorrentStore interface for mysql
type TorrentStore struct {
	db *sqlx.DB
}

func (s *TorrentStore) WhiteListDel(client model.WhiteListClient) error {
	panic("implement me")
}

func (s *TorrentStore) WhiteListAdd(client model.WhiteListClient) error {
	panic("implement me")
}

func (s *TorrentStore) WhiteListGetAll() ([]model.WhiteListClient, error) {
	panic("implement me")
}

// Close will close the underlying mysql database connection
func (s *TorrentStore) Close() error {
	return s.db.Close()
}

// GetTorrent returns a torrent for the hash provided
func (s *TorrentStore) GetTorrent(hash model.InfoHash) (*model.Torrent, error) {
	const q = `SELECT * FROM torrent WHERE info_hash = ? AND is_deleted = false`
	var t *model.Torrent
	if err := s.db.Get(t, q, hash.String()); err != nil {
		return nil, err
	}
	return t, nil
}

// AddTorrent inserts a new torrent into the backing store
func (s *TorrentStore) AddTorrent(t *model.Torrent) error {
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

// DeleteTorrent will mark a torrent as deleted in the backing store.
// If dropRow is true, it will permanently remove the torrent from the store
func (s *TorrentStore) DeleteTorrent(ih model.InfoHash, dropRow bool) error {
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
