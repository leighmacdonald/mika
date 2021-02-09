package tracker

import (
	"fmt"
	"github.com/leighmacdonald/mika/consts"
	"github.com/leighmacdonald/mika/store"
	_ "github.com/leighmacdonald/mika/store/mysql"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestBitTorrentHandler_Announce(t *testing.T) {
	unregisteredTorrent := store.GenerateTestTorrent()
	rh := NewBitTorrentHandler()

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
		{testReq{IhStr: "", PID: testLeechers[0].PeerID, IP: "12.34.56.78",
			Port: "4000", Uploaded: "0", Downloaded: "5000", left: "5000", PK: testUsers[0].Passkey},
			stateExpected{Status: msgInvalidInfoHash},
		},
		// 1. Bad InfoHash length
		{testReq{IhStr: "012345678901234567891", PID: testLeechers[0].PeerID, IP: "12.34.56.78",
			Port: "4000", Uploaded: "0", Downloaded: "5000", left: "5000", PK: testUsers[0].Passkey},
			stateExpected{Status: msgInvalidInfoHash},
		},
		// 2. Bad passkey
		{testReq{Ih: testTorrents[0].InfoHash, PID: testLeechers[0].PeerID, IP: "12.34.56.78",
			Port: "4000", Uploaded: "0", Downloaded: "5000", left: "5000", PK: "XXXXXXXXXXYYYYYYYYYY"},
			stateExpected{Status: msgInvalidAuth},
		},
		// 3. Bad port (too low)
		{testReq{Ih: testTorrents[0].InfoHash, PID: testLeechers[0].PeerID, IP: "12.34.56.78",
			Port: "1000", Uploaded: "0", Downloaded: "5000", left: "5000", PK: testUsers[0].Passkey},
			stateExpected{Status: msgInvalidPort},
		},
		// 4. Bad port (too high)
		{testReq{Ih: testTorrents[0].InfoHash, PID: testLeechers[0].PeerID, IP: "12.34.56.78",
			Port: "100000", Uploaded: "0", Downloaded: "5000", left: "5000", PK: testUsers[0].Passkey},
			stateExpected{Status: msgInvalidPort},
		},
		// 5. Non-routable ip
		{testReq{Ih: testTorrents[0].InfoHash, PID: testLeechers[0].PeerID, IP: "127.0.0.1",
			Port: "4000", Uploaded: "0", Downloaded: "5000", left: "5000", PK: testUsers[0].Passkey},
			stateExpected{Status: msgMalformedRequest},
		},
		// 6. Non-routable ip
		{testReq{Ih: testTorrents[0].InfoHash, PID: testLeechers[0].PeerID, IP: "10.0.0.10",
			Port: "4000", Uploaded: "0", Downloaded: "5000", left: "5000", PK: testUsers[0].Passkey},
			stateExpected{Status: msgMalformedRequest},
		},
		// 7. IPv6 non-routable
		{testReq{Ih: testTorrents[0].InfoHash, PID: testLeechers[0].PeerID, IP: "::1",
			Port: "4000", Uploaded: "0", Downloaded: "5000", left: "5000", PK: testUsers[0].Passkey},
			stateExpected{Status: msgMalformedRequest},
		},
		// 8. IPv6 routable
		{testReq{Ih: testTorrents[0].InfoHash, PID: testLeechers[0].PeerID, IP: "2600::1",
			Port: "4000", Uploaded: "0", Downloaded: "0", left: "5000", PK: testUsers[0].Passkey},
			stateExpected{Status: msgOk, HasPeer: true, Left: 5000, Port: 4000, IP: "2600::1", SwarmSize: 1},
		},
		// 9. IPv6 routable
		{testReq{Ih: unregisteredTorrent.InfoHash, PID: testLeechers[0].PeerID, IP: "12.34.56.78",
			Port: "4000", Uploaded: "0", Downloaded: "5000", left: "5000", PK: testUsers[0].Passkey},
			stateExpected{Status: msgInvalidInfoHash},
		},
		// 10. 1 Leecher start event
		{testReq{Ih: testTorrents[0].InfoHash, PID: testLeechers[0].PeerID, IP: testLeechers[0].IP.String(),
			Port: "4000", Uploaded: "0", Downloaded: "0", left: "10000", PK: testUsers[0].Passkey, event: string(consts.STARTED)},
			stateExpected{Uploaded: 0, Downloaded: 0, Left: 10000,
				Seeders: 0, Leechers: 1, Snatches: 0, Port: 4000, IP: "2600::1", HasPeer: true, Status: msgOk, SwarmSize: 1},
		},
		// 11. 1 Leecher announce event
		{testReq{Ih: testTorrents[0].InfoHash, PID: testLeechers[0].PeerID, IP: testLeechers[0].IP.String(),
			Port: "5000", Uploaded: "0", Downloaded: "5000", left: "5000", PK: testUsers[0].Passkey},
			stateExpected{Uploaded: 0, Downloaded: 5000, Left: 5000,
				Seeders: 0, Leechers: 1, Snatches: 0, Port: 4000, IP: "2600::1", HasPeer: true, Status: msgOk, SwarmSize: 1},
		},
		// 12. 1 leecher / 1 seeder
		{testReq{Ih: testTorrents[0].InfoHash, PID: testSeeders[0].PeerID, IP: testSeeders[0].IP.String(),
			Port: "4000", Uploaded: "5000", Downloaded: "0", left: "0", PK: testUsers[1].Passkey, event: string(consts.STARTED)},
			stateExpected{Uploaded: 5000, Downloaded: 0, Left: 0,
				Seeders: 1, Leechers: 1, Snatches: 0, Port: 4000, IP: testSeeders[0].IP.String(), HasPeer: true, Status: msgOk,
				SwarmSize: 2},
		},
		// 13. 2 Seeders, 1 completed leecher
		{testReq{Ih: testTorrents[0].InfoHash, PID: testLeechers[0].PeerID, IP: "2600::1", event: string(consts.COMPLETED),
			Port: "4000", Uploaded: "0", Downloaded: "5000", left: "0", PK: testUsers[0].Passkey},
			stateExpected{Uploaded: 0, Downloaded: 10000, Left: 0,
				Seeders: 2, Leechers: 0, Snatches: 1, Port: 4000, IP: "2600::1", HasPeer: true, Status: msgOk,
				SwarmSize: 2},
		},
		// 14. 1 seeder left swarm
		{testReq{Ih: testTorrents[0].InfoHash, PID: testSeeders[0].PeerID, IP: testSeeders[0].IP.String(), event: string(consts.STOPPED),
			Port: fmt.Sprintf("%d", testSeeders[0].Port), Uploaded: "10000", Downloaded: "0", left: "0", PK: testUsers[1].Passkey},
			stateExpected{Uploaded: 15000, Downloaded: 0, Left: 0,
				Seeders: 1, Leechers: 0, Snatches: 1, Port: testSeeders[0].Port, IP: testSeeders[0].IP.String(), HasPeer: false, Status: msgOk,
				SwarmSize: 1},
		},
		// 15. 2 seeders, 1 paused
		{testReq{Ih: testTorrents[0].InfoHash, PID: testSeeders[0].PeerID, IP: testSeeders[0].IP.String(), event: string(consts.PAUSED),
			Port: fmt.Sprintf("%d", testSeeders[0].Port), Uploaded: "0", Downloaded: "0", left: "5000", PK: testUsers[1].Passkey},
			stateExpected{Uploaded: 0, Downloaded: 0, Left: 5000,
				Seeders: 2, Leechers: 0, Snatches: 1, Port: testSeeders[0].Port, IP: testSeeders[0].IP.String(), HasPeer: true, Status: msgOk,
				SwarmSize: 2},
		},
	}
	for i, a := range announces {
		u := fmt.Sprintf("/announce/%s?%s", a.req.PK, a.req.ToValues().Encode())
		w := performRequest(rh, "GET", u, nil, nil)
		//time.Sleep(time.Millisecond * 200) // Wait for batch update call (100ms)
		require.EqualValues(t, a.state.Status, errCode(w.Code),
			fmt.Sprintf("%s (%d)", responseStringMap[errCode(w.Code)], i))
		if w.Code == 200 {
			// Additional validations for ok announces
			tor, err := TorrentGet(a.req.Ih, false)
			require.NoError(t, err)
			if a.state.HasPeer {
				// If we expect a peer (!stopped event)
				peer, err := tor.Peers.Get(a.req.PID)
				require.NoError(t, err, "Failed to get peer (%d)", i)
				require.Equal(t, int(a.state.Uploaded), int(peer.Uploaded), "Invalid uploaded (%d)", i)
				require.Equal(t, int(a.state.Downloaded), int(peer.Downloaded), "Invalid downloaded (%d)", i)
				require.Equal(t, int(a.state.Left), int(peer.Left), "Invalid left (%d)", i)
				require.Equal(t, int(a.state.Port), int(peer.Port), "Invalid port (%d)", i)
				require.Equal(t, a.state.IP, peer.IP.String(), "Invalid ip (%d)", i)
			} else {
				_, err2 := tor.Peers.Get(a.req.PID)
				require.Error(t, err2, "Got peer when we shouldn't (%d)", i)
			}
			peers, err := tor.Peers.GetN(1000)
			require.NoError(t, err, "Failed to fetch all peers (%d)", i)
			torrent, err := TorrentGet(tor.InfoHash, false)
			require.NoError(t, err)
			require.Equal(t, a.state.SwarmSize, len(peers), "Invalid swarm size (%d)", i)
			require.Equal(t, int(a.state.Seeders), int(torrent.Seeders), "Invalid seeder count (%d)", i)
			require.Equal(t, int(a.state.Leechers), int(torrent.Leechers), "Invalid leecher count (%d)", i)
			require.Equal(t, int(a.state.Snatches), int(torrent.Snatches), "invalid snatch count (%d)", i)
		}
	}
}
