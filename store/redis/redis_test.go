package redis

import (
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"mika/config"
	"mika/consts"
	"mika/model"
	"mika/util"
	"testing"
)

func createTestStore() *TorrentStore {
	host := viper.GetString(config.CacheHost)
	port := viper.GetInt(config.CachePort)
	password := viper.GetString(config.CachePassword)
	db := viper.GetInt(config.CacheDB)
	return NewTorrentStore(host, port, password, db)
}

func TestTorrentStore(t *testing.T) {
	config.Read("")
	s := createTestStore()
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
