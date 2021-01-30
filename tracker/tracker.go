package tracker

import (
	"context"
	"github.com/leighmacdonald/mika/config"
	"github.com/leighmacdonald/mika/consts"
	"github.com/leighmacdonald/mika/geo"
	"github.com/leighmacdonald/mika/store"
	log "github.com/sirupsen/logrus"
	"time"

	// Imported for side-effects for NewTestTracker
	_ "github.com/leighmacdonald/mika/store/memory"
	"sync"
)

var (
	storeMu         *sync.RWMutex
	db              store.Store
	geodb           geo.Provider
	torrents        map[store.InfoHash]*store.Torrent
	users           map[string]*store.User
	StateUpdateChan chan store.UpdateState
	// whitelist and whitelist lock
	whitelist   map[string]store.WhiteListClient
	whitelistMu *sync.RWMutex
)

func init() {
	storeMu = &sync.RWMutex{}
	StateUpdateChan = make(chan store.UpdateState, 1000)
	whitelist = make(map[string]store.WhiteListClient)
	whitelistMu = &sync.RWMutex{}

	memCfg := config.StoreConfig{Type: "memory"}
	ts, _ := store.NewStore(memCfg)
	db = ts
	torrents = make(map[store.InfoHash]*store.Torrent)
	users = make(map[string]*store.User)
}

func Init() {
	ts, err := store.NewStore(config.TorrentStore)
	if err != nil {
		log.Fatalf("Failed to setup torrent store: %s", err)
	}
	storeMu.Lock()
	db = ts
	storeMu.Unlock()
	var newGeodb geo.Provider
	if config.GeoDB.Enabled {
		newGeodb, err = geo.New(config.GeoDB.Path)
		if err != nil {
			log.Fatalf("Could not validate geo database. You may need to run ./mika updategeo")
		}
	} else {
		newGeodb = &geo.DummyProvider{}
	}
	geodb = newGeodb

	_ = LoadWhitelist()
}

// PeerReaper will call the store.PeerStore.Reap() function periodically. This is
// used to clean peers that have not announced in a while from the swarm.
func PeerReaper(ctx context.Context) {
	peerTimer := time.NewTimer(config.Tracker.ReaperIntervalParsed)
	for {
		select {
		case <-peerTimer.C:
			// TODO FIXME
			//expired := peers.Reap()
			//if PeerCache != nil {
			//	for _, ph := range expired {
			//		PeerCache.Delete(ph.InfoHash(), ph.PeerID())
			//	}
			//}
			// We use a timer here so that config updates for the interval get applied
			// on the next tick
			peerTimer.Reset(config.Tracker.ReaperIntervalParsed)
		case <-ctx.Done():
			return
		}
	}
}

// StatWorker handles summing up stats for users/peers/db to be sent to the
// backing stores for long term storage.
// No locking required for these data sets
func StatWorker(ctx context.Context) {
	syncTimer := time.NewTimer(config.Tracker.BatchUpdateIntervalParsed)
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
			if err := UserSync(userBatchCopy); err != nil {
				log.Errorf(err.Error())
			}
			//log.Debugf("Calling Sync() on %d peers", len(userBatchCopy))
			//if err := PeerSync(peerBatchCopy); err != nil {
			//	log.Errorf(err.Error())
			//}
			log.Debugf("Calling Sync() on %d db", len(userBatchCopy))
			if err := TorrentSync(torrentBatchCopy); err != nil {
				log.Errorf(err.Error())
			}
			syncTimer.Reset(config.Tracker.BatchUpdateIntervalParsed)
		case u := <-StateUpdateChan:
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
			if err := TorrentGet(&torrent, u.InfoHash, false); err != nil {
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
				if err := peerDelete(u.InfoHash, u.PeerID); err != nil {
					log.Errorf("Could not remove peer from swarm: %s", err.Error())
				}
			}
			userBatch[u.Passkey] = ub
			torrentBatch[u.InfoHash] = tb
			peerBatch[pHash] = pb
		case <-ctx.Done():
			log.Debugf("Batch context closed")
			return
		}
	}
}

