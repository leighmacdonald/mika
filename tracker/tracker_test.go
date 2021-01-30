package tracker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/chihaya/bencode"
	"github.com/leighmacdonald/mika/consts"
	"github.com/leighmacdonald/mika/store"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

func performRequest(r http.Handler, method, path string, body interface{}, recv interface{}) *httptest.ResponseRecorder {
	var req *http.Request
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			panic("Failed to marshal test body")
		}
		req, _ = http.NewRequest(method, path, bytes.NewReader(b))
	} else {
		req, _ = http.NewRequest(method, path, nil)
	}
	req.RemoteAddr = "172.16.1.22:9000"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code == http.StatusOK && recv != nil {
		if err := json.Unmarshal(w.Body.Bytes(), recv); err != nil {
			w.Code = 999
		}
	}
	return w
}

type testReq struct {
	PK         string
	Ih         store.InfoHash
	IhStr      string
	PID        store.PeerID
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
	InfoHashes []store.InfoHash
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
	ctx := context.Background()
	torrent0 := store.GenerateTestTorrent()
	leecher0 := store.GenerateTestPeer()
	seeder0 := store.GenerateTestPeer()
	user0 := store.GenerateTestUser()
	user1 := store.GenerateTestUser()

	Init()
	time.Sleep(time.Millisecond * 200)
	go StatWorker(ctx)
	go PeerReaper(ctx)
	rh := NewBitTorrentHandler()

	require.NoError(t, db.Add(torrent0), "Failed to add test torrent")
	for _, u := range []store.User{user0, user1} {
		require.NoError(t, users.Add(u), "Failed to add test user")
	}
	require.NoError(t, peers.Add(torrent0.InfoHash, leecher0))
	require.NoError(t, peers.Add(torrent0.InfoHash, seeder0))
	require.NoError(t, db.Sync(map[store.InfoHash]store.TorrentStats{
		torrent0.InfoHash: {
			Seeders:    1,
			Leechers:   1,
			Snatches:   2,
			Uploaded:   0,
			Downloaded: 0,
			Announces:  2,
		},
	}))

	scrapes := []sr{{
		req: scrapeReq{
			PK:         user0.Passkey,
			InfoHashes: []store.InfoHash{torrent0.InfoHash}},
		exp: scrapeExpect{status: msgOk}},
	}

	for i, a := range scrapes {
		u := fmt.Sprintf("/scrape/%s?%s", a.req.PK, a.req.ToValues().Encode())
		w := performRequest(rh, "GET", u, nil, nil)
		require.EqualValues(t, a.exp.status, errCode(w.Code),
			fmt.Sprintf("%s (%d)", responseStringMap[errCode(w.Code)], i))

		v, err := bencode.NewDecoder(w.Body).Decode()
		require.NoError(t, err, "Failed to decode scrape: (%d)", i)
		d := v.(bencode.Dict)
		require.Equal(t, int64(1), d[torrent0.InfoHash.String()].(bencode.Dict)["complete"].(int64))
		require.Equal(t, int64(1), d[torrent0.InfoHash.String()].(bencode.Dict)["incomplete"].(int64))
		require.Equal(t, int64(2), d[torrent0.InfoHash.String()].(bencode.Dict)["downloaded"].(int64))
	}
}

