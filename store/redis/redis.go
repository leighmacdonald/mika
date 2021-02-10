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
	prefixRole      = "r"
	prefixUserID    = "user_id_pk"
	prefixRoleID    = "role_id_pk"
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

func roleIDKey(roleID uint32) string {
	return fmt.Sprintf("%s:%d", prefixRole, roleID)
}

// Driver is the redis backed store.StoreI implementation
type Driver struct {
	client  *redis.Client
	pubSub  *redis.PubSub
	peerTTL time.Duration
}

func (d *Driver) TorrentSave(torrent *store.Torrent) error {
	panic("implement me")
}

func (d *Driver) Migrate() error {
	return nil
}

func (d *Driver) Users() (store.Users, error) {
	panic("implement me")
}

func (d *Driver) Torrents() (store.Torrents, error) {
	panic("implement me")
}

func (d *Driver) RoleSave(role *store.Role) error {
	if role.RoleID == 0 {
		return d.RoleAdd(role)
	}
	pipe := d.client.TxPipeline()
	pipe.HSet(roleIDKey(role.RoleID), roleMap(role))
	if _, err := pipe.Exec(); err != nil {
		return errors.Wrap(err, "Failed to add user to store")
	}
	return nil
}

func (d *Driver) RoleByID(roleID uint32) (*store.Role, error) {
	r, err := d.client.HGetAll(roleIDKey(roleID)).Result()
	if err != nil {
		return nil, err
	}
	var role store.Role
	resultToRole(r, &role)
	return &role, err
}

func resultToRole(r map[string]string, role *store.Role) {
	role.RoleID = util.StringToUInt32(r["role_id"], 0)
	role.RemoteID = util.StringToUInt64(r["remote_id"], 0)
	role.RoleName = r["role_name"]
	role.Priority = util.StringToInt32(r["priority"], 0)
	role.MultiUp = util.StringToFloat64(r["multi_up"], 1.0)
	role.MultiDown = util.StringToFloat64(r["multi_down"], 1.0)
	role.DownloadEnabled = util.StringToBool(r["download_enabled"], true)
	role.UploadEnabled = util.StringToBool(r["upload_enabled"], true)
	role.CreatedOn = util.StringToTime(r["created_on"])
	role.UpdatedOn = util.StringToTime(r["updated_on"])
}

func (d *Driver) nextRoleID() (uint32, error) {
	newID, err := d.client.Incr(prefixRole + "_id_seq").Result()
	if err != nil {
		return 0, err
	}
	return uint32(newID), nil
}

func (d *Driver) nextUserID() (uint32, error) {
	newID, err := d.client.Incr(prefixUser + "_id_seq").Result()
	if err != nil {
		return 0, err
	}
	return uint32(newID), nil
}

func (d *Driver) Roles() (store.Roles, error) {
	roleIds, err := d.client.Keys(prefixRole + ":*").Result()
	if err != nil {
		return nil, err
	}
	roles := store.Roles{}
	for _, roleID := range roleIds {
		res, err := d.client.HGetAll(roleID).Result()
		if err != nil {
			return nil, err
		}
		var r store.Role
		resultToRole(res, &r)
		roles[r.RoleID] = &r
	}
	return roles, err
}

func (d *Driver) RoleAdd(role *store.Role) error {
	newID, err := d.nextRoleID()
	if err != nil {
		return err
	}
	role.RoleID = newID
	if _, err := d.client.HSet(roleIDKey(role.RoleID), roleMap(role)).Result(); err != nil {
		return errors.Wrap(err, "Failed to add user to store")
	}
	return nil
}

func (d *Driver) RoleDelete(roleID uint32) error {
	r, err := d.client.Del(roleIDKey(roleID)).Result()
	if err != nil {
		return errors.Wrap(err, "Could not delete role")
	}
	if r <= 0 {
		return consts.ErrInvalidRole
	}
	return nil
}

