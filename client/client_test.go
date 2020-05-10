package client

import (
	"github.com/leighmacdonald/mika/config"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	"log"
	"os"
	"testing"
)

var host string

func TestClient_Ping(t *testing.T) {
	c := New(host)
	require.NoError(t, c.Ping())
}

func TestMain(m *testing.M) {
	if err := config.Read("mika_testing"); err != nil {
		log.Fatalf("failed to read config")
	}
	host = viper.GetString(string(config.APIListen))
	os.Exit(m.Run())
}
