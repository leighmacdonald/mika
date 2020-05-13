// Package exampleapi implements a trivial reference server implementation
// of the required API routes to communicate as a frontend server for the tracker.
//
package exampleapi

import (
	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/mika/consts"
	"github.com/leighmacdonald/mika/model"
	"github.com/leighmacdonald/mika/store"
	"net/http"
	"sync"
	"time"
)

type errMsg struct {
	Status  int    `json:"status"`
	Message string `json:"message"`
}

func errResponse(c *gin.Context, code int, msg string) {
	c.JSON(code, errMsg{code, msg})
}

// ServerExample provides and example / demo server implementation that conforms to mika's HTTP store backend.
type ServerExample struct {
	Addr        string
	Router      *gin.Engine
	Users       model.Users
	UsersMx     *sync.RWMutex
	Peers       map[model.InfoHash]model.Swarm
	PeersMx     *sync.RWMutex
	Torrents    map[model.InfoHash]model.Torrent
	TorrentsMx  *sync.RWMutex
	WhiteList   map[string]model.WhiteListClient
	WhiteListMx *sync.RWMutex
}

func (s *ServerExample) getTorrent(c *gin.Context) {
	infoHashStr := c.Param("info_hash")
	if infoHashStr == "" {
		errResponse(c, http.StatusNotFound, "Unknown info_hash")
		return
	}
	var infoHash model.InfoHash
	if err := model.InfoHashFromString(&infoHash, infoHashStr); err != nil {
		errResponse(c, http.StatusBadRequest, "Malformed info_hash")
		return
	}
	s.TorrentsMx.RLock()
	t, found := s.Torrents[infoHash]
	if !found || t.IsDeleted == true {
		errResponse(c, http.StatusNotFound, "Unknown info_hash")
		return
	}
	c.PureJSON(http.StatusOK, t)
}

func (s *ServerExample) getUser(c *gin.Context) {
	passKey := c.Param("passkey")
	if passKey == "" {
		errResponse(c, http.StatusUnauthorized, "Unauthorized")
		return
	}
	u, err := s.getUserByPasskey(passKey)
	if err != nil {
		errResponse(c, http.StatusUnauthorized, "Unauthorized")
		return
	}
	c.JSON(http.StatusOK, u)
}

func (s *ServerExample) getWhitelist(c *gin.Context) {
	var cwl []model.WhiteListClient
	s.WhiteListMx.RLock()
	defer s.WhiteListMx.RUnlock()
	for _, wl := range s.WhiteList {
		cwl = append(cwl, wl)
	}
	c.JSON(http.StatusOK, cwl)
}

func (s *ServerExample) deleteWhitelist(c *gin.Context) {
	prefix := c.Param("prefix")
	if prefix == "" || len(prefix) != 2 {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	if _, ok := s.WhiteList[prefix]; !ok {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}
	s.WhiteListMx.Lock()
	delete(s.WhiteList, prefix)
	s.WhiteListMx.Unlock()
	c.JSON(http.StatusOK, gin.H{})
}

func (s *ServerExample) addWhitelist(c *gin.Context) {
	var wlc model.WhiteListClient
	if err := c.BindJSON(&wlc); err != nil {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	if _, found := s.WhiteList[wlc.ClientPrefix]; found {
		c.AbortWithStatusJSON(http.StatusConflict, gin.H{"error": "duplicate client prefix"})
		return
	}
	s.WhiteListMx.Lock()
	s.WhiteList[wlc.ClientPrefix] = wlc
	s.WhiteListMx.Unlock()
	c.JSON(http.StatusOK, gin.H{})
}

func (s *ServerExample) getUserByPasskey(passkey string) (model.User, error) {
	s.UsersMx.RLock()
	defer s.UsersMx.RUnlock()
	for _, u := range s.Users {
		if u.Passkey == passkey {
			return u, nil
		}
	}
	return model.User{}, consts.ErrUnauthorized
}

// New returns an example HTTP server implementation to test against and learn from
func New() *http.Server {
	userCount := 10
	torrentCount := 100
	swarmSize := 10 // Swarm per torrent
	s := &ServerExample{
		Addr:       "localhost:35000",
		Router:     gin.Default(),
		UsersMx:    &sync.RWMutex{},
		PeersMx:    &sync.RWMutex{},
		TorrentsMx: &sync.RWMutex{},
		Torrents:   make(map[model.InfoHash]model.Torrent),
		Peers:      make(map[model.InfoHash]model.Swarm),
		Users:      model.Users{},
		WhiteList: map[string]model.WhiteListClient{
			"qB": {
				ClientPrefix: "qB",
				ClientName:   "qBittorrent",
			},
			"UT": {
				ClientPrefix: "UT",
				ClientName:   "uTorrent",
			},
			"TR": {
				ClientPrefix: "TR",
				ClientName:   "Transmission",
			},
		},
		WhiteListMx: &sync.RWMutex{},
	}
	for i := 0; i < userCount; i++ {
		usr := store.GenerateTestUser()
		if i == 0 {
			// Give user 0 a known passkey for testing
			usr.Passkey = "12345678901234567890"
		}
		s.Users = append(s.Users, usr)
	}
	for i := 0; i < torrentCount; i++ {
		t := store.GenerateTestTorrent()
		s.Torrents[t.InfoHash] = t
	}
	for k := range s.Torrents {
		var swarm model.Swarm
		for i := 0; i < swarmSize; i++ {
			swarm = append(swarm, store.GenerateTestPeer())
		}
		s.Peers[k] = swarm
	}
	s.Router.GET("/api/whitelist", s.getWhitelist)
	s.Router.DELETE("/api/whitelist/:prefix", s.deleteWhitelist)
	s.Router.POST("/api/whitelist", s.addWhitelist)
	s.Router.GET("/api/torrent/:info_hash", s.getTorrent)
	s.Router.GET("/api/user/pk/:passkey", s.getUser)
	return &http.Server{
		Addr:           s.Addr,
		Handler:        s.Router,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
}
