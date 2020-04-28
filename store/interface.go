// Package store provides the underlying interfaces and glue for the backend storage drivers.
//
// We define 2 distinct interfaces to allow for flexibility in storage options.
// TorrentStore is meant as a persistent storage backend which is backed to permanent storage
// PeerStore is meant as a cache to store ephemeral peer/swarm data, it does not need to be backed
// by persistent storage, but the option is there if desired.
//
// NOTE defer calls should not be used anywhere in the store packages to reduce as much overhead as possible.
package store

import (
	log "github.com/sirupsen/logrus"
	"mika/consts"
	"mika/model"
	"sync"
)

var (
	userDriverMutex     = sync.RWMutex{}
	peerDriversMutex    = sync.RWMutex{}
	torrentDriversMutex = sync.RWMutex{}
	userDrivers         = make(map[string]UserDriver)
	peerDrivers         = make(map[string]PeerDriver)
	torrentDrivers      = make(map[string]TorrentDriver)
)

type TorrentDriver interface {
	NewTorrentStore(config interface{}) (TorrentStore, error)
}

type PeerDriver interface {
	NewPeerStore(config interface{}) (PeerStore, error)
}

type UserDriver interface {
	NewUserStore(config interface{}) (UserStore, error)
}

func AddPeerDriver(name string, driver PeerDriver) {
	peerDriversMutex.Lock()
	defer peerDriversMutex.Unlock()
	peerDrivers[name] = driver
	log.Debugf("Registered peer storage driver: %s", name)
}

func AddTorrentDriver(name string, driver TorrentDriver) {
	torrentDriversMutex.Lock()
	defer torrentDriversMutex.Unlock()
	torrentDrivers[name] = driver
	log.Debugf("Registered torrent storage driver: %s", name)
}

func AddUserDriver(name string, driver UserDriver) {
	userDriverMutex.Lock()
	defer userDriverMutex.Unlock()
	userDrivers[name] = driver
	log.Debugf("Registered user storage driver: %s", name)
}

// UserStore defines a interface used to retrieve user data from a backing store.
// These should be cached indefinitely, we treat any known user as allowed to connect.
// To disable a user they MUST be deleted from the active user cache
type UserStore interface {
	GetUserByPasskey(passkey string) (model.User, error)
	GetUserById(userId uint32) (model.User, error)
	DeleteUser(user model.User) error
	// Close should cleanup and clone the underlying storage driver
	Close() error
}

// Torrent store defines where we can store permanent torrent data
// The drivers should always persist the data to disk
type TorrentStore interface {
	// Add a new torrent to the backing store
	AddTorrent(t *model.Torrent) error
	DeleteTorrent(t *model.Torrent, dropRow bool) error
	GetTorrent(hash model.InfoHash) (*model.Torrent, error)
	Close() error
}

// PeerStore defines our interface for storing peer data
// This doesnt need to be persisted to disk, but it will help warm up times
// if its backed by something that can restore its in memory state, such as redis
type PeerStore interface {
	AddPeer(t *model.Torrent, p *model.Peer) error
	UpdatePeer(t *model.Torrent, p *model.Peer) error
	DeletePeer(t *model.Torrent, p *model.Peer) error
	GetPeers(t *model.Torrent, limit int) ([]*model.Peer, error)
	GetScrape(t *model.Torrent)
	Close() error
}

func NewTorrentStore(storeType string, config interface{}) (TorrentStore, error) {
	torrentDriversMutex.RLock()
	defer torrentDriversMutex.RUnlock()
	var driver TorrentDriver
	driver, found := torrentDrivers[storeType]
	if !found {
		return nil, consts.ErrInvalidDriver
	}
	return driver.NewTorrentStore(config)
}

func NewPeerStore(storeType string, config interface{}) (PeerStore, error) {
	peerDriversMutex.RLock()
	defer peerDriversMutex.RUnlock()
	var driver PeerDriver
	driver, found := peerDrivers[storeType]
	if !found {
		return nil, consts.ErrInvalidDriver
	}
	return driver.NewPeerStore(config)
}

func NewUserStore(storeType string, config interface{}) (UserStore, error) {
	userDriverMutex.RLock()
	defer userDriverMutex.RUnlock()
	var driver UserDriver
	driver, found := userDrivers[storeType]
	if !found {
		return nil, consts.ErrInvalidDriver
	}
	return driver.NewUserStore(config)
}
