package tracker_test

import (
	"git.totdev.in/totv/mika/tracker"
	"github.com/gin-gonic/gin"
	"net/http/httptest"
	"testing"
)

var ()

func createTestContext() (ctx *gin.Context, w *httptest.ResponseRecorder, r *gin.Engine) {
	w = httptest.NewRecorder()
	r = gin.New()
//	ctx = &gin.Context{engine: r}
//	ctx.reset()
//	ctx.writermem.reset(w)
	return
}

func TestHandleVersion(t *testing.T) {
	ctx, _, _ := createTestContext()
	trk := tracker.MakeTestTracker()
	trk.HandleVersion(ctx)
}
