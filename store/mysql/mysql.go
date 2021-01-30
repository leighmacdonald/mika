// Package mysql provides mysql/mariadb backed persistent storage
//
// NOTE this requires MySQL 8.0+ / MariaDB 10.5+ (maybe 10.4?) due to the POINT column type
package mysql

import (
	"context"
	// imported for side-effects
	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"github.com/leighmacdonald/mika/config"
	"github.com/leighmacdonald/mika/consts"
	"github.com/leighmacdonald/mika/store"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"sync"
	"time"
)

const (
	driverName = "mysql"
)

// ErrNoResults is the string returned from the driver when no rows are returned
const ErrNoResults = "sql: no rows in result set"

// MariaDBStore is the MariaDB backed store.Store implementation
type MariaDBStore struct {
	db *sqlx.DB
}

func (s *MariaDBStore) UserRoles(userID uint32) ([]store.Role, error) {
	const q = `
		SELECT r.role_id, r.role_name, r.priority, r.multi_up, r.multi_down, 
		       r.download_enabled, r.upload_enabled, r.created_on, r.updated_on 
		FROM roles r
		LEFT JOIN user_roles ur on r.role_id = ur.role_id
		WHERE ur.user_id = ?`
	var roles store.Roles
	if err := s.db.Select(&roles, q, userID); err != nil {
		return nil, err
	}
	return roles, nil
}

func (s *MariaDBStore) RoleAdd(role store.Role) error {
	const q = `
		INSERT INTO roles 
		    (role_name, priority, multi_up, multi_down, download_enabled, upload_enabled, created_on, updated_on) 
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`
	_, err := s.db.Exec(q, role.RoleName, role.Priority, role.MultiUp, role.MultiDown, role.DownloadEnabled,
		role.UploadEnabled, role.CreatedOn, role.UpdateOn)
	if err != nil {
		return errors.Wrap(err, "Failed to create role")
	}
	return nil
}

func (s *MariaDBStore) RoleDelete(roleID int) error {
	const q = `DELETE FROM roles WHERE role_id = ?`
	if _, err := s.db.Exec(q, roleID); err != nil {
		return errors.Wrap(err, "Failed to delete role")
	}
	return nil
}

func (s *MariaDBStore) Roles() (store.Roles, error) {
	const q = `
		SELECT 
		    role_id, role_name, priority, multi_up, multi_down, download_enabled, 
       		upload_enabled, created_on, updated_on 
		FROM roles`
	var roles store.Roles
	if err := s.db.Select(&roles, q); err != nil {
		return nil, err
	}
	return roles, nil
}

// Sync batch updates the backing store with the new UserStats provided
func (s *MariaDBStore) UserSync(b map[string]store.UserStats) error {
	const q = ` UPDATE users
    SET announces  = (announces + ?),
        uploaded   = (uploaded + ?),
        downloaded = (downloaded + ?)
    WHERE passkey = ?;`
	// TODO use ctx for timeout
	ctx := context.Background()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "Failed to being user Sync() tx")
	}
	stmt, err := tx.Prepare(q)
	if err != nil {
		return errors.Wrap(err, "Failed to prepare user Sync() tx")
	}
	for passkey, stats := range b {
		_, err := stmt.Exec(passkey, stats.Announces, stats.Uploaded, stats.Downloaded)
		if err != nil {
			if err := tx.Rollback(); err != nil {
				log.Errorf("Failed to roll back user Sync() tx")
			}
			return errors.Wrap(err, "Failed to exec user Sync() tx")
		}
	}
	if err := tx.Commit(); err != nil {
		return errors.Wrap(err, "Failed to commit user Sync() tx")
	}

	return nil
}

// Add will add a new user to the backing store
func (s *MariaDBStore) UserAdd(user store.User) error {
	const q = `INSERT INTO users
    (user_id, passkey, download_enabled, is_deleted, downloaded, uploaded, announces)
    VALUES (?, ?, ?, ?, ?, ?, ?);`
	_, err := s.db.Exec(q, user.UserID, user.Passkey, user.DownloadEnabled,
		user.IsDeleted, user.Downloaded, user.Uploaded, user.Announces)
	if err != nil {
		return errors.Wrap(err, "Failed to add user to store")
	}
	return nil
}

