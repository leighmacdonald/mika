package redis

import (
	"fmt"
	"github.com/go-redis/redis/v7"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"mika/config"
	"mika/consts"
	"mika/geo"
	"mika/model"
	"mika/store"
	"mika/util"
	"net"
	"strconv"
	"sync"
)

const (
	driverName = "redis"
	clientName = "mika"
)

func torrentKey(t model.InfoHash) string {
	return fmt.Sprintf("t:%s", t.String())
}

func torrentPeersKey(t model.InfoHash) string {
	return fmt.Sprintf("p:%s:*", t.String())
}

func peerKey(t model.InfoHash, p model.PeerID) string {
	return fmt.Sprintf("p:%s:%s", t.String(), p.String())
}

func userKey(passkey string) string {
	return fmt.Sprintf("u:%s", passkey)
}

func userIDKey(userID uint32) string {
	return fmt.Sprintf("uid_pk:%d", userID)
}

// UserStore is the redis backed store.TorrentStore implementation
type UserStore struct {
	client *redis.Client
}

// AddUser inserts a user into redis via at the string provided by the userKey function
// This additionally sets the passkey->user_id mapping
func (us UserStore) AddUser(u *model.User) error {
	pipe := us.client.TxPipeline()
	pipe.HSet(userKey(u.Passkey), map[string]interface{}{
		"user_id":          u.UserID,
		"passkey":          u.Passkey,
		"download_enabled": true,
		"is_deleted":       false,
	})
	pipe.Set(userIDKey(u.UserID), u.Passkey, 0)
	if _, err := pipe.Exec(); err != nil {
		return errors.Wrap(err, "Failed to add user to store")
	}
	return nil
}

// GetUserByPasskey returns the hash values set of the passkey and maps it to a User struct
func (us UserStore) GetUserByPasskey(passkey string) (*model.User, error) {
	v, err := us.client.HGetAll(userKey(passkey)).Result()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to retrieve user by passkey")
	}
	var user model.User
	user.Passkey = v["passkey"]
	user.UserID = util.StringToUInt32(v["user_id"], 0)
	if !user.Valid() {
		return nil, consts.ErrInvalidState
	}
	return &user, nil
}

// GetUserByID will query the passkey:user_id index for the passkey and return the matching user
func (us UserStore) GetUserByID(userID uint32) (*model.User, error) {
	passkey, err := us.client.Get(userIDKey(userID)).Result()
	if err != nil {
		log.Warnf("Failed to lookup user by ID, no passkey mapped: %d", userID)
		return nil, consts.ErrInvalidUser
	}
	if passkey == "" {
		return nil, consts.ErrInvalidUser
	}
	return us.GetUserByPasskey(passkey)
}

// DeleteUser drops a user from redis.
// TODO Needs to also drop the index value
func (us UserStore) DeleteUser(user *model.User) error {
	if err := us.client.Del(userKey(user.Passkey)).Err(); err != nil {
		return errors.Wrap(err, "Could not remove torrent from store")
	}
	return nil
}

// Close will shutdown the underlying redis connection
func (us UserStore) Close() error {
	return us.client.Close()
}

// TorrentStore is the redis backed store.TorrentStore implementation
type TorrentStore struct {
	client *redis.Client
}

func (ts *TorrentStore) WhiteListDel(client model.WhiteListClient) error {
	panic("implement me")
}

func (ts *TorrentStore) WhiteListAdd(client model.WhiteListClient) error {
	panic("implement me")
}

func (ts *TorrentStore) WhiteListGetAll() ([]model.WhiteListClient, error) {
	panic("implement me")
}

// AddTorrent adds a new torrent to the redis backing store
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

// DeleteTorrent will mark a torrent as deleted in the backing store.
// If dropRow is true, it will permanently remove the torrent from the store
func (ts *TorrentStore) DeleteTorrent(ih model.InfoHash, dropRow bool) error {
	if dropRow {
		if err := ts.client.Del(torrentKey(ih)).Err(); err != nil {
			return errors.Wrap(err, "Could not remove torrent from store")
		}
		return nil
	}
	if err := ts.client.HSet(torrentKey(ih), "is_deleted", 1).Err(); err != nil {
		return errors.Wrap(err, "Could not mark torrent as deleted")
	}
	return nil
}

// GetTorrent returns the Torrent matching the infohash
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

// Close will close the underlying redis client and clear the caches
func (ts *TorrentStore) Close() error {
	return ts.client.Close()
}

// PeerStore is the redis backed store.PeerStore implementation
type PeerStore struct {
	client *redis.Client
}

