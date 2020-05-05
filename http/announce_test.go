package http

import (
	"fmt"
	"github.com/leighmacdonald/mika/config"
	"github.com/leighmacdonald/mika/tracker"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func performRequest(r http.Handler, method, path string) *httptest.ResponseRecorder {
	req, _ := http.NewRequest(method, path, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestBitTorrentHandler_Announce(t *testing.T) {
	config.Read("")
	tkr, torrents, users, peers := tracker.NewTestTracker()
	rh := NewBitTorrentHandler(tkr)
	type testAnn struct {
		key  string
		v    url.Values
		resp int
	}
	v := []testAnn{
		{users[0].Passkey,
			url.Values{
				"info_hash":  {torrents[0].InfoHash.RawString()},
				"peer_id":    {peers[0].PeerID.RawString()},
				"ip":         {"255.255.255.255"},
				"port":       {"6881"},
				"uploaded":   {"5678"},
				"downloaded": {"1234"},
				"left":       {"9234"},
				"event":      {""},
			},
			200,
		},
	}
	for _, ann := range v {
		u := fmt.Sprintf("/%s/announce?%s", ann.key, ann.v.Encode())
		w := performRequest(rh, "GET", u)
		assert.EqualValues(t, w.Code, ann.resp)
	}
}
