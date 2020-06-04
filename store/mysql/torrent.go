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
	"github.com/leighmacdonald/mika/store"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

const (
	driverName = "mysql"
)

// TorrentStore implements the store.TorrentStore interface for mysql
type TorrentStore struct {
	db *sqlx.DB
}

func (s *TorrentStore) Name() string {
	return driverName
}

func (s *TorrentStore) Update(torrent store.Torrent) error {
	const q = `
		UPDATE 
		    torrent 
		SET
			release_name = ?,
		    info_hash = ?,
		    total_completed = ?,
		    total_uploaded = ?,
		    total_downloaded = ?,
		    is_deleted = ?,
		    is_enabled = ?,
		    reason = ?,
		    multi_up = ?,
		    multi_dn = ?,
		    announces = ?
		WHERE
			info_hash = ?
			`
	_, err := s.db.Exec(q,
		torrent.ReleaseName,
		torrent.InfoHash.Bytes(),
		torrent.Snatches,
		torrent.Uploaded,
		torrent.Downloaded,
		torrent.IsDeleted,
		torrent.IsEnabled,
		torrent.Reason,
		torrent.MultiUp,
		torrent.MultiDn,
		torrent.Announces,
		torrent.InfoHash.Bytes())
	if err != nil {
		return errors.Wrap(err, "Failed to update torrent")
	}
	return nil
}

// Sync batch updates the backing store with the new TorrentStats provided
func (s *TorrentStore) Sync(b map[store.InfoHash]store.TorrentStats) error {
	const q = `CALL torrent_update_stats(?, ?, ?, ?, ?, ?, ?)`
	tx, err := s.db.Begin()
	if err != nil {
		return errors.Wrap(err, "Failed to being torrent Sync() tx")
	}
	stmt, err2 := tx.Prepare(q)
	if err2 != nil {
		return errors.Wrap(err2, "Failed to prepare torrent Sync() tx")
	}
	for ih, stats := range b {
		if _, err := stmt.Exec(
			ih.Bytes(),
			stats.Downloaded,
			stats.Uploaded,
			stats.Announces,
			stats.Snatches,
			stats.Seeders,
			stats.Leechers,
		); err != nil {
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
func (s *TorrentStore) WhiteListDelete(client store.WhiteListClient) error {
	const q = `CALL whitelist_delete_by_prefix(?)`
	if _, err := s.db.Exec(q, client.ClientPrefix); err != nil {
		return errors.Wrap(err, "Failed to delete client whitelist")
	}
	return nil
}

// WhiteListAdd will insert a new client prefix into the allowed clients list
func (s *TorrentStore) WhiteListAdd(client store.WhiteListClient) error {
	const q = `CALL whitelist_add(?, ?)`
	if _, err := s.db.Exec(q, client.ClientPrefix, client.ClientName); err != nil {
		return errors.Wrap(err, "Failed to insert new whitelist entry")
	}
	return nil
}

// WhiteListGetAll fetches all known whitelisted clients
func (s *TorrentStore) WhiteListGetAll() ([]store.WhiteListClient, error) {
	var wl []store.WhiteListClient
	const q = `CALL whitelist_all()`
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
func (s *TorrentStore) Get(t *store.Torrent, hash store.InfoHash, deletedOk bool) error {
	const q = `CALL torrent_by_infohash(?, ?)`
	err := s.db.Get(t, q, hash.Bytes(), deletedOk)
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			return consts.ErrInvalidInfoHash
		}
		return err
	}
	if t.IsDeleted && !deletedOk {
		return consts.ErrInvalidInfoHash
	}
	return nil
}

// Add inserts a new torrent into the backing store
func (s *TorrentStore) Add(t store.Torrent) error {
	const q = `CALL torrent_add(?, ?)`
	_, err := s.db.Exec(q, t.InfoHash.Bytes(), t.ReleaseName)
	if err != nil {
		return err
	}
	return nil
}

// Delete will mark a torrent as deleted in the backing store.
// If dropRow is true, it will permanently remove the torrent from the store
func (s *TorrentStore) Delete(ih store.InfoHash, dropRow bool) error {
	var err error
	if dropRow {
		const dropQ = `CALL torrent_delete(?)`
		_, err = s.db.Exec(dropQ, ih.Bytes())
	} else {
		const updateQ = `CALL torrent_disable(?)`
		_, err = s.db.NamedExec(updateQ, ih.Bytes())

	}
	if err != nil {
		return err
	}
	return nil
}

type torrentDriver struct{}

// New initialize a TorrentStore implementation using the mysql backing store
func (td torrentDriver) New(cfg interface{}) (store.TorrentStore, error) {
	c, ok := cfg.(*config.StoreConfig)
	if !ok {
		return nil, consts.ErrInvalidConfig
	}
	db, err := sqlx.Connect(driverName, c.DSN())
	if err != nil {
		return nil, err
	}
	return &TorrentStore{
		db: db,
	}, nil
}

func init() {
	store.AddTorrentDriver(driverName, torrentDriver{})
}
