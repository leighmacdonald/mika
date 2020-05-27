package config

import (
	"github.com/leighmacdonald/mika/consts"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestRead(t *testing.T) {
	require.Equal(t, consts.ErrInvalidConfig, Read("invalid_config_path.yaml"))
}

func TestLogger(t *testing.T) {
	require.Panics(t, func() {
		setupLogger("invalid", false)
	})
}

func TestStoreConfig_DSN(t *testing.T) {
	c := StoreConfig{
		Type:       "postgres",
		Host:       "localhost",
		Port:       5432,
		Username:   "test",
		Password:   "pass",
		Database:   "db",
		Properties: "arg1=foo&arg2=bar",
	}
	require.Equal(t,
		"test:pass@tcp(localhost:5432)/db?arg1=foo&arg2=bar",
		c.DSN())
}
