// api package is used to remotely control all aspects of the tracker.
package http

import (
	"github.com/gin-gonic/gin"
	"mika/tracker"
)

type AdminAPI struct {
	t *tracker.Tracker
}

func (a *AdminAPI) TorrentDelete(c *gin.Context) {

}

func (a *AdminAPI) TorrentUpdate(c *gin.Context) {

}

func (a *AdminAPI) UserUpdate(c *gin.Context) {

}

func (a *AdminAPI) UserDelete(c *gin.Context) {

}

func (a *AdminAPI) ConfigUpdate(c *gin.Context) {

}

func (a *AdminAPI) Stats(c *gin.Context) {

}
