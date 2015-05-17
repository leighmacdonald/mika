package tracker

import (
	"fmt"
	"git.totdev.in/totv/mika"
	"git.totdev.in/totv/mika/db"
	"git.totdev.in/totv/mika/util"
	log "github.com/Sirupsen/logrus"
	"github.com/garyburd/redigo/redis"
	"github.com/labstack/echo"
	"net/http"
	"strconv"
)

type ResponseOK struct {
	Msg string `json:"message"`
}

type ResponseErr struct {
	Err string `json:"error"`
}

type UserPayload struct {
	UserID uint64 `json:"user_id"`
}

type UserCreatePayload struct {
	UserPayload
	Passkey  string `json:"passkey"`
	CanLeech bool   `json:"can_leech"`
	Name     string `json:"name"`
}

type UserUpdatePayload struct {
	UserPayload
	UserCreatePayload
	CanLeech   bool   `json:"can_leech"`
	Downloaded uint64 `json:"downloaded"`
	Uploaded   uint64 `json:"uploaded"`
	Enabled    bool   `json:"enabled"`
}

type TorrentPayload struct {
	TorrentID uint64 `json:"torrent_id"`
}

type TorrentAddPayload struct {
	TorrentPayload
	InfoHash string `json:"info_hash"`
	Name     string `json:"name"`
}

type UserTorrentsResponse struct {
	Active     []string `json:"active"`
	HNR        []string `json:"hnr"`
	Complete   []string `json:"complete"`
	Incomplete []string `json:"incomplete"`
}

type TorrentDelPayload struct {
	TorrentPayload
	Reason string
}
type WhitelistPayload struct {
	Prefix string `json:"prefix"`
}

type WhitelistAddPayload struct {
	WhitelistPayload
	Client string `json:"client"`
}

var (
	resp_ok = ResponseOK{"ok"}
)

func (t *Tracker) HandleVersion(c *echo.Context) {
	c.String(http.StatusOK, mika.VersionStr())
}

func (t *Tracker) HandleUptime(c *echo.Context) {
	c.String(http.StatusOK, fmt.Sprintf("%d", util.Unixtime()-mika.StartTime))
}

func (t *Tracker) HandleTorrentGet(c *echo.Context) *echo.HTTPError {
	info_hash := c.Param("info_hash")
	torrent := t.FindTorrentByInfoHash(info_hash)
	if torrent == nil {
		err := c.JSON(http.StatusNotFound, ResponseErr{"Unknown info hash"})
		if err != nil {
			log.Error("ERR: ", err)
		}
	} else {
		log.Debug("HandleTorrentGet: Fetched torrent", info_hash)
		err := c.JSON(http.StatusOK, torrent)
		if err != nil {
			log.Error("ERR2: ", err)
			log.Error(torrent)
		}
	}
	return nil
}

