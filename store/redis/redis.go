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
	"time"
)

const (
	driverName = "redis"
	clientName = "mika"
)

const (
	prefixWhitelist = "whitelist"
	prefixTorrent   = "t"
	prefixPeer      = "p"
	prefixUser      = "u"
	prefixUserID    = "user_id_pk"
)

func whiteListKey(prefix string) string {
	return fmt.Sprintf("%s%s", prefixWhitelist, prefix)
}

func torrentKey(t model.InfoHash) string {
	return fmt.Sprintf("%s:%s", prefixTorrent, t.String())
}

func torrentPeersKey(t model.InfoHash) string {
	return fmt.Sprintf("%s:%s:*", prefixPeer, t.String())
}

func peerKey(t model.InfoHash, p model.PeerID) string {
	return fmt.Sprintf("%s:%s:%s", prefixPeer, t.String(), p.String())
}

func userKey(passkey string) string {
	return fmt.Sprintf("%s:%s", prefixUser, passkey)
}

func userIDKey(userID uint32) string {
	return fmt.Sprintf("%s:%d", prefixUserID, userID)
}

// UserStore is the redis backed store.TorrentStore implementation
type UserStore struct {
	client *redis.Client
}

// Sync batch updates the backing store with the new UserStats provided
// TODO leverage cache layer so we can pipeline the updates w/o query first
func (us UserStore) Sync(b map[string]model.UserStats) error {
	for passkey, stats := range b {
		old, err := us.client.HGetAll(userKey(passkey)).Result()
		if err != nil {
			return errors.Wrap(err, "Failed to get user from redis")
		}
		var downloaded uint64
		var uploaded uint64
		var announces uint32
		downloadedStr, found := old["downloaded"]
		if found {
			downloaded = util.StringToUInt64(downloadedStr, 0)
		}
		uploadedStr, found := old["uploaded"]
		if found {
			uploaded = util.StringToUInt64(uploadedStr, 0)
		}
		announcesStr, found := old["announces"]
		if found {
			announces = util.StringToUInt32(announcesStr, 0)
		}
		us.client.HSet(userKey(passkey), map[string]interface{}{
			"downloaded": downloaded + stats.Downloaded,
			"uploaded":   uploaded + stats.Uploaded,
			"announces":  announces + stats.Announces,
		})
	}
	return nil
}

// Add inserts a user into redis via at the string provided by the userKey function
// This additionally sets the passkey->user_id mapping
func (us UserStore) Add(u model.User) error {
	pipe := us.client.TxPipeline()
	pipe.HSet(userKey(u.Passkey), map[string]interface{}{
		"user_id":          u.UserID,
		"passkey":          u.Passkey,
		"download_enabled": u.DownloadEnabled,
		"is_deleted":       u.IsDeleted,
		"downloaded":       u.Downloaded,
		"uploaded":         u.Uploaded,
		"announces":        u.Announces,
	})
	pipe.Set(userIDKey(u.UserID), u.Passkey, 0)
	if _, err := pipe.Exec(); err != nil {
		return errors.Wrap(err, "Failed to add user to store")
	}
	return nil
}

// GetByPasskey returns the hash values set of the passkey and maps it to a User struct
func (us UserStore) GetByPasskey(user *model.User, passkey string) error {
	v, err := us.client.HGetAll(userKey(passkey)).Result()
	if err != nil {
		return errors.Wrap(err, "Failed to retrieve user by passkey")
	}
	user.Passkey = v["passkey"]
	user.UserID = util.StringToUInt32(v["user_id"], 0)
	user.Downloaded = util.StringToUInt64(v["downloaded"], 0)
	user.Uploaded = util.StringToUInt64(v["uploaded"], 0)
	user.Announces = util.StringToUInt32(v["announces"], 0)
	user.DownloadEnabled = util.StringToBool(v["download_enabled"], false)
	user.IsDeleted = util.StringToBool(v["is_deleted"], false)
	if !user.Valid() {
		return consts.ErrInvalidState
	}
	return nil
}

// GetByID will query the passkey:user_id index for the passkey and return the matching user
func (us UserStore) GetByID(user *model.User, userID uint32) error {
	passkey, err := us.client.Get(userIDKey(userID)).Result()
	if err != nil {
		log.Warnf("Failed to lookup user by ID, no passkey mapped: %d", userID)
		return consts.ErrInvalidUser
	}
	if passkey == "" {
		return consts.ErrInvalidUser
	}
	return us.GetByPasskey(user, passkey)
}

