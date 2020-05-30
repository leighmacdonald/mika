package tracker

import (
	"context"
	"fmt"
	"github.com/leighmacdonald/mika/consts"
	"github.com/leighmacdonald/mika/geo"
	"github.com/leighmacdonald/mika/model"
	"github.com/leighmacdonald/mika/store"
	"github.com/leighmacdonald/mika/store/memory"
	log "github.com/sirupsen/logrus"
	"time"

	// Imported for side-effects for NewTestTracker
	_ "github.com/leighmacdonald/mika/store/memory"
	"sync"
)

// Tracker is the main application struct used to tie all the discreet components together
type Tracker struct {
	*sync.RWMutex
	// ctx is the master context used in the tracker, children contexts must use
	// this for their parent
	ctx      context.Context
	Torrents store.TorrentStore
	Peers    store.PeerStore
	Users    store.UserStore
	Geodb    geo.Provider
	// GeodbEnabled will enable the lookup of location data for peers
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
	Whitelist map[string]model.WhiteListClient
}

// Opts is used to configure tracker instances
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
	// How often we sync batch updates to backing stores
	BatchInterval time.Duration
	// MaxPeers is the max number of peers we send in an announce
	MaxPeers int
}

// NewDefaultOpts returns a new tracker configuration using in-memory
// stores and default interval values
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
	peerTimer := time.NewTimer(t.ReaperInterval)
	for {
		select {
		case <-peerTimer.C:
			t.Peers.Reap()
			// We use a timer here so that config updates for the interval get applied
			// on the next tick
			peerTimer.Reset(t.ReaperInterval)
		case <-t.ctx.Done():
			return
		}
	}
}

// StatWorker handles summing up stats for users/peers/torrents to be sent to the
// backing stores for long term storage.
// No locking required for these data sets
func (t *Tracker) StatWorker() {
	syncTimer := time.NewTimer(t.BatchInterval)
	userBatch := make(map[string]model.UserStats)
	peerBatch := make(map[model.PeerHash]model.PeerStats)
	torrentBatch := make(map[model.InfoHash]model.TorrentStats)
	for {
		select {
		case <-syncTimer.C:
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
			syncTimer.Reset(t.BatchInterval)
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
			var torrent model.Torrent
			// Keep deleted true so that we can record any buffered stat updates from
			// the client even though we deleted/disabled the torrent itself.
			if err := t.Torrents.Get(&torrent, u.InfoHash, true); err != nil {
				log.Errorf("No torrent found in batch update")
				continue
			}
			// Global user stats
			ub.Uploaded += uint64(float64(u.Uploaded) * torrent.MultiUp)
			ub.Downloaded += uint64(float64(u.Downloaded) * torrent.MultiDn)
			ub.Announces++

			// Peer stats
			pb.Downloaded += u.Downloaded
			pb.Uploaded += u.Uploaded
			pb.LastAnnounce = u.Timestamp
			pb.Left = u.Left
			pb.Announces++

			// Global torrent stats
			tb.Announces++
			tb.Uploaded += u.Uploaded
			tb.Downloaded += u.Downloaded

			switch u.Event {
			case consts.PAUSED:
				if !pb.Paused {
					tb.Seeders++
				}
			case consts.STARTED:
				if u.Left == 0 {
					tb.Seeders++
				} else {
					tb.Leechers++
				}
			case consts.COMPLETED:
				tb.Snatches++
				tb.Seeders++
				tb.Leechers--
			case consts.STOPPED:
				// Paused considered a seeder
				if u.Paused || u.Left == 0 {
					tb.Seeders--
				} else {
					tb.Leechers--
				}
				if err := t.Peers.Delete(u.InfoHash, u.PeerID); err != nil {
					log.Errorf("Could not remove peer from swarm: %s", err.Error())
				}
			}
			userBatch[u.Passkey] = ub
			torrentBatch[u.InfoHash] = tb
			peerBatch[pHash] = pb
		case <-t.ctx.Done():
			log.Debugf("Batch context closed")
			return
		}
	}
}

// New creates a new Tracker instance with configured backend stores
func New(ctx context.Context, opts *Opts) (*Tracker, error) {
	return &Tracker{
		RWMutex:          &sync.RWMutex{},
		ctx:              ctx,
		Torrents:         opts.Torrents,
		Peers:            opts.Peers,
		Users:            opts.Users,
		Geodb:            opts.Geodb,
		GeodbEnabled:     opts.GeodbEnabled,
		AllowNonRoutable: opts.AllowNonRoutable,
		ReaperInterval:   opts.ReaperInterval,
		AnnInterval:      opts.AnnInterval,
		AnnIntervalMin:   opts.AnnIntervalMin,
		BatchInterval:    opts.BatchInterval,
		MaxPeers:         opts.MaxPeers,
		StateUpdateChan:  make(chan model.UpdateState, 1000),
		Whitelist:        make(map[string]model.WhiteListClient),
	}, nil
}

// NewTestTracker sets up a tracker with fake data for testing
// This shouldn't really exist here, but its currently needed by other packages so its exported
func NewTestTracker() (*Tracker, error) {
	ctx := context.Background()
	userCount := 10
	torrentCount := 100
	opts := NewDefaultOpts()
	opts.BatchInterval = time.Millisecond * 100
	opts.ReaperInterval = time.Second * 10
	opts.AnnInterval = time.Second * 10
	opts.AnnIntervalMin = time.Second * 5
	opts.MaxPeers = 50
	opts.AllowNonRoutable = false
	tracker, err := New(ctx, opts)
	if err != nil {
		return nil, err
	}
	if err := tracker.LoadWhitelist(); err != nil {
		return nil, err
	}
	for i := 0; i < userCount; i++ {
		usr := store.GenerateTestUser()
		usr.Passkey = fmt.Sprintf("1234567890123456789%d", i)
		if err := tracker.Users.Add(usr); err != nil {
			return nil, err
		}
	}
	for i := 0; i < torrentCount; i++ {
		if err := tracker.Torrents.Add(store.GenerateTestTorrent()); err != nil {
			log.Panicf("Error adding torrent: %s", err.Error())
		}
	}
	return tracker, nil
}

// LoadWhitelist will read the client white list from the tracker store and
// load it into memory for quick lookups.
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
	t.Lock()
	t.Whitelist = whitelist
	t.Unlock()
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
