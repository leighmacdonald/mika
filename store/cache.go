package store

import (
	"sync"
)

type UserCache struct {
	*sync.RWMutex
	users map[string]User
}

// TorrentCache is a simple (dumb) in-memory cache for torrent items
type TorrentCache struct {
	*sync.RWMutex
	torrents map[InfoHash]Torrent
}

type PeerCache struct {
	*sync.RWMutex
	swarms map[InfoHash]Swarm
}

func NewUserCache() *UserCache {
	return &UserCache{
		RWMutex: &sync.RWMutex{},
		users:   make(map[string]User),
	}
}

// NewTorrentCache configures and returns a new instance of TorrentCache
func NewTorrentCache() *TorrentCache {
	return &TorrentCache{
		RWMutex:  &sync.RWMutex{},
		torrents: make(map[InfoHash]Torrent),
	}
}

func NewPeerCache() *PeerCache {
	return &PeerCache{
		RWMutex: &sync.RWMutex{},
		swarms:  map[InfoHash]Swarm{},
	}
}

// Add inserts a torrent into the cache
func (cache *TorrentCache) Set(t Torrent) {
	cache.Lock()
	cache.torrents[t.InfoHash] = t
	cache.Unlock()
}

func (cache *TorrentCache) Update(infoHash InfoHash, stats TorrentStats) {
	var t Torrent
	if !cache.Get(&t, infoHash) {
		return
	}
	t.Announces += stats.Announces
	t.Downloaded += stats.Downloaded
	t.Uploaded += stats.Uploaded
	t.Snatches += stats.Snatches
	t.Leechers += stats.Leechers
	t.Seeders += stats.Seeders
	cache.Set(t)
}

// Delete will mark a torrent as deleted in the backing store.
// If dropRow is true, it will permanently remove the torrent from the store
func (cache *TorrentCache) Delete(ih InfoHash, dropRow bool) {
	cache.Lock()
	defer cache.Unlock()
	if dropRow {
		delete(cache.torrents, ih)
	} else {
		t, found := cache.torrents[ih]
		if !found {
			return
		}
		t.IsDeleted = true
		cache.torrents[ih] = t
	}
}

// Get returns the Torrent matching the infohash.
// consts.ErrInvalidInfoHash is returned on failed lookup
func (cache *TorrentCache) Get(torrent *Torrent, hash InfoHash) bool {
	t, found := cache.torrents[hash]
	if !found {
		return false
	}
	*torrent = t
	return true
}

func (cache *UserCache) Set(user User) {
	cache.Lock()
	cache.users[user.Passkey] = user
	cache.Unlock()
}

func (cache *UserCache) Get(user *User, passkey string) bool {
	cache.RLock()
	u, found := cache.users[passkey]
	cache.RUnlock()
	if !found {
		return false
	}
	*user = u
	return true
}

func (cache *UserCache) Delete(passkey string) {
	cache.Lock()
	delete(cache.users, passkey)
	cache.Unlock()
}

func (cache *PeerCache) Set(infoHash InfoHash, peer Peer) {
	cache.RLock()
	_, found := cache.swarms[infoHash]
	cache.RUnlock()
	if !found {
		cache.swarms[infoHash] = NewSwarm()
	}
	cache.Lock()
	cache.swarms[infoHash].Add(peer)
	cache.Unlock()
}

func (cache *PeerCache) Get(peer *Peer, infoHash InfoHash, peerID PeerID) bool {
	cache.RLock()
	_, found := cache.swarms[infoHash]
	cache.RUnlock()
	if !found {
		return false
	}
	cache.RLock()
	if err := cache.swarms[infoHash].Get(peer, peerID); err != nil {
		cache.RUnlock()
		return false
	}
	cache.RUnlock()
	return true
}

func (cache *PeerCache) Delete(infoHash InfoHash, peerID PeerID) {
	cache.RLock()
	_, found := cache.swarms[infoHash]
	cache.RUnlock()
	if !found {
		return
	}
	cache.RLock()
	cache.swarms[infoHash].Remove(peerID)
	cache.RUnlock()
}
