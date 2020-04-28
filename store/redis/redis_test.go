package redis

import (
	"github.com/go-redis/redis/v7"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	"mika/config"
	"mika/store"
	"testing"
)

func createTestTorrentStore(c *redis.Client) (store.TorrentStore, error) {
	var ts torrentDriver
	return ts.NewTorrentStore(&Config{
		Host:     viper.GetString(config.CacheHost),
		Port:     viper.GetInt(config.CachePort),
		Password: viper.GetString(config.CachePassword),
		DB:       viper.GetInt(config.CacheDB),
		Conn:     c,
	})
}

func createTestPeerStore(c *redis.Client) (store.PeerStore, error) {
	var ts peerDriver
	return ts.NewPeerStore(&Config{
		Host:     viper.GetString(config.CacheHost),
		Port:     viper.GetInt(config.CachePort),
		Password: viper.GetString(config.CachePassword),
		DB:       viper.GetInt(config.CacheDB),
		Conn:     c,
	})
}

func TestRedisTorrentStore(t *testing.T) {
	config.Read("")
	ts, err := createTestTorrentStore(nil)
	require.NotNil(t, err)
	store.TestTorrentStore(t, ts)
}

func TestRedisPeerStore(t *testing.T) {
	config.Read("")
	ts, err := createTestTorrentStore(nil)
	require.NotNil(t, err)
	ps, err := createTestPeerStore(nil)
	require.NotNil(t, err)
	store.TestPeerStore(t, ps, ts)
}

func clearDB(c *redis.Client) {
	for _, k := range c.Keys("*").Val() {
		c.Del(k)
	}
}
