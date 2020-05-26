package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/leighmacdonald/mika/consts"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"net"
	"net/http"
	"time"
)

// NewClient returns a http.Client with reasonable default configuration values, notably
// actual timeout values.
// TODO use context instead for timeouts
func NewClient() *http.Client {
	//noinspection GoDeprecation
	return &http.Client{
		Transport: &http.Transport{
			Dial: (&net.Dialer{
				Timeout: time.Second * 5,
			}).Dial,
			TLSHandshakeTimeout: time.Second * 5,
		},
		CheckRedirect: nil,
		Jar:           nil,
		Timeout:       time.Second * 5,
	}
}

// AuthedClient represents a http client which supports basic authentication methods
type AuthedClient struct {
	*http.Client
	authKey  string
	basePath string
}

// NewAuthedClient create a new default AuthedClient instance
func NewAuthedClient(authKey string, basePath string) *AuthedClient {
	return &AuthedClient{
		Client:   NewClient(),
		authKey:  authKey,
		basePath: basePath,
	}
}

func (c AuthedClient) u(path string) string {
	return fmt.Sprintf("%s%s", c.basePath, path)
}

// Opts defines the request and response parameters of a HTTP operation
type Opts struct {
	Method  string
	Path    string
	JSON    interface{}
	Data    []byte
	Headers map[string]string
	Recv    interface{}
}

// Exec handles http requests & response initialization and (un)marshalling of JSON payloads.
// If JSON is not nil, it will be JSON encoded before sending to the host, otherwise Data will
// be sent instead.
// If Recv is not nil the response will be unmarshalled into its address
// If a response gets a non-2xx response code, it will exit early and not read the body. The http.Response
// will however get returned in that case.
func (c *AuthedClient) Exec(opts Opts) (*http.Response, error) {
	var err error
	var payload []byte
	if opts.JSON != nil {
		payload, err = json.Marshal(opts.JSON)
		if err != nil {
			if syntaxErr, ok := err.(*json.SyntaxError); ok {
				return nil, syntaxErr
			}
			return nil, err
		}
	} else {
		payload = opts.Data
	}
	url := c.u(opts.Path)
	req, err2 := http.NewRequest(opts.Method, url, bytes.NewReader(payload))
	if err2 != nil {
		return nil, err2
	}
	// Set authorization first so it can be overridden if the user requires it
	if c.authKey != "" {
		req.Header.Set("Authorization", c.authKey)
	}
	for k, v := range opts.Headers {
		req.Header.Set(k, v)
	}
	resp, err3 := c.Do(req)
	if err3 != nil {
		return nil, err3
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		// Let the caller handle this condition
		return resp, consts.ErrBadResponseCode
	}
	if opts.Recv != nil {
		recvPayload, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return resp, errors.Wrapf(err, "Could not read response body")
		}
		defer func() {
			if err := resp.Body.Close(); err != nil {
				log.Warnf("Failed to close response body: %s", err.Error())
			}
		}()
		if err := json.Unmarshal(recvPayload, &opts.Recv); err != nil {
			return resp, errors.Wrapf(err, "Could not decode response body")
		}
	}
	return resp, nil
}
