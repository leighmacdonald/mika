package store

import (
	"github.com/leighmacdonald/mika/consts"
	"github.com/leighmacdonald/mika/model"
	"sync"
)

// TorrentCache is a simple (dumb) in-memory cache for torrent items
// TODO replace with something that wont invoke so much GC overhead
type TorrentCache struct {
	*sync.RWMutex
	torrents map[model.InfoHash]model.Torrent
	enabled  bool
}

// NewTorrentCache configures and returns a new instance of TorrentCache
func NewTorrentCache(enabled bool) *TorrentCache {
	return &TorrentCache{
		RWMutex:  &sync.RWMutex{},
		torrents: make(map[model.InfoHash]model.Torrent),
		enabled:  enabled,
	}
}

// Add inserts a torrent into the cache
func (cache *TorrentCache) Add(t model.Torrent) {
	if !cache.enabled {
		return
	}
	cache.Lock()
	cache.torrents[t.InfoHash] = t
	cache.Unlock()
}

// Delete will mark a torrent as deleted in the backing store.
// If dropRow is true, it will permanently remove the torrent from the store
func (cache *TorrentCache) Delete(ih model.InfoHash, dropRow bool) {
	if !cache.enabled {
		return
	}
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
func (cache *TorrentCache) Get(torrent *model.Torrent, hash model.InfoHash) error {
	if !cache.enabled {
		return consts.ErrInvalidInfoHash
	}
	t, found := cache.torrents[hash]
	if !found {
		return consts.ErrInvalidInfoHash
	}
	*torrent = t
	return nil
}
