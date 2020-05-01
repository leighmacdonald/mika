package store

import (
	"fmt"
	"github.com/stretchr/testify/require"
	"math/rand"
	"mika/consts"
	"mika/model"
	"mika/util"
	"net"
	"testing"
)

// GenerateTestUser creates a peer using fake data. Used for testing.
func GenerateTestUser() *model.User {
	passkey, _ := util.GenRandomBytes(20)
	return &model.User{
		UserID:  uint32(rand.Intn(10000)),
		Passkey: string(passkey),
	}
}

// GenerateTestTorrent creates a torrent using fake data. Used for testing.
func GenerateTestTorrent() *model.Torrent {
	token, _ := util.GenRandomBytes(20)
	ih := model.InfoHashFromString(string(token))
	return model.NewTorrent(ih, fmt.Sprintf("Show.Title.%d.S03E07.720p.WEB.h264-GRP", rand.Intn(1000000)), uint32(rand.Intn(1000000)))
}

// GenerateTestPeer creates a peer using fake data for the provided user. Used for testing.
func GenerateTestPeer(user *model.User) *model.Peer {
	token, _ := util.GenRandomBytes(20)
	ih := model.PeerIDFromString(string(token))
	p := model.NewPeer(
		uint32(rand.Intn(1000000)),
		ih,
		net.ParseIP("1.2.3.4"),
		uint16(rand.Intn(60000)))
	p.User = user
	return p
}

func findPeer(peers []*model.Peer, p1 *model.Peer) *model.Peer {
	for _, p := range peers {
		if p.PeerID == p1.PeerID {
			return p
		}
	}
	return nil
}

// TestPeerStore tests the interface implementation
func TestPeerStore(t *testing.T, ps PeerStore, ts TorrentStore) {
	//clearDB(ps.client)
	torrentA := GenerateTestTorrent()
	defer func() { _ = ts.DeleteTorrent(torrentA, true) }()
	require.NoError(t, ts.AddTorrent(torrentA))
	peers := []*model.Peer{
		GenerateTestPeer(nil),
		GenerateTestPeer(nil),
		GenerateTestPeer(nil),
		GenerateTestPeer(nil),
		GenerateTestPeer(nil),
	}
	for _, peer := range peers {
		require.NoError(t, ps.AddPeer(torrentA, peer))
	}
	fetchedPeers, err := ps.GetPeers(torrentA, 5)
	require.NoError(t, err)
	require.Equal(t, len(peers), len(fetchedPeers))
	for _, peer := range peers {
		fp := findPeer(peers, peer)
		require.NotNil(t, fp)
		require.Equal(t, fp.PeerID, peer.PeerID)
		require.Equal(t, fp.IP, peer.IP)
		require.Equal(t, fp.Port, peer.Port)
		require.Equal(t, fp.Location, peer.Location)
		require.Equal(t, util.TimeToString(fp.CreatedOn), util.TimeToString(peer.CreatedOn))
	}
	p1 := peers[2]
	p1.Announces = 5
	p1.TotalTime = 5000
	p1.Downloaded = 10000
	p1.Uploaded = 10000
	require.NoError(t, ps.UpdatePeer(torrentA, p1))
	updatedPeers, err := ps.GetPeers(torrentA, 5)
	require.NoError(t, err)
	p1Updated := findPeer(updatedPeers, p1)
	require.Equal(t, p1.Announces, p1Updated.Announces)
	require.Equal(t, p1.TotalTime, p1Updated.TotalTime)
	require.Equal(t, p1.Downloaded, p1Updated.Downloaded)
	require.Equal(t, p1.Uploaded, p1Updated.Uploaded)
	for _, peer := range peers {
		require.NoError(t, ps.DeletePeer(torrentA, peer))
	}
}

// TestTorrentStore tests the interface implementation
func TestTorrentStore(t *testing.T, ts TorrentStore) {
	torrentA := GenerateTestTorrent()
	require.NoError(t, ts.AddTorrent(torrentA))
	fetchedTorrent, err := ts.GetTorrent(torrentA.InfoHash)
	require.NoError(t, err)
	require.Equal(t, torrentA.TorrentID, fetchedTorrent.TorrentID)
	require.Equal(t, torrentA.InfoHash, fetchedTorrent.InfoHash)
	require.Equal(t, torrentA.IsDeleted, fetchedTorrent.IsDeleted)
	require.Equal(t, torrentA.IsEnabled, fetchedTorrent.IsEnabled)
	require.Equal(t, util.TimeToString(torrentA.CreatedOn), util.TimeToString(fetchedTorrent.CreatedOn))
	require.NoError(t, ts.DeleteTorrent(torrentA, true))
	deletedTorrent, err := ts.GetTorrent(torrentA.InfoHash)
	require.Nil(t, deletedTorrent)
	require.Equal(t, consts.ErrInvalidInfoHash, err)
}
