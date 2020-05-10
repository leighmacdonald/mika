package tracker

import (
	"context"
	"github.com/leighmacdonald/mika/config"
	"github.com/leighmacdonald/mika/consts"
	"github.com/leighmacdonald/mika/geo"
	"github.com/leighmacdonald/mika/model"
	"github.com/leighmacdonald/mika/store"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"time"

	// Imported for side-effects for NewTestTracker
	_ "github.com/leighmacdonald/mika/store/memory"
	"sync"
)

// Tracker is the main application struct used to tie all the discreet components together
type Tracker struct {
	ctx               context.Context
	Torrents          store.TorrentStore
	Peers             store.PeerStore
	Users             store.UserStore
	Geodb             *geo.DB
	AnnInterval       int
	AnnIntervalMin    int
	MaxPeers          int
	StateUpdateChan   chan model.UpdateState
	TorrentUpdateChan chan model.TorrentStats
	// Whitelist and whitelist lock
	WhitelistMutex *sync.RWMutex
	Whitelist      map[string]model.WhiteListClient
}

func (t *Tracker) PeerReaper() {
	peerTicker := time.NewTicker(30 * time.Second)
	for {
		select {
		case <-peerTicker.C:
			t.Peers.Reap()
		case <-t.ctx.Done():
			return
		}
	}

}

func (t *Tracker) StatWorker() {
	for {
		select {
		case u := <-t.StateUpdateChan:
			switch u.Event {

			case consts.COMPLETED:
				// TODO does a complete event get sent for a torrent when the user only downloads a specific file from the torrent
				// Do we force left=0 for this? Or trust the client?
				//tor.TotalCompleted++
			case consts.STOPPED:
				if err := t.Peers.Delete(u.InfoHash, u.PeerID); err != nil {
					log.Errorf("Could not remove peer from swarm: %s", err.Error())
				}
			}
			t.Torrents.UpdateState(u.InfoHash, model.TorrentStats{
				Seeders:  0,
				Leechers: 0,
				Snatches: 0,
				Event:    u.Event,
			})
		case <-t.ctx.Done():
			return
		}
	}
}

// New creates a new Tracker instance with configured backend stores
func New(ctx context.Context) (*Tracker, error) {
	var err error
	runMode := viper.GetString(string(config.GeneralRunMode))
	s, err := store.NewTorrentStore(
		viper.GetString(string(config.StoreTorrentType)),
		config.GetStoreConfig(config.Torrent))
	if err != nil {
		return nil, errors.Wrap(err, "Failed to setup torrent store")
	}
	p, err2 := store.NewPeerStore(viper.GetString(string(config.StorePeersType)),
		config.GetStoreConfig(config.Peers))
	if err2 != nil {
		return nil, errors.Wrap(err2, "Failed to setup peer store")
	}
	u, err3 := store.NewUserStore(viper.GetString(string(config.StoreUsersType)),
		config.GetStoreConfig(config.Users))
	if err3 != nil {
		return nil, errors.Wrap(err3, "Failed to setup user store")
	}

	geodb := geo.New(viper.GetString(string(config.GeodbPath)), runMode == "release")
	whitelist := make(map[string]model.WhiteListClient)
	wl, err4 := s.WhiteListGetAll()
	if err4 != nil {
		log.Warnf("Whitelist empty, all clients are allowed")
	} else {
		for _, cw := range wl {
			whitelist[cw.ClientPrefix] = cw
		}
	}
	return &Tracker{
		ctx:               ctx,
		StateUpdateChan:   make(chan model.UpdateState),
		TorrentUpdateChan: make(chan model.TorrentStats),
		Torrents:          s,
		Peers:             p,
		Users:             u,
		Geodb:             geodb,
		Whitelist:         whitelist,
		WhitelistMutex:    &sync.RWMutex{},
		MaxPeers:          50,
		AnnInterval:       viper.GetInt(string(config.TrackerAnnounceInterval)),
		AnnIntervalMin:    viper.GetInt(string(config.TrackerAnnounceIntervalMin)),
	}, nil
}

// NewTestTracker sets up a tracker with fake data for testing
// This shouldn't really exist here, but its currently needed by other packages so its exported
func NewTestTracker() (*Tracker, model.Torrents, model.Users, model.Swarm) {
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
	var users model.Users
	for i := 0; i < userCount; i++ {
		usr := store.GenerateTestUser()
		if i == 0 {
			// Give user 0 a known passkey for testing
			usr.Passkey = "12345678901234567890"
		}
		_ = us.Add(usr)
		users = append(users, usr)
	}
	if users == nil {
		log.Panicf("Failed to instantiate users")
		return nil, nil, nil, nil
	}
	var torrents model.Torrents
	for i := 0; i < torrentCount; i++ {
		t := store.GenerateTestTorrent()
		if err := ts.Add(t); err != nil {
			log.Panicf("Error adding torrent: %s", err.Error())
		}
		torrents = append(torrents, t)
	}
	wl, err := ts.WhiteListGetAll()
	if err != nil {
		log.Warnf("Failed to read any client whitelists, all clients allowed")
	}
	wlm := make(map[string]model.WhiteListClient)
	for _, cw := range wl {
		wlm[cw.ClientPrefix] = cw
	}
	var peers model.Swarm
	for _, t := range torrents {
		for i := 0; i < swarmSize; i++ {
			p := store.GenerateTestPeer()
			if err := ps.Add(t.InfoHash, p); err != nil {
				log.Panicf("Error adding peer: %s", err.Error())
			}
			peers = append(peers, p)
		}
	}
	return &Tracker{
		Torrents:       ts,
		Peers:          ps,
		Users:          us,
		Geodb:          geo.New(viper.GetString(string(config.GeodbPath)), false),
		WhitelistMutex: &sync.RWMutex{},
		Whitelist:      wlm,
		MaxPeers:       50,
		AnnInterval:    viper.GetInt(string(config.TrackerAnnounceInterval)),
		AnnIntervalMin: viper.GetInt(string(config.TrackerAnnounceIntervalMin)),
	}, torrents, users, peers
}
