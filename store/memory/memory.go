package memory

import (
	"github.com/leighmacdonald/mika/consts"
	"github.com/leighmacdonald/mika/model"
	"github.com/leighmacdonald/mika/store"
	"sync"
)

const (
	driverName = "memory"
)

// TorrentStore is the memory backed store.TorrentStore implementation
type TorrentStore struct {
	sync.RWMutex
	torrents  map[model.InfoHash]*model.Torrent
	whitelist []model.WhiteListClient
}

// WhiteListDelete removes a client from the global whitelist
func (ts *TorrentStore) WhiteListDelete(client model.WhiteListClient) error {
	ts.Lock()
	// Remove removes a peer from a slice
	for i := len(ts.whitelist) - 1; i >= 0; i-- {
		if ts.whitelist[i].ClientPrefix == client.ClientPrefix {
			ts.whitelist = append(ts.whitelist[:i], ts.whitelist[i+1:]...)
			return nil
		}
	}
	return consts.ErrInvalidClient
}

// WhiteListAdd will insert a new client prefix into the allowed clients list
func (ts *TorrentStore) WhiteListAdd(client model.WhiteListClient) error {
	ts.Lock()
	ts.whitelist = append(ts.whitelist, client)
	ts.Unlock()
	return nil
}

// WhiteListGetAll fetches all known whitelisted clients
func (ts *TorrentStore) WhiteListGetAll() ([]model.WhiteListClient, error) {
	ts.RLock()
	wl := ts.whitelist
	ts.RUnlock()
	return wl, nil
}

// Close will delete/free all the underlying torrent data
func (ts *TorrentStore) Close() error {
	ts.Lock()
	ts.torrents = map[model.InfoHash]*model.Torrent{}
	ts.Unlock()
	return nil
}

// Get returns the Torrent matching the infohash
func (ts *TorrentStore) Get(hash model.InfoHash) (*model.Torrent, error) {
	ts.RLock()
	t, found := ts.torrents[hash]
	ts.RUnlock()
	if !found || t.IsDeleted {
		return nil, consts.ErrInvalidInfoHash
	}
	return t, nil
}

// PeerStore is a memory backed store.PeerStore implementation
// TODO shard peer storage
type PeerStore struct {
	sync.RWMutex
	peers map[model.InfoHash]model.Swarm
}

// Get will fetch the peer from the swarm if it exists
func (ps *PeerStore) Get(ih model.InfoHash, p model.PeerID) (*model.Peer, error) {
	ps.RLock()
	defer ps.RUnlock()
	for _, peer := range ps.peers[ih] {
		if peer.PeerID == p {
			return peer, nil
		}
	}
	return nil, consts.ErrInvalidPeerID
}

// Close flushes allocated memory
// TODO flush mem
func (ps *PeerStore) Close() error {
	ps.Lock()
	ps.peers = make(map[model.InfoHash]model.Swarm)
	ps.Unlock()
	return nil
}

// Add inserts a peer into the active swarm for the torrent provided
func (ps *PeerStore) Add(ih model.InfoHash, p *model.Peer) error {
	ps.Lock()
	ps.peers[ih] = append(ps.peers[ih], p)
	ps.Unlock()
	return nil
}

// Update is a no-op for memory backed store
func (ps *PeerStore) Update(_ model.InfoHash, _ *model.Peer) error {
	// no-op for in-memory store
	return nil
}

// Delete will remove a user from a torrents swarm
func (ps *PeerStore) Delete(ih model.InfoHash, p *model.Peer) error {
	ps.Lock()
	ps.peers[ih].Remove(p)
	ps.Unlock()
	return nil
}

// GetN will fetch peers for a torrents active swarm up to N users
func (ps *PeerStore) GetN(ih model.InfoHash, limit int) (model.Swarm, error) {
	ps.RLock()
	p, found := ps.peers[ih]
	ps.RUnlock()
	if !found {
		return nil, consts.ErrInvalidTorrentID
	}
	return p[0:limit], nil
}

// Add adds a new torrent to the memory store
func (ts *TorrentStore) Add(t *model.Torrent) error {
	ts.RLock()
	_, found := ts.torrents[t.InfoHash]
	ts.RUnlock()
	if found {
		return consts.ErrDuplicate
	}
	ts.Lock()
	ts.torrents[t.InfoHash] = t
	ts.Unlock()
	return nil
}

// Delete will mark a torrent as deleted in the backing store.
// NOTE the memory store always permanently deletes the torrent
func (ts *TorrentStore) Delete(ih model.InfoHash, _ bool) error {
	ts.Lock()
	delete(ts.torrents, ih)
	ts.Unlock()
	return nil
}

type torrentDriver struct{}

// NewTorrentStore initialize a TorrentStore implementation using the memory backing store
func (td torrentDriver) NewTorrentStore(_ interface{}) (store.TorrentStore, error) {
	return &TorrentStore{
		sync.RWMutex{},
		make(map[model.InfoHash]*model.Torrent),
		[]model.WhiteListClient{},
	}, nil
}

type peerDriver struct{}

// NewPeerStore initialize a NewPeerStore implementation using the memory backing store
func (pd peerDriver) NewPeerStore(_ interface{}) (store.PeerStore, error) {
	return &PeerStore{
		sync.RWMutex{},
		make(map[model.InfoHash]model.Swarm),
	}, nil
}

// UserStore is the memory backed store.UserStore implementation
type UserStore struct {
	sync.RWMutex
	users map[string]*model.User
}

// Add will add a new user to the backing store
func (u *UserStore) Add(usr *model.User) error {
	u.Lock()
	u.users[usr.Passkey] = usr
	u.Unlock()
	return nil
}

// GetByPasskey will lookup and return the user via their passkey used as an identifier
// The errors returned for this method should be very generic and not reveal any info
// that could possibly help attackers gain any insight. All error cases MUST
// return ErrUnauthorized.
func (u *UserStore) GetByPasskey(passkey string) (*model.User, error) {
	u.RLock()
	user, found := u.users[passkey]
	u.RUnlock()
	if !found {
		return nil, consts.ErrUnauthorized
	}
	return user, nil
}

// GetByID returns a user matching the userId
func (u *UserStore) GetByID(userID uint32) (*model.User, error) {
	u.RLock()
	defer u.RUnlock()
	for _, usr := range u.users {
		if usr.UserID == userID {
			return usr, nil
		}
	}
	return nil, consts.ErrUnauthorized
}

// Delete removes a user from the backing store
func (u *UserStore) Delete(user *model.User) error {
	u.Lock()
	delete(u.users, user.Passkey)
	u.Unlock()
	return nil
}

// Close will delete/free the underlying memory store
func (u *UserStore) Close() error {
	u.Lock()
	u.users = make(map[string]*model.User)
	u.Unlock()
	return nil
}

type userDriver struct{}

// NewUserStore creates a new memory backed user store.
func (pd userDriver) NewUserStore(_ interface{}) (store.UserStore, error) {
	return &UserStore{
		sync.RWMutex{},
		make(map[string]*model.User),
	}, nil
}

func init() {
	store.AddUserDriver(driverName, userDriver{})
	store.AddPeerDriver(driverName, peerDriver{})
	store.AddTorrentDriver(driverName, torrentDriver{})
}
