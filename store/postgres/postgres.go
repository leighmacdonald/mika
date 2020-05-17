// Package postgres provides the backing store for postgresql
// TODO create domains for the uint types, eg: create domain uint64 as numeric(20,0);
package postgres

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v4"
	"github.com/leighmacdonald/mika/config"
	"github.com/leighmacdonald/mika/consts"
	"github.com/leighmacdonald/mika/model"
	"github.com/leighmacdonald/mika/store"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"time"
)

const (
	driverName = "postgres"
)

// UserStore is the postgres backed store.UserStore implementation
type UserStore struct {
	db  *pgx.Conn
	ctx context.Context
}

// Sync batch updates the backing store with the new UserStats provided
func (us UserStore) Sync(batch map[string]model.UserStats) error {
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
	c, cancel := context.WithDeadline(us.ctx, time.Now().Add(time.Second*10))
	defer cancel()
	tx, err := us.db.Begin(c)
	if err != nil {
		return errors.Wrap(err, "postgres.UserStore.Sync Failed to being transaction")
	}
	defer func() { _ = tx.Rollback(c) }()
	_, err = tx.Prepare(c, txName, q)
	if err != nil {
		return errors.Wrap(err, "postgres.UserStore.Sync Failed to being transaction")
	}

	for passkey, stats := range batch {
		if _, err := tx.Exec(c, txName, stats.Downloaded, stats.Uploaded, stats.Announces, passkey); err != nil {
			return errors.Wrapf(err, "postgres.UserStore.Sync failed to Exec tx")
		}
	}
	if err := tx.Commit(c); err != nil {
		return errors.Wrapf(err, "postgres.UserStore.Sync failed to commit tx")
	}
	return nil
}

