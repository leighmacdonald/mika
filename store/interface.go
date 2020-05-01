// Package store provides the underlying interfaces and glue for the backend storage drivers.
//
// We define 3 distinct interfaces to allow for flexibility in storage options.
// UserStore is meant as a persistent storage backend for user data which is backed to permanent storage
// TorrentStore is meant as a persistent storage backend for torrent data which is backed to permanent storage
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

// TorrentDriver provides a interface to enable registration of TorrentStore drivers
type TorrentDriver interface {
	// NewUserStore instantiates a new TorrentStore
	NewTorrentStore(config interface{}) (TorrentStore, error)
}

// PeerDriver provides a interface to enable registration of PeerStore drivers
type PeerDriver interface {
	// NewUserStore instantiates a new PeerStore
	NewPeerStore(config interface{}) (PeerStore, error)
}

// UserDriver provides a interface to enable registration of UserStore drivers
type UserDriver interface {
	// NewUserStore instantiates a new UserStore
	NewUserStore(config interface{}) (UserStore, error)
}

// AddPeerDriver will register a new driver able to instantiate a PeerStore
func AddPeerDriver(name string, driver PeerDriver) {
	peerDriversMutex.Lock()
	defer peerDriversMutex.Unlock()
	peerDrivers[name] = driver
	log.Debugf("Registered peer storage driver: %s", name)
}

// AddTorrentDriver will register a new driver able to instantiate a TorrentStore
func AddTorrentDriver(name string, driver TorrentDriver) {
	torrentDriversMutex.Lock()
	defer torrentDriversMutex.Unlock()
	torrentDrivers[name] = driver
	log.Debugf("Registered torrent storage driver: %s", name)
}

// AddUserDriver will register a new driver able to instantiate a UserStore
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
	// AddUser will add a new user to the backing store
	AddUser(u *model.User) error
	// GetUserByPasskey returns a user matching the passkey
	GetUserByPasskey(passkey string) (*model.User, error)
	// GetUserByID returns a user matching the userId
	GetUserByID(userID uint32) (*model.User, error)
	// DeleteUser removes a user from the backing store
	DeleteUser(user *model.User) error
	// Close will cleanup and close the underlying storage driver if necessary
	Close() error
}

// TorrentStore defines where we can store permanent torrent data
// The backing drivers should always persist the data to disk
type TorrentStore interface {
	// AddTorrent adds a new torrent to the backing store
	AddTorrent(t *model.Torrent) error
	// DeleteTorrent will mark a torrent as deleted in the backing store.
	// If dropRow is true, it will permanently remove the torrent from the store
	DeleteTorrent(t *model.Torrent, dropRow bool) error
	// GetTorrent returns the Torrent matching the infohash
	GetTorrent(hash model.InfoHash) (*model.Torrent, error)
	// Close will cleanup and close the underlying storage driver if necessary
	Close() error
}

// PeerStore defines our interface for storing peer data
// This doesnt need to be persisted to disk, but it will help warm up times
// if its backed by something that can restore its in memory state, such as redis
type PeerStore interface {
	// AddPeer inserts a peer into the active swarm for the torrent provided
	AddPeer(ih model.InfoHash, p *model.Peer) error
	// UpdatePeer will sync any new peer data with the backing store
	UpdatePeer(ih model.InfoHash, p *model.Peer) error
	// DeletePeer will remove a user from a torrents swarm
	DeletePeer(ih model.InfoHash, p *model.Peer) error
	// GetPeers will fetch peers for a torrents active swarm up to N users
	GetPeers(ih model.InfoHash, limit int) (model.Swarm, error)
	// GetPeer will fetch the peer from the swarm if it exists
	GetPeer(ih model.InfoHash, id model.PeerID) (*model.Peer, error)
	// GetScrape returns scrape data for the torrent provided
	GetScrape(ih model.InfoHash)
	// Close will cleanup and close the underlying storage driver if necessary
	Close() error
}

// NewTorrentStore will attempt to initialize a TorrentStore using the driver name provided
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

// NewPeerStore will attempt to initialize a PeerStore using the driver name provided
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

// NewUserStore will attempt to initialize a UserStore using the driver name provided
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
