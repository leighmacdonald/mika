// Package http is used to remotely control all aspects of the tracker.
package http

import (
	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/mika/config"
	"github.com/leighmacdonald/mika/consts"
	"github.com/leighmacdonald/mika/model"
	"github.com/leighmacdonald/mika/tracker"
	"net/http"
)

// AdminAPI is the interface for administering a live server over HTTP
type AdminAPI struct {
	t *tracker.Tracker
}

func infoHashFromCtx(c *gin.Context) (model.InfoHash, bool) {
	ihStr := c.Param("info_hash")
	if ihStr == "" {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{
			"message": "Invalid info hash",
		})
		return model.InfoHash{}, false
	}
	return model.InfoHashFromString(ihStr), true
}
func (a *AdminAPI) torrentDelete(c *gin.Context) {
	ih, ok := infoHashFromCtx(c)
	if !ok {
		return
	}
	if err := a.t.Torrents.Delete(ih, true); err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{})
		return
	}
	c.JSON(http.StatusOK, gin.H{})
}

// TorrentUpdatePrams defines what parameters we accept for updating a torrent. This is only
// a subset of the fields as not all should be considered mutable
type TorrentUpdatePrams struct {
	IsDeleted bool   `json:"is_deleted"`
	IsEnabled bool   `json:"is_enabled"`
	Reason    string `json:"reason"`
}

func (a *AdminAPI) torrentUpdate(c *gin.Context) {
	ih, ok := infoHashFromCtx(c)
	if !ok {
		return
	}
	t, err := a.t.Torrents.Get(ih)
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

func (a *AdminAPI) userUpdate(c *gin.Context) {

}

func (a *AdminAPI) userDelete(c *gin.Context) {

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
			a.t.AnnInterval = v.(int)
		case config.TrackerAnnounceIntervalMin:
			a.t.AnnIntervalMin = v.(int)
		}
	}
	c.JSON(http.StatusOK, gin.H{})
}

func (a *AdminAPI) stats(c *gin.Context) {

}