//// NewTestTracker sets up a tracker with fake data for testing
//// This shouldn't really exist here, but its currently needed by other packages so its exported
//func NewTestTracker() (*Tracker, error) {
//	ctx := context.Background()
//	userCount := 10
//	torrentCount := 100
//	opts := NewDefaultOpts()
//	opts.BatchInterval = time.Millisecond * 100
//	opts.ReaperInterval = time.Second * 10
//	opts.AnnInterval = time.Second * 10
//	opts.AnnIntervalMin = time.Second * 5
//	opts.AutoRegister = false
//	opts.MaxPeers = 50
//	opts.AllowNonRoutable = false
//	opts.AllowClientIP = true
//	tracker, err := New(ctx, opts)
//	if err != nil {
//		return nil, err
//	}
//	if err := tracker.LoadWhitelist(); err != nil {
//		return nil, err
//	}
//	for i := 0; i < userCount; i++ {
//		usr := store.GenerateTestUser()
//		usr.Passkey = fmt.Sprintf("1234567890123456789%d", i)
//		if err := tracker.users.Add(usr); err != nil {
//			return nil, err
//		}
//	}
//	for i := 0; i < torrentCount; i++ {
//		if err := tracker.db.Add(store.GenerateTestTorrent()); err != nil {
//			log.Panicf("Error adding torrent: %s", err.Error())
//		}
//	}
//	return tracker, nil
//}

func ClientWhitelisted(peerID store.PeerID) bool {
	whitelistMu.RLock()
	_, found := whitelist[string(peerID[0:8])]
	whitelistMu.RUnlock()
	return found
}

// LoadWhitelist will read the client white list from the tracker store and
// load it into memory for quick lookups.
func LoadWhitelist() error {
	newWhitelist := make(map[string]store.WhiteListClient)
	wl, err4 := db.WhiteListGetAll()
	if err4 != nil {
		log.Warnf("whitelist empty, all clients are allowed")
	} else {
		for _, cw := range wl {
			newWhitelist[cw.ClientPrefix] = cw
		}
	}
	whitelistMu.Lock()
	whitelist = newWhitelist
	whitelistMu.Unlock()
	return nil
}
func TorrentAdd(torrent *store.Torrent) error {
	return db.TorrentAdd(torrent)
}

func TorrentGet(torrent *store.Torrent, hash store.InfoHash, deletedOk bool) error {
	// TODO
	if err := db.TorrentGet(torrent, hash, deletedOk); err != nil {
		return err
	}
	return nil
}

func UserGet(user *store.User, passkey string) error {
	// TODO
	if err := db.UserGetByPasskey(user, passkey); err != nil {
		return err
	}
	return nil
}

func UserAdd(user *store.User) error {
	if err := db.UserAdd(user); err != nil {
		return err
	}
	return nil
}

func PeerGet(infoHash store.InfoHash, peerID store.PeerID) (*store.Peer, error) {
	// TODO
	tor, found := torrents[infoHash]
	if !found {
		return nil, consts.ErrInvalidInfoHash
	}
	p, err := tor.Peers.Get(peerID)
	if err != nil {
		return nil, consts.ErrInvalidPeerID
	}
	return p, consts.ErrInvalidPeer
}

func PeerGetN(infoHash store.InfoHash, max int) (*store.Swarm, error) {
	_, found := torrents[infoHash]
	if !found {
		return nil, consts.ErrInvalidInfoHash
	}
	swarm := &store.Swarm{}
	// TODO get swarm
	return swarm, nil
}

func PeerAdd(infoHash store.InfoHash, peer *store.Peer) error {
	// TODO
	//if err := peers.Add(infoHash, peer); err != nil {
	//	return err
	//}
	return nil
}

func peerDelete(infoHash store.InfoHash, peerID store.PeerID) error {
	// TODO
	//PeerCache.Delete(infoHash, peerID)
	//return peers.Delete(infoHash, peerID)
	return nil
}

func UserSync(batch map[string]store.UserStats) error {
	if err := db.UserSync(batch); err != nil {
		return err
	}
	return nil
}

//func PeerSync(batch map[store.PeerHash]store.PeerStats) error {
//	if err := peers.Sync(batch); err != nil {
//		return err
//	}
//	if PeerCache != nil {
//		var peer store.Peer
//		for ph, stats := range batch {
//			sum := stats.Totals()
//			if PeerCache.Get(&peer, ph.InfoHash(), ph.PeerID()) {
//				peer.Downloaded += sum.TotalDn
//				peer.Uploaded += sum.TotalUp
//				peer.SpeedDN = uint32(sum.SpeedDn)
//				peer.SpeedUP = uint32(sum.SpeedUp)
//				peer.SpeedDNMax = util.UMax32(peer.SpeedDNMax, uint32(sum.SpeedDn))
//				peer.SpeedUPMax = util.UMax32(peer.SpeedUPMax, uint32(sum.SpeedUp))
//				PeerCache.Set(ph.InfoHash(), peer)
//			}
//		}
//	}
//	return nil
//}

func TorrentSync(batch map[store.InfoHash]store.TorrentStats) error {
	if err := db.TorrentSync(batch); err != nil {
		return err
	}
	return nil
}

// Stats returns the current cumulative stats for the tracker
func Stats() GlobalStats {
	var s GlobalStats

	return s
}

// GlobalStats holds basic stats for the running tracker
type GlobalStats struct {
}
