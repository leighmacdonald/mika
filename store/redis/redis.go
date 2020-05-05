package redis

import (
	"fmt"
	"github.com/go-redis/redis/v7"
	"github.com/leighmacdonald/mika/config"
	"github.com/leighmacdonald/mika/consts"
	"github.com/leighmacdonald/mika/geo"
	"github.com/leighmacdonald/mika/model"
	"github.com/leighmacdonald/mika/store"
	"github.com/leighmacdonald/mika/util"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"net"
	"strconv"
	"sync"
)

const (
	driverName = "redis"
	clientName = "mika"
)

const (
	prefixWhitelist    = "whitelist:"
	prefixTorrent      = "t:"
	prefixTorrentPeers = "tp:"
	prefixPeer         = "p:"
	prefixUser         = "u:"
	prefixUserID       = "user_id_pk:"
)

func whiteListKey(prefix string) string {
	return fmt.Sprintf("%s%s", prefixWhitelist, prefix)
}

func torrentKey(t model.InfoHash) string {
	return fmt.Sprintf("%s%s", prefixTorrent, t.String())
}

func torrentPeersKey(t model.InfoHash) string {
	return fmt.Sprintf("%s:%s:*", prefixTorrentPeers, t.String())
}

func peerKey(t model.InfoHash, p model.PeerID) string {
	return fmt.Sprintf("%s%s:%s", prefixPeer, t.String(), p.String())
}

func userKey(passkey string) string {
	return fmt.Sprintf("%s%s", prefixUser, passkey)
}

func userIDKey(userID uint32) string {
	return fmt.Sprintf("%s%d", prefixUserID, userID)
}

// UserStore is the redis backed store.TorrentStore implementation
type UserStore struct {
	client *redis.Client
}

// Add inserts a user into redis via at the string provided by the userKey function
// This additionally sets the passkey->user_id mapping
func (us UserStore) Add(u *model.User) error {
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

// GetByPasskey returns the hash values set of the passkey and maps it to a User struct
func (us UserStore) GetByPasskey(passkey string) (*model.User, error) {
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

// GetByID will query the passkey:user_id index for the passkey and return the matching user
func (us UserStore) GetByID(userID uint32) (*model.User, error) {
	passkey, err := us.client.Get(userIDKey(userID)).Result()
	if err != nil {
		log.Warnf("Failed to lookup user by ID, no passkey mapped: %d", userID)
		return nil, consts.ErrInvalidUser
	}
	if passkey == "" {
		return nil, consts.ErrInvalidUser
	}
	return us.GetByPasskey(passkey)
}

// Delete drops a user from redis.
func (us UserStore) Delete(user *model.User) error {
	if err := us.client.Del(userKey(user.Passkey)).Err(); err != nil {
		return errors.Wrap(err, "Could not remove user from store")
	}
	if err := us.client.Del(userIDKey(user.UserID)).Err(); err != nil {
		return errors.Wrap(err, "Could not remove user pk index from store")
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

func (ts *TorrentStore) Conn() interface{} {
	return ts.client
}

// WhiteListDelete removes a client from the global whitelist
func (ts *TorrentStore) WhiteListDelete(client model.WhiteListClient) error {
	res, err := ts.client.Del(whiteListKey(client.ClientPrefix)).Result()
	if err != nil {
		return errors.Wrap(err, "Failed to remove whitelisted client")
	}
	if res != 1 {
		return consts.ErrInvalidClient
	}
	return nil
}

// WhiteListAdd will insert a new client prefix into the allowed clients list
func (ts *TorrentStore) WhiteListAdd(client model.WhiteListClient) error {
	valueMap := map[string]string{
		"client_prefix": client.ClientPrefix,
		"client_name":   client.ClientName,
	}
	err := ts.client.HSet(whiteListKey(client.ClientPrefix), valueMap).Err()
	if err != nil {
		return errors.Wrapf(err, "failed to add new whitelisted client prefix: %s", client.ClientPrefix)
	}
	return nil
}

// WhiteListGetAll fetches all known whitelisted clients
func (ts *TorrentStore) WhiteListGetAll() ([]model.WhiteListClient, error) {
	prefixes, err := ts.client.Keys(fmt.Sprintf("%s*", prefixWhitelist)).Result()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to fetch whitelist keys")
	}
	var wl []model.WhiteListClient
	for _, prefix := range prefixes {
		valueMap, err := ts.client.HGetAll(whiteListKey(prefix)).Result()
		if err != nil {
			return nil, errors.Wrapf(err, "Failed to fetch whitelist value for: %s", whiteListKey(prefix))
		}
		wl = append(wl, model.WhiteListClient{
			ClientPrefix: valueMap["client_prefix"],
			ClientName:   valueMap["client_name"],
		})
	}
	return wl, nil
}

// Add adds a new torrent to the redis backing store
func (ts *TorrentStore) Add(t *model.Torrent) error {
	err := ts.client.HSet(torrentKey(t.InfoHash), map[string]interface{}{
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
	}).Err()
	if err != nil {
		return err
	}
	return nil
}

// Delete will mark a torrent as deleted in the backing store.
// If dropRow is true, it will permanently remove the torrent from the store
func (ts *TorrentStore) Delete(ih model.InfoHash, dropRow bool) error {
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

// Get returns the Torrent matching the infohash
func (ts *TorrentStore) Get(hash model.InfoHash) (*model.Torrent, error) {
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

// Add inserts a peer into the active swarm for the torrent provided
func (ps *PeerStore) Add(ih model.InfoHash, p *model.Peer) error {
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
		return errors.Wrap(err, "Failed to Add")
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

// Update will sync any new peer data with the backing store
func (ps *PeerStore) Update(ih model.InfoHash, p *model.Peer) error {
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
		return errors.Wrap(err, "Failed to Update")
	}
	return nil
}

// Delete will remove a user from a torrents swarm
func (ps *PeerStore) Delete(ih model.InfoHash, p *model.Peer) error {
	return ps.client.Del(peerKey(ih, p.PeerID)).Err()
}

// Get will fetch the peer from the swarm if it exists
func (ps *PeerStore) Get(ih model.InfoHash, peerID model.PeerID) (*model.Peer, error) {
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

// GetN will fetch peers for a torrents active swarm up to N users
func (ps *PeerStore) GetN(ih model.InfoHash, limit int) (model.Swarm, error) {
	var peers []*model.Peer
	for i, key := range ps.findKeys(torrentPeersKey(ih)) {
		if i == limit {
			break
		}
		v, err := ps.client.HGetAll(key).Result()
		if err != nil {
			return nil, errors.Wrap(err, "Error trying to GetN")
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
