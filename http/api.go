// Package http is used to remotely control all aspects of the tracker.
package http

import (
	"github.com/gin-gonic/gin"
	"mika/tracker"
)

type AdminAPI struct {
	t *tracker.Tracker
}

func (a *AdminAPI) torrentDelete(c *gin.Context) {

}

func (a *AdminAPI) torrentUpdate(c *gin.Context) {

}

func (a *AdminAPI) userUpdate(c *gin.Context) {

}

func (a *AdminAPI) userDelete(c *gin.Context) {

}

func (a *AdminAPI) configUpdate(c *gin.Context) {

}

func (a *AdminAPI) stats(c *gin.Context) {

}
