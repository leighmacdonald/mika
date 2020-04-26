package tracker

import (
	"github.com/spf13/viper"
	"mika/config"
	"mika/geo"
	"mika/store"
	"sync"
)

type Tracker struct {
	torrents store.TorrentStore
	peers    store.PeerStore
	geodb    *geo.DB
	// Whitelist and whitelist lock
	WhitelistMutex *sync.RWMutex
	Whitelist      []string
}

func New() *Tracker {
	s := store.NewTorrentStore(viper.GetString(config.StoreType))
	p := store.NewPeerStore(viper.GetString(config.CacheType), nil)
	geodb := geo.New(viper.GetString(config.GeodbPath))
	return &Tracker{
		torrents: s,
		peers:    p,
		geodb:    geodb,
	}
}
