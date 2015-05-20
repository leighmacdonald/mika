package tracker

import (
	"errors"
	"git.totdev.in/totv/echo.git"
	"git.totdev.in/totv/mika"
	"git.totdev.in/totv/mika/db"
	"git.totdev.in/totv/mika/util"
	log "github.com/Sirupsen/logrus"
	"github.com/garyburd/redigo/redis"
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
		UserPayload
		UserCreatePayload
		CanLeech   bool   `json:"can_leech"`
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
func (t *Tracker) HandleVersion(c *echo.Context) *echo.HTTPError {
	return c.JSON(http.StatusOK, VersionResponse{Name: "mika", Version: mika.Version})
}

// HandleUptime returns the current process uptime
func (t *Tracker) HandleUptime(c *echo.Context) *echo.HTTPError {
	info := &syscall.Sysinfo_t{}
	err := syscall.Sysinfo(info)
	if err != nil {
		return &echo.HTTPError{
			Code:    http.StatusInternalServerError,
			Fields:  log.Fields{"fn": "HandleUptime"},
			Error:   err,
			Message: "Error trying to fetch sysinfo",
		}
	}
	return c.JSON(http.StatusOK, UptimeResponse{
		Process: util.Unixtime() - mika.StartTime,
		System:  int32(info.Uptime),
	})
}

// HandleTorrentGet will find and return the requested torrent.
func (t *Tracker) HandleTorrentGet(c *echo.Context) *echo.HTTPError {
	info_hash := c.Param("info_hash")
	torrent := t.FindTorrentByInfoHash(info_hash)
	if torrent == nil {
		return &echo.HTTPError{
			Code:    http.StatusNotFound,
			Message: "Invalid info hash supplied",
		}
	} else {
		log.WithFields(log.Fields{
			"info_hash": info_hash,
		}).Debug("Fetched torrent successfully")
		err := c.JSON(http.StatusOK, torrent)
		if err != nil {
			return &echo.HTTPError{
				Code:    http.StatusInternalServerError,
				Error:   err.Error,
				Message: "Failed to encode torrent data",
				Fields: log.Fields{
					"fn": "HandleTorrentGet",
				},
			}
		}
	}
	return nil
}

// HandleTorrentAdd Adds new torrents into the active torrent set
// {torrent_id: "", info_hash: "", name: ""}
func (t *Tracker) HandleTorrentAdd(c *echo.Context) *echo.HTTPError {
	payload := &TorrentAddPayload{}
	if err := c.Bind(payload); err != nil {
		return &echo.HTTPError{
			Code:    http.StatusBadRequest,
			Error:   err.Error,
			Message: "Failed to parse payload",
			Fields:  log.Fields{"fn": "HandleTorrentGet"},
		}
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
		return &echo.HTTPError{
			Code:    http.StatusBadRequest,
			Error:   err_msg,
			Message: "Payload requirements not met",
			Fields:  log.Fields{"fn": "HandleTorrentGet"},
		}
	}
	torrent := t.FindTorrentByInfoHash(payload.InfoHash)
	if torrent == nil {
		// Add a new one
		torrent = NewTorrent(payload.InfoHash, payload.Name, payload.TorrentID)
		t.AddTorrent(torrent)
	} else {
		// Update our existing one
		// Note only a few entries can be updated at the moment
		torrent.Lock()
		torrent.Enabled = true
		torrent.Name = payload.Name
		torrent.TorrentID = payload.TorrentID
		torrent.Unlock()
	}

	// Queue torrent to be written out to redis
	SyncTorrentC <- torrent

	log.WithFields(log.Fields{
		"fn":        "HandleTorrentAdd",
		"name":      payload.Name,
		"info_hash": payload.InfoHash,
	}).Info("Added new torrent successfully")

	return c.JSON(http.StatusCreated, resp_ok)
}

// HandleTorrentDel will allow the deletion of torrents from the currently active set
// This will not remove the torrent, but instead mark it as deleted.
func (t *Tracker) HandleTorrentDel(c *echo.Context) *echo.HTTPError {
	info_hash := c.Param("info_hash")
	torrent := t.FindTorrentByInfoHash(info_hash)
	if torrent == nil {
		return &echo.HTTPError{
			Code:    http.StatusNotFound,
			Fields:  log.Fields{"fn": "HandleTorrentDel", "info_hash": info_hash},
			Error:   errors.New("Unknown torrent_id supplied"),
			Message: "Tried to delete invalid torrent",
		}
	}
	torrent.Lock()
	torrent.Enabled = false
	torrent.Unlock()
	if !torrent.InQueue {
		torrent.InQueue = true
		SyncTorrentC <- torrent
	}
	log.WithFields(log.Fields{
		"fn":        "HandleTorrentDel",
		"info_hash": info_hash,
	}).Info("Deleted torrent successfully")

	return c.JSON(http.StatusOK, resp_ok)
}

// getUser is a simple shared function used to fetch the user from a context
// instance automatically.
func (t *Tracker) getUser(c *echo.Context) (*User, error) {
	user_id_str := c.Param("user_id")
	user_id, err := strconv.ParseUint(user_id_str, 10, 64)
	if err != nil {
		log.WithFields(log.Fields{
			"err": err.Error(),
			"fn":  "getUser",
		}).Warn("Invalid user id, malformed")

		return nil, errors.New("Invalid user id")
	}
	t.UsersMutex.RLock()
	user, exists := t.Users[user_id]
	t.UsersMutex.RUnlock()
	if !exists {
		return nil, errors.New("User not found")
	}
	return user, nil
}

// HandleUserTorrents fetches the current set of torrents attached to a user.
// This returns a collection of snatched/hnr/incomplete/complete torrent_ids
func (t *Tracker) HandleUserTorrents(c *echo.Context) *echo.HTTPError {
	user, err := t.getUser(c)
	if err != nil {
		return &echo.HTTPError{
			Code:    http.StatusNotFound,
			Error:   err,
			Message: "Failed to find user",
			Fields:  log.Fields{"fn": "getUser"},
		}
	}
	response := UserTorrentsResponse{}
	r := db.Pool.Get()
	defer r.Close()

	a, err := r.Do("SMEMBERS", user.KeyActive)

	if err != nil {
		return &echo.HTTPError{
			Code:    http.StatusInternalServerError,
			Message: "Could not fetch active torrents",
			Error:   err,
			Fields:  log.Fields{"fn": "HandleUserTorrents"},
		}
	}
	active_list, err := redis.Strings(a, nil)

	a, err = r.Do("SMEMBERS", user.KeyHNR)
	if err != nil {
		return &echo.HTTPError{
			Code:    http.StatusInternalServerError,
			Message: "Could not fetch active hnr's",
			Error:   err,
			Fields:  log.Fields{"fn": "HandleUserTorrents"},
		}
	}
	hnr_list, err := redis.Strings(a, nil)

	a, err = r.Do("SMEMBERS", user.KeyComplete)
	if err != nil {
		return &echo.HTTPError{
			Code:    http.StatusInternalServerError,
			Message: "Could not fetch complete torrents",
			Error:   err,
			Fields:  log.Fields{"fn": "HandleUserTorrents"},
		}
	}
	complete_list, err := redis.Strings(a, nil)

	a, err = r.Do("SMEMBERS", user.KeyIncomplete)
	if err != nil {
		return &echo.HTTPError{
			Code:    http.StatusInternalServerError,
			Message: "Could not fetch incomplete torrents",
			Error:   err,
			Fields:  log.Fields{"fn": "HandleUserTorrents"},
		}
	}
	incomplete_list, err := redis.Strings(a, nil)

	response.Active = active_list
	response.HNR = hnr_list
	response.Incomplete = incomplete_list
	response.Complete = complete_list

	return c.JSON(http.StatusOK, response)
}

// HandleUserGet Returns the current representation of the user data struct for
// the requested user_id if available.
func (t *Tracker) HandleUserGet(c *echo.Context) *echo.HTTPError {
	user, err := t.getUser(c)
	if err != nil {
		return &echo.HTTPError{
			Code:    http.StatusNotFound,
			Error:   err,
			Message: "Could not fetch user",
			Fields:  log.Fields{"fn": "HandleUserGet"},
		}
	}
	return c.JSON(http.StatusOK, user)
}

// HandleUserCreate facilitates the adding of new users into the trackers memory.
// Mika does not check redis for valid users on each request, so this function
// must be used to add new users into a running system.
func (t *Tracker) HandleUserCreate(c *echo.Context) *echo.HTTPError {
	payload := &UserCreatePayload{}
	if err := c.Bind(payload); err != nil {
		return &echo.HTTPError{
			Code:    http.StatusBadRequest,
			Error:   err.Error,
			Message: "Failed to parse user create json payload",
			Fields:  log.Fields{"fn": "HandleUserCreate"},
		}
	}
	if payload.Passkey == "" || payload.UserID <= 0 {
		return &echo.HTTPError{
			Code:    http.StatusBadRequest,
			Message: "Invalid passkey or userid",
			Fields: log.Fields{
				"fn":      "HandleUserCreate",
				"passkey": payload.Passkey,
				"user_id": payload.UserID,
			},
		}
	}

	user := t.FindUserByID(payload.UserID)

	if user != nil {
		return &echo.HTTPError{
			Fields: log.Fields{"fn": "HandleUserCreate"},
			Code:   http.StatusConflict,
			Error:  errors.New("Tried to add duplicate user"),
			Level:  log.WarnLevel,
		}
	}

	user = NewUser(payload.UserID)
	user.Lock()
	user.Passkey = payload.Passkey
	user.CanLeech = payload.CanLeech
	user.Username = payload.Name
	if !user.InQueue {
		user.InQueue = true
		user.Unlock()
		SyncUserC <- user
	} else {
		user.Unlock()
	}
	log.WithFields(log.Fields{
		"fn":        "HandleUserCreate",
		"user_name": payload.Name,
		"user_id":   payload.UserID,
	}).Info("Created new user successfully")
	return c.JSON(http.StatusOK, resp_ok)
}

// HandleUserUpdate will update an existing users data. This is usually used to change
// a users passkey without reloading the instance.
func (t *Tracker) HandleUserUpdate(c *echo.Context) *echo.HTTPError {
	payload := &UserUpdatePayload{}
	if err := c.Bind(payload); err != nil {
		return &echo.HTTPError{
			Fields:  log.Fields{"fn": "HandleUserUpdate"},
			Message: "Failed to parse user update json payload",
			Error:   err.Error,
			Code:    http.StatusBadRequest,
		}
	}
	user_id_str := c.Param("user_id")
	user_id, err := strconv.ParseUint(user_id_str, 10, 64)
	if err != nil {
		return &echo.HTTPError{
			Message: "Failed to parse user update request",
			Error:   err,
			Code:    http.StatusBadRequest,
			Fields:  log.Fields{"fn": "HandleUserUpdate"},
		}
	}

	t.UsersMutex.RLock()
	user, exists := t.Users[user_id]
	t.UsersMutex.RUnlock()
	if !exists {
		return &echo.HTTPError{
			Code:    http.StatusNotFound,
			Fields:  log.Fields{"fn": "HandleUserUpdate"},
			Message: "User not found, cannot continue",
			Level:   log.WarnLevel,
		}
	}

	user.Lock()
	// Let us do partial updates as well
	user.Uploaded = payload.Uploaded
	user.Downloaded = payload.Downloaded
	user.Passkey = payload.Passkey
	user.CanLeech = payload.CanLeech
	user.Enabled = payload.Enabled

	if !user.InQueue {
		user.InQueue = true
		user.Unlock()
		SyncUserC <- user
	} else {
		user.Unlock()
	}
	log.WithFields(log.Fields{
		"fn":      "HandleUserUpdate",
		"user_id": user_id,
	}).Info("Updated user successfully")
	return c.JSON(http.StatusOK, resp_ok)
}

// HandleWhitelistAdd facilitates adding new torrent client prefixes to the
// allowed client whitelist
func (t *Tracker) HandleWhitelistAdd(c *echo.Context) *echo.HTTPError {
	payload := &WhitelistAddPayload{}
	if err := c.Bind(payload); err != nil {
		return &echo.HTTPError{
			Code: http.StatusBadRequest,
			Fields: log.Fields{
				"fn":  "HandleWhitelistAdd",
				"err": err.Message,
			},
			Message: "Failed to parse whitelist add json payload",
			Error:   err.Error,
		}
	}
	for _, prefix := range t.Whitelist {
		if prefix == payload.Prefix {
			return c.JSON(http.StatusConflict, resp_ok)
		}
	}

	r := db.Pool.Get()
	defer r.Close()

	r.Do("HSET", "t:whitelist", payload.Prefix, payload.Client)
	t.initWhitelist(r)
	log.WithFields(log.Fields{
		"fn":     "HandleWhitelistAdd",
		"client": payload.Prefix,
	}).Info("Added new client to whitelist")
	return c.JSON(http.StatusCreated, resp_ok)
}

// HandleWhitelistDel will remove an existing torrent client prefix from the
// active whitelist.
func (t *Tracker) HandleWhitelistDel(c *echo.Context) *echo.HTTPError {
	prefix := c.Param("prefix")
	for _, p := range t.Whitelist {
		if p == prefix {
			r := db.Pool.Get()
			defer r.Close()

			r.Do("HDEL", "t:whitelist", prefix)
			t.initWhitelist(r)
			log.WithFields(log.Fields{
				"prefix": prefix,
				"fn":     "HandleWhitelistDel",
			}).Info("Deleted client from whitelist successfully")
			return c.JSON(http.StatusOK, resp_ok)
		}
	}
	return &echo.HTTPError{
		Fields:  log.Fields{"prefix": prefix, "fn": "HandleWhitelistDel"},
		Level:   log.WarnLevel,
		Error:   errors.New("Tried to remove unknown client prefix"),
		Message: "Failed to remove client from whitelist",
		Code:    http.StatusNotFound,
	}
}

// HandleGetTorrentPeer Fetch details for a specific peer of a torrent
func (t *Tracker) HandleGetTorrentPeer(c *echo.Context) *echo.HTTPError {
	return &echo.HTTPError{
		Code:    http.StatusNotFound,
		Message: "Hi! :)",
	}
}

// HandleGetTorrentPeers returns all peers for a given info_hash
func (t *Tracker) HandleGetTorrentPeers(c *echo.Context) *echo.HTTPError {
	info_hash := c.Param("info_hash")
	torrent := t.FindTorrentByInfoHash(info_hash)
	if torrent == nil {
		return &echo.HTTPError{
			Code: http.StatusNotFound,
			Fields: log.Fields{
				"info_hash": info_hash,
				"fn":        "HandleGetTorrentPeers",
			},
			Message: "Could not fetch peers from torrent",
			Error:   errors.New("Requested unknown info_hash"),
		}
	}
	log.WithFields(log.Fields{
		"info_hash": info_hash,
		"fn":        "HandleGetTorrentPeers",
	}).Debug("Got torrent peers successfully")
	return c.JSON(http.StatusOK, torrent.Peers)
}
