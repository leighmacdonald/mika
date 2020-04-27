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
	"github.com/jmoiron/sqlx"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"mika/config"
	"mika/model"
	"mika/store/memory"
	"mika/store/mysql"
	"mika/store/redis"
)

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

func NewTorrentStore(storeType string) TorrentStore {
	var s TorrentStore
	switch storeType {
	case "memory":
		s = memory.NewTorrentStore()
	case "mysql":
		fallthrough
	case "mariadb":
		s = mysql.NewTorrentStore(config.DSN())
	case "postgres":
		log.Panicf("Unimplemented store type specified: %s", storeType)
	case "redis":
		host := viper.GetString(config.CacheHost)
		port := viper.GetInt(config.CachePort)
		password := viper.GetString(config.CachePassword)
		db := viper.GetInt(config.CacheDB)
		s = redis.NewTorrentStore(host, port, password, db)
	default:
		log.Panicf("Unknown store type specified: %s", storeType)
	}
	return s
}

func NewPeerStore(storeType string, db *sqlx.DB) PeerStore {
	var s PeerStore
	switch storeType {
	case "memory":
		s = memory.NewPeerStore()
	case "mysql":
		fallthrough
	case "mariadb":
		s = mysql.NewPeerStore(config.DSN(), db)
	case "postgres":
		log.Panicf("Unimplemented store type specified: %s", storeType)
	case "redis":
		host := viper.GetString(config.CacheHost)
		port := viper.GetInt(config.CachePort)
		password := viper.GetString(config.CachePassword)
		db := viper.GetInt(config.CacheDB)
		s = redis.NewPeerStore(host, port, password, db, nil)
	default:
		log.Panicf("Unknown store type specified: %s", storeType)
	}
	return s
}
