package tracker

import (
	"fmt"
	"github.com/leighmacdonald/mika/consts"
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

// ToValues will generate query  values
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

type scrapeReq struct {
	PK         string
	InfoHashes []model.InfoHash
}

func (r scrapeReq) ToValues() url.Values {
	v := url.Values{}
	for _, ih := range r.InfoHashes {
		v.Add("info_hash", ih.URLEncode())
	}
	return v
}

type scrapeExpect struct {
	status errCode
}

type sr struct {
	req scrapeReq
	exp scrapeExpect
}

func TestBitTorrentHandler_Scrape(t *testing.T) {
	torrent0 := store.GenerateTestTorrent()
	leecher0 := store.GenerateTestPeer()
	seeder0 := store.GenerateTestPeer()
	user0 := store.GenerateTestUser()
	user1 := store.GenerateTestUser()

	tkr, err := NewTestTracker()
	require.NoError(t, err, "Failed to init tracker")
	time.Sleep(time.Millisecond * 200)
	go tkr.StatWorker()
	go tkr.PeerReaper()
	rh := NewBitTorrentHandler(tkr)

	require.NoError(t, tkr.Torrents.Add(torrent0), "Failed to add test torrent")
	for _, u := range []model.User{user0, user1} {
		require.NoError(t, tkr.Users.Add(u), "Failed to add test user")
	}
	require.NoError(t, tkr.Peers.Add(torrent0.InfoHash, leecher0))
	require.NoError(t, tkr.Peers.Add(torrent0.InfoHash, seeder0))

	scrapes := []sr{{
		req: scrapeReq{
			PK:         user0.Passkey,
			InfoHashes: []model.InfoHash{torrent0.InfoHash}},
		exp: scrapeExpect{status: msgOk}},
	}

	for i, a := range scrapes {
		u := fmt.Sprintf("/%s/scrape?%s", a.req.PK, a.req.ToValues().Encode())
		w := performRequest(rh, "GET", u)
		require.EqualValues(t, a.exp.status, errCode(w.Code),
			fmt.Sprintf("%s (%d)", responseStringMap[errCode(w.Code)], i))
	}
}

func TestBitTorrentHandler_Announce(t *testing.T) {
	torrent0 := store.GenerateTestTorrent()
	leecher0 := store.GenerateTestPeer()
	seeder0 := store.GenerateTestPeer()
	user0 := store.GenerateTestUser()
	user1 := store.GenerateTestUser()

	unregisteredTorrent := store.GenerateTestTorrent()

	tkr, err := NewTestTracker()
	require.NoError(t, err, "Failed to init tracker")
	time.Sleep(time.Millisecond * 200)
	go tkr.StatWorker()
	go tkr.PeerReaper()
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
		HasPeer    bool
	}
	type testAnn struct {
		req   testReq
		state stateExpected
	}
	announces := []testAnn{
		// 0. Bad InfoHash
		{testReq{IhStr: "", PID: leecher0.PeerID, IP: "12.34.56.78",
			Port: "4000", Uploaded: "0", Downloaded: "5000", left: "5000", PK: user0.Passkey},
			stateExpected{Status: msgInvalidInfoHash},
		},
		// 1. Bad InfoHash length
		{testReq{IhStr: "012345678901234567891", PID: leecher0.PeerID, IP: "12.34.56.78",
			Port: "4000", Uploaded: "0", Downloaded: "5000", left: "5000", PK: user0.Passkey},
			stateExpected{Status: msgInvalidInfoHash},
		},
		// 2. Bad passkey
		{testReq{Ih: torrent0.InfoHash, PID: leecher0.PeerID, IP: "12.34.56.78",
			Port: "4000", Uploaded: "0", Downloaded: "5000", left: "5000", PK: "XXXXXXXXXXYYYYYYYYYY"},
			stateExpected{Status: msgInvalidAuth},
		},
		// 3. Bad port (too low)
		{testReq{Ih: torrent0.InfoHash, PID: leecher0.PeerID, IP: "12.34.56.78",
			Port: "1000", Uploaded: "0", Downloaded: "5000", left: "5000", PK: user0.Passkey},
			stateExpected{Status: msgInvalidPort},
		},
		// 4. Bad port (too high)
		{testReq{Ih: torrent0.InfoHash, PID: leecher0.PeerID, IP: "12.34.56.78",
			Port: "100000", Uploaded: "0", Downloaded: "5000", left: "5000", PK: user0.Passkey},
			stateExpected{Status: msgInvalidPort},
		},
		// 5. Non-routable ip
		{testReq{Ih: torrent0.InfoHash, PID: leecher0.PeerID, IP: "127.0.0.1",
			Port: "4000", Uploaded: "0", Downloaded: "5000", left: "5000", PK: user0.Passkey},
			stateExpected{Status: msgGenericError},
		},
		// 6. Non-routable ip
		{testReq{Ih: torrent0.InfoHash, PID: leecher0.PeerID, IP: "10.0.0.10",
			Port: "4000", Uploaded: "0", Downloaded: "5000", left: "5000", PK: user0.Passkey},
			stateExpected{Status: msgGenericError},
		},
		// 7. IPv6 non-routable
		{testReq{Ih: torrent0.InfoHash, PID: leecher0.PeerID, IP: "::1",
			Port: "4000", Uploaded: "0", Downloaded: "5000", left: "5000", PK: user0.Passkey},
			stateExpected{Status: msgMalformedRequest},
		},
		// 8. IPv6 routable
		{testReq{Ih: torrent0.InfoHash, PID: leecher0.PeerID, IP: "2600::1",
			Port: "4000", Uploaded: "0", Downloaded: "5000", left: "5000", PK: user0.Passkey},
			stateExpected{Status: msgMalformedRequest},
		},
		// 9. IPv6 routable
		{testReq{Ih: unregisteredTorrent.InfoHash, PID: leecher0.PeerID, IP: "12.34.56.78",
			Port: "4000", Uploaded: "0", Downloaded: "5000", left: "5000", PK: user0.Passkey},
			stateExpected{Status: msgInvalidInfoHash},
		},
		// 10. 1 Leecher start event
		{testReq{Ih: torrent0.InfoHash, PID: leecher0.PeerID, IP: "12.34.56.78",
			Port: "4000", Uploaded: "0", Downloaded: "0", left: "10000", PK: user0.Passkey, event: string(consts.STARTED)},
			stateExpected{Uploaded: 0, Downloaded: 0, Left: 10000,
				Seeders: 0, Leechers: 1, Completed: 0, Port: 4000, IP: "12.34.56.78", HasPeer: true, Status: msgOk},
		},
		// 11. 1 Leecher announce event
		{testReq{Ih: torrent0.InfoHash, PID: leecher0.PeerID, IP: "12.34.56.78",
			Port: "4000", Uploaded: "0", Downloaded: "5000", left: "5000", PK: user0.Passkey},
			stateExpected{Uploaded: 0, Downloaded: 5000, Left: 5000,
				Seeders: 0, Leechers: 1, Completed: 0, Port: 4000, IP: "12.34.56.78", HasPeer: true, Status: msgOk},
		},
		// 12. 1 leecher / 1 seeder
		{testReq{Ih: torrent0.InfoHash, PID: seeder0.PeerID, IP: "12.34.56.99",
			Port: "8001", Uploaded: "5000", Downloaded: "0", left: "0", PK: user1.Passkey},
			stateExpected{Uploaded: 5000, Downloaded: 0, Left: 0,
				Seeders: 1, Leechers: 1, Completed: 0, Port: 8001, IP: "12.34.56.99", HasPeer: true, Status: msgOk},
		},
		// 13. 2 Seeders, 1 completed leecher
		{testReq{Ih: torrent0.InfoHash, PID: leecher0.PeerID, IP: "12.34.56.78", event: string(consts.COMPLETED),
			Port: "4000", Uploaded: "0", Downloaded: "5000", left: "0", PK: user0.Passkey},
			stateExpected{Uploaded: 0, Downloaded: 10000, Left: 0,
				Seeders: 2, Leechers: 0, Completed: 1, Port: 4000, IP: "12.34.56.78", HasPeer: true, Status: msgOk},
		},
		// 14. 1 seeder left swarm
		{testReq{Ih: torrent0.InfoHash, PID: seeder0.PeerID, IP: "12.34.56.99", event: string(consts.STOPPED),
			Port: "8001", Uploaded: "10000", Downloaded: "0", left: "0", PK: user1.Passkey},
			stateExpected{Uploaded: 15000, Downloaded: 0, Left: 0,
				Seeders: 1, Leechers: 0, Completed: 1, Port: 8001, IP: "12.34.56.99", HasPeer: false, Status: msgOk},
		},
	}
	for i, a := range announces {
		u := fmt.Sprintf("/%s/announce?%s", a.req.PK, a.req.ToValues().Encode())
		w := performRequest(rh, "GET", u)
		time.Sleep(time.Millisecond * 200) // Wait for batch update call (100ms)
		require.EqualValues(t, a.state.Status, errCode(w.Code),
			fmt.Sprintf("%s (%d)", responseStringMap[errCode(w.Code)], i))
		if w.Code == 200 {
			// Additional validations for ok announces
			var peer model.Peer
			if a.state.HasPeer {
				// If we expect a peer (!stopped event)
				require.NoError(t, tkr.Peers.Get(&peer, a.req.Ih, a.req.PID), "Failed to get peer (%d)", i)
				require.Equal(t, a.state.Uploaded, peer.Uploaded, "Invalid uploaded (%d)", i)
				require.Equal(t, a.state.Downloaded, peer.Downloaded, "Invalid downloaded (%d)", i)
				require.Equal(t, a.state.Left, peer.Left, "Invalid left (%d)", i)
				require.Equal(t, a.state.Port, peer.Port, "Invalid port (%d)", i)
				require.Equal(t, a.state.IP, peer.IP.String(), "Invalid ip (%d)", i)
			} else {
				require.Error(t, tkr.Peers.Get(&peer, a.req.Ih, a.req.PID), "Got peer when we shouldn't (%d)", i)
			}
			swarm, err := tkr.Peers.GetN(torrent0.InfoHash, 1000)
			require.NoError(t, err, "Failed to fetch all peers (%d)", i)
			seeds, leechers := swarm.Counts()
			require.Equal(t, a.state.Seeders, seeds, "Invalid seeder count (%d)", i)
			require.Equal(t, a.state.Leechers, leechers, "Invalid leecher count (%d)", i)
		}
	}
}
