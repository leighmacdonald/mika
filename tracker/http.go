package tracker

import (
	"crypto/tls"
	"git.totdev.in/totv/mika/conf"
	//"git.totdev.in/totv/mika/stats"
	log "github.com/Sirupsen/logrus"
	"github.com/goji/httpauth"
	"github.com/labstack/echo"
	ghttp "net/http"
)

// Run starts all of the background goroutines related to managing the tracker
// and starts the tracker and API HTTP interfaces
func (t *Tracker) Run() {
	go t.dbStatIndexer()
	go t.syncWriter()
	go t.peerStalker()
	go t.listenTracker()
	t.listenAPI()
}

// listenTracker created a new http router, configured the routes and handlers, and
// starts the trackers HTTP server listening over HTTP. This function will not
// start the API endpoints. See listenAPI for those.
func (t *Tracker) listenTracker() {
	log.Println("Loading tracker router on:", conf.Config.ListenHost)
	// Initialize the router + middlewares
	e := echo.New()
	e.MaxParam(1)

	// Public tracker routes
	e.Get("/:passkey/announce", t.HandleAnnounce)
	e.Get("/:passkey/scrape", t.HandleScrape)

	e.Run(conf.Config.ListenHost)
}

// listenAPI created a new api request router and start the http server listening over TLS
func (t *Tracker) listenAPI() {
	log.Println("Loading API router on:", conf.Config.ListenHostAPI, "(TLS)")
	e := echo.New()
	e.MaxParam(1)
	api := e.Group("/api")

	// e.Use(stats.StatsMW)

	// Optionally enabled BasicAuth over the TLS only API
	if conf.Config.APIUsername == "" || conf.Config.APIPassword == "" {
		log.Println("[WARN] No credentials set for API. All users granted access.")
	} else {
		api.Use(httpauth.SimpleBasicAuth(conf.Config.APIUsername, conf.Config.APIPassword))
	}

	api.Get("/version", t.HandleVersion)
	api.Get("/uptime", t.HandleUptime)

	api.Get("/torrent/:info_hash", t.HandleTorrentGet)
	api.Post("/torrent", t.HandleTorrentAdd)
	api.Get("/torrent/:info_hash/peers", t.HandleGetTorrentPeers)
	api.Delete("/torrent/:info_hash", t.HandleTorrentDel)

	api.Post("/user", t.HandleUserCreate)
	api.Get("/user/:user_id", t.HandleUserGet)
	api.Post("/user/:user_id", t.HandleUserUpdate)
	api.Get("/user/:user_id/torrents", t.HandleUserTorrents)

	api.Post("/whitelist", t.HandleWhitelistAdd)
	api.Delete("/whitelist/:prefix", t.HandleWhitelistDel)
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
	srv := ghttp.Server{TLSConfig: tls_config, Addr: conf.Config.ListenHostAPI, Handler: e}
	log.Fatal(srv.ListenAndServeTLS(conf.Config.SSLCert, conf.Config.SSLPrivateKey))
}