// Sync batch updates the backing store with the new UserStats provided
// TODO leverage cache layer so we can pipeline the updates w/o query first
func (d *Driver) UserSync(b []*store.User) error {
	//for passkey, stats := range b {
	//	old, err := d.client.HGetAll(userKey(passkey)).Result()
	//	if err != nil {
	//		return errors.Wrap(err, "Failed to get user from redis")
	//	}
	//	var downloaded uint64
	//	var uploaded uint64
	//	var announces uint32
	//	downloadedStr, found := old["downloaded"]
	//	if found {
	//		downloaded = util.StringToUInt64(downloadedStr, 0)
	//	}
	//	uploadedStr, found := old["uploaded"]
	//	if found {
	//		uploaded = util.StringToUInt64(uploadedStr, 0)
	//	}
	//	announcesStr, found := old["announces"]
	//	if found {
	//		announces = util.StringToUInt32(announcesStr, 0)
	//	}
	//	d.client.HSet(userKey(passkey), map[string]interface{}{
	//		"downloaded": downloaded + stats.Downloaded,
	//		"uploaded":   uploaded + stats.Uploaded,
	//		"announces":  announces + stats.Announces,
	//	})
	//}
	return nil
}

func userMap(u *store.User) map[string]interface{} {
	return map[string]interface{}{
		"user_id":          u.UserID,
		"role_id":          u.RoleID,
		"remote_id":        u.RemoteID,
		"is_deleted":       u.IsDeleted,
		"downloaded":       u.Downloaded,
		"uploaded":         u.Uploaded,
		"announces":        u.Announces,
		"passkey":          u.Passkey,
		"download_enabled": u.DownloadEnabled,
		"created_on":       u.CreatedOn.Format(time.RFC1123Z),
		"updated_on":       u.UpdatedOn.Format(time.RFC1123Z),
	}
}

func roleMap(r *store.Role) map[string]interface{} {
	return map[string]interface{}{
		"role_id":          r.RoleID,
		"remote_id":        r.RemoteID,
		"role_name":        r.RoleName,
		"priority":         r.Priority,
		"multi_up":         r.MultiUp,
		"multi_down":       r.MultiDown,
		"download_enabled": r.DownloadEnabled,
		"upload_enabled":   r.UploadEnabled,
		"created_on":       r.CreatedOn.Format(time.RFC1123Z),
		"updated_on":       r.UpdatedOn.Format(time.RFC1123Z),
	}
}

// Add inserts a user into redis via at the string provided by the userKey function
// This additionally sets the passkey->user_id mapping
func (d *Driver) UserAdd(u *store.User) error {
	if u.RoleID <= 0 {
		return consts.ErrInvalidRole
	}
	id, err := d.nextUserID()
	if err != nil {
		return err
	}
	u.CreatedOn = util.Now()
	u.UpdatedOn = util.Now()
	u.UserID = id
	if err := d.client.HSet(userKey(u.Passkey), userMap(u)).Err(); err != nil {
		return err
	}
	if err2 := d.client.Set(userIDKey(u.UserID), u.Passkey, 0).Err(); err2 != nil {
		return errors.Wrap(err2, "Failed to add user to store")
	}
	return nil
}

// GetByPasskey returns the hash values set of the passkey and maps it to a User struct
func (d *Driver) UserGetByPasskey(passkey string) (*store.User, error) {
	var user store.User
	v, err := d.client.HGetAll(userKey(passkey)).Result()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to retrieve user by passkey")
	}
	user.Passkey = v["passkey"]
	user.UserID = util.StringToUInt32(v["user_id"], 0)
	user.RoleID = util.StringToUInt32(v["role_id"], 0)
	user.RemoteID = util.StringToUInt64(v["remote_id"], 0)
	user.Downloaded = util.StringToUInt64(v["downloaded"], 0)
	user.Uploaded = util.StringToUInt64(v["uploaded"], 0)
	user.Announces = util.StringToUInt32(v["announces"], 0)
	user.DownloadEnabled = util.StringToBool(v["download_enabled"], false)
	user.IsDeleted = util.StringToBool(v["is_deleted"], false)
	user.CreatedOn = util.StringToTime(v["created_on"])
	user.UpdatedOn = util.StringToTime(v["updated_on"])
	if !user.Valid() {
		return nil, consts.ErrInvalidState
	}
	return &user, nil
}

