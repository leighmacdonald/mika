package redis

import (
	"fmt"
	"github.com/go-redis/redis/v7"
	"mika/consts"
	"mika/model"
	"mika/util"
	"sync"
)

type TorrentStore struct {
	client *redis.Client
}

func torrentKey(t model.InfoHash) string {
	return fmt.Sprintf("t:%s", t.String())
}

func (ts TorrentStore) AddTorrent(t *model.Torrent) error {
	err := ts.client.HSet(torrentKey(t.InfoHash), map[string]interface{}{
		"torrent_id":       t.TorrentID,
		"release_name":     t.ReleaseName,
		"total_completed":  t.TotalCompleted,
		"total_downloaded": t.TotalDownloaded,
		"total_uploaded":   t.TotalUploaded,
		"reason":           t.Reason,
		"multi_up":         t.MultiUp,
		"multi_dn":         t.MultiDn,
		"info_hash":        t.InfoHash.RawString(),
		"is_deleted":       t.IsDeleted,
		"is_enabled":       t.IsEnabled,
		"created_on":       util.TimeToString(t.CreatedOn),
		"updated_on":       util.TimeToString(t.UpdatedOn),
	}).Err()
	if err != nil {
		return err
	}
	return nil
}

func (ts TorrentStore) DeleteTorrent(t *model.Torrent, dropRow bool) error {
	if dropRow {
		return ts.client.Del(torrentKey(t.InfoHash)).Err()
	}
	return ts.client.HSet(torrentKey(t.InfoHash), "is_deleted", 1).Err()
}

func (ts TorrentStore) GetTorrent(hash model.InfoHash) (*model.Torrent, error) {
	v, err := ts.client.HGetAll(torrentKey(hash)).Result()
	if err != nil {
		return nil, err
	}
	_, found := v["info_hash"]
	if !found {
		return nil, consts.ErrInvalidInfoHash
	}
	t := model.Torrent{
		RWMutex:         sync.RWMutex{},
		TorrentID:       util.StringToUInt32(v["torrent_id"], 0),
		ReleaseName:     v["release_name"],
		InfoHash:        model.InfoHashFromString(v["info_hash"]),
		TotalCompleted:  util.StringToInt16(v["total_completed"], 0),
		TotalUploaded:   util.StringToUInt32(v["total_uploaded"], 0),
		TotalDownloaded: util.StringToUInt32(v["total_downloaded"], 0),
		IsDeleted:       util.StringToBool(v["is_deleted"], false),
		IsEnabled:       util.StringToBool(v["is_enabled"], false),
		Reason:          v["reason"],
		MultiUp:         util.StringToFloat64(v["multi_up"], 1.0),
		MultiDn:         util.StringToFloat64(v["multi_dn"], 1.0),
		CreatedOn:       util.StringToTime(v["created_on"]),
		UpdatedOn:       util.StringToTime(v["updated_on"]),
	}
	return &t, nil
}

func (ts TorrentStore) Close() error {
	return ts.client.Close()
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
		Addr:     fmt.Sprintf("%s:%d", host, port),
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
			Addr:     fmt.Sprintf("%s:%d", host, port),
			Password: password,
			DB:       db,
		})
	}
	return &PeerStore{
		client: c,
	}
}
