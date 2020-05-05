package redis

import (
	"github.com/go-redis/redis/v7"
	"github.com/leighmacdonald/mika/config"
	"github.com/leighmacdonald/mika/store"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestRedisTorrentStore(t *testing.T) {
	config.Read("")
	ts, e := store.NewTorrentStore("redis", config.GetStoreConfig(config.Torrent))
	require.NoError(t, e, e)
	store.TestTorrentStore(t, ts)
}

func TestRedisPeerStore(t *testing.T) {
	config.Read("")
	ts, err := store.NewTorrentStore("redis", config.GetStoreConfig(config.Torrent))
	require.NoError(t, err)
	ps, err := store.NewPeerStore("redis", config.GetStoreConfig(config.Peers))
	require.NoError(t, err, err)
	store.TestPeerStore(t, ps, ts)
}

func clearDB(c *redis.Client) {
	for _, k := range c.Keys("*").Val() {
		c.Del(k)
	}
}
