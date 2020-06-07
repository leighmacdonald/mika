package tracker

import (
	"context"
	"fmt"
	"github.com/leighmacdonald/mika/consts"
	"github.com/leighmacdonald/mika/geo"
	"github.com/leighmacdonald/mika/store"
	"github.com/leighmacdonald/mika/store/memory"
	"github.com/leighmacdonald/mika/util"
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
	ctx context.Context

	torrents      store.TorrentStore
	TorrentsCache *store.TorrentCache

	users      store.UserStore
	UsersCache *store.UserCache

	peers     store.PeerStore
	PeerCache *store.PeerCache

	Geodb geo.Provider
	// GeodbEnabled will enable the lookup of location data for peers
	GeodbEnabled bool
	// Public if true means we dont require a passkey / authorized user
	Public bool
	// If Public is true, this will allow unknown info_hashes to be automatically tracked
	AutoRegister     bool
	AllowNonRoutable bool
	AllowClientIP    bool
	// ReaperInterval is how often we can for dead peers in swarms
	ReaperInterval time.Duration
	AnnInterval    time.Duration
	AnnIntervalMin time.Duration
	BatchInterval  time.Duration
	IPv6Only       bool
	// MaxPeers is the max number of peers we send in an announce
	MaxPeers        int
	StateUpdateChan chan store.UpdateState
	// Whitelist and whitelist lock
	Whitelist map[string]store.WhiteListClient
}

// Opts is used to configure tracker instances
type Opts struct {
	Torrents            store.TorrentStore
	Peers               store.PeerStore
	Users               store.UserStore
	TorrentCacheEnabled bool
	UserCacheEnabled    bool
	PeerCacheEnabled    bool
	Geodb               geo.Provider
	// GeodbEnabled will enable the lookup of location data for peers
	// TODO the dummy provider is probably sufficient
	GeodbEnabled bool
	// Public if true means we dont require a passkey / authorized user
	Public bool
	// If Public is true, this will allow unknown info_hashes to be automatically tracked
	AutoRegister     bool
	AllowNonRoutable bool
	AllowClientIP    bool
	// Dont enable dual-stack replies in ipv6 mode
	IPv6Only bool
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
		Torrents:            memory.NewTorrentStore(),
		Peers:               memory.NewPeerStore(),
		Users:               memory.NewUserStore(),
		UserCacheEnabled:    false,
		TorrentCacheEnabled: false,
		PeerCacheEnabled:    false,
		Geodb:               &geo.DummyProvider{},
		GeodbEnabled:        false,
		Public:              false,
		AutoRegister:        false,
		AllowNonRoutable:    false,
		AllowClientIP:       false,
		IPv6Only:            false,
		ReaperInterval:      time.Second * 300,
		AnnInterval:         time.Second * 60,
		AnnIntervalMin:      time.Second * 30,
		BatchInterval:       time.Second * 60,
		MaxPeers:            100,
	}
}

