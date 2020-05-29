package tracker

import (
	"context"
	"github.com/leighmacdonald/mika/config"
	"github.com/stretchr/testify/require"
	"net/http"
	"os"
	"testing"
	"time"
)

func newTestAPI() (*Tracker, http.Handler) {
	context.Background()
	opts := NewDefaultOpts()
	tkr, err := New(context.Background(), opts)
	if err != nil {
		os.Exit(1)
	}
	return tkr, NewAPIHandler(tkr)
}

func TestConfigUpdate(t *testing.T) {
	toDuration := func(s string) time.Duration {
		d, err := time.ParseDuration(s)
		if err != nil {
			panic("Invalid duration specified")
		}
		return d
	}
	tkr, handler := newTestAPI()
	args := ConfigUpdateRequest{
		UpdateKeys: []config.Key{
			config.TrackerAnnounceInterval,
			config.TrackerAnnounceIntervalMin,
			config.TrackerReaperInterval,
			config.TrackerBatchUpdateInterval,
			config.TrackerMaxPeers,
			config.TrackerAutoRegister,
			config.TrackerAllowNonRoutable,
			config.GeodbEnabled,
		},
		TrackerAnnounceInterval:    "60s",
		TrackerAnnounceIntervalMin: "30s",
		TrackerReaperInterval:      "30s",
		TrackerBatchUpdateInterval: "10s",
		TrackerMaxPeers:            100,
		TrackerAutoRegister:        true,
		TrackerAllowNonRoutable:    true,
		GeodbEnabled:               true,
	}
	w := performRequest(handler, "PATCH", "/config", args)
	require.Equal(t, 200, w.Code)
	require.Equal(t, toDuration(args.TrackerAnnounceInterval), tkr.AnnInterval)
	require.Equal(t, toDuration(args.TrackerAnnounceIntervalMin), tkr.AnnIntervalMin)
	require.Equal(t, toDuration(args.TrackerReaperInterval), tkr.ReaperInterval)
	require.Equal(t, toDuration(args.TrackerBatchUpdateInterval), tkr.BatchInterval)
	require.Equal(t, args.TrackerMaxPeers, tkr.MaxPeers)
	require.Equal(t, args.TrackerAutoRegister, tkr.AutoRegister)
	require.Equal(t, args.TrackerAllowNonRoutable, tkr.AllowNonRoutable)
	require.Equal(t, args.GeodbEnabled, tkr.GeodbEnabled)
}

func TestMain(m *testing.M) {
	_ = config.Read("")
	retVal := m.Run()
	os.Exit(retVal)
}
