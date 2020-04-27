package redis

import (
	"github.com/go-redis/redis/v7"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"mika/config"
	"mika/consts"
	"mika/model"
	"mika/util"
	"testing"
)

func createTestTorrentStore() *TorrentStore {
	host := viper.GetString(config.CacheHost)
	port := viper.GetInt(config.CachePort)
	password := viper.GetString(config.CachePassword)
	db := viper.GetInt(config.CacheDB)
	return NewTorrentStore(host, port, password, db)
}

func createTestPeerStore(c *redis.Client) *PeerStore {
	host := viper.GetString(config.CacheHost)
	port := viper.GetInt(config.CachePort)
	password := viper.GetString(config.CachePassword)
	db := viper.GetInt(config.CacheDB)
	return NewPeerStore(host, port, password, db, c)
}

func TestTorrentStore(t *testing.T) {
	config.Read("")
	s := createTestTorrentStore()
	clearDB(s.client)
	torrentA := model.GenerateTestTorrent()
	assert.NoError(t, s.AddTorrent(torrentA))
	fetchedTorrent, err := s.GetTorrent(torrentA.InfoHash)
	assert.NoError(t, err)
	assert.Equal(t, torrentA.TorrentID, fetchedTorrent.TorrentID)
	assert.Equal(t, torrentA.InfoHash, fetchedTorrent.InfoHash)
	assert.Equal(t, torrentA.IsDeleted, fetchedTorrent.IsDeleted)
	assert.Equal(t, torrentA.IsEnabled, fetchedTorrent.IsEnabled)
	assert.Equal(t, util.TimeToString(torrentA.CreatedOn), util.TimeToString(fetchedTorrent.CreatedOn))
	assert.NoError(t, s.DeleteTorrent(torrentA, true))
	deletedTorrent, err := s.GetTorrent(torrentA.InfoHash)
	assert.Nil(t, deletedTorrent)
	assert.Equal(t, consts.ErrInvalidInfoHash, err)
}

func TestPeerStore(t *testing.T) {
	config.Read("")
	s := createTestTorrentStore()
	p := createTestPeerStore(s.client)
	clearDB(s.client)
	torrentA := model.GenerateTestTorrent()
	defer s.DeleteTorrent(torrentA, true)
	assert.NoError(t, s.AddTorrent(torrentA))
	peers := []*model.Peer{
		model.GenerateTestPeer(),
		model.GenerateTestPeer(),
		model.GenerateTestPeer(),
		model.GenerateTestPeer(),
		model.GenerateTestPeer(),
	}
	for _, peer := range peers {
		assert.NoError(t, p.AddPeer(torrentA, peer))
	}
	fetchedPeers, err := p.GetPeers(torrentA, 5)
	assert.NoError(t, err)
	assert.Equal(t, len(peers), len(fetchedPeers))
	for _, peer := range peers {
		fp := findPeer(peers, peer)
		if fp == nil {
			t.Fatalf("Could not find matching peer")
		}
		assert.Equal(t, fp.PeerId, peer.PeerId)
		assert.Equal(t, fp.IP, peer.IP)
		assert.Equal(t, fp.Port, peer.Port)
		assert.Equal(t, fp.Location, peer.Location)
		assert.Equal(t, util.TimeToString(fp.CreatedOn), util.TimeToString(peer.CreatedOn))
	}

	p1 := peers[2]
	p1.Announces = 5
	p1.TotalTime = 5000
	p1.Downloaded = 10000
	p1.Uploaded = 10000
	assert.NoError(t, p.UpdatePeer(torrentA, p1))
	updatedPeers, err := p.GetPeers(torrentA, 5)
	assert.NoError(t, err)
	p1Updated := findPeer(updatedPeers, p1)
	assert.Equal(t, p1.Announces, p1Updated.Announces)
	assert.Equal(t, p1.TotalTime, p1Updated.TotalTime)
	assert.Equal(t, p1.Downloaded, p1Updated.Downloaded)
	assert.Equal(t, p1.Uploaded, p1Updated.Uploaded)
	for _, peer := range peers {
		assert.NoError(t, p.DeletePeer(torrentA, peer))
	}
}

func findPeer(peers []*model.Peer, p1 *model.Peer) *model.Peer {
	for _, p := range peers {
		if p.PeerId == p1.PeerId {
			return p
		}
	}
	return nil
}

func clearDB(c *redis.Client) {
	for _, k := range c.Keys("*").Val() {
		c.Del(k)
	}
}
