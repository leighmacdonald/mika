package redis

import (
	"fmt"
	"github.com/go-redis/redis/v7"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"mika/consts"
	"mika/geo"
	"mika/model"
	"mika/util"
	"net"
	"sync"
)

type TorrentStore struct {
	client *redis.Client
}

func torrentKey(t model.InfoHash) string {
	return fmt.Sprintf("t:%s", t.String())
}

func torrentPeerPrefix(t model.InfoHash) string {
	return fmt.Sprintf("p:%s:*", t.String())
}

func peerKey(t model.InfoHash, p model.PeerID) string {
	return fmt.Sprintf("p:%s:%s", t.String(), p.String())
}

func (ts *TorrentStore) AddTorrent(t *model.Torrent) error {
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

func (ts *TorrentStore) DeleteTorrent(t *model.Torrent, dropRow bool) error {
	if dropRow {
		return ts.client.Del(torrentKey(t.InfoHash)).Err()
	}
	return ts.client.HSet(torrentKey(t.InfoHash), "is_deleted", 1).Err()
}

func (ts *TorrentStore) GetTorrent(hash model.InfoHash) (*model.Torrent, error) {
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

func (ts *TorrentStore) Close() error {
	return ts.client.Close()
}

type PeerStore struct {
	client *redis.Client
}

func (ps *PeerStore) AddPeer(t *model.Torrent, p *model.Peer) error {
	err := ps.client.HSet(peerKey(t.InfoHash, p.PeerId), map[string]interface{}{
		"user_peer_id":     p.UserPeerId,
		"speed_up":         p.SpeedUP,
		"speed_dn":         p.SpeedDN,
		"speed_up_max":     p.SpeedUPMax,
		"speed_dn_max":     p.SpeedDNMax,
		"total_uploaded":   p.Uploaded,
		"total_downloaded": p.Downloaded,
		"total_left":       p.Left,
		"total_announces":  p.Announces,
		"total_time":       p.TotalTime,
		"addr_ip":          p.IP.String(),
		"addr_port":        p.Port,
		"last_announce":    util.TimeToString(p.AnnounceLast),
		"first_announce":   util.TimeToString(p.AnnounceFirst),
		"peer_id":          p.PeerId.RawString(),
		"location":         p.Location.String(),
		"user_id":          p.UserId,
		"created_on":       util.TimeToString(p.CreatedOn),
		"updated_on":       util.TimeToString(p.UpdatedOn),
	}).Err()
	if err != nil {
		return errors.Wrap(err, "Failed to AddPeer")
	}
	return nil
}

func (ps *PeerStore) findKeys(prefix string) []string {
	v, err := ps.client.Keys(prefix).Result()
	if err != nil {
		log.Errorf("Failed to query for key prefix: %s", err.Error())
	}
	return v
}

func (ps *PeerStore) UpdatePeer(t *model.Torrent, p *model.Peer) error {
	err := ps.client.HSet(peerKey(t.InfoHash, p.PeerId), map[string]interface{}{
		"speed_up":         p.SpeedUP,
		"speed_dn":         p.SpeedDN,
		"speed_up_max":     p.SpeedUPMax,
		"speed_dn_max":     p.SpeedDNMax,
		"total_uploaded":   p.Uploaded,
		"total_downloaded": p.Downloaded,
		"total_left":       p.Left,
		"total_announces":  p.Announces,
		"total_time":       p.TotalTime,
		"last_announce":    util.TimeToString(p.AnnounceLast),
		"first_announce":   util.TimeToString(p.AnnounceFirst),
		"updated_on":       util.TimeToString(p.UpdatedOn),
	}).Err()
	if err != nil {
		return errors.Wrap(err, "Failed to UpdatePeer")
	}
	return nil
}

func (ps *PeerStore) DeletePeer(t *model.Torrent, p *model.Peer) error {
	return ps.client.Del(peerKey(t.InfoHash, p.PeerId)).Err()
}

func (ps *PeerStore) GetPeers(t *model.Torrent, limit int) ([]*model.Peer, error) {
	var peers []*model.Peer
	for i, key := range ps.findKeys(torrentPeerPrefix(t.InfoHash)) {
		v, err := ps.client.HGetAll(key).Result()
		if err != nil {
			return nil, errors.Wrap(err, "Error trying to GetPeers")
		}
		p := &model.Peer{
			UserPeerId:    util.StringToUInt32(v["user_peer_id"], 0),
			SpeedUP:       util.StringToUInt32(v["speed_up"], 0),
			SpeedDN:       util.StringToUInt32(v["speed_dn"], 0),
			SpeedUPMax:    util.StringToUInt32(v["speed_dn_max"], 0),
			SpeedDNMax:    util.StringToUInt32(v["speed_up_max"], 0),
			Uploaded:      util.StringToUInt32(v["total_uploaded"], 0),
			Downloaded:    util.StringToUInt32(v["total_downloaded"], 0),
			Left:          util.StringToUInt32(v["total_left"], 0),
			Announces:     util.StringToUInt32(v["total_announces"], 0),
			TotalTime:     util.StringToUInt32(v["total_time"], 0),
			IP:            net.ParseIP(v["addr_ip"]),
			Port:          util.StringToUInt16(v["addr_port"], 0),
			AnnounceLast:  util.StringToTime(v["last_announce"]),
			AnnounceFirst: util.StringToTime(v["first_announce"]),
			PeerId:        model.PeerIDFromString(v["peer_id"]),
			Location:      geo.LatLongFromString(v["location"]),
			UserId:        util.StringToUInt32(v["user_id"], 0),
			CreatedOn:     util.StringToTime(v["created_on"]),
			UpdatedOn:     util.StringToTime(v["updated_on"]),
		}
		peers = append(peers, p)
		if i >= limit {
			break
		}
	}
	return peers, nil
}

func (ps *PeerStore) GetScrape(t *model.Torrent) {
	panic("implement me")
}

func (ps *PeerStore) Close() error {
	return ps.client.Close()
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
