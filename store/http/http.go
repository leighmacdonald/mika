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
	"github.com/leighmacdonald/mika/config"
	"github.com/leighmacdonald/mika/consts"
	h "github.com/leighmacdonald/mika/http"
	"github.com/leighmacdonald/mika/model"
	"github.com/leighmacdonald/mika/store"
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
	*h.AuthedClient
}

// Sync batch updates the backing store with the new TorrentStats provided
func (ts TorrentStore) Sync(_ map[model.InfoHash]model.TorrentStats) error {
	panic("implement me")
}

// Conn returns the underlying http client
func (ts TorrentStore) Conn() interface{} {
	return ts
}

// WhiteListDelete removes a client from the global whitelist
func (ts TorrentStore) WhiteListDelete(client model.WhiteListClient) error {
	_, err := ts.Exec(h.Opts{
		Method: "DELETE",
		Path:   fmt.Sprintf("/whitelist/%s", client.ClientPrefix),
	})
	if err != nil {
		return err
	}
	return nil
}

// WhiteListAdd will insert a new client prefix into the allowed clients list
func (ts TorrentStore) WhiteListAdd(client model.WhiteListClient) error {
	_, err := ts.Exec(h.Opts{
		Method: "POST",
		Path:   "/whitelist",
		JSON:   client,
	})
	if err != nil {
		return err
	}
	return nil
}

// WhiteListGetAll fetches all known whitelisted clients
func (ts TorrentStore) WhiteListGetAll() ([]model.WhiteListClient, error) {
	var wl []model.WhiteListClient
	_, err := ts.Exec(h.Opts{
		Method: "GET",
		Path:   "/whitelist",
		Recv:   &wl,
	})
	if err != nil {
		return nil, err
	}
	return wl, nil
}

// Add adds a new torrent to the HTTP API backing store
func (ts TorrentStore) Add(t model.Torrent) error {
	_, err := ts.Exec(h.Opts{
		Method: "POST",
		Path:   "/torrent",
		JSON:   t,
	})
	return err
}

// Delete will mark a torrent as deleted in the backing store.
// If dropRow is true, it will permanently remove the torrent from the store
func (ts TorrentStore) Delete(ih model.InfoHash, dropRow bool) error {
	if dropRow {
		_, err := ts.Exec(h.Opts{
			Method: "DELETE",
			Path:   fmt.Sprintf("/torrent/%s", ih.String()),
		})
		return err
	}
	_, err := ts.Exec(h.Opts{
		Method: "PATCH",
		Path:   fmt.Sprintf("/torrent/%s", ih.String()),
		JSON: map[string]interface{}{
			"is_deleted": true,
		}})
	return err
}

// Get returns the Torrent matching the infohash
func (ts TorrentStore) Get(t *model.Torrent, hash model.InfoHash) error {
	_, err := ts.Exec(h.Opts{
		Method: "GET",
		Path:   fmt.Sprintf("/torrent/%s", hash.String()),
		Recv:   t,
	})
	return err
}

// Close will close all the remaining http connections
func (ts TorrentStore) Close() error {
	ts.CloseIdleConnections()
	return nil
}

// PeerStore is the HTTP API backed store.PeerStore implementation
type PeerStore struct {
	*h.AuthedClient
}

// Sync batch updates the backing store with the new PeerStats provided
func (ps PeerStore) Sync(_ map[model.PeerHash]model.PeerStats) error {
	panic("implement me")
}

// Reap will loop through the peers removing any stale entries from active swarms
func (ps PeerStore) Reap() {
	panic("implement me")
}

// Add inserts a peer into the active swarm for the torrent provided
func (ps PeerStore) Add(ih model.InfoHash, p model.Peer) error {
	_, err := ps.Exec(h.Opts{
		Method: "POST",
		Path:   fmt.Sprintf("/torrent/%s/peer", ih.String()),
		JSON:   p,
	})
	return err
}

