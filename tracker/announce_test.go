package tracker

import (
	"context"
	"fmt"
	tlog "github.com/anacrolix/log"
	"github.com/anacrolix/torrent"
	"github.com/anacrolix/torrent/metainfo"
	"github.com/anacrolix/torrent/storage"
	"github.com/leighmacdonald/mika/model"
	"github.com/leighmacdonald/mika/util"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
	"time"
)

func performRequest(r http.Handler, method, path string) *httptest.ResponseRecorder {
	req, _ := http.NewRequest(method, path, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func newClientConfig(port int, storage storage.ClientImpl, name string) *torrent.ClientConfig {
	c := torrent.NewDefaultClientConfig()
	c.ListenPort = port
	c.Debug = true
	c.DisablePEX = true
	c.DisableUTP = true
	c.NoDHT = true
	c.NoDefaultPortForwarding = true
	c.Seed = true
	c.DefaultStorage = storage
	c.Logger = tlog.Logger{LoggerImpl: tlog.StreamLogger{
		W: os.Stderr,
		Fmt: func(msg tlog.Msg) []byte {
			return []byte(fmt.Sprintf("%s - %s", name, tlog.LineFormatter(msg)))
		},
	}}
	return c
}

func TestBitTorrentSession(t *testing.T) {
	tkr, _, _, _ := NewTestTracker()
	require.NoError(t, tkr.Users.Add(model.User{
		UserID:          100,
		Passkey:         "01234567890123456789", // Example torrent uses this passkey
		IsDeleted:       false,
		DownloadEnabled: true,
	}))
	server := &http.Server{Addr: "localhost:34000", Handler: NewBitTorrentHandler(tkr)}
	go func() {
		if err := server.ListenAndServe(); err != nil {
			log.Infof("Failed to shutdown tracker cleanly")
		}
	}()
	port := 4000
	torrentPath := util.FindFile("examples/data/demo_torrent_data.torrent")
	mi, _ := metainfo.LoadFromFile(torrentPath)

	var ih model.InfoHash
	copy(ih[:], mi.HashInfoBytes().Bytes())
	require.NoError(t, tkr.Torrents.Add(model.Torrent{
		ReleaseName: "demo_torrent_data",
		InfoHash:    ih,
	}), "Failed to insert test torrent")

	seeder, _ := torrent.NewClient(newClientConfig(port, storage.NewFile(util.FindFile("examples/data")), "seeder "))
	defer seeder.Close()
	seederDl, err := seeder.AddTorrentFromFile(torrentPath)
	require.NoError(t, err)
	<-seederDl.GotInfo()
	seederDl.DownloadAll()
	seeder.WaitAll()
	dir, err2 := ioutil.TempDir("", "mika-test-")
	require.NoError(t, err2, "Failed to make temp dir for torrent client")
	defer func() { _ = os.RemoveAll(dir) }()
	leecherStore := storage.NewFile(dir)
	leecher, _ := torrent.NewClient(newClientConfig(port+1, leecherStore, "leecher"))
	defer leecher.Close()
	dlT, _ := leecher.AddTorrentFromFile(torrentPath)
	<-dlT.GotInfo()
	waitChan := make(chan bool, 1)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	go func() {
		leecher.WaitAll()
		waitChan <- true
	}()
	select {
	case <-ctx.Done():
		t.Fatalf("Timeout waiting for client to finish download")
		return
	case <-waitChan:
	}
}

func TestBitTorrentHandler_Announce(t *testing.T) {
	tkr, torrents, users, peers := NewTestTracker()
	rh := NewBitTorrentHandler(tkr)
	type testAnn struct {
		key  string
		v    url.Values
		resp int
	}
	v := []testAnn{
		{users[0].Passkey,
			url.Values{
				"info_hash":  {torrents[0].InfoHash.URLEncode()},
				"peer_id":    {peers[0].PeerID.URLEncode()},
				"ip":         {"255.255.255.255"},
				"port":       {"6881"},
				"uploaded":   {"5678"},
				"downloaded": {"1234"},
				"left":       {"9234"},
			},
			200,
		},
	}
	for _, ann := range v {
		u := fmt.Sprintf("/%s/announce?%s", ann.key, ann.v.Encode())
		w := performRequest(rh, "GET", u)
		assert.EqualValues(t, ann.resp, w.Code)
	}
}