// GetByPasskey will lookup and return the user via their passkey used as an identifier
// The errors returned for this method should be very generic and not reveal any info
// that could possibly help attackers gain any insight. All error cases MUST
// return ErrUnauthorized.
func (s *MariaDBStore) UserGetByPasskey(user *store.User, passkey string) error {
	const q = `
		SELECT u.user_id,
           u.passkey,
           u.download_enabled,
           u.is_deleted,
           u.downloaded,
           u.uploaded,
           u.announces
		FROM users u
		WHERE u.passkey = ?;`
	if err := s.db.Get(user, q, passkey); err != nil {
		if err.Error() == ErrNoResults {
			return consts.ErrInvalidUser
		}
		return errors.Wrap(err, "Could not query user by passkey")
	}
	r, err := s.UserRoles(user.UserID)
	if err != nil {
		return errors.Wrap(err, "Could not query user roles by user_id")
	}
	user.Roles = r
	return nil
}

// GetByID returns a user matching the userId
func (s *MariaDBStore) UserGetByID(user *store.User, userID uint32) error {
	const q = `
		SELECT u.user_id,
           u.passkey,
           u.download_enabled,
           u.is_deleted,
           u.downloaded,
           u.uploaded,
           u.announces
    	FROM users u
    	WHERE u.user_id = ?`
	if err := s.db.Get(user, q, userID); err != nil {
		if err.Error() == ErrNoResults {
			return consts.ErrInvalidUser
		}
		return errors.Wrap(err, "Could not query user by user_id")
	}
	r, err := s.UserRoles(user.UserID)
	if err != nil {
		return errors.Wrap(err, "Could not query user roles by user_id")
	}
	user.Roles = r
	return nil
}

// Delete removes a user from the backing store
func (s *MariaDBStore) UserDelete(user store.User) error {
	if user.UserID == 0 {
		return errors.New("User doesnt have a user_id")
	}
	const q = `DELETE FROM users WHERE user_id = ?;`
	if _, err := s.db.Exec(q, user.UserID); err != nil {
		return errors.Wrap(err, "Failed to delete user")
	}
	user.UserID = 0
	return nil
}

func (s *MariaDBStore) UserUpdate(user store.User, oldPasskey string) error {
	const q = `
		UPDATE users
		SET user_id          = ?,
			passkey          = ?,
			download_enabled = ?,
			is_deleted       = ?,
			downloaded       = ?,
			uploaded         = ?,
			announces        = ?
		WHERE passkey = ?`
	if _, err := s.db.Exec(q, user.UserID, user.Passkey, user.DownloadEnabled,
		user.IsDeleted, user.Downloaded, user.Uploaded, user.Announces,
		oldPasskey); err != nil {
		return errors.Wrapf(err, "Failed to update user")
	}
	return nil
}

// Close will close the underlying database connection and clear the local caches
func (s *MariaDBStore) Close() error {
	return s.db.Close()
}

func (s *MariaDBStore) Name() string {
	return driverName
}

