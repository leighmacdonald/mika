package redis

import (
	"fmt"
	"github.com/go-redis/redis/v7"
	"mika/model"
)

type TorrentStore struct {
	client *redis.Client
}

func (t2 TorrentStore) AddTorrent(t *model.Torrent) error {
	panic("implement me")
}

func (t2 TorrentStore) DeleteTorrent(t *model.Torrent, dropRow bool) error {
	panic("implement me")
}

func (t2 TorrentStore) GetTorrent(hash model.InfoHash) (*model.Torrent, error) {
	panic("implement me")
}

func (t2 TorrentStore) Close() error {
	panic("implement me")
}

type PeerStore struct {
	client *redis.Client
}

func (p2 PeerStore) AddPeer(tid *model.Torrent, p *model.Peer) error {
	panic("implement me")
}

func (p2 PeerStore) UpdatePeer(tid *model.Torrent, p *model.Peer) error {
	panic("implement me")
}

func (p2 PeerStore) DeletePeer(tid *model.Torrent, p *model.Peer) error {
	panic("implement me")
}

func (p2 PeerStore) GetPeers(t *model.Torrent) ([]*model.Peer, error) {
	panic("implement me")
}

func (p2 PeerStore) GetScrape(t *model.Torrent) {
	panic("implement me")
}

func (p2 PeerStore) Close() error {
	panic("implement me")
}

func NewTorrentStore(host string, port int, password string, db int) *TorrentStore {
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", host, port),
		Password: password,
		DB:       db,
	})
	return &TorrentStore{
		client: client,
	}
}

// NewPeerStore will create a new mysql backed peer store
// If existingConn is defined, it will be used instead of establishing a new connection
func NewPeerStore(host string, port int, password string, db int, existingConn *redis.Client) *PeerStore {
	var c *redis.Client
	if existingConn != nil {
		c = existingConn
	} else {
		c = redis.NewClient(&redis.Options{
			Addr:     fmt.Sprintf("%s:%s", host, port),
			Password: password,
			DB:       db,
		})
	}
	return &PeerStore{
		client: c,
	}
}
