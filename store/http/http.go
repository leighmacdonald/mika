// Package http defines a storage backend over a HTTP API.
// This is meant to make basic interoperability possible for users
// who do not want to change their data model (or use views on compatible RDBMS systems)
//
// Users will only need to create compatible endpoints in their codebase that we can communicate with
// It is the users job at that point to do any conversions of data type, names, etc. required to be
// compatible with their system
package http

import (
	"fmt"
	"github.com/leighmacdonald/mika/client"
	"github.com/leighmacdonald/mika/config"
	"github.com/leighmacdonald/mika/consts"
	"github.com/leighmacdonald/mika/store"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"strings"
)

const (
	driverName = "http"
)

// authMode defines what type of authentication to use when talking to the http backing store api
//type authMode int
//
////noinspection GoUnusedConst
//const (
//	basic authMode = iota
//	bearerToken
//	keyToken
//)

type torrentDriver struct{}

// TorrentStore is the HTTP API backed store.TorrentStore implementation
type TorrentStore struct {
	*client.AuthedClient
}

func (ts TorrentStore) Name() string {
	return driverName
}

func (ts TorrentStore) Update(torrent store.Torrent) error {
	panic("implement me")
}

// Sync batch updates the backing store with the new TorrentStats provided
func (ts TorrentStore) Sync(batch map[store.InfoHash]store.TorrentStats, cache *store.TorrentCache) error {
	req := make(map[string]store.TorrentStats)
	for k, v := range batch {
		req[k.String()] = v
	}
	_, err := ts.Exec(client.Opts{
		Method: "POST",
		Path:   fmt.Sprintf("/api/torrent/sync"),
		JSON:   req,
	})
	if err != nil {
		return err
	}
	if cache != nil {
		for k, v := range batch {
			cache.Update(k, v)
		}
	}
	return nil
}

// Conn returns the underlying http client
func (ts TorrentStore) Conn() interface{} {
	return ts
}

// WhiteListDelete removes a client from the global whitelist
func (ts TorrentStore) WhiteListDelete(wlc store.WhiteListClient) error {
	_, err := ts.Exec(client.Opts{
		Method: "DELETE",
		Path:   fmt.Sprintf("/api/whitelist/%s", wlc.ClientPrefix),
	})
	if err != nil {
		return err
	}
	return nil
}

// WhiteListAdd will insert a new client prefix into the allowed clients list
func (ts TorrentStore) WhiteListAdd(wlc store.WhiteListClient) error {
	_, err := ts.Exec(client.Opts{
		Method: "POST",
		Path:   "/api/whitelist",
		JSON:   wlc,
	})
	if err != nil {
		return err
	}
	return nil
}

// WhiteListGetAll fetches all known whitelisted clients
func (ts TorrentStore) WhiteListGetAll() ([]store.WhiteListClient, error) {
	var wl []store.WhiteListClient
	_, err := ts.Exec(client.Opts{
		Method: "GET",
		Path:   "/api/whitelist",
		Recv:   &wl,
	})
	if err != nil {
		return nil, err
	}
	return wl, nil
}

// Add adds a new torrent to the HTTP API backing store
func (ts TorrentStore) Add(t store.Torrent) error {
	_, err := ts.Exec(client.Opts{
		Method: "POST",
		Path:   "/api/torrent",
		JSON:   t,
	})
	return err
}

// Delete will mark a torrent as deleted in the backing store.
// If dropRow is true, it will permanently remove the torrent from the store
func (ts TorrentStore) Delete(ih store.InfoHash, dropRow bool) error {
	if dropRow {
		_, err := ts.Exec(client.Opts{
			Method: "DELETE",
			Path:   fmt.Sprintf("/api/torrent/%s", ih.String()),
		})
		return err
	}
	_, err := ts.Exec(client.Opts{
		Method: "PATCH",
		Path:   fmt.Sprintf("/api/torrent/%s", ih.String()),
		JSON: map[string]interface{}{
			"is_deleted": true,
		}})
	return err
}