// Add will add a new user to the backing store
func (us UserStore) Add(user model.User) error {
	c, cancel := context.WithDeadline(us.ctx, time.Now().Add(5*time.Second))
	defer cancel()
	const q = `
		INSERT INTO users 
		    (user_id, passkey, download_enabled, is_deleted, downloaded, uploaded, announces) 
		VALUES
		    ($1, $2, $3, $4, $5, $6, $7)`
	_, err := us.db.Exec(c, q, user.UserID, user.Passkey, user.DownloadEnabled, user.IsDeleted,
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
func (us UserStore) GetByPasskey(user *model.User, passkey string) error {
	const q = `
		SELECT 
		    user_id, passkey, download_enabled, is_deleted, downloaded, uploaded, announces 
		FROM 
		    users 
		WHERE 
		    passkey = $1`
	c, cancel := context.WithDeadline(us.ctx, time.Now().Add(5*time.Second))
	defer cancel()
	err := us.db.QueryRow(c, q, passkey).Scan(&user.UserID, &user.Passkey, &user.DownloadEnabled, &user.IsDeleted,
		&user.Downloaded, &user.Uploaded, &user.Announces)
	if err != nil {
		return errors.Wrap(err, "Failed to fetch user by passkey")
	}
	return nil
}

// GetByID returns a user matching the userId
func (us UserStore) GetByID(user *model.User, userID uint32) error {
	const q = `
		SELECT 
		    user_id, passkey, download_enabled, is_deleted, downloaded, uploaded, announces 
		FROM 
		    users 
		WHERE 
		    user_id = $1`
	c, cancel := context.WithDeadline(us.ctx, time.Now().Add(5*time.Second))
	defer cancel()
	err := us.db.QueryRow(c, q, userID).Scan(&user.UserID, &user.Passkey, &user.DownloadEnabled, &user.IsDeleted,
		&user.Downloaded, &user.Uploaded, &user.Announces)
	if err != nil {
		return errors.Wrap(err, "Failed to fetch user by user_id")
	}
	return nil
}

// Delete removes a user from the backing store
func (us UserStore) Delete(user model.User) error {
	if user.UserID == 0 {
		return errors.New("User doesnt have a user_id")
	}
	const q = `DELETE FROM users WHERE user_id = $1`
	c, cancel := context.WithDeadline(us.ctx, time.Now().Add(5*time.Second))
	defer cancel()
	if _, err := us.db.Exec(c, q, user.UserID); err != nil {
		return errors.Wrap(err, "Failed to delete user")
	}
	user.UserID = 0
	return nil
}

// Close will close the underlying database connection and clear the local caches
func (us UserStore) Close() error {
	c, cancel := context.WithDeadline(us.ctx, time.Now().Add(15*time.Second))
	defer cancel()
	return us.db.Close(c)
}

// TorrentStore implements the store.TorrentStore interface for postgres
type TorrentStore struct {
	db  *pgx.Conn
	ctx context.Context
}

// Sync batch updates the backing store with the new TorrentStats provided
func (ts TorrentStore) Sync(_ map[model.InfoHash]model.TorrentStats) error {
	panic("implement me")
}

// Conn returns the underlying database driver
func (ts TorrentStore) Conn() interface{} {
	return ts.db
}

// Add inserts a new torrent into the backing store
func (ts TorrentStore) Add(t model.Torrent) error {
	const q = `INSERT INTO torrent (info_hash, release_name) VALUES($1::bytea, $2)`
	//log.Println(t.InfoHash.Bytes())
	c, cancel := context.WithDeadline(ts.ctx, time.Now().Add(5*time.Second))
	defer cancel()
	commandTag, err := ts.db.Exec(c, q, t.InfoHash.Bytes(), t.ReleaseName)
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
func (ts TorrentStore) Delete(ih model.InfoHash, dropRow bool) error {
	const dropQ = `DELETE FROM torrent WHERE info_hash = $1`
	const updateQ = `UPDATE torrent SET is_deleted = 1 WHERE info_hash = $1`
	var query string
	if dropRow {
		query = dropQ
	} else {
		query = updateQ
	}
	c, cancel := context.WithDeadline(ts.ctx, time.Now().Add(5*time.Second))
	defer cancel()
	commandTag, err := ts.db.Exec(c, query, ih.Bytes())
	if err != nil {
		return err
	}
	if commandTag.RowsAffected() != 1 {
		return consts.ErrInvalidInfoHash
	}
	return nil
}

// Get returns a torrent for the hash provided
func (ts TorrentStore) Get(t *model.Torrent, ih model.InfoHash) error {
	const q = `
		SELECT 
			info_hash::bytea, release_name, total_uploaded, total_downloaded, total_completed, 
			is_deleted, is_enabled, reason, multi_up, multi_dn, announces
		FROM 
		    torrent 
		WHERE 
		    info_hash = $1 AND is_deleted = false`
	c, cancel := context.WithDeadline(ts.ctx, time.Now().Add(5*time.Second))
	defer cancel()
	var b []byte
	err := ts.db.QueryRow(c, q, ih.Bytes()).Scan(
		&b, // TODO implement pgx custom types to map automatically
		&t.ReleaseName,
		&t.TotalUploaded,
		&t.TotalDownloaded,
		&t.TotalCompleted,
		&t.IsDeleted,
		&t.IsEnabled,
		&t.Reason,
		&t.MultiUp,
		&t.MultiDn,
		&t.Announces,
	)
	copy(t.InfoHash[:], b)
	if err != nil {
		if err.Error() == "no rows in result set" {
			return consts.ErrInvalidInfoHash
		}
		return err
	}
	return nil
}

// Close will close the underlying postgres database connection
func (ts TorrentStore) Close() error {
	c, cancel := context.WithDeadline(ts.ctx, time.Now().Add(15*time.Second))
	defer cancel()
	return ts.db.Close(c)
}

// WhiteListDelete removes a client from the global whitelist
func (ts TorrentStore) WhiteListDelete(client model.WhiteListClient) error {
	const q = `DELETE FROM whitelist WHERE client_prefix = $1`
	c, cancel := context.WithDeadline(ts.ctx, time.Now().Add(5*time.Second))
	defer cancel()
	commandTag, err := ts.db.Exec(c, q, client.ClientPrefix)
	if err != nil {
		return errors.Wrap(err, "Failed to delete client whitelist")
	}
	if commandTag.RowsAffected() != 1 {
		return errors.New("insert ok, but no row modified")
	}
	return nil
}

// WhiteListAdd will insert a new client prefix into the allowed clients list
func (ts TorrentStore) WhiteListAdd(client model.WhiteListClient) error {
	const q = `INSERT INTO whitelist (client_prefix, client_name) VALUES ($1, $2)`
	c, cancel := context.WithDeadline(ts.ctx, time.Now().Add(5*time.Second))
	defer cancel()
	commandTag, err := ts.db.Exec(c, q, client.ClientPrefix, client.ClientName)
	if err != nil {
		return errors.Wrap(err, "Failed to insert new whitelist entry")
	}
	if commandTag.RowsAffected() != 1 {
		return errors.New("Failed to insert, but no error?")
	}
	return nil
}

// WhiteListGetAll fetches all known whitelisted clients
func (ts TorrentStore) WhiteListGetAll() ([]model.WhiteListClient, error) {
	var wl []model.WhiteListClient
	const q = `SELECT client_prefix, client_name FROM whitelist`
	c, cancel := context.WithDeadline(ts.ctx, time.Now().Add(5*time.Second))
	defer cancel()
	rows, err := ts.db.Query(c, q)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to select client whitelists")
	}
	defer rows.Close()
	for rows.Next() {
		var client model.WhiteListClient
		err = rows.Scan(&client.ClientPrefix, &client.ClientName)
		if err != nil {
			return nil, errors.Wrap(err, "Failed to fetch client whitelist")
		}
		wl = append(wl, client)
	}
	return wl, nil
}

// PeerStore is the postgres backed implementation of store.PeerStore
type PeerStore struct {
	db  *pgx.Conn
	ctx context.Context
}

// Sync batch updates the backing store with the new PeerStats provided
func (ps PeerStore) Sync(batch map[model.PeerHash]model.PeerStats) error {
	const txName = "peerSync"
	const q = `
		UPDATE 
			peers
		SET
			downloaded = (downloaded + $1),
		    uploaded = (uploaded + $2),
		    announces = (announces + $3),
		    announce_last = $4
		WHERE
			peer_id = $5 AND info_hash = $6
`
	c, cancel := context.WithDeadline(ps.ctx, time.Now().Add(time.Second*10))
	defer cancel()
	tx, err := ps.db.Begin(c)
	if err != nil {
		return errors.Wrap(err, "postgres.PeerStore.Sync Failed to being transaction")
	}
	defer func() { _ = tx.Rollback(c) }()
	_, err = tx.Prepare(c, txName, q)
	if err != nil {
		return errors.Wrap(err, "postgres.PeerStore.Sync Failed to being transaction")
	}

	for peerHash, stats := range batch {
		if _, err := tx.Exec(c, txName, stats.Downloaded, stats.Uploaded, stats.Announces, stats.LastAnnounce,
			peerHash.PeerID().Bytes(), peerHash.InfoHash().Bytes()); err != nil {
			return errors.Wrapf(err, "postgres.PeerStore.Sync failed to Exec tx")
		}
	}
	if err := tx.Commit(c); err != nil {
		return errors.Wrapf(err, "postgres.PeerStore.Sync failed to commit tx")
	}
	return nil
}

// Reap will loop through the peers removing any stale entries from active swarms
func (ps PeerStore) Reap() {
	// NOW() - INTERVAL '15 minutes'
	const q = `DELETE FROM peers WHERE announce_last < $1`
	c, cancel := context.WithDeadline(ps.ctx, time.Now().Add(5*time.Second))
	defer cancel()
	rows, err := ps.db.Exec(c, q, time.Now().Add(-(15 * time.Minute)))
	if err != nil {
		log.Errorf("failed to reap peers: %s", err.Error())
		return
	}
	if rows.RowsAffected() > 0 {
		log.Debugf("Reaped %d peers", rows.RowsAffected())
	}
}

// Add insets the peer into the swarm of the torrent provided
func (ps PeerStore) Add(ih model.InfoHash, p model.Peer) error {
	const q = `
	INSERT INTO peers 
	    (peer_id, info_hash, addr_ip, addr_port, location, user_id, announce_first, announce_last)
	VALUES 
	    ($1, $2, $3, $4::int, ST_MakePoint($6, $5), $7, $8, $9)
	`
	c, cancel := context.WithDeadline(ps.ctx, time.Now().Add(5*time.Second))
	defer cancel()
	commandTag, err := ps.db.Exec(c, q,
		p.PeerID.Bytes(), ih.Bytes(), p.IP, p.Port, p.Location.Latitude, p.Location.Longitude, p.UserID,
		p.AnnounceFirst, p.AnnounceLast)
	if err != nil {
		return err
	}
	if commandTag.RowsAffected() != 1 {
		return errors.New("Invalid rows affected inserting peer")
	}
	return nil
}

// Delete will remove a peer from the swarm of the torrent provided
func (ps PeerStore) Delete(ih model.InfoHash, p model.PeerID) error {
	const q = `DELETE FROM peers WHERE info_hash = $1 AND peer_id = $2`
	c, cancel := context.WithDeadline(ps.ctx, time.Now().Add(5*time.Second))
	defer cancel()
	_, err := ps.db.Exec(c, q, ih.Bytes(), p.Bytes())
	return err
}

// GetN will fetch the torrents swarm member peers
func (ps PeerStore) GetN(ih model.InfoHash, limit int) (model.Swarm, error) {
	const q = `
		SELECT 
		    peer_id::bytea, info_hash::bytea, user_id, addr_ip, addr_port, downloaded, uploaded, 
			announces, speed_up, speed_dn, speed_up_max, speed_dn_max, ST_x(location), ST_y(location)
		FROM
		    peers 
		WHERE
		      info_hash = $1 
		LIMIT 
		    $2`
	var peers model.Swarm
	c, cancel := context.WithDeadline(ps.ctx, time.Now().Add(5*time.Second))
	defer cancel()
	rows, err := ps.db.Query(c, q, ih.Bytes(), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var p model.Peer
		err = rows.Scan(&p.PeerID, &p.InfoHash, &p.UserID, &p.IP, &p.Port, &p.Downloaded, &p.Uploaded,
			&p.Announces, &p.SpeedUP, &p.SpeedDN, &p.SpeedUPMax, &p.SpeedDNMax, &p.Location.Longitude, &p.Location.Latitude)
		if err != nil {
			return nil, errors.Wrap(err, "failed to fetch N peers from store")
		}
		peers = append(peers, p)
	}
	if rows.Err() != nil {
		return nil, errors.Wrap(err, "error in peer query")
	}
	return peers, nil
}

// Get will fetch the peer from the swarm if it exists
func (ps PeerStore) Get(p *model.Peer, ih model.InfoHash, peerID model.PeerID) error {
	const q = `
		SELECT 
		       peer_id, info_hash, user_id, addr_ip, addr_port, downloaded, uploaded, announces,
		       speed_up, speed_dn, speed_up_max, speed_dn_max, location
		FROM
		    peers 
		WHERE 
			info_hash = $1 AND peer_id = $2`
	c, cancel := context.WithDeadline(ps.ctx, time.Now().Add(5*time.Second))
	defer cancel()
	err := ps.db.QueryRow(c, q, ih, peerID).Scan(
		p.PeerID, p.InfoHash, p.UserID, p.IP, p.Port, p.Downloaded, p.Uploaded,
		p.Announces, p.SpeedUP, p.SpeedDN, p.SpeedUPMax, p.SpeedDNMax, p.Location)
	if err != nil {
		return errors.Wrap(err, "Unknown peer")
	}
	return nil
}

// Close will close the underlying database connection
func (ps PeerStore) Close() error {
	c, cancel := context.WithDeadline(ps.ctx, time.Now().Add(15*time.Second))
	defer cancel()
	return ps.db.Close(c)
}

type userDriver struct{}

// NewUserStore creates a new postgres backed user store.
func (ud userDriver) NewUserStore(cfg interface{}) (store.UserStore, error) {
	c, ok := cfg.(*config.StoreConfig)
	if !ok {
		return nil, consts.ErrInvalidConfig
	}
	db, err := pgx.Connect(context.Background(), c.DSN())
	if err != nil {
		return nil, errors.Wrap(err, "failed to connect to postgres user store")
	}
	return NewUserStore(db), nil
}

type peerDriver struct{}

// NewPeerStore returns a postgres backed store.PeerStore driver
func (pd peerDriver) NewPeerStore(cfg interface{}) (store.PeerStore, error) {
	c, ok := cfg.(*config.StoreConfig)
	if !ok {
		return nil, consts.ErrInvalidConfig
	}
	db, err := pgx.Connect(context.Background(), c.DSN())
	if err != nil {
		return nil, errors.Wrap(err, "failed to connect to postgres peer store")
	}
	return NewPeerStore(db), nil
}
func NewUserStore(db *pgx.Conn) *UserStore {
	return &UserStore{db: db, ctx: context.Background()}
}

func NewPeerStore(db *pgx.Conn) *PeerStore {
	return &PeerStore{db: db, ctx: context.Background()}
}
func NewTorrentStore(db *pgx.Conn) *TorrentStore {
	return &TorrentStore{db: db, ctx: context.Background()}
}

type torrentDriver struct{}

// NewTorrentStore initialize a TorrentStore implementation using the postgres backing store
func (td torrentDriver) NewTorrentStore(cfg interface{}) (store.TorrentStore, error) {
	c, ok := cfg.(*config.StoreConfig)
	if !ok {
		return nil, consts.ErrInvalidConfig
	}
	db, err := pgx.Connect(context.Background(), c.DSN())
	if err != nil {
		return nil, errors.Wrap(err, "failed to connect to postgres torrent store")
	}
	return NewTorrentStore(db), nil
}

func makeDSN(c *config.StoreConfig) string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s%s",
		c.Username, c.Password, c.Host, c.Port, c.Database, c.Properties)
}

func init() {
	store.AddUserDriver(driverName, userDriver{})
	store.AddPeerDriver(driverName, peerDriver{})
	store.AddTorrentDriver(driverName, torrentDriver{})
}
