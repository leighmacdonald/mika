package tracker

import (
	"crypto/tls"
	"git.totdev.in/totv/mika/conf"
	"github.com/Sirupsen/logrus"
	log "github.com/Sirupsen/logrus"
	"github.com/gin-gonic/gin"
	"net/http"
)

type (
	APIErrorResponse struct {
		Code    int    `json:"code"`
		Error   string `json:"error"`
		Message string `json:"message"`
	}
)

// handleTrackerErrors is used as the default error handler for tracker requests
// the error is returned to the client as a bencoded error string as defined in the
// bittorrent specs.
func handleTrackerErrors(ctx *gin.Context) {
	ctx.Next()

	error_returned := ctx.Errors.Last()
	if error_returned != nil {
		meta := error_returned.JSON().(gin.H)

		status := MSG_GENERIC_ERROR
		custom_status, found := meta["status"]
		if found {
			status = custom_status.(int)
		}
		log.Debug(meta)
		// TODO handle private/public errors separately, like sentry output for priv errors
		oops(ctx, status)
	}
}

// APIErrorHandler is used as the default error handler for API requests
func handleApiErrors(ctx *gin.Context) {
	// Execute the next handler, recording any errors to be process below
	ctx.Next()

	error_returned := ctx.Errors.Last()
	if error_returned != nil {
		meta := error_returned.JSON().(gin.H)

		status := status
		custom_status, found := meta["status"]
		if found {
			status = custom_status.(int)
		}

		// TODO handle private/public errors separately, like sentry output for priv errors
		if error_returned.Meta != nil {
			ctx.JSON(status, meta)
		}
	}
}

// Run starts all of the background goroutines related to managing the tracker
// and starts the tracker and API HTTP interfaces
func (tracker *Tracker) Run() {

	go tracker.dbStatIndexer()
	go tracker.syncWriter()
	go tracker.peerStalker()
	go tracker.listenTracker()
	go tracker.listenAPI()

	select {
	case <-tracker.stopChan:
		log.Info("Exiting on stop chan signal")
	}
}

// listenTracker created a new http router, configured the routes and handlers, and
// starts the trackers HTTP server listening over HTTP. This function will not
// start the API endpoints. See listenAPI for those.
func (tracker *Tracker) listenTracker() {
	log.WithFields(log.Fields{
		"listen_host": conf.Config.ListenHost,
		"tls":         false,
	}).Info("Loading Tracker route handlers")

	router := NewRouter()
	router.Use(handleTrackerErrors)
	router.GET("/:passkey/announce", tracker.HandleAnnounce)
	router.GET("/:passkey/scrape", tracker.HandleScrape)

	router.Run(conf.Config.ListenHost)
}

func errMeta(status int, message string, fields logrus.Fields, level logrus.Level) gin.H {
	return gin.H{
		"status":  status,
		"message": message,
		"fields":  fields,
		"level":   level,
	}
}

// NewRouter creates and returns a newly configured router instance using
// the default middleware handlers.
func NewRouter() *gin.Engine {
	router := gin.New()
	router.Use(gin.Recovery())
	return router
}

// listenAPI creates a new api request router and start the http server listening over TLS
func (tracker *Tracker) listenAPI() {
	log.WithFields(log.Fields{
		"listen_host": conf.Config.ListenHostAPI,
		"tls":         true,
	}).Info("Loading API route handlers")

	router := NewRouter()
	router.Use(handleApiErrors)

	var api *gin.RouterGroup

	// Optionally enabled BasicAuth over the TLS only API
	if conf.Config.APIUsername == "" || conf.Config.APIPassword == "" {
		log.Warn("No credentials set for API. All users granted access.")
		api = router.Group("/api")
	} else {
		api = router.Group("/api", gin.BasicAuth(gin.Accounts{
			conf.Config.APIUsername: conf.Config.APIPassword,
		}))
	}

	api.GET("/version", tracker.HandleVersion)
	api.GET("/uptime", tracker.HandleUptime)
	api.GET("/torrent/:info_hash", tracker.HandleTorrentGet)
	api.POST("/torrent", tracker.HandleTorrentAdd)
	api.GET("/torrent/:info_hash/peers", tracker.HandleGetTorrentPeers)
	api.DELETE("/torrent/:info_hash", tracker.HandleTorrentDel)

	api.POST("/user", tracker.HandleUserCreate)
	api.GET("/user/:user_id", tracker.HandleUserGet)
	api.POST("/user/:user_id", tracker.HandleUserUpdate)
	api.DELETE("/user/:user_id", tracker.HandleUserDel)
	api.GET("/user/:user_id/torrents", tracker.HandleUserTorrents)

	api.POST("/whitelist", tracker.HandleWhitelistAdd)
	api.DELETE("/whitelist/:prefix", tracker.HandleWhitelistDel)

	tls_config := &tls.Config{
		MinVersion:               tls.VersionTLS12,
		PreferServerCipherSuites: true,
		CipherSuites: []uint16{tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
			tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA,
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA,
		},
	}
	if conf.Config.SSLCert == "" || conf.Config.SSLPrivateKey == "" {
		log.Fatalln("SSL config keys not set in config!")
	}
	srv := http.Server{TLSConfig: tls_config, Addr: conf.Config.ListenHostAPI, Handler: router}
	log.Fatal(srv.ListenAndServeTLS(conf.Config.SSLCert, conf.Config.SSLPrivateKey))
}
