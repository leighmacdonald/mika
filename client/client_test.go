package client

import (
	"context"
	"fmt"
	"github.com/leighmacdonald/mika/examples/api"
	"github.com/leighmacdonald/mika/model"
	"github.com/leighmacdonald/mika/tracker"
	"github.com/stretchr/testify/require"
	"log"
	"net/http"
	"net/url"
	"os"
	"testing"
	"time"
)

var (
	host   = "http://localhost:34100"
	server *http.Server
	ihStr  = "ff503e9ca036f1647c2dfc1337b163e2c54f13f8"
)

func TestClient_Torrent(t *testing.T) {
	c := New(host, api.DefaultAuthKey)
	var ih model.InfoHash
	_ = model.InfoHashFromString(&ih, ihStr)
	require.NoError(t, c.TorrentAdd(ih, "test torrent"))
	require.NoError(t, c.TorrentDelete(ih))

}

func TestClient_Ping(t *testing.T) {
	c := New(host, api.DefaultAuthKey)
	require.NoError(t, c.Ping())
}

func TestMain(m *testing.M) {
	ctx := context.Background()
	tkr, err := tracker.NewTestTracker()
	if err != nil {
		log.Fatalf("Failed to init tracker: %s", err)
	}
	handler := tracker.NewAPIHandler(tkr)
	parsedHost, err := url.Parse(host)
	if err != nil {
		log.Fatalf("Could not parse listen host: %s", host)
	}
	httpOpts := tracker.DefaultHTTPOpts()
	httpOpts.Handler = handler
	httpOpts.ListenAddr = parsedHost.Host
	server = tracker.NewHTTPServer(httpOpts)
	go func() {
		if err := server.ListenAndServe(); err != nil {
			fmt.Printf("Error serving test api server: %s", err.Error())
		}
	}()
	// Give the server enough time to init
	time.Sleep(500 * time.Millisecond)
	retVal := m.Run()
	if err := server.Shutdown(ctx); err != nil {
		fmt.Printf("Error shutting down test api server: %s", err.Error())
	}
	os.Exit(retVal)
}
