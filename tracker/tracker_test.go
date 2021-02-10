package tracker

import (
	"bytes"
	"encoding/json"
	"github.com/leighmacdonald/mika/config"
	"github.com/leighmacdonald/mika/store"
	log "github.com/sirupsen/logrus"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
)

var (
	testRoles     []*store.Role
	testUsers     []*store.User
	testTorrents  []*store.Torrent
	testWhitelist []*store.WhiteListClient
	testLeechers  []*store.Peer
	testSeeders   []*store.Peer
)

func performRequest(r http.Handler, method, path string, body interface{}, recv interface{}) *httptest.ResponseRecorder {
	var req *http.Request
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			panic("Failed to marshal test body")
		}
		req, _ = http.NewRequest(method, path, bytes.NewReader(b))
	} else {
		req, _ = http.NewRequest(method, path, nil)
	}
	req.RemoteAddr = "50.50.50.50:9000"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code == http.StatusOK && recv != nil {
		if err := json.Unmarshal(w.Body.Bytes(), recv); err != nil {
			w.Code = 999
		}
	}
	return w
}

type testReq struct {
	PK         string
	Ih         store.InfoHash
	IhStr      string
	PID        store.PeerID
	PIDStr     string
	IP         string
	Port       string
	Uploaded   string
	Downloaded string
	left       string
	event      string
}

// ToValues will generate query  values
func (t testReq) ToValues() url.Values {
	v := url.Values{
		"ip":         {t.IP},
		"port":       {t.Port},
		"uploaded":   {t.Uploaded},
		"downloaded": {t.Downloaded},
		"left":       {t.left},
	}
	if t.IhStr != "" {
		v.Set("info_hash", t.IhStr)
	} else {
		v.Set("info_hash", t.Ih.URLEncode())
	}
	if t.PIDStr != "" {
		v.Set("peer_id", t.PIDStr)
	} else {
		v.Set("peer_id", t.PID.URLEncode())
	}
	if t.event != "" {
		v.Set("event", t.event)
	}
	return v
}

type scrapeReq struct {
	PK         string
	InfoHashes []store.InfoHash
}

func (r scrapeReq) ToValues() url.Values {
	v := url.Values{}
	for _, ih := range r.InfoHashes {
		v.Add("info_hash", ih.URLEncode())
	}
	return v
}

type scrapeExpect struct {
	status errCode
}

type sr struct {
	req scrapeReq
	exp scrapeExpect
}

func TestMain(m *testing.M) {
	config.General.RunMode = "test"
	config.Tracker.AllowNonRoutable = false
	config.Tracker.AllowClientIP = true
	Init()
	if err := seedTestTracker(); err != nil {
		log.Errorf("Failed to seed tracker for test: %v", err)
		os.Exit(1)
	}
	os.Exit(m.Run())
}

func seedTestTracker() error {
	if config.General.RunMode != "test" {
		log.Fatalf("Cant seed tracker when not in test mode")
	}
	role0 := store.GenerateTestRole()
	if err := RoleAdd(&role0); err != nil {
		return err
	}
	testRoles = append(testRoles, &role0)

	torrent0 := store.GenerateTestTorrent()
	user0 := store.GenerateTestUser()
	user0.RoleID = role0.RoleID
	user1 := store.GenerateTestUser()
	user1.RoleID = role0.RoleID
	wl1 := store.WhiteListClient{
		ClientPrefix: "-qB4330-",
		ClientName:   "qbittorrent 4.3.3",
	}
	wl2 := store.WhiteListClient{
		ClientPrefix: "-DE13F0-",
		ClientName:   "Deluge 1.3",
	}
	if err := WhiteListAdd(&wl1); err != nil {
		return err
	}
	testWhitelist = append(testWhitelist, &wl1)
	if err := WhiteListAdd(&wl2); err != nil {
		return err
	}
	testWhitelist = append(testWhitelist, &wl2)

	if err := UserAdd(&user0); err != nil {
		return err
	}
	testUsers = append(testUsers, &user0)
	if err := UserAdd(&user1); err != nil {
		return err
	}
	testUsers = append(testUsers, &user1)
	if err := TorrentAdd(&torrent0); err != nil {
		return err
	}
	testTorrents = append(testTorrents, &torrent0)

	leecher0 := store.GenerateTestPeer()
	leecher0.Left = 10000
	//torrent0.Peers.Add(leecher0)
	testLeechers = append(testLeechers, leecher0)

	seeder0 := store.GenerateTestPeer()
	seeder0.Left = 0
	//torrent0.Peers.Add(seeder0)
	testSeeders = append(testSeeders, seeder0)

	_ = WhiteListAdd(&store.WhiteListClient{
		ClientPrefix: string(leecher0.PeerID.Bytes())[0:8],
		ClientName:   "test client",
	})
	_ = WhiteListAdd(&store.WhiteListClient{
		ClientPrefix: string(seeder0.PeerID.Bytes())[0:8],
		ClientName:   "test client",
	})
	return nil
}
