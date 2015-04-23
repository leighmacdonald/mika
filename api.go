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
	payload := &TorrentDelPayload{}
	if err := c.Bind(payload); err != nil {
		return err
	}
	r := getRedisConnection()
	defer returnRedisConnection(r)
	torrent := mika.GetTorrentByID(r, payload.TorrentID, false)
	if torrent == nil {
		return errors.New("Invalid torrent id")
	}

	return c.JSON(http.StatusOK, ResponseErr{"moo", 200})
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

func HandleUserUpdatePasskey(c *echo.Context) {

}

func HandleWhitelistAdd(c *echo.Context) {

}

func HandleWhitelistDel(c *echo.Context) {

}

func HandleWhitelistUpdate(c *echo.Context) {

}

func HandleGetTorrentPeer(c *echo.Context) error {
	return c.JSON(http.StatusOK, ResponseErr{"Nope! :(", 200})
}