// Add new torrents into the active torrent set
// {torrent_id: "", info_hash: "", name: ""}
func (t *Tracker) HandleTorrentAdd(c *echo.Context) *echo.HTTPError {
	payload := &TorrentAddPayload{}
	if err := c.Bind(payload); err != nil {
		log.WithFields(log.Fields{
			"err": err.Message,
		}).Error("Failed to parse addtorrent payload")

		return &echo.HTTPError{Code: http.StatusBadRequest}
	}
	if payload.TorrentID <= 0 {
		return &echo.HTTPError{Code: http.StatusBadRequest}
	} else if len(payload.InfoHash) != 40 {
		return &echo.HTTPError{Code: http.StatusBadRequest}
	} else if payload.Name == "" {
		return &echo.HTTPError{
			Code:    http.StatusBadRequest,
			Message: "Invalid release name, cannot be empty",
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

	SyncTorrentC <- torrent

	log.WithFields(log.Fields{
		"fn":        "HandleTorrentAdd",
		"name":      payload.Name,
		"info_hash": payload.InfoHash,
	}).Info("Added new torrent successfully")

	return c.JSON(http.StatusCreated, resp_ok)
}

func (t *Tracker) HandleTorrentDel(c *echo.Context) *echo.HTTPError {
	info_hash := c.Param("info_hash")
	torrent := t.FindTorrentByInfoHash(info_hash)
	if torrent == nil {
		log.WithFields(log.Fields{
			"fn":        "HandleTorrentDel",
			"info_hash": info_hash,
		}).Warn("Tried to delete invalid torrent")
		return c.JSON(http.StatusNotFound, ResponseErr{"Invalid torrent_id"})
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

func (t *Tracker) getUser(c *echo.Context) *User {
	user_id_str := c.Param("user_id")
	user_id, err := strconv.ParseUint(user_id_str, 10, 64)
	if err != nil {
		log.Debug("getUser: ", err)
		c.JSON(http.StatusBadRequest, ResponseErr{"Invalid user id"})
		return nil
	}
	t.UsersMutex.RLock()
	user, exists := t.Users[user_id]
	t.UsersMutex.RUnlock()
	if !exists {
		c.JSON(http.StatusNotFound, ResponseErr{"User not Found"})
		return nil
	}
	return user
}

func (t *Tracker) HandleUserTorrents(c *echo.Context) *echo.HTTPError {
	user := t.getUser(c)
	if user == nil {
		return nil
	}
	response := UserTorrentsResponse{}
	r := db.Pool.Get()
	defer r.Close()

	a, err := r.Do("SMEMBERS", user.KeyActive)

	if err != nil {
		log.Error("HandleUserTorrents: Failed to fetch user active", err)
		c.JSON(http.StatusInternalServerError, ResponseErr{"Error fetching user active torrents"})
		return nil
	}
	active_list, err := redis.Strings(a, nil)

	a, err = r.Do("SMEMBERS", user.KeyHNR)
	if err != nil {
		log.Error("HandleUserTorrents: Failed to fetch user HNR", err)
		return c.JSON(http.StatusInternalServerError, ResponseErr{"Error fetching user hnr torrents"})
	}
	hnr_list, err := redis.Strings(a, nil)

	a, err = r.Do("SMEMBERS", user.KeyComplete)
	if err != nil {
		log.Error("HandleUserTorrents: Failed to fetch user completes", err)
		return c.JSON(http.StatusInternalServerError, ResponseErr{"Error fetching user completed torrents"})
	}
	complete_list, err := redis.Strings(a, nil)

	a, err = r.Do("SMEMBERS", user.KeyIncomplete)
	if err != nil {
		log.Error("HandleUserTorrents: Failed to fetch user incompletes", err)
		return c.JSON(http.StatusInternalServerError, ResponseErr{"Error fetching user incompleted torrents"})
	}
	incomplete_list, err := redis.Strings(a, nil)

	response.Active = active_list
	response.HNR = hnr_list
	response.Incomplete = incomplete_list
	response.Complete = complete_list

	return c.JSON(http.StatusOK, response)
}

func (t *Tracker) HandleUserGet(c *echo.Context) *echo.HTTPError {
	user := t.getUser(c)
	if user != nil {
		return c.JSON(http.StatusOK, user)
	}
	return nil
}

func (t *Tracker) HandleUserCreate(c *echo.Context) *echo.HTTPError {
	payload := &UserCreatePayload{}
	if err := c.Bind(payload); err != nil {
		return err
	}
	if payload.Passkey == "" || payload.UserID <= 0 {
		return c.JSON(http.StatusBadRequest, ResponseErr{"Invalid user id"})
	}

	user := t.FindUserByID(payload.UserID)

	if user != nil {
		return c.JSON(http.StatusConflict, ResponseErr{"User exists"})
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
	log.Info("HandleUserCreate: Created new user", fmt.Sprintf("[%d/%s]", payload.UserID, payload.Name))
	return c.JSON(http.StatusOK, resp_ok)
}

func (t *Tracker) HandleUserUpdate(c *echo.Context) *echo.HTTPError {
	payload := &UserUpdatePayload{}
	if err := c.Bind(payload); err != nil {
		return err
	}
	user_id_str := c.Param("user_id")
	user_id, err := strconv.ParseUint(user_id_str, 10, 64)
	if err != nil {
		log.Debug("HandleUserUpdate: Failed to parse user id", err)
		return c.JSON(http.StatusBadRequest, ResponseErr{"Invalid user id format"})
	}

	t.UsersMutex.RLock()
	user, exists := t.Users[user_id]
	t.UsersMutex.RUnlock()
	if !exists {
		return c.JSON(http.StatusNotFound, ResponseErr{"User not Found"})
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
	log.Info("HandleUserUpdate: Updated user", user_id)
	return c.JSON(http.StatusOK, resp_ok)
}

func (t *Tracker) HandleWhitelistAdd(c *echo.Context) *echo.HTTPError {
	payload := &WhitelistAddPayload{}
	if err := c.Bind(payload); err != nil {
		return &echo.HTTPError{Code: http.StatusBadRequest}
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
	log.Info("HandleWhitelistAdd: Added new client to whitelist", payload.Prefix)
	return c.JSON(http.StatusCreated, resp_ok)
}

func (t *Tracker) HandleWhitelistDel(c *echo.Context) *echo.HTTPError {
	prefix := c.Param("prefix")
	for _, p := range t.Whitelist {
		if p == prefix {
			r := db.Pool.Get()
			defer r.Close()

			r.Do("HDEL", "t:whitelist", prefix)
			t.initWhitelist(r)
			log.Info("HandleWhitelistDel: Deleted client from whitelist", prefix)
			return c.JSON(http.StatusOK, resp_ok)
		}
	}
	return c.JSON(http.StatusNotFound, ResponseErr{"User not Found"})
}

func (t *Tracker) HandleGetTorrentPeer(c *echo.Context) *echo.HTTPError {
	return c.JSON(http.StatusOK, ResponseErr{"Nope! :("})
}

func (t *Tracker) HandleGetTorrentPeers(c *echo.Context) *echo.HTTPError {
	info_hash := c.Param("info_hash")
	torrent := t.FindTorrentByInfoHash(info_hash)
	if torrent == nil {
		return c.JSON(http.StatusNotFound, ResponseErr{})
	}
	log.Debug("HandleGetTorrentPeers: Got torrent peers", info_hash)
	return c.JSON(http.StatusOK, torrent.Peers)
}
