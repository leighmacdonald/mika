package tracker

import (
	"github.com/pkg/errors"
	"github.com/spf13/viper"
	"mika/config"
	"mika/geo"
	"mika/store"
	"sync"
)

// Tracker is the main application struct used to tie all the discreet components together
type Tracker struct {
	Torrents store.TorrentStore
	Peers    store.PeerStore
	Users    store.UserStore
	Geodb    *geo.DB
	// Whitelist and whitelist lock
	WhitelistMutex *sync.RWMutex
	Whitelist      []string
}

// New creates a new Tracker instance with configured backend stores
func New() (*Tracker, error) {
	var err error
	s, err := store.NewTorrentStore(
		viper.GetString(config.StoreTorrentType),
		config.GetStoreConfig(config.Torrent))
	if err != nil {
		return nil, errors.Wrap(err, "Failed to setup torrent store")
	}
	p, err := store.NewPeerStore(viper.GetString(config.StorePeersType),
		config.GetStoreConfig(config.Peers))
	if err != nil {
		return nil, errors.Wrap(err, "Failed to setup peer store")
	}
	u, err := store.NewUserStore(viper.GetString(config.StoreUsersType),
		config.GetStoreConfig(config.Users))
	if err != nil {
		return nil, errors.Wrap(err, "Failed to setup user store")
	}
	geodb := geo.New(viper.GetString(config.GeodbPath))
	return &Tracker{
		Torrents: s,
		Peers:    p,
		Users:    u,
		Geodb:    geodb,
	}, nil
}
