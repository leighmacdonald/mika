// Package store provides the underlying interfaces and glue for the backend storage drivers.
//
// We define 3 distinct interfaces to allow for flexibility in storage options.
// Store is meant as a persistent storage backend for user data which is backed to permanent storage
// StoreI is meant as a persistent storage backend for torrent data which is backed to permanent storage
// PeerStore is meant as a cache to store ephemeral peer/swarm data, it does not need to be backed
// by persistent storage, but the option is there if desired.
//
// NOTE defer calls should not be used anywhere in the store packages to reduce as much overhead as possible.
package store

import (
	"github.com/leighmacdonald/mika/config"
	"github.com/leighmacdonald/mika/consts"
	log "github.com/sirupsen/logrus"
	"sync"
)

var (
	driverMutex = sync.RWMutex{}
	drivers     = make(map[string]Driver)
)

// Driver provides a interface to enable registration of StoreI drivers
type Driver interface {
	// New instantiates a new StoreI
	New(config config.StoreConfig) (Store, error)
}

// AddDriver will register a new driver able to instantiate a StoreI
func AddDriver(name string, driver Driver) {
	driverMutex.Lock()
	defer driverMutex.Unlock()
	drivers[name] = driver
	log.Debugf("Registered torrent storage driver: %s", name)
}

// Store defines a interface used to retrieve user data from a backing store.
// These should be cached indefinitely, we treat any known user as allowed to connect.
// To disable a user they MUST be deleted from the active user cache
type Store interface {
	Users() (Users, error)
	// UserAdd will add a new user to the backing store
	UserAdd(u *User) error
	// UserGetByPasskey returns a user matching the passkey
	UserGetByPasskey(passkey string) (*User, error)
	// UserGetByID returns a user matching the userId
	UserGetByID(userID uint32) (*User, error)
	// UserDelete removes a user from the backing store
	UserDelete(user *User) error
	// UserSave is used to change a known user
	UserSave(user *User) error
	// Close will cleanup and close the underlying storage driver if necessary
	UserSync(b []*User) error

	// Roles fetches all known groups
	Roles() (Roles, error)
	// Roles fetches all known groups
	RoleByID(roleID uint32) (*Role, error)
	// RoleAdd adds a new role to the system
	RoleAdd(role *Role) error
	// RoleDelete permanently deletes a role from the system
	RoleDelete(roleID uint32) error
	// RoleSave commits the role to persistent store
	RoleSave(role *Role) error

	// Torrents returns all torrents in the store
	Torrents() (Torrents, error)
	// TorrentAdd adds a new torrent to the backing store
	TorrentAdd(t *Torrent) error
	// TorrentDelete will mark a torrent as deleted in the backing store.
	// If dropRow is true, it will permanently remove the torrent from the store
	TorrentDelete(ih InfoHash, dropRow bool) error
	// TorrentGet returns the Torrent matching the infohash
	TorrentGet(hash InfoHash, deletedOk bool) (*Torrent, error)
	// TorrentSave will update certain parameters within the torrent
	TorrentSave(torrent *Torrent) error
	// TorrentSync batch updates the backing store with the new TorrentStats provided
	TorrentSync(b []*Torrent) error

	// WhiteListDelete removes a client from the global whitelist
	WhiteListDelete(client *WhiteListClient) error
	// WhiteListAdd will insert a new client prefix into the allowed clients list
	WhiteListAdd(client *WhiteListClient) error
	// WhiteListGetAll fetches all known whitelisted clients
	WhiteListGetAll() ([]*WhiteListClient, error)

	// Migrate
	Migrate() error

	// Conn returns the underlying connection, if any
	Conn() interface{}
	// Name returns the name of the data store type
	Name() string
	// Close will cleanup and close the underlying storage driver if necessary
	Close() error
}

// NewStore will attempt to initialize a StoreI using the driver name provided
func NewStore(config config.StoreConfig) (Store, error) {
	driverMutex.RLock()
	defer driverMutex.RUnlock()
	driver, found := drivers[config.Type]
	if !found {
		return nil, consts.ErrInvalidDriver
	}
	return driver.New(config)
}
