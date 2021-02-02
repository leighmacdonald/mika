// Package mysql provides mysql/mariadb backed persistent storage
//
// NOTE this requires MySQL 8.0+ / MariaDB 10.5+ (maybe 10.4?) due to the POINT column type
package mysql

import (
	"context"
	"github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"github.com/leighmacdonald/mika/config"
	"github.com/leighmacdonald/mika/consts"
	"github.com/leighmacdonald/mika/store"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"time"
)

const (
	driverName = "mysql"
)

// ErrNoResults is the string returned from the driver when no rows are returned
const ErrNoResults = "sql: no rows in result set"

// Driver is the MariaDB backed store.Store implementation
type Driver struct {
	db *sqlx.DB
}

func (s *Driver) Users() (store.Users, error) {
	const q = `
		SELECT user_id, role_id, is_deleted, downloaded, uploaded, 
		       announces, passkey, download_enabled 
		FROM user`
	var users []*store.User
	if err := s.db.Select(&users, q); err != nil {
		return nil, errors.Wrap(err, "Failed to get all users")
	}
	result := store.Users{}
	for _, u := range users {
		result[u.Passkey] = u
	}
	return result, nil
}

func (s *Driver) Torrents() (store.Torrents, error) {
	const q = `
		SELECT info_hash, total_uploaded, total_downloaded, total_completed, 
		       is_deleted, is_enabled, reason, multi_up, multi_dn, seeders, leechers, announces,
		       title, created_on, updated_on
		FROM torrent`
	var torrents []*store.Torrent
	if err := s.db.Select(&torrents, q); err != nil {
		return nil, errors.Wrap(err, "Failed to get all torrents")
	}
	result := store.Torrents{}
	for _, t := range torrents {
		result[t.InfoHash] = t
	}
	return result, nil
}

func (s *Driver) RoleSave(role *store.Role) error {
	const q = `
		UPDATE role 
		SET download_enabled = :download_enabled, upload_enabled = :upload_enabled, 
		    multi_down = :multi_down, multi_up = :multi_up, 
		    priority = :priority, role_name = :role_name
		WHERE role_id = ?`
	if _, err := s.db.NamedExec(q, role); err != nil {
		return errors.Wrap(err, "Failed to save role")
	}
	return nil
}

func (s *Driver) RoleByID(role *store.Role, roleID uint32) error {
	const q = `
		SELECT 
       		role_id, role_name, priority, multi_up, multi_down, 
       		download_enabled, upload_enabled, created_on, updated_on 
		FROM role 
		WHERE role_id = ?`
	if err := s.db.Get(role, q, roleID); err != nil {
		if err.Error() == ErrNoResults {
			return consts.ErrInvalidRole
		}
		return errors.Wrap(err, "Could not query user by passkey")
	}
	return nil
}

func (s *Driver) RoleAdd(role *store.Role) error {
	const q = `
		INSERT INTO role 
		    (role_name, priority, multi_up, multi_down, download_enabled, upload_enabled, created_on, updated_on) 
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`
	res, err := s.db.Exec(q, role.RoleName, role.Priority, role.MultiUp, role.MultiDown, role.DownloadEnabled,
		role.UploadEnabled, role.CreatedOn, role.UpdateOn)
	if err != nil {
		return errors.Wrap(err, "Failed to create role")
	}
	i, err := res.LastInsertId()
	if err != nil {
		return errors.Wrap(err, "Failed to get role id")
	}
	role.RoleID = uint32(i)
	return nil
}

func (s *Driver) RoleDelete(roleID uint32) error {
	const q = `DELETE FROM role WHERE role_id = ?`
	if _, err := s.db.Exec(q, roleID); err != nil {
		return errors.Wrap(err, "Failed to delete role")
	}
	return nil
}

