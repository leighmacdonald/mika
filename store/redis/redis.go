package redis

import (
	"fmt"
	"github.com/go-redis/redis/v7"
	"github.com/leighmacdonald/mika/config"
	"github.com/leighmacdonald/mika/consts"
	"github.com/leighmacdonald/mika/geo"
	"github.com/leighmacdonald/mika/store"
	"github.com/leighmacdonald/mika/util"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"net"
	"strconv"
	"strings"
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

func torrentKey(t store.InfoHash) string {
	return fmt.Sprintf("%s:%s", prefixTorrent, t.String())
}

func torrentPeersKey(t store.InfoHash) string {
	return fmt.Sprintf("%s:%s:*", prefixPeer, t.String())
}

func peerKey(t store.InfoHash, p store.PeerID) string {
	return fmt.Sprintf("%s:%s:%s", prefixPeer, t.String(), p.String())
}

func userKey(passkey string) string {
	return fmt.Sprintf("%s:%s", prefixUser, passkey)
}

func userIDKey(userID uint32) string {
	return fmt.Sprintf("%s:%d", prefixUserID, userID)
}

// UserStore is the redis backed store.StoreI implementation
type UserStore struct {
	client *redis.Client
}

func (us UserStore) Roles() (store.Roles, error) {
	panic("implement me")
}

func (us UserStore) RoleAdd(role store.Role) error {
	panic("implement me")
}

func (us UserStore) RoleDelete(roleID int) error {
	panic("implement me")
}

func (us UserStore) Name() string {
	return driverName
}

// Sync batch updates the backing store with the new UserStats provided
// TODO leverage cache layer so we can pipeline the updates w/o query first
func (us UserStore) Sync(b map[string]store.UserStats) error {
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

func userMap(u store.User) map[string]interface{} {
	return map[string]interface{}{
		"user_id":          u.UserID,
		"passkey":          u.Passkey,
		"download_enabled": u.DownloadEnabled,
		"is_deleted":       u.IsDeleted,
		"downloaded":       u.Downloaded,
		"uploaded":         u.Uploaded,
		"announces":        u.Announces,
	}
}

// Add inserts a user into redis via at the string provided by the userKey function
// This additionally sets the passkey->user_id mapping
func (us UserStore) Add(u store.User) error {
	pipe := us.client.TxPipeline()
	pipe.HSet(userKey(u.Passkey), userMap(u))
	pipe.Set(userIDKey(u.UserID), u.Passkey, 0)
	if _, err := pipe.Exec(); err != nil {
		return errors.Wrap(err, "Failed to add user to store")
	}
	return nil
}

// GetByPasskey returns the hash values set of the passkey and maps it to a User struct
func (us UserStore) GetByPasskey(user *store.User, passkey string) error {
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
func (us UserStore) GetByID(user *store.User, userID uint32) error {
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
func (us UserStore) Delete(user store.User) error {
	if err := us.client.Del(userKey(user.Passkey)).Err(); err != nil {
		return errors.Wrap(err, "Could not remove user from store")
	}
	if err := us.client.Del(userIDKey(user.UserID)).Err(); err != nil {
		return errors.Wrap(err, "Could not remove user pk index from store")
	}
	return nil
}

func (us UserStore) Update(user store.User, oldPasskey string) error {
	passkey := user.Passkey
	if oldPasskey != "" {
		passkey = oldPasskey
	}
	exists, err := us.client.Exists(userKey(passkey)).Result()
	if err != nil || exists == 0 {
		return err
	}
	pipe := us.client.TxPipeline()
	pipe.HSet(userKey(user.Passkey), userMap(user))
	pipe.Set(userIDKey(user.UserID), user.Passkey, 0)
	if oldPasskey != "" {
		// remove old user object when switching to a new passkey as its the key
		pipe.Del(userKey(passkey))
	}
	if _, err := pipe.Exec(); err != nil {
		return errors.Wrap(err, "Failed to add user to store")
	}
	return nil
}

// Close will shutdown the underlying redis connection
func (us UserStore) Close() error {
	return us.client.Close()
}

// TorrentStore is the redis backed store.StoreI implementation
type TorrentStore struct {
	client *redis.Client
}

func (ts *TorrentStore) Name() string {
	return driverName
}

// Update is just a Add call with a check for existing key first as the process
// is the same for setting both
func (ts *TorrentStore) Update(torrent store.Torrent) error {
	val, err := ts.client.Exists(torrentKey(torrent.InfoHash)).Result()
	if err != nil {
		return err
	}
	if val == 0 {
		return errors.Wrapf(consts.ErrInvalidInfoHash, "Won't update non-existent torrent")
	}
	return ts.Add(torrent)
}

// Sync batch updates the backing store with the new TorrentStats provided
func (ts *TorrentStore) Sync(batch map[store.InfoHash]store.TorrentStats) error {
	pipe := ts.client.TxPipeline()
	for ih, s := range batch {
		pipe.HIncrBy(torrentKey(ih), "seeders", int64(s.Seeders))
		pipe.HIncrBy(torrentKey(ih), "leechers", int64(s.Leechers))
		pipe.HIncrBy(torrentKey(ih), "total_completed", int64(s.Snatches))
		pipe.HIncrBy(torrentKey(ih), "total_uploaded", int64(s.Uploaded))
		pipe.HIncrBy(torrentKey(ih), "total_downloaded", int64(s.Downloaded))
		pipe.HIncrBy(torrentKey(ih), "announces", int64(s.Announces))
	}
	if _, err := pipe.Exec(); err != nil {
		return err
	}
	return nil
}

// Conn returns the underlying connection
func (ts *TorrentStore) Conn() interface{} {
	return ts.client
}

// WhiteListDelete removes a client from the global whitelist
func (ts *TorrentStore) WhiteListDelete(client store.WhiteListClient) error {
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
func (ts *TorrentStore) WhiteListAdd(client store.WhiteListClient) error {
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
func (ts *TorrentStore) WhiteListGetAll() ([]store.WhiteListClient, error) {
	prefixes, err := ts.client.Keys(fmt.Sprintf("%s*", prefixWhitelist)).Result()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to fetch whitelist keys")
	}
	var wl []store.WhiteListClient
	for _, prefix := range prefixes {
		valueMap, err := ts.client.HGetAll(whiteListKey(prefix)).Result()
		if err != nil {
			return nil, errors.Wrapf(err, "Failed to fetch whitelist value for: %s", whiteListKey(prefix))
		}
		wl = append(wl, store.WhiteListClient{
			ClientPrefix: valueMap["client_prefix"],
			ClientName:   valueMap["client_name"],
		})
	}
	return wl, nil
}

func torrentMap(t store.Torrent) map[string]interface{} {
	return map[string]interface{}{
		"total_completed":  t.Snatches,
		"total_downloaded": t.Downloaded,
		"total_uploaded":   t.Uploaded,
		"reason":           t.Reason,
		"multi_up":         t.MultiUp,
		"multi_dn":         t.MultiDn,
		"info_hash":        t.InfoHash.String(),
		"is_deleted":       t.IsDeleted,
		"is_enabled":       t.IsEnabled,
		"announces":        t.Announces,
		"seeders":          t.Seeders,
		"leechers":         t.Leechers,
	}
}

// Add adds a new torrent to the redis backing store
func (ts *TorrentStore) Add(t store.Torrent) error {
	err := ts.client.HSet(torrentKey(t.InfoHash), torrentMap(t)).Err()
	if err != nil {
		return err
	}
	return nil
}

// Delete will mark a torrent as deleted in the backing store.
// If dropRow is true, it will permanently remove the torrent from the store
func (ts *TorrentStore) Delete(ih store.InfoHash, dropRow bool) error {
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
func (ts *TorrentStore) Get(t *store.Torrent, hash store.InfoHash, deletedOk bool) error {
	v, err := ts.client.HGetAll(torrentKey(hash)).Result()
	if err != nil {
		return err
	}
	ihStr, found := v["info_hash"]
	if !found {
		return consts.ErrInvalidInfoHash
	}
	var infoHash store.InfoHash
	if err := store.InfoHashFromHex(&infoHash, ihStr); err != nil {
		return errors.Wrap(err, "Failed to decode info_hash")
	}
	isDeleted := util.StringToBool(v["is_deleted"], false)
	if isDeleted && !deletedOk {
		return consts.ErrInvalidInfoHash
	}
	t.InfoHash = infoHash
	t.Snatches = util.StringToUInt16(v["total_completed"], 0)
	t.Uploaded = util.StringToUInt64(v["total_uploaded"], 0)
	t.Downloaded = util.StringToUInt64(v["total_downloaded"], 0)
	t.IsDeleted = isDeleted
	t.IsEnabled = util.StringToBool(v["is_enabled"], false)
	t.Reason = v["reason"]
	t.MultiUp = util.StringToFloat64(v["multi_up"], 1.0)
	t.MultiDn = util.StringToFloat64(v["multi_dn"], 1.0)
	t.Announces = util.StringToUInt64(v["announces"], 0)
	t.Seeders = util.StringToUInt(v["seeders"], 0)
	t.Leechers = util.StringToUInt(v["leechers"], 0)
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

func (ps *PeerStore) Name() string {
	return driverName
}

// Sync batch updates the backing store with the new PeerStats provided
func (ps *PeerStore) Sync(batch map[store.PeerHash]store.PeerStats) error {
	pipe := ps.client.Pipeline()
	for ph, stats := range batch {
		sum := stats.Totals()
		k := peerKey(ph.InfoHash(), ph.PeerID())
		pipe.HIncrBy(k, "announces", int64(len(stats.Hist)))
		pipe.HIncrBy(k, "downloaded", int64(sum.TotalDn))
		pipe.HIncrBy(k, "uploaded", int64(sum.TotalUp))
		pipe.HSet(k, "last_announce", util.TimeToString(sum.LastAnn))
		pipe.Expire(k, ps.peerTTL)
	}
	if _, err := pipe.Exec(); err != nil {
		return errors.Wrap(err, "Error trying to Sync peerstore (redis)")
	}
	return nil
}

// Reap will loop through the peers removing any stale entries from active swarms
func (ps *PeerStore) Reap() []store.PeerHash {
	return nil
}

// Add inserts a peer into the active swarm for the torrent provided
func (ps *PeerStore) Add(ih store.InfoHash, p store.Peer) error {
	ipv6 := strings.Count(p.IP.String(), ":") > 1
	err := ps.client.HSet(peerKey(ih, p.PeerID), map[string]interface{}{
		"speed_up":       p.SpeedUP,
		"speed_dn":       p.SpeedDN,
		"speed_up_max":   p.SpeedUPMax,
		"speed_dn_max":   p.SpeedDNMax,
		"uploaded":       p.Uploaded,
		"downloaded":     p.Downloaded,
		"total_left":     p.Left,
		"total_time":     p.TotalTime,
		"ipv6":           ipv6,
		"addr_ip":        p.IP.String(),
		"addr_port":      p.Port,
		"last_announce":  util.TimeToString(p.AnnounceLast),
		"first_announce": util.TimeToString(p.AnnounceFirst),
		"peer_id":        p.PeerID.RawString(),
		"location":       p.Location.String(),
		"user_id":        p.UserID,
		"announces":      p.Announces,
		"country_code":   p.CountryCode,
		"asn":            p.ASN,
		"as_name":        p.AS,
		"crypto_level":   int(p.CryptoLevel),
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
func (ps *PeerStore) Update(ih store.InfoHash, p store.Peer) error {
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
func (ps *PeerStore) Delete(ih store.InfoHash, p store.PeerID) error {
	return ps.client.Del(peerKey(ih, p)).Err()
}

// Get will fetch the peer from the swarm if it exists
func (ps *PeerStore) Get(p *store.Peer, ih store.InfoHash, peerID store.PeerID) error {
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

func mapPeerValues(p *store.Peer, v map[string]string) {
	p.SpeedUP = util.StringToUInt32(v["speed_up"], 0)
	p.SpeedDN = util.StringToUInt32(v["speed_dn"], 0)
	p.SpeedUPMax = util.StringToUInt32(v["speed_dn_max"], 0)
	p.SpeedDNMax = util.StringToUInt32(v["speed_up_max"], 0)
	p.Uploaded = util.StringToUInt64(v["uploaded"], 0)
	p.Downloaded = util.StringToUInt64(v["downloaded"], 0)
	p.Left = util.StringToUInt32(v["total_left"], 0)
	p.Announces = util.StringToUInt32(v["announces"], 0)
	p.TotalTime = util.StringToUInt32(v["total_time"], 0)
	p.IPv6 = util.StringToBool(v["ipv6"], false)
	p.IP = net.ParseIP(v["addr_ip"])
	p.Port = util.StringToUInt16(v["addr_port"], 0)
	p.AnnounceLast = util.StringToTime(v["last_announce"])
	p.AnnounceFirst = util.StringToTime(v["first_announce"])
	p.PeerID = store.PeerIDFromString(v["peer_id"])
	p.Location = geo.LatLongFromString(v["location"])
	p.UserID = util.StringToUInt32(v["user_id"], 0)
	p.ASN = util.StringToUInt32(v["asn"], 0)
	p.AS = v["as_name"]
	p.CountryCode = v["country_code"]
	p.CryptoLevel = consts.CryptoLevel(util.StringToUInt(v["crypto_level"], 0))
}

// GetN will fetch peers for a torrents active swarm up to N users
func (ps *PeerStore) GetN(ih store.InfoHash, limit int) (store.Swarm, error) {
	swarm := store.NewSwarm()
	for i, key := range ps.findKeys(torrentPeersKey(ih)) {
		if i == limit {
			break
		}
		v, err := ps.client.HGetAll(key).Result()
		if err != nil {
			return swarm, errors.Wrap(err, "Error trying to GetN")
		}
		var p store.Peer
		mapPeerValues(&p, v)
		swarm.Peers[p.PeerID] = p
	}
	return swarm, nil
}

// Close will close the underlying redis client and clear in-memory caches
func (ps *PeerStore) Close() error {
	return ps.client.Close()
}

func newRedisConfig(c config.StoreConfig) *redis.Options {
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
func (td torrentDriver) New(cfg config.StoreConfig) (store.StoreI, error) {
	client := redis.NewClient(newRedisConfig(cfg))
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
func (pd peerDriver) New(cfg config.StoreConfig) (store.PeerStore, error) {
	client := redis.NewClient(newRedisConfig(cfg))
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
func (pd userDriver) New(cfg config.StoreConfig) (store.Store, error) {
	return &UserStore{client: redis.NewClient(newRedisConfig(cfg))}, nil
}

func init() {
	store.AddUserDriver(driverName, userDriver{})
	store.AddPeerDriver(driverName, peerDriver{})
	store.AddDriver(driverName, torrentDriver{})
}