// GetByID will query the passkey:user_id index for the passkey and return the matching user
func (d *Driver) UserGetByID(userID uint32) (*store.User, error) {
	passkey, err := d.client.Get(userIDKey(userID)).Result()
	if err != nil {
		log.Warnf("Failed to lookup user by ID, no passkey mapped: %d", userID)
		return nil, consts.ErrInvalidUser
	}
	if passkey == "" {
		return nil, consts.ErrInvalidUser
	}
	return d.UserGetByPasskey(passkey)
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

func (d *Driver) UserSave(user *store.User) error {
	exists, err := d.client.Exists(userKey(user.Passkey)).Result()
	if err != nil || exists == 0 {
		return err
	}
	pipe := d.client.TxPipeline()
	pipe.HSet(userKey(user.Passkey), userMap(user))
	pipe.Set(userIDKey(user.UserID), user.Passkey, 0)
	// TODO handle changing passkeys properly, this will not remove the old
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
func (d *Driver) TorrentSync(batch []*store.Torrent) error {
	//pipe := d.client.TxPipeline()
	//for ih, s := range batch {
	//	pipe.HIncrBy(torrentKey(ih), "seeders", int64(s.Seeders))
	//	pipe.HIncrBy(torrentKey(ih), "leechers", int64(s.Leechers))
	//	pipe.HIncrBy(torrentKey(ih), "total_completed", int64(s.Snatches))
	//	pipe.HIncrBy(torrentKey(ih), "total_uploaded", int64(s.Uploaded))
	//	pipe.HIncrBy(torrentKey(ih), "total_downloaded", int64(s.Downloaded))
	//	pipe.HIncrBy(torrentKey(ih), "announces", int64(s.Announces))
	//}
	//if _, err := pipe.Exec(); err != nil {
	//	return err
	//}
	return nil
}

// Conn returns the underlying connection
func (d *Driver) Conn() interface{} {
	return d.client
}

// WhiteListDelete removes a client from the global whitelist
func (d *Driver) WhiteListDelete(client *store.WhiteListClient) error {
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
func (d *Driver) WhiteListAdd(client *store.WhiteListClient) error {
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
func (d *Driver) WhiteListGetAll() ([]*store.WhiteListClient, error) {
	prefixes, err := d.client.Keys(fmt.Sprintf("%s*", prefixWhitelist)).Result()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to fetch whitelist keys")
	}
	var wl []*store.WhiteListClient
	for _, prefix := range prefixes {
		valueMap, err := d.client.HGetAll(whiteListKey(prefix)).Result()
		if err != nil {
			return nil, errors.Wrapf(err, "Failed to fetch whitelist value for: %s", whiteListKey(prefix))
		}
		wl = append(wl, &store.WhiteListClient{
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
func (d *Driver) TorrentGet(hash store.InfoHash, deletedOk bool) (*store.Torrent, error) {
	var t store.Torrent
	v, err := d.client.HGetAll(torrentKey(hash)).Result()
	if err != nil {
		return nil, err
	}
	ihStr, found := v["info_hash"]
	if !found {
		return nil, consts.ErrInvalidInfoHash
	}
	var infoHash store.InfoHash
	if err := store.InfoHashFromHex(&infoHash, ihStr); err != nil {
		return nil, errors.Wrap(err, "Failed to decode info_hash")
	}
	isDeleted := util.StringToBool(v["is_deleted"], false)
	if isDeleted && !deletedOk {
		return nil, consts.ErrInvalidInfoHash
	}
	t.InfoHash = infoHash
	t.Snatches = util.StringToUInt32(v["total_completed"], 0)
	t.Uploaded = util.StringToUInt64(v["total_uploaded"], 0)
	t.Downloaded = util.StringToUInt64(v["total_downloaded"], 0)
	t.IsDeleted = isDeleted
	t.IsEnabled = util.StringToBool(v["is_enabled"], false)
	t.Reason = v["reason"]
	t.MultiUp = util.StringToFloat64(v["multi_up"], 1.0)
	t.MultiDn = util.StringToFloat64(v["multi_dn"], 1.0)
	t.Announces = util.StringToUInt64(v["announces"], 0)
	t.Seeders = util.StringToUInt32(v["seeders"], 0)
	t.Leechers = util.StringToUInt32(v["leechers"], 0)
	return &t, nil
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