func (s *Driver) Roles() (store.Roles, error) {
	const q = `
		SELECT 
		    role_id, role_name, priority, multi_up, multi_down, download_enabled, 
       		upload_enabled, created_on, updated_on 
		FROM role`
	var roles []*store.Role
	if err := s.db.Select(&roles, q); err != nil {
		return nil, err
	}
	result := store.Roles{}
	for _, t := range roles {
		result[t.RoleID] = t
	}
	return result, nil
}

// Sync batch updates the backing store with the new UserStats provided
func (s *Driver) UserSync(b []*store.User) error {
	const q = ` UPDATE user
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
		_, err := stmt.Exec(stats.Announces, stats.Uploaded, stats.Downloaded, passkey)
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
func (s *Driver) UserAdd(user *store.User) error {
	if user.RoleID == 0 {
		return errors.New("Must supply at least 1 role")
	}
	const q = `INSERT INTO user
    (passkey, download_enabled, is_deleted, downloaded, uploaded, announces, role_id, remote_id)
    VALUES (?, ?, ?, ?, ?, ?, ?, ?);`
	res, err2 := s.db.Exec(q, user.Passkey, user.DownloadEnabled,
		user.IsDeleted, user.Downloaded, user.Uploaded, user.Announces, user.RoleID, user.RemoteID)
	if err2 != nil {
		return errors.Wrap(err2, "Failed to add user to store")
	}
	i, err3 := res.LastInsertId()
	if err3 != nil {
		return errors.Wrap(err3, "Failed to get role id")
	}
	user.UserID = uint32(i)
	r := &store.Role{}
	if err := s.RoleByID(r, user.RoleID); err != nil {
		return errors.Wrap(err, "Failed to load role")
	}
	user.Role = r
	return nil
}

// GetByPasskey will lookup and return the user via their passkey used as an identifier
// The errors returned for this method should be very generic and not reveal any info
// that could possibly help attackers gain any insight. All error cases MUST
// return ErrUnauthorized.
func (s *Driver) UserGetByPasskey(user *store.User, passkey string) error {
	const q = `
		SELECT 
		    u.user_id,
           	u.passkey,
           	u.download_enabled,
           	u.is_deleted,
           	u.downloaded,
           	u.uploaded,
           	u.announces,
			u.role_id
		FROM user u
		WHERE u.passkey = ?;`
	if err := s.db.Get(user, q, passkey); err != nil {
		if err.Error() == ErrNoResults {
			return consts.ErrInvalidUser
		}
		return errors.Wrap(err, "Could not query user by passkey")
	}
	r := &store.Role{}
	if err := s.RoleByID(r, user.RoleID); err != nil {
		return errors.Wrap(err, "Failed to load role")
	}
	user.Role = r
	return nil
}

// GetByID returns a user matching the userId
func (s *Driver) UserGetByID(user *store.User, userID uint32) error {
	const q = `
		SELECT 
		    u.user_id,
           	u.passkey,
           	u.download_enabled,
           	u.is_deleted,
           	u.downloaded,
           	u.uploaded,
           	u.announces,
			u.role_id
    	FROM user u
    	WHERE u.user_id = ?`
	if err := s.db.Get(user, q, userID); err != nil {
		if err.Error() == ErrNoResults {
			return consts.ErrInvalidUser
		}
		return errors.Wrap(err, "Could not query user by user_id")
	}
	r := &store.Role{}
	if err := s.RoleByID(r, user.RoleID); err != nil {
		return errors.Wrap(err, "Failed to load role")
	}
	user.Role = r
	return nil
}

// Delete removes a user from the backing store
func (s *Driver) UserDelete(user *store.User) error {
	if user.UserID == 0 {
		return errors.New("User doesnt have a user_id")
	}
	const q = `DELETE FROM user WHERE user_id = ?;`
	if _, err := s.db.Exec(q, user.UserID); err != nil {
		return errors.Wrap(err, "Failed to delete user")
	}
	user.UserID = 0
	return nil
}

func (s *Driver) UserSave(user *store.User) error {
	const q = `
		UPDATE user
		SET
			passkey          = ?,
			download_enabled = ?,
			is_deleted       = ?,
			downloaded       = ?,
			uploaded         = ?,
			announces        = ?
		WHERE user_id = ?`
	if _, err := s.db.Exec(q, user.Passkey, user.DownloadEnabled,
		user.IsDeleted, user.Downloaded, user.Uploaded, user.Announces,
		user.UserID); err != nil {
		return errors.Wrapf(err, "Failed to update user")
	}
	return nil
}

// Close will close the underlying database connection and clear the local caches
func (s *Driver) Close() error {
	return s.db.Close()
}

func (s *Driver) Name() string {
	return driverName
}

func (s *Driver) TorrentUpdate(torrent *store.Torrent) error {
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
func (s *Driver) TorrentSync(b []*store.Torrent) error {
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
	for _, t := range b {
		if _, err := stmt.Exec(
			t.Downloaded,
			t.Uploaded,
			t.Announces,
			t.Snatches,
			t.Seeders,
			t.Leechers,
			t.InfoHash.Bytes(),
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
func (s *Driver) Conn() interface{} {
	return s.db
}

// WhiteListDelete removes a client from the global whitelist
func (s *Driver) WhiteListDelete(client *store.WhiteListClient) error {
	const q = `DELETE FROM whitelist WHERE client_prefix = ?`
	if _, err := s.db.Exec(q, client.ClientPrefix); err != nil {
		return errors.Wrap(err, "Failed to delete client whitelist")
	}
	return nil
}

// WhiteListAdd will insert a new client prefix into the allowed clients list
func (s *Driver) WhiteListAdd(client *store.WhiteListClient) error {
	const q = ` INSERT INTO whitelist (client_prefix, client_name) VALUES (?, ?);`
	if _, err := s.db.Exec(q, client.ClientPrefix, client.ClientName); err != nil {
		return errors.Wrap(err, "Failed to insert new whitelist entry")
	}
	return nil
}

// WhiteListGetAll fetches all known whitelisted clients
func (s *Driver) WhiteListGetAll() ([]*store.WhiteListClient, error) {
	var wl []*store.WhiteListClient
	const q = `SELECT client_prefix, client_name FROM whitelist;`
	if err := s.db.Select(&wl, q); err != nil {
		return nil, errors.Wrap(err, "Failed to select client whitelists")
	}
	return wl, nil
}

// Get returns a torrent for the hash provided
func (s *Driver) TorrentGet(t *store.Torrent, hash store.InfoHash, deletedOk bool) error {
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
func (s *Driver) TorrentAdd(t *store.Torrent) error {
	t.CreatedOn = time.Now()
	t.UpdatedOn = t.CreatedOn
	const q = `
		INSERT INTO torrent (info_hash, multi_up, multi_dn, title, created_on, updated_on) 
		VALUES (?, ?, ?, ?, ?, ?);`
	_, err := s.db.Exec(q, t.InfoHash.Bytes(), t.MultiUp, t.MultiDn, t.Title, t.CreatedOn, t.UpdatedOn)
	if err != nil {
		myErr, ok := err.(*mysql.MySQLError)
		if ok { // MySQL error
			if myErr.Number == 1062 {
				return consts.ErrDuplicate
			}
		}
		return errors.Wrap(err, "Failed to add torrent to store")
	}
	return nil
}

// Delete will mark a torrent as deleted in the backing store.
// If dropRow is true, it will permanently remove the torrent from the store
func (s *Driver) TorrentDelete(ih store.InfoHash, dropRow bool) error {
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

type driver struct{}

// New creates a new mysql backed user store.
func (ud driver) New(cfg config.StoreConfig) (store.Store, error) {
	db, err := sqlx.Connect(driverName, cfg.DSN())
	if err != nil {
		return nil, errors.Wrap(err, "Could not connect to mysql database")
	}
	db.SetMaxOpenConns(50)
	db.SetMaxIdleConns(50)
	db.SetConnMaxLifetime(time.Second * 10)
	return &Driver{db: db}, nil
}

func init() {
	store.AddDriver(driverName, driver{})
}