func (s *MariaDBStore) TorrentUpdate(torrent store.Torrent) error {
	const q = `
		UPDATE 
		    torrent 
		SET
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
func (s *MariaDBStore) TorrentSync(b map[store.InfoHash]store.TorrentStats) error {
	const q = ` 
		UPDATE
			torrent
		SET total_downloaded = (total_downloaded + ?),
			total_uploaded   = (total_uploaded + ?),
			announces        = (announces + ?),
			total_completed  = (total_completed + ?),
			seeders          = ?,
			leechers         = ?
		WHERE info_hash = ?`
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
			stats.Downloaded,
			stats.Uploaded,
			stats.Announces,
			stats.Snatches,
			stats.Seeders,
			stats.Leechers,
			ih.Bytes(),
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
func (s *MariaDBStore) Conn() interface{} {
	return s.db
}

// WhiteListDelete removes a client from the global whitelist
func (s *MariaDBStore) WhiteListDelete(client store.WhiteListClient) error {
	const q = `DELETE FROM whitelist WHERE client_prefix = ?`
	if _, err := s.db.Exec(q, client.ClientPrefix); err != nil {
		return errors.Wrap(err, "Failed to delete client whitelist")
	}
	return nil
}

// WhiteListAdd will insert a new client prefix into the allowed clients list
func (s *MariaDBStore) WhiteListAdd(client store.WhiteListClient) error {
	const q = ` INSERT INTO whitelist (client_prefix, client_name) VALUES (?, ?);`
	if _, err := s.db.Exec(q, client.ClientPrefix, client.ClientName); err != nil {
		return errors.Wrap(err, "Failed to insert new whitelist entry")
	}
	return nil
}

// WhiteListGetAll fetches all known whitelisted clients
func (s *MariaDBStore) WhiteListGetAll() ([]store.WhiteListClient, error) {
	var wl []store.WhiteListClient
	const q = `SELECT client_prefix, client_name FROM whitelist;`
	if err := s.db.Select(&wl, q); err != nil {
		return nil, errors.Wrap(err, "Failed to select client whitelists")
	}
	return wl, nil
}

// Get returns a torrent for the hash provided
func (s *MariaDBStore) TorrentGet(t *store.Torrent, hash store.InfoHash, deletedOk bool) error {
	const q = `
		SELECT 
			info_hash,
           	total_uploaded,
           	total_downloaded,
           	total_completed,
           	is_deleted,
           	is_enabled,
           	reason,
           	multi_up,
           	multi_dn,
           	seeders,
           	leechers,
           	announces
    	FROM 
    	    torrent
    	WHERE 
    	    is_deleted = ? AND info_hash = ?`
	err := s.db.Get(t, q, deletedOk, hash.Bytes())
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
func (s *MariaDBStore) TorrentAdd(t store.Torrent) error {
	const q = `INSERT INTO torrent (info_hash) VALUES (?);`
	_, err := s.db.Exec(q, t.InfoHash.Bytes())
	if err != nil {
		return err
	}
	return nil
}

// Delete will mark a torrent as deleted in the backing store.
// If dropRow is true, it will permanently remove the torrent from the store
func (s *MariaDBStore) TorrentDelete(ih store.InfoHash, dropRow bool) error {
	var err error
	if dropRow {
		const dropQ = `DELETE FROM torrent WHERE info_hash = ?;`
		_, err = s.db.Exec(dropQ, ih.Bytes())
	} else {
		const updateQ = ` UPDATE torrent SET is_deleted = true WHERE info_hash = ?;`
		_, err = s.db.NamedExec(updateQ, ih.Bytes())
	}
	if err != nil {
		return err
	}
	return nil
}

var (
	connections   map[string]*sqlx.DB
	connectionsMu *sync.RWMutex
)

func getOrCreateConn(cfg config.StoreConfig) (*sqlx.DB, error) {
	connectionsMu.Lock()
	defer connectionsMu.Unlock()
	existing, found := connections[cfg.Host]
	if found {
		return existing, nil
	}
	db, err := sqlx.Connect(driverName, cfg.DSN())
	if err != nil {
		return nil, errors.Wrap(err, "Could not connect to mysql database")
	}
	db.SetMaxOpenConns(50)
	db.SetMaxIdleConns(50)
	db.SetConnMaxLifetime(time.Second * 10)
	connections[cfg.Host] = db
	return db, nil
}

type driver struct{}

// New creates a new mysql backed user store.
func (ud driver) New(cfg config.StoreConfig) (store.Store, error) {
	db, err := getOrCreateConn(cfg)
	if err != nil {
		return nil, err
	}
	return &MariaDBStore{db: db}, nil
}

func init() {
	connections = make(map[string]*sqlx.DB)
	connectionsMu = &sync.RWMutex{}
	store.AddDriver(driverName, driver{})
}
