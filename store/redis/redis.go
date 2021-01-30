package redis

import (
	"fmt"
	"github.com/go-redis/redis/v7"
	"github.com/leighmacdonald/mika/config"
	"github.com/leighmacdonald/mika/consts"
	"github.com/leighmacdonald/mika/store"
	"github.com/leighmacdonald/mika/util"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
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

// Driver is the redis backed store.StoreI implementation
type Driver struct {
	client  *redis.Client
	pubSub  *redis.PubSub
	peerTTL time.Duration
}

func (d *Driver) RoleByID(role *store.Role, roleID uint32) error {
	panic("implement me")
}

func (d *Driver) Roles() (store.Roles, error) {
	panic("implement me")
}

func (d *Driver) RoleAdd(role *store.Role) error {
	panic("implement me")
}

func (d *Driver) RoleDelete(roleID uint32) error {
	panic("implement me")
}

// Sync batch updates the backing store with the new UserStats provided
// TODO leverage cache layer so we can pipeline the updates w/o query first
func (d *Driver) UserSync(b map[string]store.UserStats) error {
	for passkey, stats := range b {
		old, err := d.client.HGetAll(userKey(passkey)).Result()
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
		d.client.HSet(userKey(passkey), map[string]interface{}{
			"downloaded": downloaded + stats.Downloaded,
			"uploaded":   uploaded + stats.Uploaded,
			"announces":  announces + stats.Announces,
		})
	}
	return nil
}

func userMap(u *store.User) map[string]interface{} {
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
func (d *Driver) UserAdd(u *store.User) error {
	pipe := d.client.TxPipeline()
	pipe.HSet(userKey(u.Passkey), userMap(u))
	pipe.Set(userIDKey(u.UserID), u.Passkey, 0)
	if _, err := pipe.Exec(); err != nil {
		return errors.Wrap(err, "Failed to add user to store")
	}
	return nil
}

// GetByPasskey returns the hash values set of the passkey and maps it to a User struct
func (d *Driver) UserGetByPasskey(user *store.User, passkey string) error {
	v, err := d.client.HGetAll(userKey(passkey)).Result()
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
func (d *Driver) UserGetByID(user *store.User, userID uint32) error {
	passkey, err := d.client.Get(userIDKey(userID)).Result()
	if err != nil {
		log.Warnf("Failed to lookup user by ID, no passkey mapped: %d", userID)
		return consts.ErrInvalidUser
	}
	if passkey == "" {
		return consts.ErrInvalidUser
	}
	return d.UserGetByPasskey(user, passkey)
}

// Delete drops a user from redis.
func (d *Driver) UserDelete(user *store.User) error {
	if err := d.client.Del(userKey(user.Passkey)).Err(); err != nil {
		return errors.Wrap(err, "Could not remove user from store")
	}
	if err := d.client.Del(userIDKey(user.UserID)).Err(); err != nil {
		return errors.Wrap(err, "Could not remove user pk index from store")
	}
	return nil
}

func (d *Driver) UserUpdate(user *store.User, oldPasskey string) error {
	passkey := user.Passkey
	if oldPasskey != "" {
		passkey = oldPasskey
	}
	exists, err := d.client.Exists(userKey(passkey)).Result()
	if err != nil || exists == 0 {
		return err
	}
	pipe := d.client.TxPipeline()
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

// Update is just a Add call with a check for existing key first as the process
// is the same for setting both
func (d *Driver) TorrentUpdate(torrent *store.Torrent) error {
	val, err := d.client.Exists(torrentKey(torrent.InfoHash)).Result()
	if err != nil {
		return err
	}
	if val == 0 {
		return errors.Wrapf(consts.ErrInvalidInfoHash, "Won't update non-existent torrent")
	}
	return d.TorrentAdd(torrent)
}

// Sync batch updates the backing store with the new TorrentStats provided
func (d *Driver) TorrentSync(batch map[store.InfoHash]store.TorrentStats) error {
	pipe := d.client.TxPipeline()
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
func (d *Driver) Conn() interface{} {
	return d.client
}

// WhiteListDelete removes a client from the global whitelist
func (d *Driver) WhiteListDelete(client store.WhiteListClient) error {
	res, err := d.client.Del(whiteListKey(client.ClientPrefix)).Result()
	if err != nil {
		return errors.Wrap(err, "Failed to remove whitelisted client")
	}
	if res != 1 {
		return consts.ErrInvalidClient
	}
	return nil
}

// WhiteListAdd will insert a new client prefix into the allowed clients list
func (d *Driver) WhiteListAdd(client store.WhiteListClient) error {
	valueMap := map[string]interface{}{
		"client_prefix": client.ClientPrefix,
		"client_name":   client.ClientName,
	}
	err := d.client.HSet(whiteListKey(client.ClientPrefix), valueMap).Err()
	if err != nil {
		return errors.Wrapf(err, "failed to add new whitelisted client prefix: %s", client.ClientPrefix)
	}
	return nil
}

// WhiteListGetAll fetches all known whitelisted clients
func (d *Driver) WhiteListGetAll() ([]store.WhiteListClient, error) {
	prefixes, err := d.client.Keys(fmt.Sprintf("%s*", prefixWhitelist)).Result()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to fetch whitelist keys")
	}
	var wl []store.WhiteListClient
	for _, prefix := range prefixes {
		valueMap, err := d.client.HGetAll(whiteListKey(prefix)).Result()
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

func torrentMap(t *store.Torrent) map[string]interface{} {
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
func (d *Driver) TorrentAdd(t *store.Torrent) error {
	err := d.client.HSet(torrentKey(t.InfoHash), torrentMap(t)).Err()
	if err != nil {
		return err
	}
	return nil
}

// Delete will mark a torrent as deleted in the backing store.
// If dropRow is true, it will permanently remove the torrent from the store
func (d *Driver) TorrentDelete(ih store.InfoHash, dropRow bool) error {
	if dropRow {
		if err := d.client.Del(torrentKey(ih)).Err(); err != nil {
			return errors.Wrap(err, "Could not remove torrent from store")
		}
		return nil
	}
	if err := d.client.HSet(torrentKey(ih), "is_deleted", 1).Err(); err != nil {
		return errors.Wrap(err, "Could not mark torrent as deleted")
	}
	return nil
}

// Get returns the Torrent matching the infohash
func (d *Driver) TorrentGet(t *store.Torrent, hash store.InfoHash, deletedOk bool) error {
	v, err := d.client.HGetAll(torrentKey(hash)).Result()
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

func (d *Driver) Name() string {
	return driverName
}

// Reap will loop through the peers removing any stale entries from active swarms
func (d *Driver) Reap() []store.PeerHash {
	return nil
}

func (d *Driver) findKeys(prefix string) []string {
	v, err := d.client.Keys(prefix).Result()
	if err != nil {
		log.Errorf("Failed to query for key prefix: %s", err.Error())
	}
	return v
}

// Close will close the underlying redis client and clear in-memory caches
func (d *Driver) Close() error {
	return d.client.Close()
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

type initializer struct{}

// New initialize a New implementation using the redis backing store
func (pd initializer) New(cfg config.StoreConfig) (store.Store, error) {
	return &Driver{client: redis.NewClient(newRedisConfig(cfg))}, nil
}

func init() {
	store.AddDriver(driverName, initializer{})
}
