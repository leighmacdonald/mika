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

type authMode int

const (
	Basic authMode = iota
	BearerToken
	KeyToken
)

type torrentDriver struct{}

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

func (ts TorrentStore) AddTorrent(t *model.Torrent) error {
	resp, err := doRequest(ts.client, "POST", fmt.Sprintf(ts.baseURL, "/torrent"), t)
	if err != nil {
		return err
	}
	return checkResponse(resp, http.StatusCreated)
}

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

func (ts TorrentStore) GetTorrent(hash model.InfoHash) (*model.Torrent, error) {
	resp, err := doRequest(ts.client, "GET", fmt.Sprintf(ts.baseURL, "/torrent/%s", hash.RawString()), nil)
	if err != nil {
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

func (ts TorrentStore) Close() error {
	ts.client.CloseIdleConnections()
	return nil
}

type PeerStore struct {
	client  *http.Client
	baseUrl string
}

func (ps PeerStore) AddPeer(t *model.Torrent, p *model.Peer) error {
	resp, err := doRequest(ps.client, "POST", fmt.Sprintf(ps.baseUrl, "/torrent/%s/peer", t.InfoHash), p)
	if err != nil {
		return err
	}
	return checkResponse(resp, http.StatusCreated)
}

func (ps PeerStore) UpdatePeer(t *model.Torrent, p *model.Peer) error {
	panic("implement me")
}

func (ps PeerStore) DeletePeer(t *model.Torrent, p *model.Peer) error {
	reqUrl := fmt.Sprintf(ps.baseUrl, "/torrent/%s/peer/%s", t.InfoHash, p.PeerID)
	resp, err := doRequest(ps.client, "DELETE", reqUrl, nil)
	if err != nil {
		return err
	}
	return checkResponse(resp, http.StatusOK)
}

func (ps PeerStore) GetPeers(t *model.Torrent, limit int) ([]*model.Peer, error) {
	var peers []*model.Peer
	resp, err := doRequest(ps.client, "GET", fmt.Sprintf(ps.baseUrl, "/torrent/%s/peers", t.InfoHash), nil)
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

func (ps PeerStore) GetScrape(t *model.Torrent) {
	panic("implement me")
}

func (ps PeerStore) Close() error {
	ps.client.CloseIdleConnections()
	return nil
}

func (t torrentDriver) NewTorrentStore(config interface{}) (store.TorrentStore, error) {
	c, ok := config.(*Config)
	if !ok {
		return nil, consts.ErrInvalidConfig
	}
	return TorrentStore{
		client: newClient(c),
	}, nil
}

type peerDriver struct{}

func (p peerDriver) NewPeerStore(config interface{}) (store.PeerStore, error) {
	c, ok := config.(*Config)
	if !ok {
		return nil, consts.ErrInvalidConfig
	}
	return PeerStore{
		client: newClient(c),
	}, nil
}

// Config defines how we connect to external endpoints
type Config struct {
	BaseURL    string
	AuthMethod authMode
	Timeout    time.Duration
}

func newClient(c *Config) *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			Dial: (&net.Dialer{
				Timeout: c.Timeout,
			}).Dial,
			TLSHandshakeTimeout: c.Timeout,
		},
		CheckRedirect: nil,
		Jar:           nil,
		Timeout:       c.Timeout,
	}
}
func init() {
	store.AddPeerDriver(driverName, peerDriver{})
	store.AddTorrentDriver(driverName, torrentDriver{})
}
