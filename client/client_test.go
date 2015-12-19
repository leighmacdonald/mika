package client_test

import (
	"github.com/leighmacdonald/mika/client"
	"testing"
)

const (
	listenHost = "http://localhost:34999"
)

func checkNilTest(t *testing.T, err error, msg string) {
	if err != nil {
		t.Error(msg)
	}
}

func newClient() *client.Client {
	return client.New(listenHost, "", "")
}

func TestClientVersion(t *testing.T) {
	c := newClient()
	ver, err := c.Version()
	checkNilTest(t, err, "Invalid response")
	checkNilTest(t, ver.Version == nil, "nil value returned")
}

func init() {

}
