package main

import (
	"errors"
	"fmt"
	"github.com/garyburd/redigo/redis"
	"github.com/labstack/echo"
	"log"
	"net/http"
	"strconv"
)

type Response struct {
}

type ResponseErr struct {
	Msg    string `json:"msg"`
	Status int    `json:"status"`
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

func HandleVersion(c *echo.Context) {
	c.String(http.StatusOK, fmt.Sprintf("mika/%s", version))
}

func HandleTorrentGet(c *echo.Context) error {
	r := pool.Get()
	defer r.Close()
	if r.Err() != nil {
		return c.JSON(http.StatusInternalServerError, ResponseErr{})
	}

	info_hash := c.Param("info_hash")
	torrent := mika.GetTorrentByInfoHash(r, info_hash, false)
	if torrent == nil {
		return c.JSON(http.StatusNotFound, ResponseErr{"Unknown info hash", 404})
	}
	log.Println("HandleTorrentGet: Fetched torrent", info_hash)
	return c.JSON(http.StatusOK, torrent)
}

func HandleTorrentAdd(c *echo.Context) error {
	payload := &TorrentAddPayload{}
	if err := c.Bind(payload); err != nil {
		return err
	}
	if payload.TorrentID <= 0 {
		return errors.New("Invalid torrent id")
	} else if len(payload.InfoHash) != 40 {
		return errors.New("Invalid info hash")
	}
	r := pool.Get()
	defer r.Close()

	torrent := mika.GetTorrentByInfoHash(r, payload.InfoHash, false)
	mika.TorrentsMutex.Lock()
	if torrent != nil {
		torrent.Enabled = true
	} else {
		torrent = &Torrent{
			InfoHash:  payload.InfoHash,
			Name:      payload.Name,
			TorrentID: payload.TorrentID,
			Enabled:   true,
			Peers:     []*Peer{},
		}
		mika.Torrents[payload.InfoHash] = torrent
		mika.TorrentsMutex.Unlock()
	}
	sync_torrent <- torrent

	log.Println("HandleTorrentAdd: Added new torrent:", payload.Name)
	return c.JSON(http.StatusCreated, Response{})
}

func HandleTorrentDel(c *echo.Context) error {
	r := pool.Get()
	defer r.Close()
	if r.Err() != nil {
		return c.JSON(http.StatusInternalServerError, ResponseErr{})
	}

	info_hash := c.Param("info_hash")
	torrent := mika.GetTorrentByInfoHash(r, info_hash, false)
	if torrent == nil {
		return c.JSON(http.StatusNotFound, ResponseErr{"Invalid torrent_id", http.StatusNotFound})
	}
	torrent.Lock()
	torrent.Enabled = false
	torrent.Unlock()
	if !torrent.InQueue {
		torrent.InQueue = true
		sync_torrent <- torrent
	}
	log.Println("HandleTorrentDel: Deleted torrent", info_hash)
	return c.JSON(http.StatusOK, ResponseErr{"ok", http.StatusOK})
}

func getUser(c *echo.Context) *User {
	user_id_str := c.Param("user_id")
	user_id, err := strconv.ParseUint(user_id_str, 10, 64)
	if err != nil {
		Debug("getUser: ", err)
		c.JSON(http.StatusBadRequest, ResponseErr{"Invalid user id", 0})
		return nil
	}
	mika.UsersMutex.RLock()
	user, exists := mika.Users[user_id]
	mika.UsersMutex.RUnlock()
	if !exists {
		c.JSON(http.StatusNotFound, ResponseErr{"User not Found", 404})
		return nil
	}
	return user
}

func HandleUserTorrents(c *echo.Context) error {
	user := getUser(c)
	if user == nil {
		return nil
	}
	response := UserTorrentsResponse{}
	r := pool.Get()
	defer r.Close()

	a, err := r.Do("SMEMBERS", user.KeyActive)

	if err != nil {
		log.Println("HandleUserTorrents: Failed to fetch user active", err)
		return nil
	}
	active_list, err := redis.Strings(a, nil)

	a, err = r.Do("SMEMBERS", user.KeyHNR)
	if err != nil {
		log.Println("HandleUserTorrents: Failed to fetch user HNR", err)
		return nil
	}
	hnr_list, err := redis.Strings(a, nil)

	a, err = r.Do("SMEMBERS", user.KeyComplete)
	if err != nil {
		log.Println("HandleUserTorrents: Failed to fetch user completes", err)
		return nil
	}
	complete_list, err := redis.Strings(a, nil)

	a, err = r.Do("SMEMBERS", user.KeyIncomplete)
	if err != nil {
		log.Println("HandleUserTorrents: Failed to fetch user incompletes", err)
		return nil
	}
	incomplete_list, err := redis.Strings(a, nil)

	response.Active = active_list
	response.HNR = hnr_list
	response.Incomplete = incomplete_list
	response.Complete = complete_list

	return c.JSON(http.StatusOK, response)
}

func HandleUserGet(c *echo.Context) error {
	user := getUser(c)
	if user != nil {
		return c.JSON(http.StatusOK, user)
	}
	return nil
}

func HandleUserCreate(c *echo.Context) error {
	payload := &UserCreatePayload{}
	if err := c.Bind(payload); err != nil {
		return err
	}
	if payload.Passkey == "" || payload.UserID <= 0 {
		return c.JSON(http.StatusBadRequest, ResponseErr{"Invalid user id", 0})
	}
	r := pool.Get()
	defer r.Close()
	user := GetUserByID(r, payload.UserID, false)

	if user != nil {
		return c.JSON(http.StatusConflict, ResponseErr{"User exists", http.StatusConflict})
	}

	user = GetUserByID(r, payload.UserID, true)
	user.Lock()
	user.Passkey = payload.Passkey
	user.CanLeech = payload.CanLeech
	user.Enabled = true
	user.Username = payload.Name
	if !user.InQueue {
		user.InQueue = true
		user.Unlock()
		sync_user <- user
	} else {
		user.Unlock()
	}
	log.Println("HandleUserCreate: Created new user", fmt.Sprintf("[%d/%s]", payload.UserID, payload.Name))
	return c.JSON(http.StatusOK, ResponseErr{"ok", 200})
}

func HandleUserUpdate(c *echo.Context) error {
	payload := &UserUpdatePayload{}
	if err := c.Bind(payload); err != nil {
		return err
	}
	user_id_str := c.Param("user_id")
	user_id, err := strconv.ParseUint(user_id_str, 10, 64)
	if err != nil {
		Debug("HandleUserUpdate: Failed to parse user id", err)
		return c.JSON(http.StatusBadRequest, ResponseErr{"Invalid user id format", 0})
	}

	mika.UsersMutex.RLock()
	user, exists := mika.Users[user_id]
	mika.UsersMutex.RUnlock()
	if !exists {
		return c.JSON(http.StatusNotFound, ResponseErr{"User not Found", 404})
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
		sync_user <- user
	} else {
		user.Unlock()
	}
	log.Println("HandleUserUpdate: Updated user", user_id)
	return c.JSON(http.StatusOK, ResponseErr{"ok", 200})
}

func HandleWhitelistAdd(c *echo.Context) error {
	payload := &WhitelistAddPayload{}
	if err := c.Bind(payload); err != nil {
		return err
	}
	for _, prefix := range whitelist {
		if prefix == payload.Prefix {
			return c.JSON(http.StatusConflict, ResponseErr{"ok", http.StatusConflict})
		}
	}

	r := pool.Get()
	defer r.Close()

	r.Do("HSET", "t:whitelist", payload.Prefix, payload.Client)
	mika.initWhitelist(r)
	log.Println("HandleWhitelistAdd: Added new client to whitelist", payload.Prefix)
	return c.JSON(http.StatusCreated, ResponseErr{"ok", http.StatusCreated})
}

func HandleWhitelistDel(c *echo.Context) error {
	prefix := c.Param("prefix")
	for _, p := range whitelist {
		if p == prefix {
			r := pool.Get()
			defer r.Close()

			r.Do("HDEL", "t:whitelist", prefix)
			mika.initWhitelist(r)
			log.Println("HandleWhitelistDel: Deleted client from whitelist", prefix)
			return c.JSON(http.StatusOK, ResponseErr{"ok", http.StatusOK})
		}
	}
	return c.JSON(http.StatusNotFound, ResponseErr{"User not Found", 404})
}

func HandleGetTorrentPeer(c *echo.Context) error {
	return c.JSON(http.StatusOK, ResponseErr{"Nope! :(", 200})
}

func HandleGetTorrentPeers(c *echo.Context) error {
	r := pool.Get()
	defer r.Close()
	if r.Err() != nil {
		return c.JSON(http.StatusInternalServerError, ResponseErr{})
	}

	info_hash := c.Param("info_hash")
	torrent := mika.GetTorrentByInfoHash(r, info_hash, false)
	if torrent == nil {
		return c.JSON(http.StatusNotFound, ResponseErr{})
	}
	Debug("HandleGetTorrentPeers: Got torrent peers", info_hash)
	return c.JSON(http.StatusOK, torrent.Peers)
}
