package postgres

import (
	"github.com/jmoiron/sqlx"
	"github.com/leighmacdonald/mika/config"
	"github.com/leighmacdonald/mika/consts"
	"github.com/leighmacdonald/mika/model"
	"github.com/leighmacdonald/mika/store"
	"github.com/pkg/errors"
)

const (
	driverName = "postgres"
)

// UserStore is the postgres backed store.UserStore implementation
type UserStore struct {
	db *sqlx.DB
}

// Add will add a new user to the backing store
func (us UserStore) Add(user *model.User) error {
	if user.UserID > 0 {
		return errors.New("User already has a user_id")
	}
	const q = `
		INSERT INTO users 
		    (passkey, download_enabled, is_deleted) 
		VALUES
		    (?, ?, ?)
		RETURNING 
		    (user_id)`
	var userID int
	err := us.db.QueryRowx(q, user.Passkey, true, false).Scan(&userID)
	if err != nil {
		return errors.Wrap(err, "Failed to add user to store")
	}
	user.UserID = uint32(userID)
	return nil
}

// GetByPasskey will lookup and return the user via their passkey used as an identifier
// The errors returned for this method should be very generic and not reveal any info
// that could possibly help attackers gain any insight. All error cases MUST
// return ErrUnauthorized.
func (us UserStore) GetByPasskey(passkey string) (*model.User, error) {
	var user model.User
	const q = `SELECT * FROM users WHERE passkey = ?`
	if err := us.db.Get(&user, q, passkey); err != nil {
		return nil, errors.Wrap(err, "Failed to fetch user by passkey")
	}
	return &user, nil
}

// GetByID returns a user matching the userId
func (us UserStore) GetByID(userID uint32) (*model.User, error) {
	var user model.User
	const q = `SELECT * FROM users WHERE user_id = ?`
	if err := us.db.Get(&user, q, userID); err != nil {
		return nil, errors.Wrap(err, "Failed to fetch user by user_id")
	}
	return &user, nil
}

// Delete removes a user from the backing store
func (us UserStore) Delete(user *model.User) error {
	if user.UserID <= 0 {
		return errors.New("User doesnt have a user_id")
	}
	const q = `DELETE FROM users WHERE user_id = ?`
	if _, err := us.db.Exec(q, user.UserID); err != nil {
		return errors.Wrap(err, "Failed to delete user")
	}
	user.UserID = 0
	return nil
}

// Close will close the underlying database connection and clear the local caches
func (us UserStore) Close() error {
	return us.db.Close()
}

// TorrentStore implements the store.TorrentStore interface for postgres
type TorrentStore struct {
	db *sqlx.DB
}

// Conn returns the underlying database driver
func (ts TorrentStore) Conn() interface{} {
	return ts.db
}

// Add inserts a new torrent into the backing store
func (ts TorrentStore) Add(t *model.Torrent) error {
	const q = `INSERT INTO torrent (info_hash, release_name) VALUES(?, ?)`
	_, err := ts.db.Exec(q, t.InfoHash.Bytes(), t.ReleaseName)
	if err != nil {
		return err
	}
	return nil
}

// Delete will mark a torrent as deleted in the backing store.
// If dropRow is true, it will permanently remove the torrent from the store
func (ts TorrentStore) Delete(ih model.InfoHash, dropRow bool) error {
	if dropRow {
		const dropQ = `DELETE FROM torrent WHERE info_hash = ?`
		_, err := ts.db.Exec(dropQ, ih.Bytes())
		if err != nil {
			return err
		}
	} else {
		const updateQ = `UPDATE torrent SET is_deleted = 1 WHERE info_hash = ?`
		_, err := ts.db.NamedExec(updateQ, ih.Bytes())
		if err != nil {
			return err
		}
	}
	return nil
}

// Get returns a torrent for the hash provided
func (ts TorrentStore) Get(hash model.InfoHash) (*model.Torrent, error) {
	const q = `SELECT * FROM torrent WHERE info_hash = ? AND is_deleted = false`
	var t model.Torrent
	err := ts.db.Get(&t, q, hash.Bytes())
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			return nil, consts.ErrInvalidInfoHash
		}
		return nil, err
	}
	return &t, nil
}

// Close will close the underlying postgres database connection
func (ts TorrentStore) Close() error {
	return ts.db.Close()
}

