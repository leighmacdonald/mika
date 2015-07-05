package tracker

import (
	"errors"
	"git.totdev.in/totv/mika"
	"git.totdev.in/totv/mika/db"
	"git.totdev.in/totv/mika/util"
	log "github.com/Sirupsen/logrus"
	"github.com/garyburd/redigo/redis"
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"
	"syscall"
)

type (
	ResponseOK struct {
		Msg string `json:"message"`
	}

	VersionResponse struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	}

	UptimeResponse struct {
		Process int32 `json:"process"`
		System  int32 `json:"system"`
	}

	UserPayload struct {
		UserID uint64 `json:"user_id"`
	}

	UserCreatePayload struct {
		UserPayload
		Passkey  string `json:"passkey"`
		CanLeech bool   `json:"can_leech"`
		Name     string `json:"name"`
	}

	UserUpdatePayload struct {
		UserCreatePayload
		Downloaded uint64 `json:"downloaded"`
		Uploaded   uint64 `json:"uploaded"`
		Enabled    bool   `json:"enabled"`
	}

	TorrentPayload struct {
		TorrentID uint64 `json:"torrent_id"`
	}

	TorrentAddPayload struct {
		TorrentPayload
		InfoHash string `json:"info_hash"`
		Name     string `json:"name"`
	}

	UserTorrentsResponse struct {
		Active     []string `json:"active"`
		HNR        []string `json:"hnr"`
		Complete   []string `json:"complete"`
		Incomplete []string `json:"incomplete"`
	}

	TorrentDelPayload struct {
		TorrentPayload
		Reason string
	}
	WhitelistPayload struct {
		Prefix string `json:"prefix"`
	}

	WhitelistAddPayload struct {
		WhitelistPayload
		Client string `json:"client"`
	}
)

var (
	resp_ok = ResponseOK{"ok"}
)

// HandleVersion returns the current running version
func (t *Tracker) HandleVersion(c *gin.Context) {
	c.JSON(http.StatusOK, VersionResponse{Name: "mika", Version: mika.Version})
}

// HandleUptime returns the current process uptime
func (tracker *Tracker) HandleUptime(c *gin.Context) {
	info := &syscall.Sysinfo_t{}
	err := syscall.Sysinfo(info)
	if err != nil {
		c.Error(err).SetMeta(errMeta(
			http.StatusInternalServerError,
			"Error trying to fetch sysinfo",
			log.Fields{"fn": "HandleUptime"},
			log.ErrorLevel,
		))
	} else {
		//noinspection GoUnresolvedReference
		c.JSON(http.StatusOK, UptimeResponse{
			Process: util.Unixtime() - mika.StartTime,
			System:  int32(info.Uptime),
		})
	}
}

// HandleTorrentGet will find and return the requested torrent.
func (tracker *Tracker) HandleTorrentGet(c *gin.Context) {
	info_hash := c.Param("info_hash")
	torrent := tracker.FindTorrentByInfoHash(info_hash)
	if torrent == nil {
		c.Error(errors.New("Invalid info hash supplied")).SetMeta(errMeta(
			http.StatusNotFound,
			"Invalid info hash supplied",
			log.Fields{
				"info_hash": info_hash,
				"fn":        "HandleTorrentGet",
			},
			log.ErrorLevel,
		))
	} else {
		log.WithFields(log.Fields{
			"info_hash": info_hash,
		}).Debug("Fetched torrent successfully")
		c.JSON(http.StatusOK, torrent)
	}
}

