package client

import (
	"encoding/json"
	"errors"
	"github.com/leighmacdonald/mika/tracker"
	"github.com/parnurzeal/gorequest"
	"time"
)

var (
	invalid_response = errors.New("Invalid response code")
)

type Client struct {
	api_endpoint string
	http         *gorequest.SuperAgent
	username     string
	password     string
}

func New(api_endpoint string, username string, password string) *Client {
	httpClient := gorequest.New()
	httpClient.SetBasicAuth(username, password)
	httpClient.Timeout(5 * time.Second)
	return Client{
		api_endpoint: api_endpoint,
		http:         httpClient,
		username:     username,
		password:     password,
	}
}

func (c *Client) mkURL(path string) string {
	return c.api_endpoint + path
}

func (c *Client) Version() (tracker.VersionResponse, error) {
	resp, body, errs := c.http.Get(c.mkURL("/version")).EndBytes()
	if errs != nil {
		return nil, errs[0]
	}
	if resp.StatusCode != 200 {
		return nil, invalid_response
	}
	var version tracker.VersionResponse
	err := json.Unmarshal(body, &version)
	if err != nil {
		return nil, err
	}
	return version, nil

}
