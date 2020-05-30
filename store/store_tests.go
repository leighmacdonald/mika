package store

import (
	"errors"
	"fmt"
	"github.com/leighmacdonald/mika/consts"
	"github.com/leighmacdonald/mika/util"
	"github.com/stretchr/testify/require"
	"log"
	"math/rand"
	"net"
	"testing"
	"time"
)

// GenerateTestUser creates a peer using fake data. Used for testing.
func GenerateTestUser() User {
	return User{
		UserID:          uint32(rand.Intn(10000)),
		Passkey:         util.NewPasskey(),
		IsDeleted:       false,
		DownloadEnabled: true,
		Downloaded:      1000,
		Uploaded:        2000,
		Announces:       500,
	}
}

// GenerateTestTorrent creates a torrent using fake data. Used for testing.
func GenerateTestTorrent() Torrent {
	token, _ := util.GenRandomBytes(20)
	var ih InfoHash
	if err := InfoHashFromBytes(&ih, token); err != nil {
		log.Panicf("Failed to generate info_hash: %s", err.Error())
	}
	return NewTorrent(ih, fmt.Sprintf("Show.Title.%d.S03E07.720p.WEB.h264-GRP", rand.Intn(1000000)))
}

// GenerateTestPeer creates a peer using fake data for the provided user. Used for testing.
func GenerateTestPeer() Peer {
	token, _ := util.GenRandomBytes(20)
	ih := PeerIDFromString(string(token))
	p := NewPeer(
		uint32(rand.Intn(1000000)),
		ih,
		net.ParseIP("1.2.3.4"),
		uint16(rand.Intn(60000)))
	return p
}

func findPeer(swarm Swarm, p1 Peer) (Peer, error) {
	p, found := swarm.Peers[p1.PeerID]
	if !found {
		return Peer{}, errors.New("unknown peer")
	}
	return p, nil
}

// TestPeerStore tests the interface implementation
func TestPeerStore(t *testing.T, ps PeerStore, ts TorrentStore, _ UserStore) {
	torrentA := GenerateTestTorrent()
	defer func() { _ = ts.Delete(torrentA.InfoHash, true) }()
	require.NoError(t, ts.Add(torrentA))
	swarm := NewSwarm()
	for i := 0; i < 5; i++ {
		p := GenerateTestPeer()
		p.InfoHash = torrentA.InfoHash
		swarm.Peers[p.PeerID] = p
	}
	for _, peer := range swarm.Peers {
		require.NoError(t, ps.Add(torrentA.InfoHash, peer))
	}
	fetchedPeers, err := ps.GetN(torrentA.InfoHash, 5)
	require.NoError(t, err)
	require.Equal(t, len(swarm.Peers), len(fetchedPeers.Peers))
	for _, peer := range swarm.Peers {
		fp, err := findPeer(swarm, peer)
		require.NoError(t, err)
		require.NotNil(t, fp)
		require.Equal(t, fp.PeerID, peer.PeerID)
		require.Equal(t, fp.IP, peer.IP)
		require.Equal(t, fp.Port, peer.Port)
		require.Equal(t, fp.Location, peer.Location)
	}
	if len(swarm.Peers) < 5 {
		t.Fatalf("Invalid peer count")
	}
	var p1 Peer
	for k := range swarm.Peers {
		p1 = swarm.Peers[k]
		break
	}

	ph := NewPeerHash(p1.InfoHash, p1.PeerID)
	require.NoError(t, ps.Sync(map[PeerHash]PeerStats{
		ph: {
			Uploaded:     10000,
			Downloaded:   20000,
			LastAnnounce: time.Now(),
			Announces:    5,
		},
	}, nil))
	updatedPeers, err2 := ps.GetN(torrentA.InfoHash, 5)
	require.NoError(t, err2)
	p1Updated, _ := findPeer(updatedPeers, p1)
	require.Equal(t, uint32(5), p1Updated.Announces)
	require.Equal(t, p1.TotalTime, p1Updated.TotalTime)
	require.Equal(t, uint64(20000), p1Updated.Downloaded)
	require.Equal(t, uint64(10000), p1Updated.Uploaded)
	for _, peer := range swarm.Peers {
		require.NoError(t, ps.Delete(torrentA.InfoHash, peer.PeerID))
	}
}

// TestTorrentStore tests the interface implementation
func TestTorrentStore(t *testing.T, ts TorrentStore) {
	torrentA := GenerateTestTorrent()
	require.NoError(t, ts.Add(torrentA))
	var fetchedTorrent Torrent
	require.NoError(t, ts.Get(&fetchedTorrent, torrentA.InfoHash, false))
	require.Equal(t, torrentA.InfoHash, fetchedTorrent.InfoHash)
	require.Equal(t, torrentA.IsDeleted, fetchedTorrent.IsDeleted)
	require.Equal(t, torrentA.IsEnabled, fetchedTorrent.IsEnabled)
	require.NoError(t, ts.Delete(torrentA.InfoHash, true))
	var deletedTorrent Torrent
	require.Equal(t, consts.ErrInvalidInfoHash, ts.Get(&deletedTorrent, torrentA.InfoHash, false))
	wlClients := []WhiteListClient{
		{ClientPrefix: "UT", ClientName: "uTorrent"},
		{ClientPrefix: "qT", ClientName: "QBittorrent"},
	}
	for _, c := range wlClients {
		require.NoError(t, ts.WhiteListAdd(c))
	}
	clients, err3 := ts.WhiteListGetAll()
	require.NoError(t, err3)
	require.Equal(t, len(wlClients), len(clients))
	require.NoError(t, ts.WhiteListDelete(wlClients[0]))
	clientsUpdated, _ := ts.WhiteListGetAll()
	require.Equal(t, len(wlClients)-1, len(clientsUpdated))
}

// TestUserStore tests the user store for conformance to our interface
func TestUserStore(t *testing.T, s UserStore) {
	var users []User
	for i := 0; i < 5; i++ {
		users = append(users, GenerateTestUser())
	}
	if users == nil {
		t.Fatalf("Failed to setup users")
	}
	require.NoError(t, s.Add(users[0]))
	var fetchedUserID User
	var fetchedUserPasskey User
	require.NoError(t, s.GetByID(&fetchedUserID, users[0].UserID))
	require.Equal(t, users[0], fetchedUserID)
	require.NoError(t, s.GetByPasskey(&fetchedUserPasskey, users[0].Passkey))
	require.Equal(t, users[0], fetchedUserPasskey)

	batchUpdate := map[string]UserStats{
		users[0].Passkey: {
			Uploaded:   1000,
			Downloaded: 2000,
			Announces:  10,
		},
	}
	require.NoError(t, s.Sync(batchUpdate, nil))
	time.Sleep(100 * time.Millisecond)
	var updatedUser User
	require.NoError(t, s.GetByPasskey(&updatedUser, users[0].Passkey))
	require.Equal(t, uint64(1000)+users[0].Uploaded, updatedUser.Uploaded)
	require.Equal(t, uint64(2000)+users[0].Downloaded, updatedUser.Downloaded)
	require.Equal(t, uint32(10)+users[0].Announces, updatedUser.Announces)
}

func init() {
	rand.Seed(time.Now().UnixNano())
}
