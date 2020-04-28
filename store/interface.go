// Package store provides the underlying interfaces and glue for the backend storage drivers.
//
// We define 2 distinct interfaces to allow for flexibility in storage options.
// TorrentStore is meant as a persistent storage backend which is backed to permanent storage
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
	peerDriversMutex    = sync.RWMutex{}
	torrentDriversMutex = sync.RWMutex{}
	peerDrivers         = make(map[string]PeerDriver)
	torrentDrivers      = make(map[string]TorrentDriver)
)

type TorrentDriver interface {
	NewTorrentStore(config interface{}) (TorrentStore, error)
}
type PeerDriver interface {
	NewPeerStore(config interface{}) (PeerStore, error)
}

func AddPeerDriver(name string, driver PeerDriver) {
	peerDriversMutex.Lock()
	defer peerDriversMutex.Unlock()
	peerDrivers[name] = driver
	log.Debugf("Registered peer storage driver: %s", name)
}

func AddTorrentDriver(name string, driver TorrentDriver) {
	torrentDriversMutex.Lock()
	defer torrentDriversMutex.Unlock()
	torrentDrivers[name] = driver
	log.Debugf("Registered torrent storage driver: %s", name)
}

// Torrent store defines where we can store permanent torrent data
// The drivers should always persist the data to disk
type TorrentStore interface {
	// Add a new torrent to the backing store
	AddTorrent(t *model.Torrent) error
	DeleteTorrent(t *model.Torrent, dropRow bool) error
	GetTorrent(hash model.InfoHash) (*model.Torrent, error)
	Close() error
}

// PeerStore defines our interface for storing peer data
// This doesnt need to be persisted to disk, but it will help warm up times
// if its backed by something that can restore its in memory state, such as redis
type PeerStore interface {
	AddPeer(tid *model.Torrent, p *model.Peer) error
	UpdatePeer(tid *model.Torrent, p *model.Peer) error
	DeletePeer(tid *model.Torrent, p *model.Peer) error
	GetPeers(t *model.Torrent, limit int) ([]*model.Peer, error)
	GetScrape(t *model.Torrent)
	Close() error
}

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
