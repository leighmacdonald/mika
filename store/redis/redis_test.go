package redis

import (
	"github.com/go-redis/redis/v7"
	"github.com/leighmacdonald/mika/config"
	"github.com/leighmacdonald/mika/store"
	"github.com/leighmacdonald/mika/store/memory"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
)

func TestRedisTorrentStore(t *testing.T) {
	ts, e := store.NewTorrentStore("redis", config.GetStoreConfig(config.Torrent))
	require.NoError(t, e, e)
	store.TestTorrentStore(t, ts)
}

func TestRedisUserStore(t *testing.T) {
	us, e := store.NewUserStore("redis", config.GetStoreConfig(config.Users))
	require.NoError(t, e, e)
	store.TestUserStore(t, us)
}

func TestRedisPeerStore(t *testing.T) {
	client := redis.NewClient(newRedisConfig(config.GetStoreConfig(config.Torrent)))
	setupDB(t, client)
	ts, err := store.NewTorrentStore("redis", config.GetStoreConfig(config.Torrent))
	require.NoError(t, err)
	ps, err := store.NewPeerStore("redis", config.GetStoreConfig(config.Peers))
	require.NoError(t, err, err)
	store.TestPeerStore(t, ps, ts, memory.NewUserStore())
}

func clearDB(c *redis.Client) {
	keys, err := c.Keys("*").Result()
	if err != nil {
		log.Panicf("Could not initialize redis db: %s", err.Error())
	}
	for _, k := range keys {
		c.Del(k)
	}
}

func setupDB(t *testing.T, c *redis.Client) {
	clearDB(c)
	t.Cleanup(func() {
		clearDB(c)
	})
}

func TestMain(m *testing.M) {
	if err := config.Read("mika_testing_redis"); err != nil {
		log.Info("Skipping database tests, failed to find config: mika_testing_redis.yaml")
		os.Exit(0)
		return
	}
	if config.GetString(config.GeneralRunMode) != "test" {
		log.Info("Skipping database tests, not running in testing mode")
		os.Exit(0)
		return
	}
	exitCode := m.Run()
	os.Exit(exitCode)
}