// AddPeer inserts a peer into the active swarm for the torrent provided
func (ps *PeerStore) AddPeer(ih model.InfoHash, p *model.Peer) error {
	err := ps.client.HSet(peerKey(ih, p.PeerID), map[string]interface{}{
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
		"peer_id":          p.PeerID.RawString(),
		"location":         p.Location.String(),
		"user_id":          p.UserID,
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

// UpdatePeer will sync any new peer data with the backing store
func (ps *PeerStore) UpdatePeer(ih model.InfoHash, p *model.Peer) error {
	err := ps.client.HSet(peerKey(ih, p.PeerID), map[string]interface{}{
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

// DeletePeer will remove a user from a torrents swarm
func (ps *PeerStore) DeletePeer(ih model.InfoHash, p *model.Peer) error {
	return ps.client.Del(peerKey(ih, p.PeerID)).Err()
}

// GetPeer will fetch the peer from the swarm if it exists
func (ps *PeerStore) GetPeer(ih model.InfoHash, peerID model.PeerID) (*model.Peer, error) {
	v, err := ps.client.HGetAll(peerKey(ih, peerID)).Result()
	if err != nil {
		return nil, err
	}
	p := mapPeerValues(v)
	if !p.Valid() {
		return nil, consts.ErrInvalidState
	}
	return &p, nil
}

func mapPeerValues(v map[string]string) model.Peer {
	return model.Peer{
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
		PeerID:        model.PeerIDFromString(v["peer_id"]),
		Location:      geo.LatLongFromString(v["location"]),
		UserID:        util.StringToUInt32(v["user_id"], 0),
		CreatedOn:     util.StringToTime(v["created_on"]),
		UpdatedOn:     util.StringToTime(v["updated_on"]),
	}
}

// GetPeers will fetch peers for a torrents active swarm up to N users
func (ps *PeerStore) GetPeers(ih model.InfoHash, limit int) (model.Swarm, error) {
	var peers []*model.Peer
	for i, key := range ps.findKeys(torrentPeersKey(ih)) {
		if i == limit {
			break
		}
		v, err := ps.client.HGetAll(key).Result()
		if err != nil {
			return nil, errors.Wrap(err, "Error trying to GetPeers")
		}
		p := mapPeerValues(v)
		peers = append(peers, &p)
	}
	return peers, nil
}

// Close will close the underlying redis client and clear in-memory caches
func (ps *PeerStore) Close() error {
	return ps.client.Close()
}

func newRedisConfig(c *config.StoreConfig) *redis.Options {
	database, err := strconv.ParseInt(c.Database, 10, 32)
	if err != nil {
		log.Panicf("Failed to parse redis database integer: %s", c.Database)
	}
	return &redis.Options{
		Addr:     fmt.Sprintf("%s:%d", c.Host, c.Port),
		Password: c.Password,
		DB:       int(database),
		OnConnect: func(conn *redis.Conn) error {
			if err := conn.ClientSetName(clientName).Err(); err != nil {
				log.Fatalf("Could not setname, bailing: %s", err)
			}
			return nil
		},
	}
}

type torrentDriver struct{}

// NewTorrentStore initialize a TorrentStore implementation using the redis backing store
func (td torrentDriver) NewTorrentStore(cfg interface{}) (store.TorrentStore, error) {
	c, ok := cfg.(*config.StoreConfig)
	if !ok {
		return nil, consts.ErrInvalidConfig
	}
	client := redis.NewClient(newRedisConfig(c))
	return &TorrentStore{
		client: client,
	}, nil
}

type peerDriver struct{}

// NewPeerStore initialize a NewPeerStore implementation using the redis backing store
func (pd peerDriver) NewPeerStore(cfg interface{}) (store.PeerStore, error) {
	c, ok := cfg.(*config.StoreConfig)
	if !ok {
		return nil, consts.ErrInvalidConfig
	}
	client := redis.NewClient(newRedisConfig(c))
	return &PeerStore{
		client: client,
	}, nil
}

type userDriver struct{}

// NewPeerStore initialize a NewPeerStore implementation using the redis backing store
func (pd userDriver) NewUserStore(cfg interface{}) (store.UserStore, error) {
	c, ok := cfg.(*config.StoreConfig)
	if !ok {
		return nil, consts.ErrInvalidConfig
	}
	client := redis.NewClient(newRedisConfig(c))
	return &UserStore{
		client: client,
	}, nil
}

func init() {
	store.AddUserDriver(driverName, userDriver{})
	store.AddPeerDriver(driverName, peerDriver{})
	store.AddTorrentDriver(driverName, torrentDriver{})
}
