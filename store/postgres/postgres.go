// Package postgres provides the backing store for postgresql
// TODO create domains for the uint types, eg: create domain uint64 as numeric(20,0);
package postgres

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v4"
	"github.com/leighmacdonald/mika/config"
	"github.com/leighmacdonald/mika/consts"
	"github.com/leighmacdonald/mika/store"
	"github.com/leighmacdonald/mika/util"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"time"
)

const (
	driverName = "postgres"
)

// UserStore is the postgres backed store.Store implementation
type Driver struct {
	db  *pgx.Conn
	ctx context.Context
}

func (d *Driver) TorrentSave(torrent *store.Torrent) error {
	return nil
}

func (d *Driver) Migrate() error { return nil }

func (d *Driver) Users() (store.Users, error) {
	panic("implement me")
}

func (d *Driver) Torrents() (store.Torrents, error) {
	panic("implement me")
}

func (d *Driver) RoleSave(role *store.Role) error {
	panic("implement me")
}

func (d *Driver) Roles() (store.Roles, error) {
	panic("implement me")
}

func (d *Driver) RoleByID(role *store.Role, roleID uint32) error {
	panic("implement me")
}

func (d *Driver) RoleAdd(role *store.Role) error {
	panic("implement me")
}

func (d *Driver) RoleDelete(roleID uint32) error {
	panic("implement me")
}

func (d *Driver) UserSave(user *store.User) error {
	const q = `
		UPDATE
			users
		SET
		    passkey = $1,
		    is_deleted = $2,
		    download_enabled = $3,
		    downloaded = $4,
		    uploaded = $5,
		    announces = $6
		WHERE
			user_id = $7
	`
	c, cancel := context.WithDeadline(d.ctx, time.Now().Add(5*time.Second))
	defer cancel()
	_, err := d.db.Exec(c, q, user.Passkey, user.IsDeleted, user.DownloadEnabled,
		user.Downloaded, user.Uploaded, user.Announces, user.UserID)
	if err != nil {
		return errors.Wrapf(err, "Failed to update user: %d", user.UserID)
	}
	return nil
}

// Sync batch updates the backing store with the new UserStats provided
func (d *Driver) UserSync(batch []*store.User) error {
	const txName = "userSync"
	const q = `
		UPDATE 
			users
		SET
			downloaded = (downloaded + $1),
		    uploaded = (uploaded + $2),
		    announces = (announces + $3)
		WHERE
			passkey = $4
`
	c, cancel := context.WithDeadline(d.ctx, time.Now().Add(time.Second*10))
	defer cancel()
	tx, err := d.db.Begin(c)
	if err != nil {
		return errors.Wrap(err, "postgres.Store.Sync Failed to being transaction")
	}
	defer func() { _ = tx.Rollback(c) }()
	_, err = tx.Prepare(c, txName, q)
	if err != nil {
		return errors.Wrap(err, "postgres.Store.Sync Failed to being transaction")
	}

	for passkey, stats := range batch {
		if _, err := tx.Exec(c, txName, stats.Downloaded, stats.Uploaded, stats.Announces, passkey); err != nil {
			return errors.Wrapf(err, "postgres.Store.Sync failed to Exec tx")
		}
	}
	if err := tx.Commit(c); err != nil {
		return errors.Wrapf(err, "postgres.Store.Sync failed to commit tx")
	}
	return nil
}

// Add will add a new user to the backing store
func (d *Driver) UserAdd(user *store.User) error {
	c, cancel := context.WithDeadline(d.ctx, time.Now().Add(5*time.Second))
	defer cancel()
	const q = `
		INSERT INTO users 
		    (user_id, passkey, download_enabled, is_deleted, downloaded, uploaded, announces) 
		VALUES
		    ($1, $2, $3, $4, $5, $6, $7)`
	_, err := d.db.Exec(c, q, user.UserID, user.Passkey, user.DownloadEnabled, user.IsDeleted,
		user.Downloaded, user.Uploaded, user.Announces)
	if err != nil {
		return errors.Wrap(err, "Failed to add user to store")
	}
	return nil
}

