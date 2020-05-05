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

type UserStore struct {
	db *sqlx.DB
}

func (us UserStore) Add(u *model.User) error {
	panic("implement me")
}

func (us UserStore) GetByPasskey(passkey string) (*model.User, error) {
	panic("implement me")
}

func (us UserStore) GetByID(userID uint32) (*model.User, error) {
	panic("implement me")
}

func (us UserStore) Delete(user *model.User) error {
	panic("implement me")
}

func (us UserStore) Close() error {
	panic("implement me")
}

// TorrentStore implements the store.TorrentStore interface for postgres
type TorrentStore struct {
	db *sqlx.DB
}

func (ts TorrentStore) Add(t *model.Torrent) error {
	panic("implement me")
}

func (ts TorrentStore) Delete(ih model.InfoHash, dropRow bool) error {
	panic("implement me")
}

func (ts TorrentStore) Get(hash model.InfoHash) (*model.Torrent, error) {
	panic("implement me")
}

func (ts TorrentStore) Close() error {
	panic("implement me")
}

func (ts TorrentStore) WhiteListDelete(client model.WhiteListClient) error {
	panic("implement me")
}

func (ts TorrentStore) WhiteListAdd(client model.WhiteListClient) error {
	panic("implement me")
}

func (ts TorrentStore) WhiteListGetAll() ([]model.WhiteListClient, error) {
	panic("implement me")
}

// PeerStore is the postgres backed implementation of store.PeerStore
type PeerStore struct {
	db *sqlx.DB
}

func (ps PeerStore) Add(ih model.InfoHash, p *model.Peer) error {
	panic("implement me")
}

func (ps PeerStore) Update(ih model.InfoHash, p *model.Peer) error {
	panic("implement me")
}

func (ps PeerStore) Delete(ih model.InfoHash, p *model.Peer) error {
	panic("implement me")
}

func (ps PeerStore) GetN(ih model.InfoHash, limit int) (model.Swarm, error) {
	panic("implement me")
}

func (ps PeerStore) Get(ih model.InfoHash, id model.PeerID) (*model.Peer, error) {
	panic("implement me")
}

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
