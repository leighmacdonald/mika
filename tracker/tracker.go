package tracker

import (
	"context"
	"github.com/leighmacdonald/mika/config"
	"github.com/leighmacdonald/mika/consts"
	"github.com/leighmacdonald/mika/geo"
	"github.com/leighmacdonald/mika/store"
	"github.com/leighmacdonald/mika/util"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"sort"
	"sync/atomic"
	"time"

	// Imported for side-effects for NewTestTracker
	_ "github.com/leighmacdonald/mika/store/memory"
	"sync"
)

var (
	storeMu     *sync.RWMutex
	db          store.Store
	users       store.Users
	roles       store.Roles
	whitelist   store.WhiteList
	torrents    store.Torrents
	geodb       geo.Provider
	whitelistMu *sync.RWMutex
)

func init() {
	storeMu = &sync.RWMutex{}
	whitelist = make(store.WhiteList)
	whitelistMu = &sync.RWMutex{}
	memCfg := config.StoreConfig{Type: "memory"}
	ts, _ := store.NewStore(memCfg)
	db = ts
	torrents = make(store.Torrents)
	users = make(store.Users)
	roles = make(store.Roles)
}

func Init() {
	ts, err := store.NewStore(config.Store)
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

	whitelist = loadWhitelist()
	roles = loadRoles()
	users = loadUsers()
	torrents = loadTorrents()
}

func mapRoleToUser(u *store.User) {
	u.Role = roles[u.RoleID]
}

// loadWhitelist will read the client white list from the tracker store and
// load it into memory for quick lookups.
func loadWhitelist() store.WhiteList {
	newWhitelist := make(store.WhiteList)
	wl, err4 := db.WhiteListGetAll()
	if err4 != nil {
		log.Fatalf("whitelist empty, all clients are allowed")
	} else {
		for _, cw := range wl {
			newWhitelist[cw.ClientPrefix] = cw
		}
	}
	return newWhitelist
}

func loadRoles() store.Roles {
	roleSet, err := db.Roles()
	if err != nil {
		log.Fatalf("Failed to load roles")
	}
	return roleSet
}

func loadUsers() store.Users {
	us, err := db.Users()
	if err != nil {
		log.Fatalf("Failed to load users")
	}
	for _, u := range us {
		mapRoleToUser(u)
	}
	return us
}

func loadTorrents() store.Torrents {
	torrentSet, err := db.Torrents()
	if err != nil {
		log.Fatalf("Failed to load torrents")
	}
	for _, t := range torrentSet {
		t.Peers = store.NewSwarm()
	}
	return torrentSet
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

func findDirtyUsers(n int) ([]*store.User, error) {
	var sorted []*store.User
	for _, t := range users {
		sorted = append(sorted, t)
	}
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Writes < sorted[j].Writes
	})
	return sorted[0:util.Min(n, len(sorted))], nil
}

func findDirtyTorrents(n int) ([]*store.Torrent, error) {
	var sorted []*store.Torrent
	for _, t := range torrents {
		sorted = append(sorted, t)
	}
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Writes < sorted[j].Writes
	})
	return sorted[0:util.Min(n, len(sorted))], nil
}

// StatWorker handles summing up stats for users/peers/db to be sent to the
// backing stores for long term storage.
// No locking required for these data sets
func StatWorker(ctx context.Context) {
	syncTimer := time.NewTimer(config.Tracker.BatchUpdateIntervalParsed)
	for {
		select {
		case <-syncTimer.C:
			dUsers, err := findDirtyUsers(100)
			if err != nil {
				log.Errorf("Failed to fetch dirty users: %v", err)
				continue
			}
			if err2 := userSync(dUsers); err2 != nil {
				log.Errorf("Failed to sync dirty users: %v", err2)
				continue
			}

			dTorrents, err3 := findDirtyTorrents(100)
			if err3 != nil {
				log.Errorf("Failed to fetch dirty torrents: %v", err3)
				continue
			}
			if err4 := torrentSync(dTorrents); err4 != nil {
				log.Errorf("Failed to sync dirty torrents: %v", err4)
				continue
			}
			syncTimer.Reset(config.Tracker.BatchUpdateIntervalParsed)
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
//	if err := tracker.loadWhitelist(); err != nil {
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

func Migrate() error {
	return db.Migrate()
}

func ClientWhitelisted(peerID store.PeerID) bool {
	whitelistMu.RLock()
	_, found := whitelist[string(peerID[0:8])]
	whitelistMu.RUnlock()
	return found
}

func WhiteListAdd(wl *store.WhiteListClient) error {
	if err := db.WhiteListAdd(wl); err != nil {
		return errors.Wrap(err, "Failed to add new client whitelist")
	}
	whitelistMu.Lock()
	defer whitelistMu.Unlock()
	whitelist[wl.ClientPrefix] = wl
	return nil
}

func WhiteListGet(p string) (*store.WhiteListClient, error) {
	w, found := whitelist[p]
	if !found {
		return nil, consts.ErrInvalidClient
	}
	return w, nil
}

func WhiteListDelete(wl *store.WhiteListClient) error {
	if err := db.WhiteListDelete(wl); err != nil {
		return err
	}
	delete(whitelist, wl.ClientPrefix)
	return nil
}

func WhiteList() store.WhiteList {
	return whitelist
}

func Torrents() store.Torrents {
	return torrents
}

func TorrentAdd(torrent *store.Torrent) error {
	torrent.CreatedOn = util.Now()
	torrent.UpdatedOn = util.Now()
	if err := db.TorrentAdd(torrent); err != nil {
		return errors.Wrapf(err, "Failed to add torrent")
	}
	torrents[torrent.InfoHash] = torrent
	return nil
}

func TorrentGet(hash store.InfoHash, deletedOk bool) (*store.Torrent, error) {
	t, found := torrents[hash]
	if !found {
		return nil, consts.ErrInvalidInfoHash
	}
	if !deletedOk && t.IsDeleted {
		return nil, consts.ErrInvalidInfoHash
	}
	return t, nil
}

func TorrentDelete(torrent *store.Torrent) error {
	if err := db.TorrentDelete(torrent.InfoHash, true); err != nil {
		return err
	}
	torrent.IsDeleted = true
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

func torrentSync(batch []*store.Torrent) error {
	if err := db.TorrentSync(batch); err != nil {
		return err
	}
	for _, t := range batch {
		atomic.SwapUint32(&t.Writes, 0)
	}
	return nil
}

// GlobalStats holds basic stats for the running tracker
type GlobalStats struct {
}