// GetByPasskey will lookup and return the user via their passkey used as an identifier
// The errors returned for this method should be very generic and not reveal any info
// that could possibly help attackers gain any insight. All error cases MUST
// return ErrUnauthorized.
func (d *Driver) UserGetByPasskey(user *store.User, passkey string) error {
	const q = `
		SELECT 
		    user_id, passkey, download_enabled, is_deleted, downloaded, uploaded, announces 
		FROM 
		    users 
		WHERE 
		    passkey = $1`
	c, cancel := context.WithDeadline(d.ctx, time.Now().Add(5*time.Second))
	defer cancel()
	err := d.db.QueryRow(c, q, passkey).Scan(&user.UserID, &user.Passkey, &user.DownloadEnabled, &user.IsDeleted,
		&user.Downloaded, &user.Uploaded, &user.Announces)
	if err != nil {
		return errors.Wrap(err, "Failed to fetch user by passkey")
	}
	return nil
}

// GetByID returns a user matching the userId
func (d *Driver) UserGetByID(user *store.User, userID uint32) error {
	const q = `
		SELECT 
		    user_id, passkey, download_enabled, is_deleted, downloaded, uploaded, announces 
		FROM 
		    users 
		WHERE 
		    user_id = $1`
	c, cancel := context.WithDeadline(d.ctx, time.Now().Add(5*time.Second))
	defer cancel()
	err := d.db.QueryRow(c, q, userID).Scan(&user.UserID, &user.Passkey, &user.DownloadEnabled, &user.IsDeleted,
		&user.Downloaded, &user.Uploaded, &user.Announces)
	if err != nil {
		return errors.Wrap(err, "Failed to fetch user by user_id")
	}
	return nil
}

// Delete removes a user from the backing store
func (d *Driver) UserDelete(user *store.User) error {
	if user.UserID == 0 {
		return errors.New("User doesnt have a user_id")
	}
	const q = `DELETE FROM users WHERE user_id = $1`
	c, cancel := context.WithDeadline(d.ctx, time.Now().Add(5*time.Second))
	defer cancel()
	if _, err := d.db.Exec(c, q, user.UserID); err != nil {
		return errors.Wrap(err, "Failed to delete user")
	}
	user.UserID = 0
	return nil
}

// TorrentUpdate
func (d *Driver) TorrentUpdate(torrent *store.Torrent) error {
	const q = `
		UPDATE 
		    torrent 
		SET
		    info_hash = $1,
		    total_completed = $2,
		    total_uploaded = $3, 
		    total_downloaded = $4,
		    is_deleted = $5,
		    is_enabled = $6,
		    reason = $7,
		    multi_up = $8,
		    multi_dn = $9,
		    announces = $10
		WHERE
			info_hash = $11
			`
	c, cancel := context.WithDeadline(d.ctx, time.Now().Add(5*time.Second))
	defer cancel()
	_, err := d.db.Exec(c, q, torrent.InfoHash.Bytes(), torrent.Snatches,
		torrent.Uploaded, torrent.Downloaded, torrent.IsDeleted, torrent.IsEnabled,
		torrent.Reason, torrent.MultiUp, torrent.MultiDn, torrent.Announces)
	if err != nil {
		return errors.Wrapf(err, "Failed to update torrent: %s", torrent.InfoHash.String())
	}
	return nil
}

