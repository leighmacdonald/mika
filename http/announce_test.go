package http

import (
	"fmt"
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
	rh := NewHandler()
	type testAnn struct {
		key  string
		v    url.Values
		resp int
	}
	v := []testAnn{
		{"blah",
			url.Values{
				"info_hash":  {"12345678901234567890"},
				"peer_id":    {"ABCDEFGHIJKLMNOPQRST"},
				"ip":         {"255.255.255.255"},
				"port":       {"6881"},
				"downloaded": {"1234"},
				"left":       {"9234"},
				"event":      {"stopped"},
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