// HandleTorrentAdd Adds new torrents into the active torrent set
// {torrent_id: "", info_hash: "", name: ""}
//
// If the info_hash already exists the torrent_id and name are instead
// updated to reflect the supplied values.
func (tracker *Tracker) HandleTorrentAdd(c *gin.Context) {
	payload := &TorrentAddPayload{}
	if err := c.Bind(payload); err != nil {
		c.Error(err).SetMeta(errMeta(
			http.StatusBadRequest,
			"Failed to parse payload",
			log.Fields{
				"fn": "HandleTorrentAdd",
			},
			log.ErrorLevel,
		))
		return
	}
	var err_msg error
	if payload.TorrentID <= 0 {
		err_msg = errors.New("Invalid torrent id < 0")
	} else if len(payload.InfoHash) != 40 {
		err_msg = errors.New("Invalid info_hash len != 40")
	} else if payload.Name == "" {
		err_msg = errors.New("Invalid release name, cannot be empty")
	}
	if err_msg != nil {
		c.Error(err_msg).SetMeta(errMeta(
			http.StatusBadRequest,
			"Payload requirements not met",
			log.Fields{
				"fn": "HandleTorrentAdd",
			},
			log.ErrorLevel,
		))
		return
	}
	status := http.StatusCreated
	torrent := tracker.FindTorrentByInfoHash(payload.InfoHash)
	if torrent == nil {
		// Add a new one
		torrent = NewTorrent(payload.InfoHash, payload.Name, payload.TorrentID)
		tracker.AddTorrent(torrent)

	} else {
		status = http.StatusAccepted
		// Update our existing one
		// Note only a few entries can be updated at the moment
		torrent.Lock()
		torrent.Enabled = true
		torrent.Name = payload.Name
		torrent.TorrentID = payload.TorrentID
		torrent.Unlock()
	}

	// Queue torrent to be written out to redis
	SyncEntityC <- torrent

	log.WithFields(log.Fields{
		"fn":        "HandleTorrentAdd",
		"name":      payload.Name,
		"info_hash": payload.InfoHash,
	}).Info("Added new torrent successfully")

	c.JSON(status, ResponseOK{"ok"})

}

// HandleTorrentDel will allow the deletion of torrents from the currently active set
// This will not remove the torrent, but instead mark it as deleted.
func (tracker *Tracker) HandleTorrentDel(ctx *gin.Context) {
	info_hash := ctx.Param("info_hash")
	torrent := tracker.FindTorrentByInfoHash(info_hash)
	if torrent == nil {
		ctx.Error(errors.New("Unknown torrent_id supplied")).SetMeta(errMeta(
			http.StatusNotFound,
			"Tried to delete invalid torrent",
			log.Fields{"fn": "HandleTorrentDel", "info_hash": info_hash},
			log.ErrorLevel,
		))
		return
	}

	if tracker.DelTorrent(torrent) {
		ctx.JSON(http.StatusOK, resp_ok)
	} else {
		ctx.Error(errors.New("Cannot re-disable a disabled torrent")).SetMeta(errMeta(
			http.StatusNotFound,
			"Cannot re-disable a disabled torrent",
			log.Fields{"fn": "HandleTorrentDel", "info_hash": info_hash},
			log.ErrorLevel,
		))
		return
	}
}

// getUser is a simple shared function used to fetch the user from a context
// instance automatically.
func (tracker *Tracker) getUser(ctx *gin.Context) (*User, error) {
	user_id_str := ctx.Param("user_id")
	user_id, err := strconv.ParseUint(user_id_str, 10, 64)
	if err != nil {
		log.WithFields(log.Fields{
			"err": err.Error(),
			"fn":  "getUser",
		}).Warn("Invalid user id, malformed")

		return nil, errors.New("Invalid user id")
	}
	tracker.UsersMutex.RLock()
	user, exists := tracker.Users[user_id]
	tracker.UsersMutex.RUnlock()
	if !exists {
		return nil, errors.New("User not found")
	}
	return user, nil
}

// HandleUserTorrents fetches the current set of torrents attached to a user.
// This returns a collection of snatched/hnr/incomplete/complete torrent_ids
func (tracker *Tracker) HandleUserTorrents(ctx *gin.Context) {
	user, err := tracker.getUser(ctx)
	if err != nil {
		ctx.Error(err).SetMeta(errMeta(
			http.StatusNotFound,
			"Failed to find user",
			log.Fields{"fn": "HandleUserTorrents"},
			log.ErrorLevel,
		))
		return
	}
	response := UserTorrentsResponse{}
	r := db.Pool.Get()
	defer r.Close()

	a, err := r.Do("SMEMBERS", user.KeyActive)

	if err != nil {
		ctx.Error(err).SetMeta(errMeta(
			http.StatusInternalServerError,
			"Could not fetch active torrents",
			log.Fields{"fn": "HandleUserTorrents"},
			log.ErrorLevel,
		))
		return
	}
	active_list, err := redis.Strings(a, nil)

	a, err = r.Do("SMEMBERS", user.KeyHNR)
	if err != nil {
		ctx.Error(err).SetMeta(errMeta(
			http.StatusInternalServerError,
			"Could not fetch active hnr's",
			log.Fields{"fn": "HandleUserTorrents"},
			log.ErrorLevel,
		))
		return
	}
	hnr_list, err := redis.Strings(a, nil)

	a, err = r.Do("SMEMBERS", user.KeyComplete)
	if err != nil {
		ctx.Error(err).SetMeta(errMeta(
			http.StatusInternalServerError,
			"Could not fetch complete torrents",
			log.Fields{"fn": "HandleUserTorrents"},
			log.ErrorLevel,
		))
		return
	}
	complete_list, err := redis.Strings(a, nil)

	a, err = r.Do("SMEMBERS", user.KeyIncomplete)
	if err != nil {
		ctx.Error(err).SetMeta(errMeta(
			http.StatusInternalServerError,
			"Could not fetch incomplete torrents",
			log.Fields{"fn": "HandleUserTorrents"},
			log.ErrorLevel,
		))
		return
	}
	incomplete_list, err := redis.Strings(a, nil)

	response.Active = active_list
	response.HNR = hnr_list
	response.Incomplete = incomplete_list
	response.Complete = complete_list

	ctx.JSON(http.StatusOK, response)
}