// Sync batch updates the backing store with the new TorrentStats provided
// TODO test cases
func (d *Driver) TorrentSync(batch []*store.Torrent) error {
	const txName = "torrentSync"
	const q = `
		UPDATE 
			torrent
		SET
			seeders = (seeders + $1),
		    leechers = (leechers + $2),
		    total_completed = (total_completed + $3),
		    total_downloaded = (total_downloaded + $4),
		    total_uploaded = (total_uploaded + $5),
		    announces = (announces + $6)
		WHERE
			info_hash = $7
`
	c, cancel := context.WithDeadline(d.ctx, time.Now().Add(time.Second*10))
	defer cancel()
	tx, err := d.db.Begin(c)
	if err != nil {
		return errors.Wrap(err, "postgres.Store.Sync Failed to being transaction")
	}
	defer func() { _ = tx.Rollback(c) }()
	_, err = tx.Prepare(c, txName, q)
	if err != nil {
		return errors.Wrap(err, "postgres.Store.Sync Failed to being transaction")
	}

	//for ih, stats := range batch {
	//	if _, err := tx.Exec(c, txName, stats.Seeders, stats.Leechers, stats.Snatches,
	//		stats.Downloaded, stats.Uploaded, stats.Announces, ih.Bytes()); err != nil {
	//		return errors.Wrapf(err, "postgres.Store.Sync failed to Exec tx")
	//	}
	//}
	if err := tx.Commit(c); err != nil {
		return errors.Wrapf(err, "postgres.Store.Sync failed to commit tx")
	}
	return nil
}

// Conn returns the underlying database driverInit
func (d *Driver) Conn() interface{} {
	return d.db
}

// Add inserts a new torrent into the backing store
func (d *Driver) TorrentAdd(t *store.Torrent) error {
	const q = `INSERT INTO torrent (info_hash) VALUES($1::bytea)`
	//log.Println(t.InfoHash.Bytes())
	c, cancel := context.WithDeadline(d.ctx, time.Now().Add(5*time.Second))
	defer cancel()
	commandTag, err := d.db.Exec(c, q, t.InfoHash.Bytes())
	if err != nil {
		return err
	}
	if commandTag.RowsAffected() != 1 {
		return errors.New("Failed to insert new torrent to store 0 rows affected")
	}
	return nil
}

// Delete will mark a torrent as deleted in the backing store.
// If dropRow is true, it will permanently remove the torrent from the store
func (d *Driver) TorrentDelete(ih store.InfoHash, dropRow bool) error {
	const dropQ = `DELETE FROM torrent WHERE info_hash = $1`
	const updateQ = `UPDATE torrent SET is_deleted = 1 WHERE info_hash = $1`
	var query string
	if dropRow {
		query = dropQ
	} else {
		query = updateQ
	}
	c, cancel := context.WithDeadline(d.ctx, time.Now().Add(5*time.Second))
	defer cancel()
	commandTag, err := d.db.Exec(c, query, ih.Bytes())
	if err != nil {
		return err
	}
	if commandTag.RowsAffected() != 1 {
		return consts.ErrInvalidInfoHash
	}
	return nil
}

// TorrentGet returns a torrent for the hash provided
func (d *Driver) TorrentGet(t *store.Torrent, ih store.InfoHash, deletedOk bool) error {
	const q = `
		SELECT 
			info_hash::bytea, total_uploaded, total_downloaded, total_completed, 
			is_deleted, is_enabled, reason, multi_up, multi_dn, announces, seeders, leechers
		FROM 
		    torrent 
		WHERE 
		    info_hash = $1 AND is_deleted = false`
	c, cancel := context.WithDeadline(d.ctx, time.Now().Add(5*time.Second))
	defer cancel()
	var b []byte
	err := d.db.QueryRow(c, q, ih.Bytes()).Scan(
		&b, // TODO implement pgx custom types to map automatically
		&t.Uploaded,
		&t.Downloaded,
		&t.Snatches,
		&t.IsDeleted,
		&t.IsEnabled,
		&t.Reason,
		&t.MultiUp,
		&t.MultiDn,
		&t.Announces,
		&t.Seeders,
		&t.Leechers,
	)
	copy(t.InfoHash[:], b)
	if err != nil {
		if err.Error() == "no rows in result set" {
			return consts.ErrInvalidInfoHash
		}
		return err
	}
	if t.IsDeleted && !deletedOk {
		return consts.ErrInvalidInfoHash
	}
	return nil
}