// PeerReaper will call the store.PeerStore.Reap() function periodically. This is
// used to clean peers that have not announced in a while from the swarm.
func (t *Tracker) PeerReaper() {
	peerTimer := time.NewTimer(t.ReaperInterval)
	for {
		select {
		case <-peerTimer.C:
			expired := t.peers.Reap()
			if t.PeerCache != nil {
				for _, ph := range expired {
					t.PeerCache.Delete(ph.InfoHash(), ph.PeerID())
				}
			}
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
	userBatch := make(map[string]store.UserStats)
	peerBatch := make(map[store.PeerHash]store.PeerStats)
	torrentBatch := make(map[store.InfoHash]store.TorrentStats)
	for {
		select {
		case <-syncTimer.C:
			// Copy the maps to pass into the go routine call. At the same time deleting
			// the existing values
			userBatchCopy := make(map[string]store.UserStats)
			for k, v := range userBatch {
				userBatchCopy[k] = v
				delete(userBatch, k)
			}

			peerBatchCopy := make(map[store.PeerHash]store.PeerStats)
			for k, v := range peerBatch {
				peerBatchCopy[k] = v
				delete(peerBatch, k)
			}

			torrentBatchCopy := make(map[store.InfoHash]store.TorrentStats)
			for k, v := range torrentBatch {
				torrentBatchCopy[k] = v
				delete(torrentBatch, k)
			}
			// Send current copies of data to stores
			log.Debugf("Calling Sync() on %d users", len(userBatchCopy))
			if err := t.UserSync(userBatchCopy); err != nil {
				log.Errorf(err.Error())
			}
			log.Debugf("Calling Sync() on %d peers", len(userBatchCopy))
			if err := t.PeerSync(peerBatchCopy); err != nil {
				log.Errorf(err.Error())
			}
			log.Debugf("Calling Sync() on %d torrents", len(userBatchCopy))
			if err := t.TorrentSync(torrentBatchCopy); err != nil {
				log.Errorf(err.Error())
			}
			syncTimer.Reset(t.BatchInterval)
		case u := <-t.StateUpdateChan:
			ub, found := userBatch[u.Passkey]
			if !found {
				ub = store.UserStats{}
			}
			tb, found := torrentBatch[u.InfoHash]
			if !found {
				tb = store.TorrentStats{}
			}
			pHash := store.NewPeerHash(u.InfoHash, u.PeerID)
			pb, peerFound := peerBatch[pHash]
			if !peerFound {
				pb = store.PeerStats{}
			}
			var torrent store.Torrent
			// Keep deleted true so that we can record any buffered stat updates from
			// the client even though we deleted/disabled the torrent itself.
			if err := t.torrents.Get(&torrent, u.InfoHash, false); err != nil {
				log.Errorf("No torrent found in batch update")
				continue
			}
			// Global user stats
			ub.Uploaded += uint64(float64(u.Uploaded) * torrent.MultiUp)
			ub.Downloaded += uint64(float64(u.Downloaded) * torrent.MultiDn)
			ub.Announces++

			// Peer stats
			pb.Hist = append(pb.Hist, store.AnnounceHist{
				Downloaded: u.Downloaded,
				Uploaded:   u.Uploaded,
				Timestamp:  u.Timestamp,
			})
			pb.Left = u.Left

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
				if err := t.peers.Delete(u.InfoHash, u.PeerID); err != nil {
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
	t := &Tracker{
		RWMutex:          &sync.RWMutex{},
		ctx:              ctx,
		torrents:         opts.Torrents,
		peers:            opts.Peers,
		users:            opts.Users,
		Geodb:            opts.Geodb,
		GeodbEnabled:     opts.GeodbEnabled,
		AllowNonRoutable: opts.AllowNonRoutable,
		AllowClientIP:    opts.AllowClientIP,
		IPv6Only:         opts.IPv6Only,
		AutoRegister:     opts.AutoRegister,
		ReaperInterval:   opts.ReaperInterval,
		AnnInterval:      opts.AnnInterval,
		AnnIntervalMin:   opts.AnnIntervalMin,
		BatchInterval:    opts.BatchInterval,
		MaxPeers:         opts.MaxPeers,
		StateUpdateChan:  make(chan store.UpdateState, 1000),
		Whitelist:        make(map[string]store.WhiteListClient),
	}
	// Don't enable caching if we are already configured for a memory store.
	if opts.TorrentCacheEnabled {
		switch t.torrents.(type) {
		case *memory.TorrentStore:
			log.Warnf("Not enabling cache for in-memory torrent store, already in-memory")
		default:
			t.TorrentsCache = store.NewTorrentCache()
		}
	}
	if opts.UserCacheEnabled {
		switch t.users.(type) {
		case *memory.UserStore:
			log.Warnf("Not enabling cache for in-memory user store, already in-memory.")
		default:
			t.UsersCache = store.NewUserCache()
		}
	}
	if opts.PeerCacheEnabled {
		switch t.peers.(type) {
		case *memory.PeerStore:
			log.Warnf("Not enabling cache for in-memory peer store, already in-memory.")
		default:
			t.PeerCache = store.NewPeerCache()
		}
	}
	return t, nil
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
	opts.AutoRegister = false
	opts.MaxPeers = 50
	opts.AllowNonRoutable = false
	opts.AllowClientIP = true
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
		if err := tracker.users.Add(usr); err != nil {
			return nil, err
		}
	}
	for i := 0; i < torrentCount; i++ {
		if err := tracker.torrents.Add(store.GenerateTestTorrent()); err != nil {
			log.Panicf("Error adding torrent: %s", err.Error())
		}
	}
	return tracker, nil
}

// LoadWhitelist will read the client white list from the tracker store and
// load it into memory for quick lookups.
func (t *Tracker) LoadWhitelist() error {
	whitelist := make(map[string]store.WhiteListClient)
	wl, err4 := t.torrents.WhiteListGetAll()
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
func (t *Tracker) TorrentAdd(torrent store.Torrent) error {
	return t.torrents.Add(torrent)
}

func (t *Tracker) TorrentGet(torrent *store.Torrent, hash store.InfoHash, deletedOk bool) error {
	cached := false
	if t.TorrentsCache != nil {
		cached = t.TorrentsCache.Get(torrent, hash)
		if cached {
			if torrent.IsDeleted && !deletedOk {
				return consts.ErrInvalidInfoHash
			}
			return nil
		}
	}
	if err := t.torrents.Get(torrent, hash, deletedOk); err != nil {
		return err
	}
	if t.TorrentsCache != nil && !cached {
		t.TorrentsCache.Set(*torrent)
	}
	return nil
}

func (t *Tracker) UserGet(user *store.User, passkey string) error {
	cached := false
	if t.UsersCache != nil {
		cached = t.UsersCache.Get(user, passkey)
		if cached {
			return nil
		}
	}
	if err := t.users.GetByPasskey(user, passkey); err != nil {
		return err
	}
	if t.UsersCache != nil && !cached {
		t.UsersCache.Set(*user)
	}
	return nil
}

func (t *Tracker) UserAdd(user store.User) error {
	if err := t.users.Add(user); err != nil {
		return err
	}
	if t.UsersCache != nil {
		t.UsersCache.Set(user)
	}
	return nil
}

func (t *Tracker) PeerGet(peer *store.Peer, infoHash store.InfoHash, peerID store.PeerID) error {
	if t.PeerCache != nil && t.PeerCache.Get(peer, infoHash, peerID) {
		return nil
	}
	if err := t.peers.Get(peer, infoHash, peerID); err != nil {
		return err
	}
	if t.PeerCache != nil {
		t.PeerCache.Set(infoHash, *peer)
	}
	return nil
}

func (t *Tracker) PeerGetN(infoHash store.InfoHash, max int) (store.Swarm, error) {
	swarm, err := t.peers.GetN(infoHash, max)
	if err != nil {
		return store.Swarm{}, err
	}
	return swarm, nil
}

func (t *Tracker) PeerAdd(infoHash store.InfoHash, peer store.Peer) error {
	if err := t.peers.Add(infoHash, peer); err != nil {
		return err
	}
	if t.PeerCache != nil {
		t.PeerCache.Set(infoHash, peer)
	}
	return nil
}

func (t *Tracker) UserSync(batch map[string]store.UserStats) error {
	if err := t.users.Sync(batch); err != nil {
		return err
	}
	if t.UsersCache != nil {
		var usr store.User
		for passkey, stats := range batch {
			if t.UsersCache.Get(&usr, passkey) {
				usr.Downloaded += stats.Downloaded
				usr.Uploaded += stats.Uploaded
				usr.Announces += stats.Announces
				t.UsersCache.Set(usr)
			}
		}
	}
	return nil
}

func (t *Tracker) PeerSync(batch map[store.PeerHash]store.PeerStats) error {
	if err := t.peers.Sync(batch); err != nil {
		return err
	}
	if t.PeerCache != nil {
		var peer store.Peer
		for ph, stats := range batch {
			sum := stats.Totals()
			if t.PeerCache.Get(&peer, ph.InfoHash(), ph.PeerID()) {
				peer.Downloaded += sum.TotalDn
				peer.Uploaded += sum.TotalUp
				peer.SpeedDN = uint32(sum.SpeedDn)
				peer.SpeedUP = uint32(sum.SpeedUp)
				peer.SpeedDNMax = util.UMax32(peer.SpeedDNMax, uint32(sum.SpeedDn))
				peer.SpeedUPMax = util.UMax32(peer.SpeedUPMax, uint32(sum.SpeedUp))
				t.PeerCache.Set(ph.InfoHash(), peer)
			}
		}
	}
	return nil
}

func (t *Tracker) TorrentSync(batch map[store.InfoHash]store.TorrentStats) error {
	if err := t.torrents.Sync(batch); err != nil {
		return err
	}
	if t.TorrentsCache != nil {
		for ih, stats := range batch {
			t.TorrentsCache.Update(ih, stats)
		}
	}
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
