package main

import (
	"errors"
	"fmt"
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
	Passkey string `json:"passkey"`
}

type UserUpdatePayload struct {
	UserPayload
	UserCreatePayload
	CanLeech   bool   `json:"can_leech"`
	Downloaded uint64 `json:"downloaded"`
	Uploaded   uint64 `json:"uploaded"`
}

type TorrentPayload struct {
	TorrentID uint64 `json:"torrent_id"`
}

type TorrentAddPayload struct {
	TorrentPayload
	InfoHash string `json:"info_hash"`
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
	r := getRedisConnection()
	defer returnRedisConnection(r)
	if r.Err() != nil {
		return c.JSON(http.StatusInternalServerError, ResponseErr{})
	}

	torrent_id_str := c.Param("torrent_id")
	torrent_id, err := strconv.ParseUint(torrent_id_str, 10, 64)
	if err != nil {
		Debug(err)
		return c.JSON(http.StatusNotFound, ResponseErr{})
	}
	torrent := mika.GetTorrentByID(r, torrent_id, false)
	if torrent == nil {
		return c.JSON(http.StatusNotFound, ResponseErr{})
	}
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
	r := getRedisConnection()
	defer returnRedisConnection(r)

	_, err := r.Do("SET", payload.InfoHash, payload.TorrentID)
	if err != nil {
		return errors.New("Failed to insert torrent")
	}

	torrent := mika.GetTorrentByID(r, payload.TorrentID, true)
	torrent.Enabled = true

	log.Println("Added new torrent:", payload)
	return c.JSON(http.StatusCreated, Response{})
}

func HandleTorrentDel(c *echo.Context) error {
	r := getRedisConnection()
	defer returnRedisConnection(r)
	if r.Err() != nil {
		return c.JSON(http.StatusInternalServerError, ResponseErr{})
	}

	torrent_id_str := c.Param("torrent_id")
	torrent_id, err := strconv.ParseUint(torrent_id_str, 10, 64)
	if err != nil {
		Debug(err)
		return c.JSON(http.StatusNotFound, ResponseErr{})
	}
	torrent := mika.GetTorrentByID(r, torrent_id, false)
	if torrent == nil {
		return c.JSON(http.StatusNotFound, ResponseErr{"Invalid torrent_id", http.StatusNotFound})
	}
	mika.Lock()
	torrent.Enabled = false
	mika.Unlock()
	if !torrent.InQueue {
		torrent.InQueue = true
		sync_torrent <- torrent
	}

	return c.JSON(http.StatusOK, ResponseErr{"ok", http.StatusOK})
}

func HandleUserGetActive(c *echo.Context) {

}

func HandleUserGet(c *echo.Context) error {
	user_id_str := c.Param("user_id")
	user_id, err := strconv.ParseUint(user_id_str, 10, 64)
	if err != nil {
		Debug(err)
		return c.JSON(http.StatusBadRequest, ResponseErr{"Invalid user id", 0})
	}

	mika.RLock()
	defer mika.RUnlock()
	user, exists := mika.Users[user_id]
	if !exists {
		return c.JSON(http.StatusNotFound, ResponseErr{"Not Found", 404})
	}
	return c.JSON(http.StatusOK, user)
}

func HandleUserCreate(c *echo.Context) error {
	payload := &UserCreatePayload{}
	if err := c.Bind(payload); err != nil {
		return err
	}
	if payload.Passkey == "" || payload.UserID <= 0 {
		return c.JSON(http.StatusBadRequest, ResponseErr{"Invalid user id", 0})
	}
	r := getRedisConnection()
	defer returnRedisConnection(r)
	user := GetUserByID(r, payload.UserID, false)

	if user != nil {
		return c.JSON(http.StatusConflict, ResponseErr{"User exists", http.StatusConflict})
	}

	user = GetUserByID(r, payload.UserID, true)
	mika.Lock()
	user.Passkey = payload.Passkey
	user.CanLeech = true
	user.Enabled = true
	mika.Unlock()
	if !user.InQueue {
		user.InQueue = true
		sync_user <- user
	}
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
		Debug(err)
		return c.JSON(http.StatusBadRequest, ResponseErr{"Invalid user id format", 0})
	}

	mika.RLock()
	user, exists := mika.Users[user_id]
	mika.RUnlock()
	if !exists {
		return c.JSON(http.StatusNotFound, ResponseErr{"User not Found", 404})
	}

	user.Lock()
	user.Uploaded = payload.Uploaded
	user.Downloaded = payload.Downloaded
	user.Passkey = payload.Passkey
	user.CanLeech = payload.CanLeech
	user.Unlock()

	if !user.InQueue {
		user.InQueue = true
		sync_user <- user
	}
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
	whitelist = append(whitelist, payload.Prefix)

	r := getRedisConnection()
	defer returnRedisConnection(r)

	r.Do("HSET", "t:whitelist", payload.Prefix, payload.Client)

	return c.JSON(http.StatusCreated, ResponseErr{"ok", http.StatusCreated})
}

func HandleWhitelistDel(c *echo.Context) error {
	prefix := c.Param("prefix")
	for _, p := range whitelist {
		if p == prefix {
			r := getRedisConnection()
			defer returnRedisConnection(r)

			r.Do("HDEL", "t:whitelist", prefix)
			go initWhitelist(r)
			return c.JSON(http.StatusOK, ResponseErr{"ok", http.StatusOK})
		}
	}
	return c.JSON(http.StatusNotFound, ResponseErr{"User not Found", 404})
}

func HandleGetTorrentPeer(c *echo.Context) error {
	return c.JSON(http.StatusOK, ResponseErr{"Nope! :(", 200})
}
