package tracker

import (
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"mika/config"
	"mika/geo"
	"mika/model"
	"mika/store"
	// Imported for side-effects for NewTestTracker
	_ "mika/store/memory"
	"sync"
)

// Tracker is the main application struct used to tie all the discreet components together
type Tracker struct {
	Torrents       store.TorrentStore
	Peers          store.PeerStore
	Users          store.UserStore
	Geodb          *geo.DB
	AnnInterval    int
	AnnIntervalMin int
	MaxPeers       int
	// Whitelist and whitelist lock
	WhitelistMutex *sync.RWMutex
	Whitelist      map[string]model.WhiteListClient
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
		Torrents:       s,
		Peers:          p,
		Users:          u,
		Geodb:          geodb,
		MaxPeers:       50,
		AnnInterval:    viper.GetInt(config.TrackerAnnounceInterval),
		AnnIntervalMin: viper.GetInt(config.TrackerAnnounceIntervalMin),
	}, nil
}

// NewTestTracker sets up a tracker with fake data for testing
// This shouldn't really exist here, but its currently needed by other packages so its exported
func NewTestTracker() (*Tracker, []*model.Torrent, []*model.User, []*model.Peer) {
	userCount := 10
	torrentCount := 100
	swarmSize := 10 // Swarm per torrent
	ps, err := store.NewPeerStore("memory", config.StoreConfig{})
	if err != nil {
		log.Panicf("Failed to setup peer store: %s", err)
	}
	ts, err := store.NewTorrentStore("memory", config.StoreConfig{})
	if err != nil {
		log.Panicf("Failed to setup torrent store: %s", err)
	}
	us, err := store.NewUserStore("memory", config.StoreConfig{})
	if err != nil {
		log.Panicf("Failed to setup user store: %s", err)
	}
	var users []*model.User
	for i := 0; i < userCount; i++ {
		usr := store.GenerateTestUser()
		if i == 0 {
			// Give user 0 a known passkey for testing
			usr.Passkey = "12345678901234567890"
		}
		_ = us.AddUser(usr)
		users = append(users, usr)
	}
	if users == nil {
		log.Panicf("Failed to instantiate users")
		return nil, nil, nil, nil
	}
	var torrents []*model.Torrent
	for i := 0; i < torrentCount; i++ {
		t := store.GenerateTestTorrent()
		if err := ts.AddTorrent(t); err != nil {
			log.Panicf("Error adding torrent: %s", err.Error())
		}
		torrents = append(torrents, t)
	}
	var peers []*model.Peer
	for _, t := range torrents {
		for i := 0; i < swarmSize; i++ {
			p := store.GenerateTestPeer(users[i])
			if err := ps.AddPeer(t.InfoHash, p); err != nil {
				log.Panicf("Error adding peer: %s", err.Error())
			}
			peers = append(peers, p)
		}
	}
	return &Tracker{
		Torrents:       ts,
		Peers:          ps,
		Users:          us,
		Geodb:          geo.New(viper.GetString(config.GeodbPath)),
		WhitelistMutex: nil,
		Whitelist:      nil,
		MaxPeers:       50,
		AnnInterval:    viper.GetInt(config.TrackerAnnounceInterval),
		AnnIntervalMin: viper.GetInt(config.TrackerAnnounceIntervalMin),
	}, torrents, users, peers
}