// Close will close the underlying postgres database connection
func (d *Driver) Close() error {
	c, cancel := context.WithDeadline(d.ctx, time.Now().Add(15*time.Second))
	defer cancel()
	return d.db.Close(c)
}

// WhiteListDelete removes a client from the global whitelist
func (d *Driver) WhiteListDelete(client *store.WhiteListClient) error {
	const q = `DELETE FROM whitelist WHERE client_prefix = $1`
	c, cancel := context.WithDeadline(d.ctx, time.Now().Add(5*time.Second))
	defer cancel()
	commandTag, err := d.db.Exec(c, q, client.ClientPrefix)
	if err != nil {
		return errors.Wrap(err, "Failed to delete client whitelist")
	}
	if commandTag.RowsAffected() != 1 {
		return errors.New("insert ok, but no row modified")
	}
	return nil
}

// WhiteListAdd will insert a new client prefix into the allowed clients list
func (d *Driver) WhiteListAdd(client *store.WhiteListClient) error {
	const q = `INSERT INTO whitelist (client_prefix, client_name) VALUES ($1, $2)`
	c, cancel := context.WithDeadline(d.ctx, time.Now().Add(5*time.Second))
	defer cancel()
	commandTag, err := d.db.Exec(c, q, client.ClientPrefix, client.ClientName)
	if err != nil {
		return errors.Wrap(err, "Failed to insert new whitelist entry")
	}
	if commandTag.RowsAffected() != 1 {
		return errors.New("Failed to insert, but no error?")
	}
	return nil
}

// WhiteListGetAll fetches all known whitelisted clients
func (d *Driver) WhiteListGetAll() ([]*store.WhiteListClient, error) {
	var wl []*store.WhiteListClient
	const q = `SELECT client_prefix, client_name FROM whitelist`
	c, cancel := context.WithDeadline(d.ctx, time.Now().Add(5*time.Second))
	defer cancel()
	rows, err := d.db.Query(c, q)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to select client whitelists")
	}
	defer rows.Close()
	for rows.Next() {
		var client store.WhiteListClient
		err = rows.Scan(&client.ClientPrefix, &client.ClientName)
		if err != nil {
			return nil, errors.Wrap(err, "Failed to fetch client whitelist")
		}
		wl = append(wl, &client)
	}
	return wl, nil
}

func (d *Driver) Name() string {
	return driverName
}

// Reap will loop through the peers removing any stale entries from active swarms
// TODO fetch peer hashes for expired peers to flush local caches
func (d *Driver) Reap() []store.PeerHash {
	// NOW() - INTERVAL '15 minutes'
	var peerHashes []store.PeerHash
	const q = `DELETE FROM peers WHERE announce_last < $1`
	c, cancel := context.WithDeadline(d.ctx, util.Now().Add(5*time.Second))
	defer cancel()
	rows, err := d.db.Exec(c, q, util.Now().Add(-(15 * time.Minute)))
	if err != nil {
		log.Errorf("failed to reap peers: %s", err.Error())
		return nil
	}
	if rows.RowsAffected() > 0 {
		log.Debugf("Reaped %d peers", rows.RowsAffected())
	}
	return peerHashes
}

type driverInit struct{}

// New initialize a Store implementation using the postgres backing store
func (td driverInit) New(cfg config.StoreConfig) (store.Store, error) {
	db, err := pgx.Connect(context.Background(), cfg.DSN())
	if err != nil {
		return nil, errors.Wrap(err, "failed to connect to postgres torrent store")
	}
	return &Driver{db: db, ctx: context.Background()}, nil
}

func makeDSN(c config.StoreConfig) string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s%s",
		c.User, c.Password, c.Host, c.Port, c.Database, c.Properties)
}

func init() {
	store.AddDriver(driverName, driverInit{})
}
