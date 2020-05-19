// Package exampleapi implements a trivial reference server implementation
// of the required API routes to communicate as a frontend server for the tracker.
//
package api

import (
	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/mika/model"
	"github.com/leighmacdonald/mika/store"
	"github.com/leighmacdonald/mika/store/memory"
	"github.com/leighmacdonald/mika/tracker"
	"github.com/leighmacdonald/mika/util"
	"log"
	"net/http"
	"sync"
	"time"
)

const (
	DefaultAuthKey = "12345678901234567890"
)

type (
	// ServerExample provides and example / demo server implementation that conforms to mika's HTTP store backend.
	ServerExample struct {
		Addr        string
		Router      *gin.Engine
		Users       store.UserStore
		Peers       store.PeerStore
		Torrents    store.TorrentStore
		WhiteList   map[string]model.WhiteListClient
		WhiteListMx *sync.RWMutex
	}

	errMsg struct {
		Status int    `json:"status"`
		Error  string `json:"error"`
	}
	okMsg struct {
		Message string `json:"message"`
	}
)

func errResponse(c *gin.Context, code int, msg string) {
	c.JSON(code, errMsg{code, msg})
}

func okResponse(c *gin.Context, msg string) {
	c.JSON(http.StatusOK, okMsg{msg})
}

func getInfoHashParam(ih *model.InfoHash, c *gin.Context) bool {
	infoHashStr := c.Param("info_hash")
	if infoHashStr == "" {
		errResponse(c, http.StatusNotFound, "Unknown info_hash")
		return false
	}
	if err := model.InfoHashFromHex(ih, infoHashStr); err != nil {
		errResponse(c, http.StatusBadRequest, "Malformed info_hash")
		return false
	}
	return true
}

func getPeerIDParam(peerId *model.PeerID, c *gin.Context) bool {
	peerIdStr := c.Param("peer_id")
	if peerIdStr == "" {
		errResponse(c, http.StatusNotFound, "Unknown info_hash")
		return false
	}
	*peerId = model.PeerIDFromString(peerIdStr)
	return true
}