// Get will fetch the peer from the swarm if it exists
func (ps PeerStore) Get(_ *model.Peer, _ model.InfoHash, _ model.PeerID) error {
	panic("implement me")
}

// Delete will remove a user from a torrents swarm
func (ps PeerStore) Delete(ih model.InfoHash, p model.PeerID) error {
	_, err := ps.Exec(h.Opts{
		Method: "DELETE",
		Path:   fmt.Sprintf("/torrent/%s/peer/%s", ih.String(), p.String()),
	})
	return err
}

// GetN will fetch peers for a torrents active swarm up to N users
func (ps PeerStore) GetN(ih model.InfoHash, limit int) (model.Swarm, error) {
	var peers model.Swarm
	_, err := ps.Exec(h.Opts{
		Method: "GET",
		Path:   fmt.Sprintf("/torrent/%s/peers/%d", ih.String(), limit),
		Recv:   peers,
	})
	return peers, err
}

// Close will close all the remaining http connections
func (ps PeerStore) Close() error {
	ps.CloseIdleConnections()
	return nil
}

func NewTorrentStore(key string, baseUrl string) *TorrentStore {
	return &TorrentStore{h.NewAuthedClient(key, fullSchema(baseUrl))}
}

// NewTorrentStore initialize a TorrentStore implementation using the HTTP API backing store
func (t torrentDriver) NewTorrentStore(cfg interface{}) (store.TorrentStore, error) {
	c, ok := cfg.(*config.StoreConfig)
	if !ok {
		return nil, consts.ErrInvalidConfig
	}
	return NewTorrentStore(c.Password, c.Host), nil
}

func NewPeerStore(key string, baseUrl string) *PeerStore {
	return &PeerStore{h.NewAuthedClient(key, fullSchema(baseUrl))}
}

type peerDriver struct {
	*h.AuthedClient
}

// NewPeerStore initialize a NewPeerStore implementation using the HTTP API backing store
func (p peerDriver) NewPeerStore(cfg interface{}) (store.PeerStore, error) {
	c, ok := cfg.(*config.StoreConfig)
	if !ok {
		return nil, consts.ErrInvalidConfig
	}
	return NewPeerStore(c.Password, c.Host), nil
}

// UserStore is the HTTP API backed store.UserStore implementation
type UserStore struct {
	*h.AuthedClient
}

// Sync batch updates the backing store with the new UserStats provided
func (u *UserStore) Sync(_ map[string]model.UserStats) error {
	panic("implement me")
}

// Add will add a new user to the backing store
func (u *UserStore) Add(user model.User) error {
	_, err := u.Exec(h.Opts{
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
func (u *UserStore) GetByPasskey(usr *model.User, passkey string) error {
	if len(passkey) != 20 {
		return consts.ErrUnauthorized
	}
	_, err := u.Exec(h.Opts{
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
func (u *UserStore) GetByID(_ *model.User, _ uint32) error {
	panic("implement me")
}

// Delete removes a user from the backing store
func (u *UserStore) Delete(_ model.User) error {
	panic("implement me")
}

// Close will close all the remaining http connections
func (u *UserStore) Close() error {
	u.CloseIdleConnections()
	return nil
}

func NewUserStore(key string, baseUrl string) *UserStore {
	return &UserStore{h.NewAuthedClient(key, fullSchema(baseUrl))}
}

// TODO handle https
func fullSchema(host string) string {
	if !strings.HasPrefix(host, "http://") || !strings.HasPrefix(host, "https://") {
		return "http://" + host
	}
	return host
}

type userDriver struct{}

// NewUserStore creates a new http api backed user store.
// the config key store_users_host should be a full url prefix, including port.
// This should be everything up to the /api/... path
// eg: http://localhost:35000 will be translated into:
// http://localhost:35000/api/user/pk/12345678901234567890
func (p userDriver) NewUserStore(cfg interface{}) (store.UserStore, error) {
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
