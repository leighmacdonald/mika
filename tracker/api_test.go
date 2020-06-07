package tracker

import (
	"context"
	"fmt"
	"github.com/leighmacdonald/mika/config"
	"github.com/leighmacdonald/mika/consts"
	"github.com/leighmacdonald/mika/store"
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

func TestPing(t *testing.T) {
	_, handler := newTestAPI()
	req := PingRequest{Ping: "test"}
	var resp PingResponse
	w := performRequest(handler, "POST", "/ping", req, &resp)
	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, req.Ping, resp.Pong)
	w2 := performRequest(handler, "POST", "/ping", "bad data", nil)
	require.Equal(t, http.StatusBadRequest, w2.Code)
}

func equalUser(t *testing.T, a store.User, b store.User) {
	require.Equal(t, a.UserID, b.UserID)
	require.Equal(t, a.Passkey, b.Passkey)
	require.Equal(t, a.IsDeleted, b.IsDeleted)
	require.Equal(t, a.DownloadEnabled, b.DownloadEnabled)
	require.Equal(t, a.Downloaded, b.Downloaded)
	require.Equal(t, a.Uploaded, b.Uploaded)
	require.Equal(t, a.Announces, b.Announces)
}

func TestUserAdd(t *testing.T) {
	user0 := store.GenerateTestUser()
	tkr, handler := newTestAPI()
	w := performRequest(handler, "POST", "/user", user0, nil)
	require.Equal(t, 200, w.Code)
	var userByPK store.User
	var userByID store.User
	require.NoError(t, tkr.users.GetByPasskey(&userByPK, user0.Passkey))
	require.NoError(t, tkr.users.GetByID(&userByID, user0.UserID))
	for _, u := range []store.User{userByPK, userByID} {
		equalUser(t, user0, u)
	}
}

func TestUserDelete(t *testing.T) {
	user0 := store.GenerateTestUser()
	tkr, handler := newTestAPI()
	require.NoError(t, tkr.users.Add(user0))
	u := fmt.Sprintf("/user/pk/%s", user0.Passkey)
	w := performRequest(handler, "DELETE", u, nil, nil)
	require.Equal(t, 200, w.Code)
	require.Error(t, tkr.users.GetByPasskey(&user0, user0.Passkey))
}

func TestUserUpdate(t *testing.T) {
	user0 := store.GenerateTestUser()
	user1 := store.GenerateTestUser()
	var user2 store.User
	tkr, handler := newTestAPI()
	require.NoError(t, tkr.users.Add(user0))
	u := fmt.Sprintf("/user/pk/%s", user0.Passkey)
	w := performRequest(handler, "PATCH", u, user1, nil)
	require.Equal(t, 200, w.Code)
	require.NoError(t, tkr.users.GetByPasskey(&user2, user1.Passkey))
	equalUser(t, user1, user2)
}

func TestTorrentAdd(t *testing.T) {
	tor0 := store.GenerateTestTorrent()
	tkr, handler := newTestAPI()
	tadd := TorrentAddRequest{
		InfoHash: tor0.InfoHash.String(),
		MultiUp:  1.0,
		MultiDn:  -1,
	}
	w := performRequest(handler, "POST", "/torrent", tadd, nil)
	require.Equal(t, 200, w.Code)
	var tor1 store.Torrent
	require.NoError(t, tkr.torrents.Get(&tor1, tor0.InfoHash, false))
	require.Equal(t, tadd.MultiUp, tor1.MultiUp)
	require.Equal(t, float64(0), tor1.MultiDn)
}

func TestTorrentDelete(t *testing.T) {
	tor0 := store.GenerateTestTorrent()
	tkr, handler := newTestAPI()
	require.NoError(t, tkr.torrents.Add(tor0))
	u := fmt.Sprintf("/torrent/%s", tor0.InfoHash.String())
	w := performRequest(handler, "DELETE", u, nil, nil)
	require.Equal(t, 200, w.Code)
	var tor1 store.Torrent
	require.Error(t, tkr.torrents.Get(&tor1, tor0.InfoHash, false))

}

func TestTorrentUpdate(t *testing.T) {
	tor0 := store.GenerateTestTorrent()
	tkr, handler := newTestAPI()
	require.NoError(t, tkr.torrents.Add(tor0))
	tup := store.TorrentUpdate{
		Keys:        []string{"release_name", "is_deleted", "is_enabled", "reason", "multi_up", "multi_dn"},
		ReleaseName: "new_name",
		IsDeleted:   false,
		IsEnabled:   false,
		Reason:      "reason",
		MultiUp:     2.0,
		MultiDn:     0.5,
	}
	p := fmt.Sprintf("/torrent/%s", tor0.InfoHash.String())
	w := performRequest(handler, "PATCH", p, tup, nil)
	require.Equal(t, 200, w.Code)
	var tor1 store.Torrent
	require.NoError(t, tkr.torrents.Get(&tor1, tor0.InfoHash, true))
	require.Equal(t, tup.IsDeleted, tor1.IsDeleted)
	require.Equal(t, tup.IsEnabled, tor1.IsEnabled)
	require.Equal(t, tup.Reason, tor1.Reason)
	require.Equal(t, tup.MultiUp, tor1.MultiUp)
	require.Equal(t, tup.MultiDn, tor1.MultiDn)

	// Deleted torrents should not be fetchable after update
	w2 := performRequest(handler, "PATCH", p, store.TorrentUpdate{
		Keys:      []string{"is_deleted"},
		IsDeleted: true,
	}, nil)
	require.Equal(t, 200, w2.Code)
	var tor2 store.Torrent
	require.Equal(t, consts.ErrInvalidInfoHash, tkr.torrents.Get(&tor2, tor0.InfoHash, false))
}

func TestConfigUpdate(t *testing.T) {
	toDuration := func(seconds int) time.Duration {
		d, err := time.ParseDuration(fmt.Sprintf("%ds", seconds))
		if err != nil {
			panic("Invalid duration specified")
		}
		return d
	}
	tkr, handler := newTestAPI()
	args := ConfigRequest{
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
		TrackerAnnounceInterval:    60,
		TrackerAnnounceIntervalMin: 30,
		TrackerReaperInterval:      30,
		TrackerBatchUpdateInterval: 10,
		TrackerMaxPeers:            100,
		TrackerAutoRegister:        true,
		TrackerAllowNonRoutable:    true,
		GeodbEnabled:               true,
	}
	w := performRequest(handler, "PATCH", "/config", args, nil)
	require.Equal(t, 200, w.Code)
	require.Equal(t, toDuration(args.TrackerAnnounceInterval), tkr.AnnInterval)
	require.Equal(t, toDuration(args.TrackerAnnounceIntervalMin), tkr.AnnIntervalMin)
	require.Equal(t, toDuration(args.TrackerReaperInterval), tkr.ReaperInterval)
	require.Equal(t, toDuration(args.TrackerBatchUpdateInterval), tkr.BatchInterval)
	require.Equal(t, args.TrackerMaxPeers, tkr.MaxPeers)
	require.Equal(t, args.TrackerAutoRegister, tkr.AutoRegister)
	require.Equal(t, args.TrackerAllowNonRoutable, tkr.AllowNonRoutable)
}

func TestMain(m *testing.M) {
	_ = config.Read("")
	retVal := m.Run()
	os.Exit(retVal)
}
