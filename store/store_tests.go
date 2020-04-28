package store

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"mika/consts"
	"mika/model"
	"mika/util"
	"testing"
)

func findPeer(peers []*model.Peer, p1 *model.Peer) *model.Peer {
	for _, p := range peers {
		if p.PeerId == p1.PeerId {
			return p
		}
	}
	return nil
}

// TestPeerStore tests the interface implementation
func TestPeerStore(t *testing.T, ps PeerStore, ts TorrentStore) {
	//clearDB(ps.client)
	torrentA := model.GenerateTestTorrent()
	defer ts.DeleteTorrent(torrentA, true)
	require.NoError(t, ts.AddTorrent(torrentA))
	peers := []*model.Peer{
		model.GenerateTestPeer(),
		model.GenerateTestPeer(),
		model.GenerateTestPeer(),
		model.GenerateTestPeer(),
		model.GenerateTestPeer(),
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
		require.Equal(t, fp.PeerId, peer.PeerId)
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
	torrentA := model.GenerateTestTorrent()
	assert.NoError(t, ts.AddTorrent(torrentA))
	fetchedTorrent, err := ts.GetTorrent(torrentA.InfoHash)
	assert.NoError(t, err)
	assert.Equal(t, torrentA.TorrentID, fetchedTorrent.TorrentID)
	assert.Equal(t, torrentA.InfoHash, fetchedTorrent.InfoHash)
	assert.Equal(t, torrentA.IsDeleted, fetchedTorrent.IsDeleted)
	assert.Equal(t, torrentA.IsEnabled, fetchedTorrent.IsEnabled)
	assert.Equal(t, util.TimeToString(torrentA.CreatedOn), util.TimeToString(fetchedTorrent.CreatedOn))
	assert.NoError(t, ts.DeleteTorrent(torrentA, true))
	deletedTorrent, err := ts.GetTorrent(torrentA.InfoHash)
	assert.Nil(t, deletedTorrent)
	assert.Equal(t, consts.ErrInvalidInfoHash, err)
}
