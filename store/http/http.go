// Package http defines a storage backend over a HTTP API.
// This is meant to make basic interoperability possible for users
// who do not want to change their data model (or use views on compatible RDBMS systems)
//
// Users will only need to create compatible endpoints in their codebases that we can communicate with
// It is the users job at that point to do any conversions of data type, names, etc. required to be
// compatible with their system
package http

import (
	"bytes"
	"encoding/json"
	"fmt"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"mika/config"
	"mika/consts"
	"mika/model"
	"mika/store"
	"net"
	"net/http"
	"time"
)

const (
	driverName = "http"
)

// authMode defines what type of authentication to use when talking to the http backing store api
type authMode int

const (
	basic authMode = iota
	bearerToken
	keyToken
)

type torrentDriver struct{}

// TorrentStore is the HTTP API backed store.TorrentStore implementation
type TorrentStore struct {
	client  *http.Client
	baseURL string
}

func checkResponse(resp *http.Response, code int) error {
	switch resp.StatusCode {
	case code:
		return nil
	default:
		log.Errorf("Received invalid response code from server: %d", resp.StatusCode)
		return consts.ErrInvalidResponseCode
	}
}

func doRequest(client *http.Client, method string, path string, data interface{}) (*http.Response, error) {
	b, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(method, path, bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	return client.Do(req)
}

// AddTorrent adds a new torrent to the HTTP API backing store
func (ts TorrentStore) AddTorrent(t *model.Torrent) error {
	resp, err := doRequest(ts.client, "POST", fmt.Sprintf(ts.baseURL, "/torrent"), t)
	if err != nil {
		return err
	}
	return checkResponse(resp, http.StatusCreated)
}

// DeleteTorrent will mark a torrent as deleted in the backing store.
// If dropRow is true, it will permanently remove the torrent from the store
func (ts TorrentStore) DeleteTorrent(t *model.Torrent, dropRow bool) error {
	if dropRow {
		resp, err := doRequest(ts.client, "DELETE", fmt.Sprintf(ts.baseURL, "/torrent"), t)
		if err != nil {
			return err
		}
		return checkResponse(resp, http.StatusOK)
	}
	resp, err := doRequest(ts.client, "PATCH", fmt.Sprintf(ts.baseURL, "/torrent"), map[string]interface{}{
		"total_downloaded": t.TotalDownloaded,
		"total_uploaded":   t.TotalUploaded,
	})
	if err != nil {
		return err
	}
	return checkResponse(resp, http.StatusOK)
}

// GetTorrent returns the Torrent matching the infohash
func (ts TorrentStore) GetTorrent(hash model.InfoHash) (*model.Torrent, error) {
	url := fmt.Sprintf("%s/torrent/%s", ts.baseURL, hash.String())
	resp, err := doRequest(ts.client, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	if err := checkResponse(resp, http.StatusOK); err != nil {
		return nil, err
	}
	b, err := ioutil.ReadAll(resp.Body)
	defer func() {
		_ = resp.Body.Close()
	}()
	var t *model.Torrent
	if err := json.Unmarshal(b, t); err != nil {
		return nil, err
	}
	return t, nil
}

// Close will close all the remaining http connections
func (ts TorrentStore) Close() error {
	ts.client.CloseIdleConnections()
	return nil
}

// PeerStore is the HTTP API backed store.PeerStore implementation
type PeerStore struct {
	client  *http.Client
	baseURL string
}

// AddPeer inserts a peer into the active swarm for the torrent provided
func (ps PeerStore) AddPeer(t *model.Torrent, p *model.Peer) error {
	resp, err := doRequest(ps.client, "POST", fmt.Sprintf(ps.baseURL, "/torrent/%s/peer", t.InfoHash), p)
	if err != nil {
		return err
	}
	return checkResponse(resp, http.StatusCreated)
}

// UpdatePeer will sync any new peer data with the backing store
func (ps PeerStore) UpdatePeer(t *model.Torrent, p *model.Peer) error {
	panic("implement me")
}

// DeletePeer will remove a user from a torrents swarm
func (ps PeerStore) DeletePeer(t *model.Torrent, p *model.Peer) error {
	reqUrl := fmt.Sprintf(ps.baseURL, "/torrent/%s/peer/%s", t.InfoHash, p.PeerID)
	resp, err := doRequest(ps.client, "DELETE", reqUrl, nil)
	if err != nil {
		return err
	}
	return checkResponse(resp, http.StatusOK)
}

// GetPeers will fetch peers for a torrents active swarm up to N users
func (ps PeerStore) GetPeers(t *model.Torrent, limit int) ([]*model.Peer, error) {
	var peers []*model.Peer
	resp, err := doRequest(ps.client, "GET", fmt.Sprintf(ps.baseURL, "/torrent/%s/peers", t.InfoHash), nil)
	if err != nil {
		return nil, err
	}
	if err := checkResponse(resp, http.StatusOK); err != nil {
		return nil, err
	}
	b, err := ioutil.ReadAll(resp.Body)
	defer func() {
		_ = resp.Body.Close()
	}()
	if err := json.Unmarshal(b, &peers); err != nil {
		return nil, err
	}
	return peers, nil
}

// GetScrape returns scrape data for the torrent provided
func (ps PeerStore) GetScrape(t *model.Torrent) {
	panic("implement me")
}

// Close will close all the remaining http connections
func (ps PeerStore) Close() error {
	ps.client.CloseIdleConnections()
	return nil
}

// NewTorrentStore initialize a TorrentStore implementation using the HTTP API backing store
func (t torrentDriver) NewTorrentStore(cfg interface{}) (store.TorrentStore, error) {
	c, ok := cfg.(*config.StoreConfig)
	if !ok {
		return nil, consts.ErrInvalidConfig
	}
	return TorrentStore{
		client:  newClient(c),
		baseURL: c.Host,
	}, nil
}

type peerDriver struct{}

// NewPeerStore initialize a NewPeerStore implementation using the HTTP API backing store
func (p peerDriver) NewPeerStore(cfg interface{}) (store.PeerStore, error) {
	c, ok := cfg.(*config.StoreConfig)
	if !ok {
		return nil, consts.ErrInvalidConfig
	}
	return PeerStore{
		client:  newClient(c),
		baseURL: c.Host,
	}, nil
}

// UserStore is the HTTP API backed store.UserStore implementation
type UserStore struct {
	client  *http.Client
	baseURL string
}

// GetUserByPasskey will lookup and return the user via their passkey used as an identifier
// The errors returned for this method should be very generic and not reveal any info
// that could possibly help attackers gain any insight. All error cases MUST
// return ErrUnauthorized.
func (u *UserStore) GetUserByPasskey(passkey string) (model.User, error) {
	var usr model.User
	if passkey == "" || len(passkey) != 20 {
		return usr, consts.ErrUnauthorized
	}
	path := fmt.Sprintf("%s/api/user/pk/%s", u.baseURL, passkey)
	resp, err := doRequest(u.client, "GET", path, nil)
	if err != nil {
		log.Errorf("Failed to make api call to backing http api: %s", err)
		return usr, consts.ErrUnauthorized
	}
	if err := checkResponse(resp, http.StatusOK); err != nil {
		return usr, consts.ErrUnauthorized
	}
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Errorf("Could not read response body from backing http api: %s", err.Error())
		return usr, consts.ErrUnauthorized
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if err := json.Unmarshal(b, &usr); err != nil {
		log.Warnf("Failed to decode user data from backing http api: %s", err.Error())
		return usr, consts.ErrUnauthorized
	}
	if !usr.Valid() {
		log.Warnf("Received invalid user data from backing http api")
		return usr, consts.ErrUnauthorized
	}
	return usr, nil
}

// GetUserByID returns a user matching the userId
func (u *UserStore) GetUserByID(userId uint32) (model.User, error) {
	panic("implement me")
}

// DeleteUser removes a user from the backing store
func (u *UserStore) DeleteUser(user model.User) error {
	panic("implement me")
}

// Close will close all the remaining http connections
func (u *UserStore) Close() error {
	u.client.CloseIdleConnections()
	return nil
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
	return &UserStore{
		client:  newClient(c),
		baseURL: c.Host,
	}, nil
}

func newClient(c *config.StoreConfig) *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			Dial: (&net.Dialer{
				Timeout: time.Second * 5,
			}).Dial,
			TLSHandshakeTimeout: time.Second * 5,
		},
		CheckRedirect: nil,
		Jar:           nil,
		Timeout:       time.Second * 5,
	}
}
func init() {
	store.AddUserDriver(driverName, userDriver{})
	store.AddPeerDriver(driverName, peerDriver{})
	store.AddTorrentDriver(driverName, torrentDriver{})
}