// Get returns the Torrent matching the infohash
func (ts TorrentStore) Get(t *store.Torrent, hash store.InfoHash, deletedOk bool) error {
	resp, err := ts.Exec(client.Opts{
		Method: "GET",
		Path:   fmt.Sprintf("/api/torrent/%s", hash.String()),
		Recv:   t,
	})
	if err != nil && resp != nil {
		if resp.StatusCode == 404 {
			return consts.ErrInvalidInfoHash
		}
	} else if err != nil {
		return err
	}
	return nil
}

// Close will close all the remaining http connections
func (ts TorrentStore) Close() error {
	ts.CloseIdleConnections()
	return nil
}

// PeerStore is the HTTP API backed store.PeerStore implementation
type PeerStore struct {
	*client.AuthedClient
}

func (ps PeerStore) Name() string {
	return driverName
}

// Sync batch updates the backing store with the new PeerStats provided
func (ps PeerStore) Sync(batch map[store.PeerHash]store.PeerStats, cache *store.PeerCache) error {
	rb := make(map[string]store.PeerStats)
	for k, v := range batch {
		rb[k.String()] = v
	}
	_, err := ps.Exec(client.Opts{
		Method: "POST",
		Path:   "/api/peers/sync",
		JSON:   rb,
	})
	return err
}

// Reap will loop through the peers removing any stale entries from active swarms
func (ps PeerStore) Reap(cache *store.PeerCache) {
	panic("implement me")
}

// Add inserts a peer into the active swarm for the torrent provided
func (ps PeerStore) Add(ih store.InfoHash, p store.Peer) error {
	_, err := ps.Exec(client.Opts{
		Method: "POST",
		Path:   fmt.Sprintf("/api/peer/create/%s", ih.String()),
		JSON:   p,
	})
	return err
}

// Get will fetch the peer from the swarm if it exists
func (ps PeerStore) Get(_ *store.Peer, _ store.InfoHash, _ store.PeerID) error {
	panic("implement me")
}

// Delete will remove a user from a torrents swarm
func (ps PeerStore) Delete(ih store.InfoHash, p store.PeerID) error {
	_, err := ps.Exec(client.Opts{
		Method: "DELETE",
		Path:   fmt.Sprintf("/api/peers/delete/%s/%s", ih.String(), p.String()),
	})
	return err
}

// GetN will fetch peers for a torrents active swarm up to N users
func (ps PeerStore) GetN(ih store.InfoHash, limit int) (store.Swarm, error) {
	swarm := store.NewSwarm()
	var peers []store.Peer
	_, err := ps.Exec(client.Opts{
		Method: "GET",
		Path:   fmt.Sprintf("/api/peers/swarm/%s/%d", ih.String(), limit),
		Recv:   &peers,
	})
	for _, peer := range peers {
		swarm.Peers[peer.PeerID] = peer
	}
	return swarm, err
}

// Close will close all the remaining http connections
func (ps PeerStore) Close() error {
	ps.CloseIdleConnections()
	return nil
}

// NewTorrentStore instantiates a new http torrent store
func NewTorrentStore(key string, baseURL string) *TorrentStore {
	return &TorrentStore{client.NewAuthedClient(key, fullSchema(baseURL))}
}

// New initialize a TorrentStore implementation using the HTTP API backing store
func (t torrentDriver) New(cfg interface{}) (store.TorrentStore, error) {
	c, ok := cfg.(*config.StoreConfig)
	if !ok {
		return nil, consts.ErrInvalidConfig
	}
	return NewTorrentStore(c.Password, c.Host), nil
}

// NewPeerStore instantiates a new http peer store
func NewPeerStore(key string, baseURL string) *PeerStore {
	return &PeerStore{client.NewAuthedClient(key, fullSchema(baseURL))}
}

type peerDriver struct {
	*client.AuthedClient
}

