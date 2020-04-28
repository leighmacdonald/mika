// api package is used to remotely control all aspects of the tracker.
package http

import (
	"github.com/gin-gonic/gin"
	"mika/tracker"
)

type Api struct {
	t *tracker.Tracker
}

func (a *Api) TorrentDelete(c *gin.Context) {

}

func (a *Api) TorrentUpdate(c *gin.Context) {

}

func (a *Api) UserUpdate(c *gin.Context) {

}

func (a *Api) UserDelete(c *gin.Context) {

}

func (a *Api) ConfigUpdate(c *gin.Context) {

}

func (a *Api) Stats(c *gin.Context) {

}
