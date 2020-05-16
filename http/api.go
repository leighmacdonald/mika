// Package http is used to remotely control all aspects of the tracker.
package http

import (
	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/mika/config"
	"github.com/leighmacdonald/mika/consts"
	"github.com/leighmacdonald/mika/model"
	"github.com/leighmacdonald/mika/tracker"
	"github.com/leighmacdonald/mika/util"
	log "github.com/sirupsen/logrus"
	"net/http"
	"time"
)

// StatusResp is a generic response struct used when simple responses are all that
// is required.
type StatusResp struct {
	Err     string `json:"error,omitempty"`
	Message string `json:"message,omitempty"`
}

// Error implements the error interface for our response
func (s StatusResp) Error() string {
	return s.Err
}

// AdminAPI is the interface for administering a live server over HTTP
type AdminAPI struct {
	t *tracker.Tracker
}

// PingRequest represents a JSON ping request
type PingRequest struct {
	Ping string `json:"ping"`
}

// PingResponse represents a JSON ping response
type PingResponse struct {
	Pong string `json:"pong"`
}

func (a *AdminAPI) whitelistAdd(c *gin.Context) {
	var wcl model.WhiteListClient
	if err := c.BindJSON(&wcl); err != nil {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	if wcl.ClientPrefix == "" || wcl.ClientName == "" {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	if err := a.t.Torrents.WhiteListAdd(wcl); err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
	}
	a.t.WhitelistMutex.Lock()
	a.t.Whitelist[wcl.ClientPrefix] = wcl
	a.t.WhitelistMutex.Unlock()
	c.JSON(http.StatusOK, nil)
}

func (a *AdminAPI) whitelistDelete(c *gin.Context) {
	prefix := c.Param("prefix")
	if prefix == "" {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	a.t.WhitelistMutex.RLock()
	defer a.t.WhitelistMutex.RUnlock()
	wlc := a.t.Whitelist[prefix]
	if err := a.t.Torrents.WhiteListDelete(wlc); err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	newWL := make(map[string]model.WhiteListClient)
	wl, err := a.t.Torrents.WhiteListGetAll()
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	for _, w := range wl {
		newWL[w.ClientPrefix] = w
	}
	a.t.WhitelistMutex.Lock()
	a.t.Whitelist = newWL
	a.t.WhitelistMutex.Unlock()
	c.JSON(http.StatusOK, nil)
}

func (a *AdminAPI) whitelistGet(c *gin.Context) {
	var wl []model.WhiteListClient
	a.t.WhitelistMutex.RLock()
	defer a.t.WhitelistMutex.RUnlock()
	for _, c := range a.t.Whitelist {
		wl = append(wl, c)
	}
	c.JSON(http.StatusOK, wl)
}

func (a *AdminAPI) ping(c *gin.Context) {
	var r PingRequest
	if err := c.BindJSON(&r); err != nil {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	c.JSON(http.StatusOK, PingResponse{Pong: r.Ping})
}

func infoHashFromCtx(infoHash *model.InfoHash, c *gin.Context) bool {
	ihStr := c.Param("info_hash")
	if ihStr == "" {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{
			"message": "Invalid info hash",
		})
		return false
	}
	if err := model.InfoHashFromString(infoHash, ihStr); err != nil {
		log.Warnf("failed to parse info hash from request context: %s", err.Error())
		return false
	}
	return true
}

// TorrentAddRequest represents a JSON request for adding a new torrent
type TorrentAddRequest struct {
	Name     string `json:"name"`
	InfoHash string `json:"info_hash"`
}

func (a *AdminAPI) torrentAdd(c *gin.Context) {
	var req TorrentAddRequest
	if err := c.BindJSON(&req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, StatusResp{Err: "Malformed request"})
		return
	}
	var t model.Torrent
	var ih model.InfoHash
	if err := model.InfoHashFromString(&ih, req.InfoHash); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, StatusResp{Err: err.Error()})
		return
	}
	t.ReleaseName = req.Name
	t.InfoHash = ih
	if err := a.t.Torrents.Add(t); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, StatusResp{Err: err.Error()})
		return
	}
	c.JSON(http.StatusOK, StatusResp{Message: "Torrent added successfully"})
}

