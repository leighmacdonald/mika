// Package example_api implements a trivial reference server implementation
// of the required API routes to communicate as a frontend server for the tracker.
package example_api

import (
	"github.com/gin-gonic/gin"
	"math/rand"
	"mika/model"
	"mika/store"
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

type ServerExample struct {
	Addr       string
	Router     *gin.Engine
	Users      map[string]*model.User
	UsersMx    sync.RWMutex
	Peers      map[model.InfoHash]*model.Peer
	PeersMx    sync.RWMutex
	Torrents   []*model.Torrent
	TorrentsMx sync.RWMutex
}

func (s *ServerExample) getTorrent(c *gin.Context) {
	infoHash := c.Param("info_hash")
	if infoHash == "" {
		errResponse(c, http.StatusNotFound, "Unknown info_hash")
		return
	}
}

func (s *ServerExample) getUser(c *gin.Context) {
	passKey := c.Param("passkey")
	if passKey == "" {
		errResponse(c, http.StatusUnauthorized, "Unauthorized")
		return
	}
}

func (s *ServerExample) Run() error {
	return s.Router.Run(s.Addr)
}

func New() *http.Server {
	userCount := 10
	torrentCount := 100
	peerCount := 1000
	s := &ServerExample{
		Addr:       "localhost:8080",
		Router:     gin.Default(),
		UsersMx:    sync.RWMutex{},
		PeersMx:    sync.RWMutex{},
		TorrentsMx: sync.RWMutex{},
		Peers:      map[model.InfoHash]*model.Peer{},
		Users:      map[string]*model.User{},
	}
	for i := 0; i < userCount; i++ {
		usr := store.GenerateTestUser()
		s.Users[usr.Passkey] = usr
	}
	for i := 0; i < torrentCount; i++ {
		s.Torrents = append(s.Torrents, store.GenerateTestTorrent())
	}
	for i := 0; i < peerCount; i++ {
		s.Peers[s.Torrents[rand.Int31n(int32(torrentCount-1))].InfoHash] = store.GenerateTestPeer()
	}
	s.Router.GET("/api/torrent/:info_hash", s.getTorrent)
	s.Router.GET("/api/user/pk/:passkey", s.getUser)
	return &http.Server{
		Addr:           s.Addr,
		Handler:        s.Router,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
		TLSConfig:      nil,
	}
}