// HandleUserGet Returns the current representation of the user data struct for
// the requested user_id if available.
func (tracker *Tracker) HandleUserGet(ctx *gin.Context) {
	user, err := tracker.getUser(ctx)
	if err != nil {
		ctx.Error(err).SetMeta(errMeta(
			http.StatusNotFound,
			"Could not fetch user",
			log.Fields{"fn": "HandleUserGet"},
			log.ErrorLevel,
		))
		return
	}
	ctx.JSON(http.StatusOK, user)
}

func (tracker *Tracker) HandleUserDel(ctx *gin.Context) {
	user, err := tracker.getUser(ctx)
	if err != nil {
		ctx.Error(err).SetMeta(errMeta(
			http.StatusNotFound,
			"Could not fetch user",
			log.Fields{"fn": "HandleUserGet"},
			log.ErrorLevel,
		))
		return
	}
	tracker.DelUser(user)
	ctx.JSON(http.StatusOK, resp_ok)
}

// HandleUserCreate facilitates the adding of new users into the trackers memory.
// Mika does not check redis for valid users on each request, so this function
// must be used to add new users into a running system.
func (tracker *Tracker) HandleUserCreate(ctx *gin.Context) {
	payload := &UserCreatePayload{}
	if err := ctx.Bind(payload); err != nil {
		ctx.Error(err).SetMeta(errMeta(
			http.StatusBadRequest,
			"Failed to parse user create json payload",
			log.Fields{"fn": "HandleUserCreate"},
			log.ErrorLevel,
		))
		return
	}
	if payload.Passkey == "" || payload.UserID <= 0 {
		ctx.Error(errors.New("Invalid passkey or userid")).SetMeta(errMeta(
			http.StatusBadRequest,
			"Invalid passkey or userid",
			log.Fields{
				"fn":      "HandleUserCreate",
				"passkey": payload.Passkey,
				"user_id": payload.UserID,
			},
			log.ErrorLevel,
		))
		return
	}

	user := tracker.FindUserByID(payload.UserID)

	if user != nil {
		ctx.Error(errors.New("Tried to add duplicate user")).SetMeta(errMeta(
			http.StatusConflict,
			"Tried to add duplicate user",
			log.Fields{"fn": "HandleUserCreate"},
			log.WarnLevel,
		))
		return
	}

	user = NewUser(payload.UserID)
	user.Lock()
	user.Passkey = payload.Passkey
	user.CanLeech = payload.CanLeech
	user.Username = payload.Name
	user.Unlock()

	SyncEntityC <- user

	tracker.AddUser(user)

	log.WithFields(log.Fields{
		"fn":        "HandleUserCreate",
		"user_name": payload.Name,
		"user_id":   payload.UserID,
	}).Info("Created new user successfully")
	ctx.JSON(http.StatusOK, resp_ok)
}