func TestBitTorrentHandler_Announce(t *testing.T) {
	ctx := context.Background()
	torrent0 := store.GenerateTestTorrent()
	leecher0 := store.GenerateTestPeer()
	seeder0 := store.GenerateTestPeer()
	user0 := store.GenerateTestUser()
	user1 := store.GenerateTestUser()

	unregisteredTorrent := store.GenerateTestTorrent()

	Init()
	// whitelist our random peerids
	whitelist[string(leecher0.PeerID[0:8])] = store.WhiteListClient{
		ClientPrefix: string(leecher0.PeerID[0:8]),
		ClientName:   "Test-Leecher",
	}
	whitelist[string(seeder0.PeerID[0:8])] = store.WhiteListClient{
		ClientPrefix: string(seeder0.PeerID[0:8]),
		ClientName:   "Test-Seeder",
	}
	time.Sleep(time.Millisecond * 200)
	go StatWorker(ctx)
	go PeerReaper(ctx)
	rh := NewBitTorrentHandler()

	require.NoError(t, db.Add(torrent0), "Failed to add test torrent")
	for _, u := range []store.User{user0, user1} {
		require.NoError(t, users.Add(u), "Failed to add test user")
	}

	type stateExpected struct {
		Uploaded   uint64
		Downloaded uint64
		Left       uint32
		Seeders    int
		Leechers   int
		Port       uint16
		IP         string
		Status     errCode
		HasPeer    bool
		Snatches   uint16
		SwarmSize  int
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
			stateExpected{Status: msgMalformedRequest},
		},
		// 6. Non-routable ip
		{testReq{Ih: torrent0.InfoHash, PID: leecher0.PeerID, IP: "10.0.0.10",
			Port: "4000", Uploaded: "0", Downloaded: "5000", left: "5000", PK: user0.Passkey},
			stateExpected{Status: msgMalformedRequest},
		},
		// 7. IPv6 non-routable
		{testReq{Ih: torrent0.InfoHash, PID: leecher0.PeerID, IP: "::1",
			Port: "4000", Uploaded: "0", Downloaded: "5000", left: "5000", PK: user0.Passkey},
			stateExpected{Status: msgMalformedRequest},
		},
		// 8. IPv6 routable
		{testReq{Ih: torrent0.InfoHash, PID: leecher0.PeerID, IP: "2600::1",
			Port: "4000", Uploaded: "0", Downloaded: "0", left: "5000", PK: user0.Passkey},
			stateExpected{Status: msgOk, HasPeer: true, Left: 5000, Port: 4000, IP: "2600::1", SwarmSize: 1},
		},
		// 9. IPv6 routable
		{testReq{Ih: unregisteredTorrent.InfoHash, PID: leecher0.PeerID, IP: "12.34.56.78",
			Port: "4000", Uploaded: "0", Downloaded: "5000", left: "5000", PK: user0.Passkey},
			stateExpected{Status: msgInvalidInfoHash},
		},
		// 10. 1 Leecher start event
		{testReq{Ih: torrent0.InfoHash, PID: leecher0.PeerID, IP: "2600::1",
			Port: "4000", Uploaded: "0", Downloaded: "0", left: "10000", PK: user0.Passkey, event: string(consts.STARTED)},
			stateExpected{Uploaded: 0, Downloaded: 0, Left: 10000,
				Seeders: 0, Leechers: 1, Snatches: 0, Port: 4000, IP: "2600::1", HasPeer: true, Status: msgOk, SwarmSize: 1},
		},
		// 11. 1 Leecher announce event
		{testReq{Ih: torrent0.InfoHash, PID: leecher0.PeerID, IP: "2600::1",
			Port: "4000", Uploaded: "0", Downloaded: "5000", left: "5000", PK: user0.Passkey},
			stateExpected{Uploaded: 0, Downloaded: 5000, Left: 5000,
				Seeders: 0, Leechers: 1, Snatches: 0, Port: 4000, IP: "2600::1", HasPeer: true, Status: msgOk, SwarmSize: 1},
		},
		// 12. 1 leecher / 1 seeder
		{testReq{Ih: torrent0.InfoHash, PID: seeder0.PeerID, IP: "12.34.56.99",
			Port: "8001", Uploaded: "5000", Downloaded: "0", left: "0", PK: user1.Passkey, event: string(consts.STARTED)},
			stateExpected{Uploaded: 5000, Downloaded: 0, Left: 0,
				Seeders: 1, Leechers: 1, Snatches: 0, Port: 8001, IP: "12.34.56.99", HasPeer: true, Status: msgOk,
				SwarmSize: 2},
		},
		// 13. 2 Seeders, 1 completed leecher
		{testReq{Ih: torrent0.InfoHash, PID: leecher0.PeerID, IP: "2600::1", event: string(consts.COMPLETED),
			Port: "4000", Uploaded: "0", Downloaded: "5000", left: "0", PK: user0.Passkey},
			stateExpected{Uploaded: 0, Downloaded: 10000, Left: 0,
				Seeders: 2, Leechers: 0, Snatches: 1, Port: 4000, IP: "2600::1", HasPeer: true, Status: msgOk,
				SwarmSize: 2},
		},
		// 14. 1 seeder left swarm
		{testReq{Ih: torrent0.InfoHash, PID: seeder0.PeerID, IP: "12.34.56.99", event: string(consts.STOPPED),
			Port: "8001", Uploaded: "10000", Downloaded: "0", left: "0", PK: user1.Passkey},
			stateExpected{Uploaded: 15000, Downloaded: 0, Left: 0,
				Seeders: 1, Leechers: 0, Snatches: 1, Port: 8001, IP: "12.34.56.99", HasPeer: false, Status: msgOk,
				SwarmSize: 1},
		},
		// 15. 2 seeders, 1 paused
		{testReq{Ih: torrent0.InfoHash, PID: seeder0.PeerID, IP: "12.34.56.99", event: string(consts.PAUSED),
			Port: "8001", Uploaded: "0", Downloaded: "0", left: "5000", PK: user1.Passkey},
			stateExpected{Uploaded: 0, Downloaded: 0, Left: 5000,
				Seeders: 2, Leechers: 0, Snatches: 1, Port: 8001, IP: "12.34.56.99", HasPeer: true, Status: msgOk,
				SwarmSize: 2},
		},
	}
	for i, a := range announces {
		u := fmt.Sprintf("/announce/%s?%s", a.req.PK, a.req.ToValues().Encode())
		w := performRequest(rh, "GET", u, nil, nil)
		time.Sleep(time.Millisecond * 200) // Wait for batch update call (100ms)
		require.EqualValues(t, a.state.Status, errCode(w.Code),
			fmt.Sprintf("%s (%d)", responseStringMap[errCode(w.Code)], i))
		if w.Code == 200 {
			// Additional validations for ok announces
			var peer store.Peer
			if a.state.HasPeer {
				// If we expect a peer (!stopped event)
				require.NoError(t, peers.Get(&peer, a.req.Ih, a.req.PID), "Failed to get peer (%d)", i)
				require.Equal(t, a.state.Uploaded, peer.Uploaded, "Invalid uploaded (%d)", i)
				require.Equal(t, a.state.Downloaded, peer.Downloaded, "Invalid downloaded (%d)", i)
				require.Equal(t, a.state.Left, peer.Left, "Invalid left (%d)", i)
				require.Equal(t, a.state.Port, peer.Port, "Invalid port (%d)", i)
				require.Equal(t, a.state.IP, peer.IP.String(), "Invalid ip (%d)", i)
			} else {
				require.Error(t, peers.Get(&peer, a.req.Ih, a.req.PID), "Got peer when we shouldn't (%d)", i)
			}
			swarm, err := peers.GetN(torrent0.InfoHash, 1000)
			require.NoError(t, err, "Failed to fetch all peers (%d)", i)
			var torrent store.Torrent
			require.NoError(t, db.Get(&torrent, torrent0.InfoHash, false))
			require.Equal(t, a.state.SwarmSize, len(swarm.Peers), "Invalid swarm size (%d)", i)
			require.Equal(t, a.state.Seeders, torrent.Seeders, "Invalid seeder count (%d)", i)
			require.Equal(t, a.state.Leechers, torrent.Leechers, "Invalid leecher count (%d)", i)
			require.Equal(t, a.state.Snatches, torrent.Snatches, "invalid snatch count (%d)", i)
		}
	}
}
