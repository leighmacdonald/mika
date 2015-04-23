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

type TorrentAddPayload struct {
	TorrentID uint64 `json:"torrent_id"`
	InfoHash  string `json:"info_hash"`
}

func HandleVersion(c *echo.Context) {
	c.String(http.StatusOK, fmt.Sprintf("mika/%s", version))
}

func HandleTorrentGet(c *echo.Context) error {
	r := getRedisConnection()
	defer returnRedisConnection(r)
	if r.Err() != nil {
		CaptureMessage(r.Err().Error())
		log.Println("TorrentInfo redis conn:", r.Err().Error())
		return errors.New("Internal error")
	}

	torrent_id_str := c.Param("torrent_id")
	torrent_id, err := strconv.ParseUint(torrent_id_str, 10, 64)
	if err != nil {
		log.Println(err)
		return c.String(http.StatusNotFound, err.Error())
	}
	torrent := mika.GetTorrentByID(r, torrent_id)

	return c.JSON(http.StatusOK, torrent)
}

func HandleTorrentAdd(c *echo.Context) error {
	payload := &TorrentAddPayload{}
	if err := c.Bind(payload); err != nil {
		return err
	}
	log.Println(payload)
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

	return c.JSON(http.StatusCreated, Response{})
}

func HandleTorrentDel(c *echo.Context) {

}

func HandleUserGetActive(c *echo.Context) {

}

func HandleUserGet(c *echo.Context) {

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
