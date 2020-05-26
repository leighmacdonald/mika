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
	IhStr      string
	PID        model.PeerID
	PIDStr     string
	IP         string
	Port       string
	Uploaded   string
	Downloaded string
	left       string
	event      string
}

func (t testReq) ToValues() url.Values {
	v := url.Values{
		"ip":         {t.IP},
		"port":       {t.Port},
		"uploaded":   {t.Uploaded},
		"downloaded": {t.Downloaded},
		"left":       {t.left},
	}
	if t.IhStr != "" {
		v.Set("info_hash", t.IhStr)
	} else {
		v.Set("info_hash", t.Ih.URLEncode())
	}
	if t.PIDStr != "" {
		v.Set("peer_id", t.PIDStr)
	} else {
		v.Set("peer_id", t.PID.URLEncode())
	}
	if t.event != "" {
		v.Set("event", t.event)
	}
	return v
}

func TestBitTorrentHandler_Announce(t *testing.T) {
	torrent0 := store.GenerateTestTorrent()
	peer0 := store.GenerateTestPeer()
	peer1 := store.GenerateTestPeer()
	user0 := store.GenerateTestUser()
	user1 := store.GenerateTestUser()

	unregisteredTorrent := store.GenerateTestTorrent()

	tkr, err := NewTestTracker()
	require.NoError(t, err, "Failed to init tracker")
	time.Sleep(time.Millisecond * 200)
	go tkr.StatWorker()
	rh := NewBitTorrentHandler(tkr)

	require.NoError(t, tkr.Torrents.Add(torrent0), "Failed to add test torrent")
	for _, u := range []model.User{user0, user1} {
		require.NoError(t, tkr.Users.Add(u), "Failed to add test user")
	}

	type stateExpected struct {
		Uploaded   uint64
		Downloaded uint64
		Left       uint32
		Seeders    uint
		Leechers   uint
		Completed  uint
		Port       uint16
		IP         string
		Status     errCode
	}
	type testAnn struct {
		req   testReq
		state stateExpected
	}
	announces := []testAnn{
		// Bad InfoHash
		{testReq{IhStr: "", PID: peer0.PeerID, IP: "12.34.56.78",
			Port: "4000", Uploaded: "0", Downloaded: "5000", left: "5000", PK: user0.Passkey},
			stateExpected{Status: msgInvalidInfoHash},
		},
		// Bad InfoHash length
		{testReq{IhStr: "012345678901234567891", PID: peer0.PeerID, IP: "12.34.56.78",
			Port: "4000", Uploaded: "0", Downloaded: "5000", left: "5000", PK: user0.Passkey},
			stateExpected{Status: msgInvalidInfoHash},
		},
		// Bad passkey
		{testReq{Ih: torrent0.InfoHash, PID: peer0.PeerID, IP: "12.34.56.78",
			Port: "4000", Uploaded: "0", Downloaded: "5000", left: "5000", PK: "XXXXXXXXXXYYYYYYYYYY"},
			stateExpected{Status: msgInvalidAuth},
		},
		// Bad port (too low)
		{testReq{Ih: torrent0.InfoHash, PID: peer0.PeerID, IP: "12.34.56.78",
			Port: "1000", Uploaded: "0", Downloaded: "5000", left: "5000", PK: user0.Passkey},
			stateExpected{Status: msgInvalidPort},
		},
		// Bad port (too high)
		{testReq{Ih: torrent0.InfoHash, PID: peer0.PeerID, IP: "12.34.56.78",
			Port: "100000", Uploaded: "0", Downloaded: "5000", left: "5000", PK: user0.Passkey},
			stateExpected{Status: msgInvalidPort},
		},
		// Non-routable ip
		{testReq{Ih: torrent0.InfoHash, PID: peer0.PeerID, IP: "127.0.0.1",
			Port: "4000", Uploaded: "0", Downloaded: "5000", left: "5000", PK: user0.Passkey},
			stateExpected{Status: msgGenericError},
		},
		// Non-routable ip
		{testReq{Ih: torrent0.InfoHash, PID: peer0.PeerID, IP: "10.0.0.10",
			Port: "4000", Uploaded: "0", Downloaded: "5000", left: "5000", PK: user0.Passkey},
			stateExpected{Status: msgGenericError},
		},
		// IPv6 non-routable
		{testReq{Ih: torrent0.InfoHash, PID: peer0.PeerID, IP: "::1",
			Port: "4000", Uploaded: "0", Downloaded: "5000", left: "5000", PK: user0.Passkey},
			stateExpected{Status: msgMalformedRequest},
		},
		// IPv6 routable
		{testReq{Ih: torrent0.InfoHash, PID: peer0.PeerID, IP: "2600::1",
			Port: "4000", Uploaded: "0", Downloaded: "5000", left: "5000", PK: user0.Passkey},
			stateExpected{Status: msgMalformedRequest},
		},
		// IPv6 routable
		{testReq{Ih: unregisteredTorrent.InfoHash, PID: peer0.PeerID, IP: "12.34.56.78",
			Port: "4000", Uploaded: "0", Downloaded: "5000", left: "5000", PK: user0.Passkey},
			stateExpected{Status: msgInvalidInfoHash},
		},
		// 1 Leecher
		{testReq{Ih: torrent0.InfoHash, PID: peer0.PeerID, IP: "12.34.56.78",
			Port: "4000", Uploaded: "0", Downloaded: "5000", left: "5000", PK: user0.Passkey},
			stateExpected{Uploaded: 0, Downloaded: 5000, Left: 5000,
				Seeders: 0, Leechers: 1, Completed: 0, Port: 4000, IP: "12.34.56.78", Status: msgOk},
		},
		// 1 leecher / 1 seeder
		{testReq{Ih: torrent0.InfoHash, PID: peer1.PeerID, IP: "12.34.56.99",
			Port: "8001", Uploaded: "5000", Downloaded: "0", left: "0", PK: user1.Passkey},
			stateExpected{Uploaded: 5000, Downloaded: 0, Left: 0,
				Seeders: 1, Leechers: 1, Completed: 0, Port: 8001, IP: "12.34.56.99", Status: msgOk},
		},
	}
	for i, a := range announces {
		u := fmt.Sprintf("/%s/announce?%s", a.req.PK, a.req.ToValues().Encode())
		w := performRequest(rh, "GET", u)
		time.Sleep(time.Millisecond * 200) // Wait for batch update call (100ms)
		require.EqualValues(t, a.state.Status, errCode(w.Code),
			fmt.Sprintf("%s (%d)", responseStringMap[errCode(w.Code)], i))
		if w.Code == 200 {
			var peer model.Peer
			require.NoError(t, tkr.Peers.Get(&peer, a.req.Ih, a.req.PID), "Failed to get peer (%d)", i)
			require.Equal(t, a.state.Uploaded, peer.Uploaded, "Invalid uploaded (%d)", i)
			require.Equal(t, a.state.Downloaded, peer.Downloaded, "Invalid downloaded (%d)", i)
			require.Equal(t, a.state.Left, peer.Left, "Invalid left (%d)", i)
			require.Equal(t, a.state.Port, peer.Port, "Invalid port (%d)", i)
			require.Equal(t, a.state.IP, peer.IP.String(), "Invalid ip (%d)", i)
			swarm, err := tkr.Peers.GetN(torrent0.InfoHash, 1000)
			require.NoError(t, err, "Failed to fetch all peers (%d)", i)
			seeds, leechers := swarm.Counts()
			require.Equal(t, a.state.Seeders, seeds, "Invalid seeder count (%d)", i)
			require.Equal(t, a.state.Leechers, leechers, "Invalid leecher count (%d)", i)
		}
	}
}
