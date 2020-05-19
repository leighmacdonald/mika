package http

import (
	"context"
	"github.com/leighmacdonald/mika/examples/api"
	"github.com/leighmacdonald/mika/store"
	"github.com/leighmacdonald/mika/store/memory"
	"os"
	"testing"
	"time"
)

const (
	listenAddr = "localhost:35999"
	pathPrefix = ""
)

func TestUserStore(t *testing.T) {
	store.TestUserStore(t, NewUserStore(api.DefaultAuthKey, listenAddr))
}

func TestPeerStore(t *testing.T) {
	store.TestPeerStore(t, NewPeerStore(api.DefaultAuthKey, listenAddr), memory.NewTorrentStore(), memory.NewUserStore())
}

func TestTorrentStore(t *testing.T) {
	store.TestTorrentStore(t, NewTorrentStore(api.DefaultAuthKey, listenAddr))
}

func TestMain(m *testing.M) {
	ctx := context.Background()
	server := api.New(listenAddr, pathPrefix, api.DefaultAuthKey)
	go func() { _ = server.ListenAndServe() }()
	defer func() { _ = server.Shutdown(ctx) }()
	// Give the server enough time to init
	time.Sleep(500 * time.Millisecond)
	retVal := m.Run()
	os.Exit(retVal)
}