func (a *AdminAPI) torrentDelete(c *gin.Context) {
	var infoHash model.InfoHash
	if !infoHashFromCtx(&infoHash, c) {
		return
	}
	if err := a.t.Torrents.Delete(infoHash, true); err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{})
		return
	}
	c.JSON(http.StatusOK, StatusResp{Message: "Deleted successfully"})
}

// TorrentUpdatePrams defines what parameters we accept for updating a torrent. This is only
// a subset of the fields as not all should be considered mutable
type TorrentUpdatePrams struct {
	IsDeleted bool   `json:"is_deleted"`
	IsEnabled bool   `json:"is_enabled"`
	Reason    string `json:"reason"`
}

func (a *AdminAPI) torrentUpdate(c *gin.Context) {
	var ih model.InfoHash
	if !infoHashFromCtx(&ih, c) {
		return
	}
	var t model.Torrent
	err := a.t.Torrents.Get(&t, ih)
	if err == consts.ErrInvalidInfoHash {
		c.JSON(http.StatusNotFound, gin.H{})
		return
	}
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{})
		return
	}
	var tup TorrentUpdatePrams
	if err := c.BindJSON(&tup); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{})
		return
	}
	// TODO use update channel
	t.Reason = tup.Reason
	t.IsDeleted = tup.IsDeleted
	t.IsEnabled = tup.IsEnabled
	c.JSON(http.StatusOK, tup)

}

//func (a *AdminAPI) userUpdate(_ *gin.Context) {
//
//}

// UserDeleteRequest represents a JSON API requests to delete a user via passkey
type UserDeleteRequest struct {
	Passkey string `json:"passkey"`
}

func (a *AdminAPI) userDelete(c *gin.Context) {
	pk := c.Param("passkey")
	var user model.User
	if err := a.t.Users.GetByPasskey(&user, pk); err != nil {
		c.AbortWithStatusJSON(http.StatusNotFound, StatusResp{Err: "User not found"})
		return
	}
	if err := a.t.Users.Delete(user); err != nil {
		c.AbortWithStatusJSON(http.StatusNotFound, StatusResp{Err: "Failed to delete user"})
		return
	}
	c.JSON(http.StatusOK, StatusResp{Message: "Deleted user successfully"})
}

// UserAddRequest represents a JSON API requests to add a user
type UserAddRequest struct {
	UserID  uint32 `json:"user_id,omitempty"`
	Passkey string `json:"passkey,omitempty"`
}

// UserAddResponse represents a JSON API response to adding a user
type UserAddResponse struct {
	Passkey string `json:"passkey"`
}

func (a *AdminAPI) userAdd(c *gin.Context) {
	var user model.User
	var req UserAddRequest
	if err := c.BindJSON(&req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, StatusResp{Err: "Malformed request"})
		return
	}
	user.DownloadEnabled = true
	if req.Passkey == "" {
		user.Passkey = util.NewPasskey()
	} else {
		user.Passkey = req.Passkey
	}
	if req.UserID > 0 {
		user.UserID = req.UserID
	}
	if err := a.t.Users.Add(user); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, StatusResp{Err: "Failed to add user"})
		return
	}
	c.JSON(http.StatusOK, UserAddResponse{Passkey: user.Passkey})
}

func (a *AdminAPI) configUpdate(c *gin.Context) {
	var configValues map[config.Key]interface{}
	if err := c.BindJSON(&configValues); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{})
		return
	}
	for k, v := range configValues {
		// TODO lock tracker
		switch k {
		case config.TrackerAnnounceInterval:
			d, err := time.ParseDuration(v.(string))
			if err != nil {
				c.AbortWithStatusJSON(http.StatusBadRequest, StatusResp{Err: "Announce interval invalid format"})
				return
			}
			a.t.AnnInterval = d
		case config.TrackerAnnounceIntervalMin:
			d, err := time.ParseDuration(v.(string))
			if err != nil {
				c.AbortWithStatusJSON(http.StatusBadRequest, StatusResp{Err: "Announce interval min invalid format"})
				return
			}
			a.t.AnnIntervalMin = d
		}
	}
	c.JSON(http.StatusOK, gin.H{})
}

func (a *AdminAPI) stats(_ *gin.Context) {

}
