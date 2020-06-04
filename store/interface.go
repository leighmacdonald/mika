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
	"github.com/leighmacdonald/mika/consts"
	log "github.com/sirupsen/logrus"
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
	// New instantiates a new TorrentStore
	New(config interface{}) (TorrentStore, error)
}

// PeerDriver provides a interface to enable registration of PeerStore drivers
type PeerDriver interface {
	// New instantiates a new PeerStore
	New(config interface{}) (PeerStore, error)
}

// UserDriver provides a interface to enable registration of UserStore drivers
type UserDriver interface {
	// New instantiates a new UserStore
	New(config interface{}) (UserStore, error)
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
	// Add will add a new user to the backing store
	Add(u User) error
	// GetByPasskey returns a user matching the passkey
	GetByPasskey(user *User, passkey string) error
	// GetByID returns a user matching the userId
	GetByID(user *User, userID uint32) error
	// Delete removes a user from the backing store
	Delete(user User) error
	// Update is used to change a known user
	Update(user User, oldPasskey string) error
	// Close will cleanup and close the underlying storage driver if necessary
	Close() error
	// Sync batch updates the backing store with the new UserStats provided
	Sync(b map[string]UserStats) error
	// Name returns the name of the data store type
	Name() string
}

// TorrentStore defines where we can store permanent torrent data
// The backing drivers should always persist the data to disk
type TorrentStore interface {
	// Add adds a new torrent to the backing store
	Add(t Torrent) error
	// Delete will mark a torrent as deleted in the backing store.
	// If dropRow is true, it will permanently remove the torrent from the store
	Delete(ih InfoHash, dropRow bool) error
	// Get returns the Torrent matching the infohash
	Get(torrent *Torrent, hash InfoHash, deletedOk bool) error
	// Update will update certain parameters within the torrent
	Update(torrent Torrent) error
	// Close will cleanup and close the underlying storage driver if necessary
	Close() error
	// WhiteListDelete removes a client from the global whitelist
	WhiteListDelete(client WhiteListClient) error
	// WhiteListAdd will insert a new client prefix into the allowed clients list
	WhiteListAdd(client WhiteListClient) error
	// WhiteListGetAll fetches all known whitelisted clients
	WhiteListGetAll() ([]WhiteListClient, error)
	// Sync batch updates the backing store with the new TorrentStats provided
	Sync(b map[InfoHash]TorrentStats) error
	// Conn returns the underlying connection, if any
	Conn() interface{}
	// Name returns the name of the data store type
	Name() string
}

// PeerStore defines our interface for storing peer data
// This doesnt need to be persisted to disk, but it will help warm up times
// if its backed by something that can restore its in memory state, such as redis
type PeerStore interface {
	// Add inserts a peer into the active swarm for the torrent provided
	Add(ih InfoHash, p Peer) error
	// Delete will remove a user from a torrents swarm
	Delete(ih InfoHash, p PeerID) error
	// GetN will fetch peers for a torrents active swarm up to N users
	GetN(ih InfoHash, limit int) (Swarm, error)
	// Get will fetch the peer from the swarm if it exists
	Get(peer *Peer, ih InfoHash, id PeerID) error
	// Close will cleanup and close the underlying storage driver if necessary
	Close() error
	// Reap will loop through the peers removing any stale entries from active swarms
	Reap() []PeerHash
	// Sync batch updates the backing store with the new PeerStats provided
	Sync(b map[PeerHash]PeerStats) error
	// Name returns the name of the data store type
	Name() string
}

// NewTorrentStore will attempt to initialize a TorrentStore using the driver name provided
func NewTorrentStore(storeType string, config interface{}) (TorrentStore, error) {
	torrentDriversMutex.RLock()
	defer torrentDriversMutex.RUnlock()
	driver, found := torrentDrivers[storeType]
	if !found {
		return nil, consts.ErrInvalidDriver
	}
	return driver.New(config)
}

// NewPeerStore will attempt to initialize a PeerStore using the driver name provided
func NewPeerStore(storeType string, config interface{}) (PeerStore, error) {
	peerDriversMutex.RLock()
	defer peerDriversMutex.RUnlock()
	driver, found := peerDrivers[storeType]
	if !found {
		return nil, consts.ErrInvalidDriver
	}
	return driver.New(config)
}

// NewUserStore will attempt to initialize a UserStore using the driver name provided
func NewUserStore(storeType string, config interface{}) (UserStore, error) {
	userDriverMutex.RLock()
	defer userDriverMutex.RUnlock()
	driver, found := userDrivers[storeType]
	if !found {
		return nil, consts.ErrInvalidDriver
	}
	return driver.New(config)
}