func (s *ServerExample) getTorrent(c *gin.Context) {
	var infoHash model.InfoHash
	if !getInfoHashParam(&infoHash, c) {
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
	okResponse(c, "Deleted prefix successfully")
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
	c.JSON(http.StatusOK, wlc)
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

func (s *ServerExample) deleteTorrent(c *gin.Context) {
	var infoHash model.InfoHash
	if !getInfoHashParam(&infoHash, c) {
		return
	}
	if err := s.Torrents.Delete(infoHash, true); err != nil {
		errResponse(c, http.StatusInternalServerError, err.Error())
		return
	}
	okResponse(c, "Deleted torrent successfully")
}

func (s *ServerExample) addTorrent(c *gin.Context) {
	var torrent model.Torrent
	if err := c.BindJSON(&torrent); err != nil {
		errResponse(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := s.Torrents.Add(torrent); err != nil {
		errResponse(c, http.StatusInternalServerError, err.Error())
		return
	}
	okResponse(c, "Added torrent successfully")
}

func (s *ServerExample) userAdd(c *gin.Context) {
	var userReq tracker.UserAddRequest
	if err := c.BindJSON(&userReq); err != nil {
		errResponse(c, http.StatusBadRequest, err.Error())
		return
	}
	user := model.User{
		UserID:  userReq.UserID,
		Passkey: userReq.Passkey,
	}
	if err := s.Users.Add(user); err != nil {
		errResponse(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := s.Users.GetByID(&user, user.UserID); err != nil {
		errResponse(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, user)
}

func (s *ServerExample) userSync(c *gin.Context) {
	var batch map[string]model.UserStats
	if err := c.BindJSON(&batch); err != nil {
		errResponse(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := s.Users.Sync(batch); err != nil {
		errResponse(c, http.StatusInternalServerError, err.Error())
		return
	}
	okResponse(c, "sync successful")
}

func (s *ServerExample) userDelete(c *gin.Context) {
	var us tracker.UserDeleteRequest
	if err := c.BindJSON(&us); err != nil {
		errResponse(c, http.StatusBadRequest, err.Error())
		return
	}
	var user model.User
	if err := s.Users.GetByPasskey(&user, us.Passkey); err != nil {
		errResponse(c, http.StatusNotFound, "User does not exist")
		return
	}
	if err := s.Users.Delete(user); err != nil {
		errResponse(c, http.StatusInternalServerError, err.Error())
		return
	}
	okResponse(c, "Deleted user successfully")
}

func (s *ServerExample) torrentSync(c *gin.Context) {
	var batch map[model.InfoHash]model.TorrentStats
	if err := c.BindJSON(batch); err != nil {
		errResponse(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := s.Torrents.Sync(batch); err != nil {
		errResponse(c, http.StatusInternalServerError, err.Error())
		return
	}
	okResponse(c, "torrent sync successful")
}

func (s *ServerExample) peersSync(c *gin.Context) {
	var batch map[string]model.PeerStats
	if err := c.BindJSON(&batch); err != nil {
		errResponse(c, http.StatusBadRequest, err.Error())
		return
	}
	rb := make(map[model.PeerHash]model.PeerStats)
	for k, v := range batch {
		var ph model.PeerHash
		if err := model.PeerHashFromHex(&ph, k); err != nil {
			errResponse(c, http.StatusBadRequest, err.Error())
			return
		}
		rb[ph] = v
	}
	if err := s.Peers.Sync(rb); err != nil {
		errResponse(c, http.StatusInternalServerError, err.Error())
		return
	}
	okResponse(c, "peer sync successful")
}

func (s *ServerExample) peersAdd(c *gin.Context) {
	var (
		peer     model.Peer
		infoHash model.InfoHash
	)
	if !getInfoHashParam(&infoHash, c) {
		return
	}
	if err := c.BindJSON(&peer); err != nil {
		errResponse(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := s.Peers.Add(infoHash, peer); err != nil {
		errResponse(c, http.StatusInternalServerError, err.Error())
		return
	}
	okResponse(c, "Peer added to swarm successfully")
}

func (s *ServerExample) peersDelete(c *gin.Context) {
	var (
		peerID   model.PeerID
		infoHash model.InfoHash
	)
	if !getPeerIDParam(&peerID, c) || !getInfoHashParam(&infoHash, c) {
		return
	}
	if err := s.Peers.Delete(infoHash, peerID); err != nil {
		errResponse(c, http.StatusInternalServerError, err.Error())
		return
	}
	okResponse(c, "Successfully deleted peer")
}

func (s *ServerExample) peersGetN(c *gin.Context) {
	var infoHash model.InfoHash
	if !getInfoHashParam(&infoHash, c) {
		return
	}
	maxLimit := 100
	limit := 25
	limitStr, found := c.GetQuery("limit")
	if found {
		limit = util.StringToUInt(limitStr, limit)
	}
	if limit > maxLimit {
		limit = maxLimit
	}
	peers, err := s.Peers.GetN(infoHash, limit)
	if err != nil {
		errResponse(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, peers)
}

func (s *ServerExample) peersGet(c *gin.Context) {
	var (
		peer     model.Peer
		peerID   model.PeerID
		infoHash model.InfoHash
	)
	if !getPeerIDParam(&peerID, c) || !getInfoHashParam(&infoHash, c) {
		return
	}
	if err := s.Peers.Get(&peer, infoHash, peerID); err != nil {
		errResponse(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, peer)
}

func (s *ServerExample) peersReap(c *gin.Context) {
	s.Peers.Reap()
	okResponse(c, "reaped")
}

// New returns an example HTTP server implementation to test against and learn from
// Set the Authorization header to the Auth
func New(listenAddr string, pathPrefix string, authKey string) *http.Server {
	userCount := 10
	torrentCount := 100
	swarmSize := 10 // Swarm per torrent
	router := gin.New()
	router.Use(func(c *gin.Context) {
		clientKey := c.GetHeader("Authorization")
		if authKey != clientKey {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized token"})
			return
		}
		// Continue down the chain to handler etc
		c.Next()
	})
	s := &ServerExample{
		Addr:        listenAddr,
		Router:      router,
		Torrents:    memory.NewTorrentStore(),
		Peers:       memory.NewPeerStore(),
		Users:       memory.NewUserStore(),
		WhiteList:   map[string]model.WhiteListClient{},
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

	// Conn() and Close() do not need any endpoints, they are noop when using http backed
	// stores.

	// UserStore implementations

	// UserStore.Add
	s.Router.POST(pathPrefix+"/api/user", s.userAdd)
	// UserStore.Delete
	s.Router.DELETE(pathPrefix+"/api/user/:passkey", s.userDelete)
	// UserStore.Sync
	s.Router.POST(pathPrefix+"/api/user/sync", s.userSync)
	// UserStore.GetByPasskey
	s.Router.GET(pathPrefix+"/api/user/pk/:passkey", s.getUserByPasskey)
	// UserStore.GetByID
	s.Router.GET(pathPrefix+"/api/user/id/:user_id", s.getUserByID)

	// TorrentStore implementations

	// TorrentStore.WhiteListGetAll
	s.Router.GET(pathPrefix+"/api/whitelist", s.getWhitelist)
	// TorrentStore.WhiteListDelete
	s.Router.DELETE(pathPrefix+"/api/whitelist/:prefix", s.deleteWhitelist)
	// TorrentStore.WhiteListAdd
	s.Router.POST(pathPrefix+"/api/whitelist", s.addWhitelist)
	// TorrentStore.Add
	s.Router.POST(pathPrefix+"/api/torrent", s.addTorrent)
	// TorrentStore.Get
	s.Router.GET(pathPrefix+"/api/torrent/:info_hash", s.getTorrent)
	// TorrentStore.Delete
	s.Router.DELETE(pathPrefix+"/api/torrent/:info_hash", s.deleteTorrent)
	// TorrentStore.Sync
	s.Router.POST(pathPrefix+"/api/torrent/sync", s.torrentSync)

	// PeerStore implementations

	// PeerStore.Reap
	s.Router.GET(pathPrefix+"/api/peers/reap", s.peersReap)
	// PeerStore.Sync
	s.Router.POST(pathPrefix+"/api/peers/sync", s.peersSync)
	// PeerStore.Add
	s.Router.POST(pathPrefix+"/api/peer/create/:info_hash", s.peersAdd)
	// PeerStore.Delete
	s.Router.DELETE(pathPrefix+"/api/peers/delete/:info_hash/:peer_id", s.peersDelete)
	// PeerStore.GetN
	s.Router.GET(pathPrefix+"/api/peers/swarm/:info_hash/:count", s.peersGetN)
	// PeerStore.Get
	s.Router.GET(pathPrefix+"/api/peer/:info_hash/:peer_id", s.peersGet)

	return &http.Server{
		Addr:           s.Addr,
		Handler:        s.Router,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
}
