package memory

import (
	"github.com/leighmacdonald/mika/config"
	"github.com/leighmacdonald/mika/consts"
	"github.com/leighmacdonald/mika/store"
	"github.com/pkg/errors"
	"strings"
	"sync"
)

const (
	driverName = "memory"
)

func (d *Driver) Name() string {
	return driverName
}

func (d *Driver) TorrentUpdate(_ *store.Torrent) error {
	return nil
}

// Add adds a new torrent to the memory store
func (d *Driver) TorrentAdd(t *store.Torrent) error {
	d.torrentsMu.Lock()
	defer d.torrentsMu.Unlock()
	_, found := d.torrents[t.InfoHash]
	if found {
		return consts.ErrDuplicate
	}
	d.torrents[t.InfoHash] = t
	return nil
}

// Delete will mark a torrent as deleted in the backing store.
// NOTE the memory store always permanently deletes the torrent
func (d *Driver) TorrentDelete(ih store.InfoHash, _ bool) error {
	d.torrentsMu.Lock()
	delete(d.torrents, ih)
	d.torrentsMu.Unlock()
	return nil
}

// Sync batch updates the backing store with the new TorrentStats provided
func (d *Driver) TorrentSync(b map[store.InfoHash]store.TorrentStats) error {
	d.torrentsMu.Lock()
	defer d.torrentsMu.Unlock()
	for ih, stats := range b {
		t, found := d.torrents[ih]
		if !found {
			// Deleted torrent before sync occurred
			continue
		}
		t.Uploaded += stats.Uploaded
		t.Downloaded += stats.Downloaded
		t.Snatches += stats.Snatches
		t.Seeders += stats.Seeders
		t.Leechers += stats.Leechers
		t.Announces += stats.Announces
		d.torrents[ih] = t
	}
	return nil
}

// Conn always returns nil for in-memory store
func (d *Driver) Conn() interface{} {
	return nil
}

// WhiteListDelete removes a client from the global whitelist
func (d *Driver) WhiteListDelete(client store.WhiteListClient) error {
	d.whitelistMu.Lock()
	defer d.whitelistMu.Unlock()
	// Remove removes a peer from a slice
	for i := len(d.whitelist) - 1; i >= 0; i-- {
		if d.whitelist[i].ClientPrefix == client.ClientPrefix {
			d.whitelist = append(d.whitelist[:i], d.whitelist[i+1:]...)
			return nil
		}
	}
	return consts.ErrInvalidClient
}

// WhiteListAdd will insert a new client prefix into the allowed clients list
func (d *Driver) WhiteListAdd(client store.WhiteListClient) error {
	d.whitelistMu.Lock()
	defer d.whitelistMu.Unlock()
	d.whitelist = append(d.whitelist, client)
	return nil
}

// WhiteListGetAll fetches all known whitelisted clients
func (d *Driver) WhiteListGetAll() ([]store.WhiteListClient, error) {
	d.whitelistMu.RLock()
	defer d.whitelistMu.RUnlock()
	wl := d.whitelist
	return wl, nil
}

// Get returns the Torrent matching the infohash
func (d *Driver) TorrentGet(torrent *store.Torrent, hash store.InfoHash, deletedOk bool) error {
	d.torrentsMu.RLock()
	t, found := d.torrents[hash]
	d.torrentsMu.RUnlock()
	if !found {
		return consts.ErrInvalidInfoHash
	}
	if t.IsDeleted && !deletedOk {
		return consts.ErrInvalidInfoHash
	}
	torrent = t
	return nil
}

// NewPeerStore instantiates a new in-memory peer store
func NewDriver() *Driver {
	return &Driver{
		swarms:     make(map[store.InfoHash]store.Swarm),
		users:      make(map[string]*store.User),
		roles:      nil,
		torrents:   make(map[store.InfoHash]*store.Torrent),
		rolesMu:    &sync.RWMutex{},
		torrentsMu: &sync.RWMutex{},
		usersMu:    &sync.RWMutex{},
		whitelist:  nil,
	}
}

// Driver is the memory backed store.Store implementation
type Driver struct {
	swarms      map[store.InfoHash]store.Swarm
	users       map[string]*store.User
	roles       []*store.Role
	torrents    map[store.InfoHash]*store.Torrent
	whitelist   []store.WhiteListClient
	rolesMu     *sync.RWMutex
	torrentsMu  *sync.RWMutex
	usersMu     *sync.RWMutex
	whitelistMu *sync.RWMutex
}

