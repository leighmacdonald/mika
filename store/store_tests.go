package store

import (
	"errors"
	"fmt"
	"github.com/leighmacdonald/mika/consts"
	"github.com/leighmacdonald/mika/model"
	"github.com/leighmacdonald/mika/util"
	"github.com/stretchr/testify/require"
	"log"
	"math/rand"
	"net"
	"testing"
	"time"
)

// GenerateTestUser creates a peer using fake data. Used for testing.
func GenerateTestUser() model.User {
	passkey, _ := util.GenRandomBytes(20)
	return model.User{
		UserID:  uint32(rand.Intn(10000)),
		Passkey: string(passkey),
	}
}

// GenerateTestTorrent creates a torrent using fake data. Used for testing.
func GenerateTestTorrent() model.Torrent {
	token, _ := util.GenRandomBytes(20)
	var ih model.InfoHash
	if err := model.InfoHashFromString(&ih, string(token)); err != nil {
		log.Panicf("Failed to generate info_hash: %s", err.Error())
	}
	return model.NewTorrent(ih, fmt.Sprintf("Show.Title.%d.S03E07.720p.WEB.h264-GRP", rand.Intn(1000000)))
}

// GenerateTestPeer creates a peer using fake data for the provided user. Used for testing.
func GenerateTestPeer() model.Peer {
	token, _ := util.GenRandomBytes(20)
	ih := model.PeerIDFromString(string(token))
	p := model.NewPeer(
		uint32(rand.Intn(1000000)),
		ih,
		net.ParseIP("1.2.3.4"),
		uint16(rand.Intn(60000)))
	return p
}

func findPeer(peers model.Swarm, p1 model.Peer) (model.Peer, error) {
	for _, p := range peers {
		if p.PeerID == p1.PeerID {
			return p, nil
		}
	}
	return model.Peer{}, errors.New("unknown peer")
}

// TestPeerStore tests the interface implementation
func TestPeerStore(t *testing.T, ps PeerStore, ts TorrentStore) {
	torrentA := GenerateTestTorrent()
	defer func() { _ = ts.Delete(torrentA.InfoHash, true) }()
	require.NoError(t, ts.Add(torrentA))
	var peers model.Swarm
	for i := 0; i < 5; i++ {
		peers = append(peers, GenerateTestPeer())
	}
	for _, peer := range peers {
		require.NoError(t, ps.Add(torrentA.InfoHash, peer))
	}
	fetchedPeers, err := ps.GetN(torrentA.InfoHash, 5)
	require.NoError(t, err)
	require.Equal(t, len(peers), len(fetchedPeers))
	for _, peer := range peers {
		fp, err := findPeer(peers, peer)
		require.NoError(t, err)
		require.NotNil(t, fp)
		require.Equal(t, fp.PeerID, peer.PeerID)
		require.Equal(t, fp.IP, peer.IP)
		require.Equal(t, fp.Port, peer.Port)
		require.Equal(t, fp.Location, peer.Location)
	}
	if peers == nil || len(peers) < 5 {
		t.Fatalf("Invalid peer count")
	}
	p1 := peers[2]
	p1.Announces = 5
	p1.TotalTime = 5000
	p1.Downloaded = 10000
	p1.Uploaded = 10000

	updatedPeers, err2 := ps.GetN(torrentA.InfoHash, 5)
	require.NoError(t, err2)
	p1Updated, _ := findPeer(updatedPeers, p1)
	require.Equal(t, p1.Announces, p1Updated.Announces)
	require.Equal(t, p1.TotalTime, p1Updated.TotalTime)
	require.Equal(t, p1.Downloaded, p1Updated.Downloaded)
	require.Equal(t, p1.Uploaded, p1Updated.Uploaded)
	for _, peer := range peers {
		require.NoError(t, ps.Delete(torrentA.InfoHash, peer.PeerID))
	}
}

// TestTorrentStore tests the interface implementation
func TestTorrentStore(t *testing.T, ts TorrentStore) {
	torrentA := GenerateTestTorrent()
	require.NoError(t, ts.Add(torrentA))
	var fetchedTorrent model.Torrent
	require.NoError(t, ts.Get(&fetchedTorrent, torrentA.InfoHash))
	require.Equal(t, torrentA.InfoHash, fetchedTorrent.InfoHash)
	require.Equal(t, torrentA.IsDeleted, fetchedTorrent.IsDeleted)
	require.Equal(t, torrentA.IsEnabled, fetchedTorrent.IsEnabled)
	require.NoError(t, ts.Delete(torrentA.InfoHash, true))
	var deletedTorrent model.Torrent
	require.Equal(t, consts.ErrInvalidInfoHash, ts.Get(&deletedTorrent, torrentA.InfoHash))
	wlClients := []model.WhiteListClient{
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

func init() {
	rand.Seed(time.Now().UnixNano())
}
