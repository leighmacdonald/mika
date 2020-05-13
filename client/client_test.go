package client

import (
	"context"
	"fmt"
	"github.com/leighmacdonald/mika/config"
	h "github.com/leighmacdonald/mika/http"
	"github.com/leighmacdonald/mika/model"
	"github.com/leighmacdonald/mika/tracker"
	"github.com/stretchr/testify/require"
	"log"
	"net/http"
	"os"
	"testing"
	"time"
)

var (
	host   = "localhost:34100"
	server *http.Server
	ihStr  = "ff503e9ca036f1647c2dfc1337b163e2c54f13f8"
)

func TestClient_Torrent(t *testing.T) {
	c := New(host)
	var ih model.InfoHash
	_ = model.InfoHashFromString(&ih, ihStr)
	require.NoError(t, c.TorrentAdd(ih, "test torrent"))
	require.NoError(t, c.TorrentDelete(ih))

}

func TestClient_Ping(t *testing.T) {
	c := New(host)
	require.NoError(t, c.Ping())
}

func TestMain(m *testing.M) {
	ctx := context.Background()
	if err := config.Read("mika_testing"); err != nil {
		log.Fatalf("failed to read config")
	}
	tkr, _, _, _ := tracker.NewTestTracker()
	handler := h.NewAPIHandler(tkr)
	server = h.CreateServer(handler, host, false)
	go func() {
		if err := server.ListenAndServe(); err != nil {
			fmt.Printf("Error serving test api server: %s", err.Error())
		}
	}()
	time.Sleep(1 * time.Second)
	retVal := m.Run()
	if err := server.Shutdown(ctx); err != nil {
		fmt.Printf("Error shutting down test api server: %s", err.Error())
	}
	os.Exit(retVal)
}
