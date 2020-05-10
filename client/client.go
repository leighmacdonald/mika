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
