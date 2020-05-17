// Package exampleapi implements a trivial reference server implementation
// of the required API routes to communicate as a frontend server for the tracker.
//
package api

import (
	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/mika/model"
	"github.com/leighmacdonald/mika/store"
	"github.com/leighmacdonald/mika/store/memory"
	"github.com/leighmacdonald/mika/util"
	"log"
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
	Users       store.UserStore
	Peers       store.PeerStore
	Torrents    store.TorrentStore
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
	var t model.Torrent
	if err := s.Torrents.Get(&t, infoHash); err != nil || t.IsDeleted {
		errResponse(c, http.StatusNotFound, "Unknown info_hash")
		return
	}
	c.JSON(http.StatusOK, t)
}

func (s *ServerExample) getUserByID(c *gin.Context) {
	userIdStr := c.Param("user_id")
	if userIdStr == "" {
		errResponse(c, http.StatusUnauthorized, "Unauthorized")
		return
	}
	userId := util.StringToUInt32(userIdStr, 0)
	if userId == 0 {
		errResponse(c, http.StatusUnauthorized, "Unauthorized")
		return
	}
	var u model.User
	if err := s.Users.GetByID(&u, userId); err != nil {
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

func (s *ServerExample) getUserByPasskey(c *gin.Context) {
	passkey := c.Param("passkey")
	if passkey == "" {
		errResponse(c, http.StatusUnauthorized, "Unauthorized")
		return
	}
	var u model.User
	if err := s.Users.GetByPasskey(&u, passkey); err != nil {
		errResponse(c, http.StatusUnauthorized, "Unauthorized")
		return
	}
	c.JSON(http.StatusOK, u)
}

// New returns an example HTTP server implementation to test against and learn from
func New() *http.Server {
	userCount := 10
	torrentCount := 100
	swarmSize := 10 // Swarm per torrent
	s := &ServerExample{
		Addr:     "localhost:35000",
		Router:   gin.Default(),
		Torrents: memory.NewTorrentStore(),
		Peers:    memory.NewPeerStore(),
		Users:    memory.NewUserStore(),
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
		if err := s.Users.Add(usr); err != nil {
			log.Panicf("Failed to add user")
		}
	}
	var torrents model.Torrents
	for i := 0; i < torrentCount; i++ {
		t := store.GenerateTestTorrent()
		if err := s.Torrents.Add(t); err != nil {
			log.Panicf("Failed to add torrent")
		}
		torrents = append(torrents, t)
	}
	for _, t := range torrents {
		for i := 0; i < swarmSize; i++ {
			if err := s.Peers.Add(t.InfoHash, store.GenerateTestPeer()); err != nil {
				log.Panicf("Failed to add peer")
			}
		}
	}
	s.Router.GET("/api/whitelist", s.getWhitelist)
	s.Router.DELETE("/api/whitelist/:prefix", s.deleteWhitelist)
	s.Router.POST("/api/whitelist", s.addWhitelist)
	s.Router.GET("/api/torrent/:info_hash", s.getTorrent)
	s.Router.GET("/api/user/pk/:passkey", s.getUserByPasskey)
	s.Router.GET("/api/user/id/:user_id", s.getUserByID)
	return &http.Server{
		Addr:           s.Addr,
		Handler:        s.Router,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
}
