package tracker

import (
	"context"
	"github.com/leighmacdonald/mika/config"
	"github.com/leighmacdonald/mika/consts"
	"github.com/leighmacdonald/mika/geo"
	"github.com/leighmacdonald/mika/model"
	"github.com/leighmacdonald/mika/store"
	"github.com/leighmacdonald/mika/store/memory"
	"github.com/leighmacdonald/mika/util"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"time"

	// Imported for side-effects for NewTestTracker
	_ "github.com/leighmacdonald/mika/store/memory"
	"sync"
)

// Tracker is the main application struct used to tie all the discreet components together
type Tracker struct {
	// ctx is the master context used in the tracker, children contexts must use
	// this for their parent
	ctx      context.Context
	Torrents store.TorrentStore
	Peers    store.PeerStore
	Users    store.UserStore
	Geodb    geo.Provider
	// GeodbEnabled will enable the lookup of location data for peers
	// TODO the dummy provider is probably sufficient
	GeodbEnabled bool
	// Public if true means we dont require a passkey / authorized user
	Public bool
	// If Public is true, this will allow unknown info_hashes to be automatically tracked
	AutoRegister     bool
	AllowNonRoutable bool
	// ReaperInterval is how often we can for dead peers in swarms
	ReaperInterval time.Duration
	AnnInterval    time.Duration
	AnnIntervalMin time.Duration
	BatchInterval  time.Duration
	// MaxPeers is the max number of peers we send in an announce
	MaxPeers        int
	StateUpdateChan chan model.UpdateState
	// Whitelist and whitelist lock
	WhitelistMutex *sync.RWMutex
	Whitelist      map[string]model.WhiteListClient
}

type Opts struct {
	Torrents store.TorrentStore
	Peers    store.PeerStore
	Users    store.UserStore
	Geodb    geo.Provider
	// GeodbEnabled will enable the lookup of location data for peers
	// TODO the dummy provider is probably sufficient
	GeodbEnabled bool
	// Public if true means we dont require a passkey / authorized user
	Public bool
	// If Public is true, this will allow unknown info_hashes to be automatically tracked
	AutoRegister     bool
	AllowNonRoutable bool
	// ReaperInterval is how often we can for dead peers in swarms
	ReaperInterval time.Duration
	AnnInterval    time.Duration
	AnnIntervalMin time.Duration
	BatchInterval  time.Duration
	// MaxPeers is the max number of peers we send in an announce
	MaxPeers int
}

func NewDefaultOpts() *Opts {
	return &Opts{
		Torrents:         memory.NewTorrentStore(),
		Peers:            memory.NewPeerStore(),
		Users:            memory.NewUserStore(),
		Geodb:            &geo.DummyProvider{},
		GeodbEnabled:     false,
		Public:           false,
		AutoRegister:     false,
		AllowNonRoutable: false,
		ReaperInterval:   time.Second * 300,
		AnnInterval:      time.Second * 60,
		AnnIntervalMin:   time.Second * 30,
		BatchInterval:    time.Second * 60,
		MaxPeers:         100,
	}
}

// PeerReaper will call the store.PeerStore.Reap() function periodically. This is
// used to clean peers that have not announced in a while from the swarm.
func (t *Tracker) PeerReaper() {
	peerTicker := time.NewTicker(t.ReaperInterval)
	for {
		select {
		case <-peerTicker.C:
			t.Peers.Reap()
		case <-t.ctx.Done():
			return
		}
	}
}