// WhiteListDelete removes a client from the global whitelist
func (ts TorrentStore) WhiteListDelete(client model.WhiteListClient) error {
	const q = `DELETE FROM whitelist WHERE client_prefix = ?`
	if _, err := ts.db.Exec(q, client.ClientPrefix); err != nil {
		return errors.Wrap(err, "Failed to delete client whitelist")
	}
	return nil
}

// WhiteListAdd will insert a new client prefix into the allowed clients list
func (ts TorrentStore) WhiteListAdd(client model.WhiteListClient) error {
	const q = `INSERT INTO whitelist (client_prefix, client_name) VALUES (:client_prefix, :client_name)`
	if _, err := ts.db.NamedExec(q, client); err != nil {
		return errors.Wrap(err, "Failed to insert new whitelist entry")
	}
	return nil
}

// WhiteListGetAll fetches all known whitelisted clients
func (ts TorrentStore) WhiteListGetAll() ([]model.WhiteListClient, error) {
	var wl []model.WhiteListClient
	const q = `SELECT * FROM whitelist`
	if err := ts.db.Select(&wl, q); err != nil {
		return nil, errors.Wrap(err, "Failed to select client whitelists")
	}
	return wl, nil
}

// PeerStore is the postgres backed implementation of store.PeerStore
type PeerStore struct {
	db *sqlx.DB
}

// Add insets the peer into the swarm of the torrent provided
func (ps PeerStore) Add(ih model.InfoHash, p *model.Peer) error {
	const q = `
	INSERT INTO peers 
	    (peer_id, info_hash, addr_ip, addr_port, location, user_id, created_on, updated_on)
	VALUES 
	    (:peer_id, :info_hash, :addr_ip, :addr_port, :location, :user_id, now(), :updated_on)
	`
	_, err := ps.db.Exec(q, p.PeerID, ih, p.IP, p.Port, p.Location, p.UserID)
	if err != nil {
		return err
	}
	return nil
}

// Update will sync the new peer data with the backing store
func (ps PeerStore) Update(ih model.InfoHash, p *model.Peer) error {
	panic("implement me")
}

// Delete will remove a peer from the swarm of the torrent provided
func (ps PeerStore) Delete(ih model.InfoHash, p *model.Peer) error {
	const q = `DELETE FROM peers WHERE info_hash = ? AND peer_id = ?`
	_, err := ps.db.Exec(q, ih, p.PeerID)
	return err
}

// GetN will fetch the torrents swarm member peers
func (ps PeerStore) GetN(ih model.InfoHash, limit int) (model.Swarm, error) {
	const q = `SELECT * FROM peers WHERE info_hash = ? LIMIT ?`
	var peers []*model.Peer
	if err := ps.db.Select(&peers, q, ih, limit); err != nil {
		return nil, err
	}
	return peers, nil
}

// Get will fetch the peer from the swarm if it exists
func (ps PeerStore) Get(ih model.InfoHash, peerID model.PeerID) (*model.Peer, error) {
	const q = `SELECT * FROM peers WHERE info_hash = ? AND peer_id = ? LIMIT 1`
	var peer model.Peer
	if err := ps.db.Get(&peer, q, ih, peerID); err != nil {
		return nil, errors.Wrap(err, "Unknown peer")
	}
	return &peer, nil
}

// Close will close the underlying database connection
func (ps PeerStore) Close() error {
	return ps.db.Close()
}

type userDriver struct{}

// NewUserStore creates a new postgres backed user store.
func (ud userDriver) NewUserStore(cfg interface{}) (store.UserStore, error) {
	c, ok := cfg.(*config.StoreConfig)
	if !ok {
		return nil, consts.ErrInvalidConfig
	}
	db := sqlx.MustConnect(driverName, c.DSN())
	return &UserStore{
		db: db,
	}, nil
}

type peerDriver struct{}

// NewPeerStore returns a postgres backed store.PeerStore driver
func (pd peerDriver) NewPeerStore(cfg interface{}) (store.PeerStore, error) {
	c, ok := cfg.(*config.StoreConfig)
	if !ok {
		return nil, consts.ErrInvalidConfig
	}
	db := sqlx.MustConnect(driverName, c.DSN())
	return &PeerStore{
		db: db,
	}, nil
}

type torrentDriver struct{}

// NewTorrentStore initialize a TorrentStore implementation using the postgres backing store
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
	store.AddUserDriver(driverName, userDriver{})
	store.AddPeerDriver(driverName, peerDriver{})
	store.AddTorrentDriver(driverName, torrentDriver{})
}