// Delete drops a user from redis.
func (us UserStore) Delete(user model.User) error {
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

// Sync batch updates the backing store with the new TorrentStats provided
func (ts *TorrentStore) Sync(_ map[model.InfoHash]model.TorrentStats) error {
	panic("implement me")
}

// Conn returns the underlying connection
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
	valueMap := map[string]interface{}{
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
func (ts *TorrentStore) Add(t model.Torrent) error {
	err := ts.client.HSet(torrentKey(t.InfoHash), map[string]interface{}{
		"release_name":     t.ReleaseName,
		"total_completed":  t.TotalCompleted,
		"total_downloaded": t.TotalDownloaded,
		"total_uploaded":   t.TotalUploaded,
		"reason":           t.Reason,
		"multi_up":         t.MultiUp,
		"multi_dn":         t.MultiDn,
		"info_hash":        t.InfoHash.String(),
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
func (ts *TorrentStore) Get(t *model.Torrent, hash model.InfoHash) error {
	v, err := ts.client.HGetAll(torrentKey(hash)).Result()
	if err != nil {
		return err
	}
	ihStr, found := v["info_hash"]
	if !found {
		return consts.ErrInvalidInfoHash
	}
	var infoHash model.InfoHash
	if err := model.InfoHashFromHex(&infoHash, ihStr); err != nil {
		return errors.Wrap(err, "Failed to decode info_hash")
	}
	t.ReleaseName = v["release_name"]
	t.InfoHash = infoHash
	t.TotalCompleted = util.StringToUInt16(v["total_completed"], 0)
	t.TotalUploaded = util.StringToUInt64(v["total_uploaded"], 0)
	t.TotalDownloaded = util.StringToUInt64(v["total_downloaded"], 0)
	t.IsDeleted = util.StringToBool(v["is_deleted"], false)
	t.IsEnabled = util.StringToBool(v["is_enabled"], false)
	t.Reason = v["reason"]
	t.MultiUp = util.StringToFloat64(v["multi_up"], 1.0)
	t.MultiDn = util.StringToFloat64(v["multi_dn"], 1.0)

	return nil
}

// Close will close the underlying redis client and clear the caches
func (ts *TorrentStore) Close() error {
	return ts.client.Close()
}

// PeerStore is the redis backed store.PeerStore implementation
type PeerStore struct {
	client  *redis.Client
	pubSub  *redis.PubSub
	peerTTL time.Duration
}

// Sync batch updates the backing store with the new PeerStats provided
func (ps *PeerStore) Sync(batch map[model.PeerHash]model.PeerStats) error {
	pipe := ps.client.Pipeline()
	for ph, stats := range batch {
		k := peerKey(ph.InfoHash(), ph.PeerID())
		pipe.HIncrBy(k, "announces", int64(stats.Announces))
		pipe.HIncrBy(k, "downloaded", int64(stats.Downloaded))
		pipe.HIncrBy(k, "uploaded", int64(stats.Uploaded))
		pipe.HSet(k, "last_announce", util.TimeToString(stats.LastAnnounce))
		pipe.Expire(k, ps.peerTTL)
	}
	if _, err := pipe.Exec(); err != nil {
		return errors.Wrap(err, "Error trying to Sync peerstore (redis)")
	}
	return nil
}

// Reap will loop through the peers removing any stale entries from active swarms
func (ps *PeerStore) Reap() {
	log.Debugf("Implement reaping peers..")
}

// Add inserts a peer into the active swarm for the torrent provided
func (ps *PeerStore) Add(ih model.InfoHash, p model.Peer) error {
	err := ps.client.HSet(peerKey(ih, p.PeerID), map[string]interface{}{
		"speed_up":       p.SpeedUP,
		"speed_dn":       p.SpeedDN,
		"speed_up_max":   p.SpeedUPMax,
		"speed_dn_max":   p.SpeedDNMax,
		"uploaded":       p.Uploaded,
		"downloaded":     p.Downloaded,
		"total_left":     p.Left,
		"total_time":     p.TotalTime,
		"addr_ip":        p.IP.String(),
		"addr_port":      p.Port,
		"last_announce":  util.TimeToString(p.AnnounceLast),
		"first_announce": util.TimeToString(p.AnnounceFirst),
		"peer_id":        p.PeerID.RawString(),
		"location":       p.Location.String(),
		"user_id":        p.UserID,
		"announces":      p.Announces,
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
// Note that this OVERWRITES the values, doesnt add
func (ps *PeerStore) Update(ih model.InfoHash, p model.Peer) error {
	err := ps.client.HSet(peerKey(ih, p.PeerID), map[string]interface{}{
		"speed_up":       p.SpeedUP,
		"speed_dn":       p.SpeedDN,
		"speed_up_max":   p.SpeedUPMax,
		"speed_dn_max":   p.SpeedDNMax,
		"uploaded":       p.Uploaded,
		"downloaded":     p.Downloaded,
		"total_left":     p.Left,
		"announces":      p.Announces,
		"total_time":     p.TotalTime,
		"last_announce":  util.TimeToString(p.AnnounceLast),
		"first_announce": util.TimeToString(p.AnnounceFirst),
	}).Err()
	if err != nil {
		return errors.Wrap(err, "Failed to Update")
	}
	return nil
}

// Delete will remove a user from a torrents swarm
func (ps *PeerStore) Delete(ih model.InfoHash, p model.PeerID) error {
	return ps.client.Del(peerKey(ih, p)).Err()
}

// Get will fetch the peer from the swarm if it exists
func (ps *PeerStore) Get(p *model.Peer, ih model.InfoHash, peerID model.PeerID) error {
	v, err := ps.client.HGetAll(peerKey(ih, peerID)).Result()
	if err != nil {
		return err
	}
	mapPeerValues(p, v)
	if !p.Valid() {
		return consts.ErrInvalidState
	}
	return nil
}

func mapPeerValues(p *model.Peer, v map[string]string) {
	p.SpeedUP = util.StringToUInt32(v["speed_up"], 0)
	p.SpeedDN = util.StringToUInt32(v["speed_dn"], 0)
	p.SpeedUPMax = util.StringToUInt32(v["speed_dn_max"], 0)
	p.SpeedDNMax = util.StringToUInt32(v["speed_up_max"], 0)
	p.Uploaded = util.StringToUInt64(v["uploaded"], 0)
	p.Downloaded = util.StringToUInt64(v["downloaded"], 0)
	p.Left = util.StringToUInt32(v["total_left"], 0)
	p.Announces = util.StringToUInt32(v["announces"], 0)
	p.TotalTime = util.StringToUInt32(v["total_time"], 0)
	p.IP = net.ParseIP(v["addr_ip"])
	p.Port = util.StringToUInt16(v["addr_port"], 0)
	p.AnnounceLast = util.StringToTime(v["last_announce"])
	p.AnnounceFirst = util.StringToTime(v["first_announce"])
	p.PeerID = model.PeerIDFromString(v["peer_id"])
	p.Location = geo.LatLongFromString(v["location"])
	p.UserID = util.StringToUInt32(v["user_id"], 0)
}

// GetN will fetch peers for a torrents active swarm up to N users
func (ps *PeerStore) GetN(ih model.InfoHash, limit int) (model.Swarm, error) {
	swarm := model.NewSwarm()
	for i, key := range ps.findKeys(torrentPeersKey(ih)) {
		if i == limit {
			break
		}
		v, err := ps.client.HGetAll(key).Result()
		if err != nil {
			return swarm, errors.Wrap(err, "Error trying to GetN")
		}
		var p model.Peer
		mapPeerValues(&p, v)
		swarm.Peers[p.PeerID] = p
	}
	return swarm, nil
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
				log.Fatalf("Could not SetName, bailing: %s", err)
			}
			return nil
		},
	}
}

type torrentDriver struct{}

// New initialize a TorrentStore implementation using the redis backing store
func (td torrentDriver) New(cfg interface{}) (store.TorrentStore, error) {
	c, ok := cfg.(*config.StoreConfig)
	if !ok {
		return nil, consts.ErrInvalidConfig
	}
	client := redis.NewClient(newRedisConfig(c))
	return &TorrentStore{
		client: client,
	}, nil
}

func (ps *PeerStore) peerExpireHandler() {
	psChan := ps.pubSub.Channel()
	for {
		m := <-psChan
		// TODO cleanup any cache if it exists?
		log.Println(m)
	}
}

type peerDriver struct{}

// New initialize a New implementation using the redis backing store
func (pd peerDriver) New(cfg interface{}) (store.PeerStore, error) {
	c, ok := cfg.(*config.StoreConfig)
	if !ok {
		return nil, consts.ErrInvalidConfig
	}
	client := redis.NewClient(newRedisConfig(c))
	ps := &PeerStore{
		client:  client,
		pubSub:  client.Subscribe("peer_expired"),
		peerTTL: time.Minute * 10,
	}
	go ps.peerExpireHandler()
	return ps, nil
}

type userDriver struct{}

// New initialize a New implementation using the redis backing store
func (pd userDriver) New(cfg interface{}) (store.UserStore, error) {
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