func (d *Driver) RoleByID(role *store.Role, roleID uint32) error {
	for _, r := range d.roles {
		if r.RoleID == roleID {
			role = r
			return nil
		}
	}
	return consts.ErrInvalidRole
}

func (d *Driver) RoleAdd(role *store.Role) error {
	d.rolesMu.Lock()
	defer d.rolesMu.Unlock()
	maxID := uint32(0)
	for _, r := range d.roles {
		if r.RoleID > maxID {
			maxID = r.RoleID
		}
	}
	for _, r := range d.roles {
		if strings.ToLower(r.RoleName) == strings.ToLower(role.RoleName) {
			return errors.Errorf("duplicate role_name: %s", role.RoleName)
		}
		if r.RoleID == role.RoleID {
			return errors.Errorf("duplicate role_Id: %d", r.RoleID)
		}
	}
	role.RoleID = maxID + 1
	d.roles = append(d.roles, role)
	return nil
}

func (d *Driver) RoleDelete(roleID uint32) error {
	d.rolesMu.Lock()
	defer d.rolesMu.Unlock()
	conflicts := 0
	for _, u := range d.users {
		if u.RoleID == roleID {
			conflicts++
		}
	}
	if conflicts > 0 {
		return errors.Errorf("Found %d users with only a single role, cannot remove only role", conflicts)
	}
	found := false
	for i := len(d.roles) - 1; i >= 0; i-- {
		if d.roles[i].RoleID == roleID {
			d.roles = append(d.roles[:i], d.roles[i+1:]...)
			found = true
			break
		}
	}
	if !found {
		return errors.New("Unknown role_id")
	}
	return nil
}

func (d *Driver) Roles() (store.Roles, error) {
	return d.roles, nil
}

// Update is used to change a known user
func (d *Driver) UserUpdate(_ *store.User, _ string) error {
	return nil
}

// Sync batch updates the backing store with the new UserStats provided
func (d *Driver) UserSync(b map[string]store.UserStats) error {
	d.usersMu.Lock()
	defer d.usersMu.Unlock()
	for passkey, stats := range b {
		user, found := d.users[passkey]
		if !found {
			// Deleted user
			continue
		}
		user.Announces += stats.Announces
		user.Downloaded += stats.Downloaded
		user.Uploaded += stats.Uploaded
		d.users[passkey] = user
	}
	return nil
}

// Add will add a new user to the backing store
func (d *Driver) UserAdd(usr *store.User) error {
	d.usersMu.Lock()
	defer d.usersMu.Unlock()
	for _, existing := range d.users {
		if existing.UserID == usr.UserID {
			return consts.ErrDuplicate
		}
	}
	d.users[usr.Passkey] = usr
	return nil
}

// GetByPasskey will lookup and return the user via their passkey used as an identifier
// The errors returned for this method should be very generic and not reveal any info
// that could possibly help attackers gain any insight. All error cases MUST
// return ErrUnauthorized.
func (d *Driver) UserGetByPasskey(usr *store.User, passkey string) error {
	d.usersMu.RLock()
	user, found := d.users[passkey]
	d.usersMu.RUnlock()
	if !found {
		return consts.ErrUnauthorized
	}
	usr = user
	return nil
}

// GetByID returns a user matching the userId
func (d *Driver) UserGetByID(user *store.User, userID uint32) error {
	d.usersMu.RLock()
	defer d.usersMu.RUnlock()
	for _, usr := range d.users {
		if usr.UserID == userID {
			user = usr
			return nil
		}
	}
	return consts.ErrUnauthorized
}

// Delete removes a user from the backing store
func (d *Driver) UserDelete(user *store.User) error {
	d.usersMu.Lock()
	delete(d.users, user.Passkey)
	d.usersMu.Unlock()
	return nil
}

// Close will delete/free the underlying memory store
func (d *Driver) Close() error {
	d.usersMu.Lock()
	d.users = make(map[string]*store.User)
	d.usersMu.Unlock()
	d.torrentsMu.Lock()
	d.torrents = make(map[store.InfoHash]*store.Torrent)
	d.torrentsMu.Unlock()
	return nil
}

type initializer struct{}

// New creates a new memory backed user store.
func (d initializer) New(_ config.StoreConfig) (store.Store, error) {
	return NewDriver(), nil
}

func init() {
	store.AddDriver(driverName, initializer{})
}
