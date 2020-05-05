package postgres

import (
	"github.com/jmoiron/sqlx"
	"github.com/leighmacdonald/mika/config"
	"github.com/leighmacdonald/mika/consts"
	"github.com/leighmacdonald/mika/model"
	"github.com/leighmacdonald/mika/store"
)

const (
	driverName = "postgres"
)

// UserStore is the postgres backed store.UserStore implementation
type UserStore struct {
	db *sqlx.DB
}

// Add will add a new user to the backing store
func (us UserStore) Add(u *model.User) error {
	panic("implement me")
}

// GetByPasskey will lookup and return the user via their passkey used as an identifier
// The errors returned for this method should be very generic and not reveal any info
// that could possibly help attackers gain any insight. All error cases MUST
// return ErrUnauthorized.
func (us UserStore) GetByPasskey(passkey string) (*model.User, error) {
	panic("implement me")
}

// GetByID returns a user matching the userId
func (us UserStore) GetByID(userID uint32) (*model.User, error) {
	panic("implement me")
}

// Delete removes a user from the backing store
func (us UserStore) Delete(user *model.User) error {
	panic("implement me")
}

// Close will close the underlying database connection and clear the local caches
func (us UserStore) Close() error {
	panic("implement me")
}

// TorrentStore implements the store.TorrentStore interface for postgres
type TorrentStore struct {
	db *sqlx.DB
}

// Add inserts a new torrent into the backing store
func (ts TorrentStore) Add(t *model.Torrent) error {
	panic("implement me")
}

// Delete will mark a torrent as deleted in the backing store.
// If dropRow is true, it will permanently remove the torrent from the store
func (ts TorrentStore) Delete(ih model.InfoHash, dropRow bool) error {
	panic("implement me")
}

// Get returns a torrent for the hash provided
func (ts TorrentStore) Get(hash model.InfoHash) (*model.Torrent, error) {
	panic("implement me")
}

// Close will close the underlying postgres database connection
func (ts TorrentStore) Close() error {
	panic("implement me")
}

// WhiteListDelete removes a client from the global whitelist
func (ts TorrentStore) WhiteListDelete(client model.WhiteListClient) error {
	panic("implement me")
}

// WhiteListAdd will insert a new client prefix into the allowed clients list
func (ts TorrentStore) WhiteListAdd(client model.WhiteListClient) error {
	panic("implement me")
}

// WhiteListGetAll fetches all known whitelisted clients
func (ts TorrentStore) WhiteListGetAll() ([]model.WhiteListClient, error) {
	panic("implement me")
}

// PeerStore is the postgres backed implementation of store.PeerStore
type PeerStore struct {
	db *sqlx.DB
}

// Add insets the peer into the swarm of the torrent provided
func (ps PeerStore) Add(ih model.InfoHash, p *model.Peer) error {
	panic("implement me")
}

// Update will sync the new peer data with the backing store
func (ps PeerStore) Update(ih model.InfoHash, p *model.Peer) error {
	panic("implement me")
}

// Delete will remove a peer from the swarm of the torrent provided
func (ps PeerStore) Delete(ih model.InfoHash, p *model.Peer) error {
	panic("implement me")
}

// GetN will fetch the torrents swarm member peers
func (ps PeerStore) GetN(ih model.InfoHash, limit int) (model.Swarm, error) {
	panic("implement me")
}

// Get will fetch the peer from the swarm if it exists
func (ps PeerStore) Get(ih model.InfoHash, id model.PeerID) (*model.Peer, error) {
	panic("implement me")
}

// Close will close the underlying database connection
func (ps PeerStore) Close() error {
	panic("implement me")
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
