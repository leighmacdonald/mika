package client

import (
	"fmt"
	h "github.com/leighmacdonald/mika/http"
	"github.com/leighmacdonald/mika/model"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"net/http"
	"time"
)

// Client is the API client implementation
type Client struct {
	host    string
	authKey string
	client  *http.Client
}

// New initializes an API client for the specified host
func New(host string, authKey string) *Client {
	c := h.NewClient()
	return &Client{
		host:    host,
		authKey: authKey,
		client:  c,
	}
}

func (c *Client) u(path string) string {
	return fmt.Sprintf("http://%s%s", c.host, path)
}

func (c *Client) headers() map[string]string {
	if c.authKey == "" {
		return nil
	}
	return map[string]string{
		"Authorization": c.authKey,
	}
}

// TorrentDelete will delete the torrent matching the info_hash provided
func (c *Client) TorrentDelete(ih model.InfoHash) error {
	_, err := h.Do(c.client, h.Opts{
		Method:  "DELETE",
		URL:     c.u(fmt.Sprintf("/torrent/%s", ih.String())),
		Headers: c.headers(),
	})
	return err
}

// TorrentAdd add a new info_hash and associated name to be tracked
func (c *Client) TorrentAdd(ih model.InfoHash, name string) error {
	_, err := h.Do(c.client, h.Opts{
		Method: "POST",
		URL:    c.u("/torrent"),
		JSON: h.TorrentAddRequest{
			InfoHash: ih.String(),
			Name:     name,
		},
		Headers: c.headers(),
	})
	return err
}

// UserDelete deletes the user matching the passkey provided
func (c *Client) UserDelete(passkey string) error {
	_, err := h.Do(c.client, h.Opts{
		Method:  "DELETE",
		URL:     c.u(fmt.Sprintf("/user/pk/%s", passkey)),
		Headers: c.headers(),
	})
	return err
}

// UserAdd creates a new user with the passkey provided
func (c *Client) UserAdd(passkey string) error {
	var uar h.UserAddResponse
	_, err := h.Do(c.client, h.Opts{
		Method: "POST",
		URL:    c.u("/user"),
		JSON: h.UserAddRequest{
			Passkey: passkey,
		},
		Headers: c.headers(),
		Recv:    &uar,
	})
	return err
}

// Ping tests communication between the API server and the client
func (c *Client) Ping() error {
	const msg = "hello world"
	var pong h.PingResponse
	t0 := time.Now()
	_, err := h.Do(c.client, h.Opts{
		Method:  "POST",
		URL:     c.u("/ping"),
		JSON:    h.PingRequest{Ping: msg},
		Headers: c.headers(),
		Recv:    &pong,
	})
	if err != nil {
		return err
	}
	if pong.Pong != msg {
		return errors.New("invalid response to message")
	}
	log.Debugf("Ping successful: %s", time.Since(t0).String())
	return nil
}