// New initialize a New implementation using the HTTP API backing store
func (p peerDriver) New(cfg interface{}) (store.PeerStore, error) {
	c, ok := cfg.(*config.StoreConfig)
	if !ok {
		return nil, consts.ErrInvalidConfig
	}
	return NewPeerStore(c.Password, c.Host), nil
}

// UserStore is the HTTP API backed store.UserStore implementation
type UserStore struct {
	*client.AuthedClient
}

func (u *UserStore) Name() string {
	return driverName
}

// Sync batch updates the backing store with the new UserStats provided
func (u *UserStore) Sync(batch map[string]store.UserStats, cache *store.UserCache) error {
	_, err := u.Exec(client.Opts{
		Method: "POST",
		Path:   "/api/user/sync",
		JSON:   batch,
	})
	return err
}

// Add will add a new user to the backing store
func (u *UserStore) Add(user store.User) error {
	_, err := u.Exec(client.Opts{
		Method: "POST",
		Path:   "/api/user",
		JSON:   user,
	})
	if err != nil {
		log.Errorf("Failed to make api call to backing http api: %s", err)
		return consts.ErrUnauthorized
	}
	return nil
}

// GetByPasskey will lookup and return the user via their passkey used as an identifier
// The errors returned for this method should be very generic and not reveal any info
// that could possibly help attackers gain any insight. All error cases MUST
// return ErrUnauthorized.
func (u *UserStore) GetByPasskey(usr *store.User, passkey string) error {
	if len(passkey) != 20 {
		return consts.ErrUnauthorized
	}
	_, err := u.Exec(client.Opts{
		Method: "GET",
		Path:   fmt.Sprintf("/api/user/pk/%s", passkey),
		Recv:   usr,
	})
	if err != nil {
		log.Errorf("Failed to make api call to backing http api: %s", err)
		return consts.ErrUnauthorized
	}
	if !usr.Valid() {
		log.Warnf("Received invalid user data from backing http api")
		return consts.ErrUnauthorized
	}
	return nil
}

// GetByID returns a user matching the userId
func (u *UserStore) GetByID(user *store.User, userID uint32) error {
	if userID == 0 {
		return consts.ErrUnauthorized
	}
	_, err := u.Exec(client.Opts{
		Method: "GET",
		Path:   fmt.Sprintf("/api/user/id/%d", userID),
		Recv:   user,
	})
	if err != nil {
		return errors.Wrapf(err, "Failed to fetch user from backing store api")
	}
	if !user.Valid() {
		return consts.ErrUnauthorized
	}
	return nil
}

// Update updates a user from the backing store
func (u *UserStore) Update(user store.User, oldPasskey string) error {
	panic("implement me")
}

// Delete removes a user from the backing store
func (u *UserStore) Delete(_ store.User) error {
	panic("implement me")
}

// Close will close all the remaining http connections
func (u *UserStore) Close() error {
	u.CloseIdleConnections()
	return nil
}

// NewUserStore instantiated a new http user store
func NewUserStore(key string, baseURL string) *UserStore {
	return &UserStore{client.NewAuthedClient(key, fullSchema(baseURL))}
}

// TODO handle https
func fullSchema(host string) string {
	if !strings.HasPrefix(host, "http://") || !strings.HasPrefix(host, "https://") {
		return "http://" + host
	}
	return host
}

type userDriver struct{}

// New creates a new http api backed user store.
// the config key store_users_host should be a full url prefix, including port.
// This should be everything up to the /api/... path
// eg: http://localhost:35000 will be translated into:
// http://localhost:35000/api/user/pk/12345678901234567890
func (p userDriver) New(cfg interface{}) (store.UserStore, error) {
	c, ok := cfg.(*config.StoreConfig)
	if !ok {
		return nil, consts.ErrInvalidConfig
	}

	return NewUserStore(c.Password, c.Host), nil
}

func init() {
	store.AddUserDriver(driverName, userDriver{})
	store.AddPeerDriver(driverName, peerDriver{})
	store.AddTorrentDriver(driverName, torrentDriver{})
}