// StatWorker handles summing up stats for users/peers/torrents to be sent to the
// backing stores for long term storage.
// No locking required for these data sets
func (t *Tracker) StatWorker() {
	syncTicker := time.NewTicker(t.BatchInterval)
	userBatch := make(map[string]model.UserStats)
	peerBatch := make(map[model.PeerHash]model.PeerStats)
	torrentBatch := make(map[model.InfoHash]model.TorrentStats)
	for {
		select {
		case <-syncTicker.C:
			// Copy the maps to pass into the go routine call. At the same time deleting
			// the existing values
			userBatchCopy := make(map[string]model.UserStats)
			for k, v := range userBatch {
				userBatchCopy[k] = v
				delete(userBatch, k)
			}
			peerBatchCopy := make(map[model.PeerHash]model.PeerStats)
			for k, v := range peerBatch {
				peerBatchCopy[k] = v
				delete(peerBatch, k)
			}
			torrentBatchCopy := make(map[model.InfoHash]model.TorrentStats)
			for k, v := range torrentBatch {
				torrentBatchCopy[k] = v
				delete(torrentBatch, k)
			}
			// TODO make sure we dont exec this more than once at a time
			go func() {
				// Send current copies of data to stores
				if err := t.Users.Sync(userBatchCopy); err != nil {
					log.Errorf(err.Error())
				}
				if err := t.Peers.Sync(peerBatchCopy); err != nil {
					log.Errorf(err.Error())
				}
				if err := t.Torrents.Sync(torrentBatchCopy); err != nil {
					log.Errorf(err.Error())
				}
			}()
		case u := <-t.StateUpdateChan:
			ub, found := userBatch[u.Passkey]
			if !found {
				ub = model.UserStats{}
			}
			tb, found := torrentBatch[u.InfoHash]
			if !found {
				tb = model.TorrentStats{}
			}
			pHash := model.NewPeerHash(u.InfoHash, u.PeerID)
			pb, found := peerBatch[pHash]
			if !found {
				pb = model.PeerStats{}
			}
			// Global user stats
			ub.Uploaded += u.Uploaded
			ub.Downloaded += u.Downloaded
			ub.Announces++

			// Peer stats
			pb.Downloaded += u.Downloaded
			pb.Uploaded += u.Uploaded
			pb.LastAnnounce = u.Timestamp
			pb.Announces++

			// Global torrent stats
			tb.Announces++
			tb.Uploaded += u.Uploaded
			tb.Downloaded += u.Downloaded

			switch u.Event {
			case consts.ANNOUNCE:

			case consts.STARTED:
				if u.Left == 0 {
					tb.Seeders++
				} else {
					tb.Leechers++
				}
			case consts.COMPLETED:
				// TODO does a complete event get sent for a torrent when the user only downloads a specific file from the torrent
				// Do we force left=0 for this? Or trust the client?
				tb.Snatches++
				tb.Seeders++
				tb.Leechers--
			case consts.STOPPED:
				if u.Left > 0 {
					tb.Leechers--
				} else {
					tb.Seeders--
				}
				if err := t.Peers.Delete(u.InfoHash, u.PeerID); err != nil {
					log.Errorf("Could not remove peer from swarm: %s", err.Error())
				}
			}
			userBatch[u.Passkey] = ub
			torrentBatch[u.InfoHash] = tb
			peerBatch[pHash] = pb
		case <-t.ctx.Done():
			return
		}
	}
}

// New creates a new Tracker instance with configured backend stores
// TODO pass these in as deps
func New(ctx context.Context, opts *Opts) (*Tracker, error) {
	return &Tracker{
		ctx:              ctx,
		StateUpdateChan:  make(chan model.UpdateState, 1000),
		Torrents:         opts.Torrents,
		Peers:            opts.Peers,
		Users:            opts.Users,
		Geodb:            opts.Geodb,
		GeodbEnabled:     opts.GeodbEnabled,
		Whitelist:        make(map[string]model.WhiteListClient),
		WhitelistMutex:   &sync.RWMutex{},
		MaxPeers:         opts.MaxPeers,
		BatchInterval:    opts.BatchInterval,
		ReaperInterval:   opts.ReaperInterval,
		AnnInterval:      opts.AnnInterval,
		AnnIntervalMin:   opts.AnnIntervalMin,
		AllowNonRoutable: opts.AllowNonRoutable,
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
	geoPath := util.FindFile(viper.GetString(string(config.GeodbPath)))
	var geodb geo.Provider
	if config.GetBool(config.GeodbEnabled) {
		geodb = geo.New(geoPath, false)
	} else {
		geodb = &geo.DummyProvider{}
	}
	return &Tracker{
		Torrents:         ts,
		Peers:            ps,
		Users:            us,
		Geodb:            geodb,
		GeodbEnabled:     viper.GetBool(string(config.GeodbEnabled)),
		WhitelistMutex:   &sync.RWMutex{},
		Whitelist:        wlm,
		MaxPeers:         50,
		StateUpdateChan:  make(chan model.UpdateState, 1000),
		ReaperInterval:   viper.GetDuration(string(config.TrackerReaperInterval)),
		AnnInterval:      viper.GetDuration(string(config.TrackerAnnounceInterval)),
		AnnIntervalMin:   viper.GetDuration(string(config.TrackerAnnounceIntervalMin)),
		AllowNonRoutable: true,
	}, torrents, users, peers
}

func (t *Tracker) LoadWhitelist() error {
	whitelist := make(map[string]model.WhiteListClient)
	wl, err4 := t.Torrents.WhiteListGetAll()
	if err4 != nil {
		log.Warnf("Whitelist empty, all clients are allowed")
	} else {
		for _, cw := range wl {
			whitelist[cw.ClientPrefix] = cw
		}
	}
	t.WhitelistMutex.Lock()
	t.Whitelist = whitelist
	t.WhitelistMutex.Unlock()
	return nil
}

// Stats returns the current cumulative stats for the tracker
func (t *Tracker) Stats() GlobalStats {
	var s GlobalStats

	return s
}

// GlobalStats holds basic stats for the running tracker
type GlobalStats struct {
}
