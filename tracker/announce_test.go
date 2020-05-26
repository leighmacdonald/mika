package tracker

import (
	"fmt"
	"github.com/leighmacdonald/mika/model"
	"github.com/leighmacdonald/mika/store"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

func performRequest(r http.Handler, method, path string) *httptest.ResponseRecorder {
	req, _ := http.NewRequest(method, path, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

type testReq struct {
	PK         string
	Ih         model.InfoHash
	PID        model.PeerID
	IP         string
	Port       int
	Uploaded   int
	Downloaded int
	left       int
}

func (t testReq) ToValues() url.Values {
	return url.Values{
		"info_hash":  {t.Ih.URLEncode()},
		"peer_id":    {t.PID.URLEncode()},
		"ip":         {t.IP},
		"port":       {fmt.Sprintf("%d", t.Port)},
		"uploaded":   {fmt.Sprintf("%d", t.Uploaded)},
		"downloaded": {fmt.Sprintf("%d", t.Downloaded)},
		"left":       {fmt.Sprintf("%d", t.left)},
	}
}

func TestBitTorrentHandler_Announce(t *testing.T) {
	torrent0 := store.GenerateTestTorrent()
	peer0 := store.GenerateTestPeer()
	user0 := store.GenerateTestUser()
	tkr, err := NewTestTracker()
	require.NoError(t, err, "Failed to init tracker")
	time.Sleep(time.Millisecond * 200)
	go tkr.StatWorker()
	rh := NewBitTorrentHandler(tkr)

	require.NoError(t, tkr.Torrents.Add(torrent0), "Failed to add test torrent")
	require.NoError(t, tkr.Users.Add(user0), "Failed to add test user")

	type stateExpected struct {
		Uploaded   uint64
		Downloaded uint64
		Left       uint32
		Seeders    uint
		Leechers   uint
		Completed  uint
		Port       uint16
		IP         string
		Status     int
	}
	type testAnn struct {
		req   testReq
		state stateExpected
	}
	v := []testAnn{
		// 1 Leecher
		{testReq{Ih: torrent0.InfoHash, PID: peer0.PeerID, IP: "12.34.56.78",
			Port: 4000, Uploaded: 5678, Downloaded: 1000, left: 5000, PK: user0.Passkey},
			stateExpected{Uploaded: 5678, Downloaded: 1000, Left: 5000,
				Seeders: 0, Leechers: 1, Completed: 0, Port: 4000, IP: "12.34.56.78", Status: 200},
		},
	}
	for i, ann := range v {
		u := fmt.Sprintf("/%s/announce?%s", ann.req.PK, ann.req.ToValues().Encode())
		w := performRequest(rh, "GET", u)
		time.Sleep(time.Millisecond * 200) // Wait for batch update call
		require.EqualValues(t, ann.state.Status, w.Code)
		var peer model.Peer
		require.NoError(t, tkr.Peers.Get(&peer, ann.req.Ih, ann.req.PID), "Failed to get peer (%d)", i)
		require.Equal(t, ann.state.Uploaded, peer.Uploaded, "Invalid uploaded (%d)", i)
		require.Equal(t, ann.state.Downloaded, peer.Downloaded, "Invalid downloaded (%d)", i)
		require.Equal(t, ann.state.Left, peer.Left, "Invalid left (%d)", i)
		require.Equal(t, ann.state.Port, peer.Port, "Invalid port (%d)", i)
		require.Equal(t, ann.state.IP, peer.IP.String(), "Invalid ip (%d)", i)
		swarm, err := tkr.Peers.GetN(torrent0.InfoHash, 1000)
		require.NoError(t, err, "Failed to fetch all peers (%d)", i)
		seeds, leechers := swarm.Counts()
		require.Equal(t, ann.state.Seeders, seeds)
		require.Equal(t, ann.state.Leechers, leechers)
	}
}
