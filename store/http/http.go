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
	"mika/consts"
	"mika/model"
	"mika/store"
	"net/http"
)

const (
	driverName = "http"
)

type AuthMode int

const (
	Basic AuthMode = iota
	BearerToken
	KeyToken
)

type torrentDriver struct{}

type TorrentStore struct {
	client  *http.Client
	baseUrl string
}

func checkResponse(resp *http.Response) error {
	switch resp.StatusCode {
	case http.StatusCreated:
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
	resp, err := doRequest(ts.client, "POST", fmt.Sprintf(ts.baseUrl, "/torrent"), t)
	if err != nil {
		return err
	}
	return checkResponse(resp)
}

func (ts TorrentStore) DeleteTorrent(t *model.Torrent, dropRow bool) error {
	if dropRow {
		resp, err := doRequest(ts.client, "DELETE", fmt.Sprintf(ts.baseUrl, "/torrent"), t)
		if err != nil {
			return err
		}
		return checkResponse(resp)
	} else {
		resp, err := doRequest(ts.client, "PATCH", fmt.Sprintf(ts.baseUrl, "/torrent"), t)
		if err != nil {
			return err
		}
		return checkResponse(resp)
	}

}

func (ts TorrentStore) GetTorrent(hash model.InfoHash) (*model.Torrent, error) {
	panic("implement me")
}

func (ts TorrentStore) Close() error {
	ts.client.CloseIdleConnections()
	return nil
}

type PeerStore struct {
	client *http.Client
}

func (ps PeerStore) AddPeer(t *model.Torrent, p *model.Peer) error {

}

func (ps PeerStore) UpdatePeer(t *model.Torrent, p *model.Peer) error {
	panic("implement me")
}

func (ps PeerStore) DeletePeer(t *model.Torrent, p *model.Peer) error {
	panic("implement me")
}

func (ps PeerStore) GetPeers(t *model.Torrent, limit int) ([]*model.Peer, error) {
	panic("implement me")
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

type Config struct {
	BaseUrl    string
	AuthMethod AuthMode
}

func newClient(c *Config) *http.Client {
	return &http.Client{
		Transport:     nil,
		CheckRedirect: nil,
		Jar:           nil,
		Timeout:       0,
	}
}
func init() {
	store.AddPeerDriver(driverName, peerDriver{})
	store.AddTorrentDriver(driverName, torrentDriver{})
}
