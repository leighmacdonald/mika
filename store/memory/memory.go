package memory

import (
	"mika/consts"
	"mika/model"
	"mika/store"
	"sync"
)

const (
	driverName = "memory"
)

type TorrentStore struct {
	sync.RWMutex
	torrents map[model.InfoHash]*model.Torrent
}

func (ts *TorrentStore) Close() error {
	return nil
}

func (ts *TorrentStore) GetTorrent(hash model.InfoHash) (*model.Torrent, error) {
	ts.RLock()
	t, found := ts.torrents[hash]
	ts.RUnlock()
	if !found || t.IsDeleted {
		return nil, consts.ErrInvalidInfoHash
	}
	return t, nil
}

// PeerStore is a memory backed store.PeerStore implementation
// TODO shard peer storage
type PeerStore struct {
	sync.RWMutex
	peers map[uint32][]*model.Peer
}

// Close flushes allocated memory
// TODO flush mem
func (ps *PeerStore) Close() error {
	return nil
}

func (ps *PeerStore) AddPeer(tid *model.Torrent, p *model.Peer) error {
	ps.Lock()
	ps.peers[tid.TorrentID] = append(ps.peers[tid.TorrentID], p)
	ps.Unlock()
	return nil
}

func (ps *PeerStore) UpdatePeer(_ *model.Torrent, _ *model.Peer) error {
	// no-op for in-memory
	return nil
}

func removePeer(peers []*model.Peer, p *model.Peer) []*model.Peer {
	for i := len(peers) - 1; i >= 0; i-- {
		if peers[i].UserPeerID == p.UserPeerID {
			return append(peers[:i], peers[i+1:]...)
		}
	}
	return peers
}

func (ps *PeerStore) DeletePeer(tid *model.Torrent, p *model.Peer) error {
	ps.Lock()
	ps.peers[tid.TorrentID] = removePeer(ps.peers[tid.TorrentID], p)
	ps.Unlock()
	return nil
}

func (ps *PeerStore) GetPeers(t *model.Torrent, limit int) ([]*model.Peer, error) {
	ps.RLock()
	p, found := ps.peers[t.TorrentID]
	ps.RUnlock()
	if !found {
		return nil, consts.ErrInvalidTorrentID
	}
	return p[0:limit], nil
}

func (ps *PeerStore) GetScrape(t *model.Torrent) {
	panic("implement me")
}

func (ts *TorrentStore) AddTorrent(t *model.Torrent) error {
	ts.RLock()
	_, found := ts.torrents[t.InfoHash]
	ts.RUnlock()
	if found {
		return consts.ErrDuplicate
	}
	ts.Lock()
	ts.torrents[t.InfoHash] = t
	ts.Unlock()
	return nil
}

func (ts *TorrentStore) DeleteTorrent(t *model.Torrent, _ bool) error {
	ts.Lock()
	delete(ts.torrents, t.InfoHash)
	ts.Unlock()
	return nil
}

type torrentDriver struct{}

func (td torrentDriver) NewTorrentStore(_ interface{}) (store.TorrentStore, error) {
	return &TorrentStore{
		sync.RWMutex{},
		map[model.InfoHash]*model.Torrent{},
	}, nil
}

type peerDriver struct{}

func (pd peerDriver) NewPeerStore(_ interface{}) (store.PeerStore, error) {
	return &PeerStore{
		sync.RWMutex{},
		map[uint32][]*model.Peer{},
	}, nil
}

func init() {
	store.AddPeerDriver(driverName, peerDriver{})
	store.AddTorrentDriver(driverName, torrentDriver{})
}
