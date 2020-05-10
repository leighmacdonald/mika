package client

import (
	"encoding/json"
	"errors"
	"fmt"
	h "github.com/leighmacdonald/mika/http"
	"github.com/leighmacdonald/mika/model"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"time"
)

type Client struct {
	host   string
	client *http.Client
}

func New(host string) *Client {
	c := h.NewClient(nil)
	return &Client{
		host:   host,
		client: c,
	}
}

func (c *Client) u(path string) string {
	return fmt.Sprintf("http://%s%s", c.host, path)
}

func (c *Client) TorrentDelete(ih model.InfoHash) error {
	resp, err := h.DoRequest(c.client, "DELETE", c.u(fmt.Sprintf("/torrent/%s", ih.String())), nil)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return readStatus(resp)
	}
	log.Debugf("Torrent deleted successfully: %s", ih.String())
	return nil
}

func (c *Client) TorrentAdd(ih model.InfoHash, name string) error {
	tar := h.TorrentAddRequest{
		InfoHash: ih.String(),
		Name:     name,
	}
	resp, err := h.DoRequest(c.client, "POST", c.u("/torrent"), tar)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return readStatus(resp)
	}
	log.Debugf("Torrent added successfully: %s", name)
	return nil
}

func readStatus(resp *http.Response) error {
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	var sr h.StatusResp
	if err := json.Unmarshal(b, &sr); err != nil {
		return err
	}
	return sr
}

func (c *Client) UserDelete(passkey string) error {
	resp, err := h.DoRequest(c.client, "DELETE", c.u(fmt.Sprintf("/user/pk/%s", passkey)), nil)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return readStatus(resp)
	}
	log.Debugf("User deleted successfully: %s", passkey)
	return nil
}

func (c *Client) UserAdd(passkey string) error {
	var req h.UserAddRequest
	req.Passkey = passkey
	resp, err := h.DoRequest(c.client, "POST", c.u("/user"), req)
	if err != nil {
		return err
	}
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		var sr h.StatusResp
		if err := json.Unmarshal(b, &sr); err != nil {
			return err
		}
		return sr
	}
	var uar h.UserAddResponse
	if err := json.Unmarshal(b, &uar); err != nil {
		return err
	}
	log.Debugf("User added successfully w/passkey: %s", uar.Passkey)
	return nil
}

func (c *Client) Ping() error {
	const msg = "hello world"
	t0 := time.Now()
	resp, err := h.DoRequest(c.client, "POST", c.u("/ping"), h.PingRequest{Ping: msg})
	if err != nil {
		return err
	}
	b, err := ioutil.ReadAll(resp.Body)
	defer func() { _ = resp.Body.Close() }()
	var pong h.PingResponse
	if err := json.Unmarshal(b, &pong); err != nil {
		return err
	}
	log.Debugf("Ping successful: %s", time.Now().Sub(t0).String())
	if pong.Pong != msg {
		return errors.New("invalid response to message")
	}
	return nil
}