// HandleUserUpdate will update an existing users data. This is usually used to change
// a users passkey without reloading the instance.
//
// Be aware that there is a race condition regardnig updating these values. When you fetch the user
// initially, your copy of the current data will become out of date upon the next announce or
// periodic update for that user. If you do not update before that happens you may lose
// user stats for that specific delta.
func (tracker *Tracker) HandleUserUpdate(ctx *gin.Context) {
	payload := &UserUpdatePayload{}
	if err := ctx.Bind(payload); err != nil {
		ctx.Error(err).SetMeta(errMeta(
			http.StatusBadRequest,
			"Failed to parse user update json payload",
			log.Fields{"fn": "HandleUserUpdate"},
			log.ErrorLevel,
		))
		return
	}
	user_id_str := ctx.Param("user_id")
	user_id, err := strconv.ParseUint(user_id_str, 10, 64)
	if err != nil {
		ctx.Error(err).SetMeta(errMeta(
			http.StatusBadRequest,
			"Failed to parse user update request",
			log.Fields{"fn": "HandleUserUpdate"},
			log.ErrorLevel,
		))
		return
	}

	tracker.UsersMutex.RLock()
	user, exists := tracker.Users[user_id]
	tracker.UsersMutex.RUnlock()
	if !exists || user == nil {
		ctx.Error(errors.New("Invalid user")).SetMeta(errMeta(
			http.StatusNotFound,
			"User not found, cannot continue",
			log.Fields{"fn": "HandleUserUpdate"},
			log.WarnLevel,
		))
		return
	}

	user.Lock()
	// Let us do partial updates as well
	user.Uploaded = payload.Uploaded
	user.Downloaded = payload.Downloaded
	user.Passkey = payload.Passkey
	user.CanLeech = payload.CanLeech
	user.Enabled = payload.Enabled
	user.Unlock()

	SyncEntityC <- user

	log.WithFields(log.Fields{
		"fn":      "HandleUserUpdate",
		"user_id": user_id,
	}).Info("Updated user successfully")
	ctx.JSON(http.StatusAccepted, resp_ok)
}

// HandleWhitelistAdd facilitates adding new torrent client prefixes to the
// allowed client whitelist
func (tracker *Tracker) HandleWhitelistAdd(ctx *gin.Context) {
	payload := &WhitelistAddPayload{}
	if err := ctx.Bind(payload); err != nil {
		ctx.Error(err).SetMeta(errMeta(
			http.StatusBadRequest,
			"Failed to parse whitelist add json payload",
			log.Fields{"fn": "HandleWhitelistAdd"},
			log.WarnLevel,
		))
		return
	}
	for _, prefix := range tracker.Whitelist {
		if prefix == payload.Prefix {
			ctx.JSON(http.StatusConflict, resp_ok)
			return
		}
	}

	r := db.Pool.Get()
	defer r.Close()

	r.Do("HSET", "t:whitelist", payload.Prefix, payload.Client)
	tracker.initWhitelist(r)
	log.WithFields(log.Fields{
		"fn":     "HandleWhitelistAdd",
		"client": payload.Prefix,
	}).Info("Added new client to whitelist")
	ctx.JSON(http.StatusCreated, resp_ok)
}

// HandleWhitelistDel will remove an existing torrent client prefix from the
// active whitelist.
func (tracker *Tracker) HandleWhitelistDel(ctx *gin.Context) {
	prefix := ctx.Param("prefix")
	for _, p := range tracker.Whitelist {
		if p == prefix {
			r := db.Pool.Get()
			defer r.Close()

			r.Do("HDEL", "t:whitelist", prefix)
			tracker.initWhitelist(r)
			log.WithFields(log.Fields{
				"prefix": prefix,
				"fn":     "HandleWhitelistDel",
			}).Info("Deleted client from whitelist successfully")
			ctx.JSON(http.StatusOK, resp_ok)
		}
	}
	ctx.Error(errors.New("Tried to delete unknown client prefix")).SetMeta(errMeta(
		http.StatusNotFound,
		"Tried to delete unknown client prefix",
		log.Fields{"fn": "HandleWhitelistDel", "prefix": prefix},
		log.WarnLevel,
	))
}

// HandleGetTorrentPeer Fetch details for a specific peer of a torrent
func (tracker *Tracker) HandleGetTorrentPeer(c *gin.Context) {
}

// HandleGetTorrentPeers returns all peers for a given info_hash
func (tracker *Tracker) HandleGetTorrentPeers(ctx *gin.Context) {
	info_hash := ctx.Param("info_hash")
	torrent := tracker.FindTorrentByInfoHash(info_hash)
	if torrent == nil {
		ctx.Error(errors.New("Invalid torrent")).SetMeta(errMeta(
			http.StatusNotFound,
			"Invalid torrent",
			log.Fields{"fn": "HandleGetTorrentPeers", "info_hash": info_hash},
			log.WarnLevel,
		))
		return
	}
	log.WithFields(log.Fields{
		"info_hash": info_hash,
		"fn":        "HandleGetTorrentPeers",
	}).Debug("Got torrent peers successfully")
	ctx.JSON(http.StatusOK, torrent.Peers)
}
