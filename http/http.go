package http

import (
	"bytes"
	"crypto/tls"
	"github.com/chihaya/bencode"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
	"mika/tracker"
	"net"
	"net/http"
	"strings"
	"time"
)

type AnnounceType string

// Announce types
const (
	STARTED   AnnounceType = "started"
	STOPPED   AnnounceType = "stopped"
	COMPLETED AnnounceType = "completed"
	ANNOUNCE  AnnounceType = ""
)

func parseAnnounceType(t string) AnnounceType {
	switch t {
	case "started":
		return STARTED
	case "stopped":
		return STOPPED
	case "completed":
		return COMPLETED
	default:
		return ANNOUNCE
	}
}

type errorResponse struct {
	FailReason string `bencode:"failure reason"`
}

type trackerErrCode int

const (
	MsgInvalidReqType       trackerErrCode = 100
	MsgMissingInfoHash      trackerErrCode = 101
	MsgMissingPeerId        trackerErrCode = 102
	MsgMissingPort          trackerErrCode = 103
	MsgInvalidPort          trackerErrCode = 104
	MsgInvalidInfoHash      trackerErrCode = 150
	MsgInvalidPeerId        trackerErrCode = 151
	MsgInvalidNumWant       trackerErrCode = 152
	MsgOk                   trackerErrCode = 200
	MsgInfoHashNotFound     trackerErrCode = 480
	MsgInvalidAuth          trackerErrCode = 490
	MsgClientRequestTooFast trackerErrCode = 500
	MsgGenericError         trackerErrCode = 900
	MsgMalformedRequest     trackerErrCode = 901
	MsgQueryParseFail       trackerErrCode = 902
)

var (
	// Error code to message mappings
	responseStringMap = map[trackerErrCode]error{
		MsgInvalidReqType:       errors.New("Invalid request type"),
		MsgMissingInfoHash:      errors.New("info_hash missing from request"),
		MsgMissingPeerId:        errors.New("peer_id missing from request"),
		MsgMissingPort:          errors.New("port missing from request"),
		MsgInvalidPort:          errors.New("Invalid port"),
		MsgInvalidAuth:          errors.New("Invalid passkey supplied"),
		MsgInvalidInfoHash:      errors.New("Torrent info hash must be 20 characters"),
		MsgInvalidPeerId:        errors.New("Peer ID Invalid"),
		MsgInvalidNumWant:       errors.New("num_want invalid"),
		MsgInfoHashNotFound:     errors.New("Unknown infohash"),
		MsgClientRequestTooFast: errors.New("Slow down there jimmy."),
		MsgMalformedRequest:     errors.New("Malformed request"),
		MsgGenericError:         errors.New("Generic Error"),
		MsgQueryParseFail:       errors.New("Could not parse request"),
	}
)

func TrackerErr(code trackerErrCode) error {
	return responseStringMap[code]
}

// getIP Parses and returns a IP from a string
func getIP(q *Query, c *gin.Context) (net.IP, error) {
	ipStr, found := q.Params[paramIP]
	if found {
		ip := net.ParseIP(ipStr)
		if ip != nil {
			return ip.To4(), nil
		}
	}
	// Look for forwarded ip in header then default to remote address
	forwardedIp := c.Request.Header.Get("X-Forwarded-For")
	if forwardedIp != "" {
		ip := net.ParseIP(forwardedIp)
		if ip != nil {
			return ip.To4(), nil
		}
		return ip, nil
	} else {
		s := strings.Split(c.Request.RemoteAddr, ":")
		ipReq, _ := s[0], s[1]
		ip := net.ParseIP(ipReq)
		if ip != nil {
			return ip.To4(), nil
		}
		return ip, nil
	}
}

// oops will output a bencoded error code to the torrent client using
// a preset message code constant
func oops(ctx *gin.Context, errCode trackerErrCode) {
	msg, exists := responseStringMap[errCode]
	if !exists {
		msg = responseStringMap[MsgGenericError]
	}
	ctx.String(int(errCode), responseError(msg.Error()))

	log.Println("Error in request (", errCode, "):", msg)
	log.Println("From:", ctx.Request.RequestURI)
}

// handleTrackerErrors is used as the default error handler for tracker requests
// the error is returned to the client as a bencoded error string as defined in the
// bittorrent specs.
func handleTrackerErrors(ctx *gin.Context) {
	// Run request handler
	ctx.Next()

	// Handle any errors recorded
	errorReturned := ctx.Errors.Last()
	if errorReturned != nil {
		meta := errorReturned.JSON().(gin.H)

		status := MsgGenericError
		customStatus, found := meta["status"]
		if found {
			status = customStatus.(trackerErrCode)
		}

		// TODO handle private/public errors separately, like sentry output for priv errors
		oops(ctx, status)
	}
}

// responseError generates a bencoded error response for the torrent client to
// parse and display to the user
//
// Note that this function does not generate or support a warning reason, which are rarely if
// ever used.
func responseError(message string) string {
	var buf bytes.Buffer
	encoder := bencode.NewEncoder(&buf)
	if err := encoder.Encode(bencode.Dict{
		"failure reason": message,
	}); err != nil {
		log.Errorf("Failed to encode error response: %s", err)
	}
	return buf.String()
}

// NewRouter creates and returns a newly configured router instance using
// the default middleware handlers.
func newRouter() *gin.Engine {
	router := gin.New()
	router.Use(gin.Recovery())
	return router
}

func NewBitTorrentHandler(tkr *tracker.Tracker) *gin.Engine {
	r := newRouter()
	r.Use(handleTrackerErrors)
	h := BitTorrentHandler{
		t: tkr,
	}
	r.GET("/:passkey/announce", h.Announce)
	r.GET("/:passkey/scrape", h.Scrape)
	return r
}

func NewAPIHandler(tkr *tracker.Tracker) *gin.Engine {
	r := newRouter()
	h := AdminAPI{
		t: tkr,
	}
	r.GET("/tracker/stats", h.Stats)
	r.DELETE("/torrent/:info_hash", h.TorrentDelete)
	r.PATCH("/torrent/:info_hash", h.TorrentUpdate)
	return r
}

func CreateServer(router http.Handler, addr string, useTLS bool) *http.Server {
	var tlsCfg *tls.Config
	if useTLS {
		tlsCfg = &tls.Config{
			MinVersion:               tls.VersionTLS12,
			CurvePreferences:         []tls.CurveID{tls.CurveP521, tls.CurveP384, tls.CurveP256},
			PreferServerCipherSuites: true,
			CipherSuites: []uint16{
				tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
				tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_RSA_WITH_AES_256_CBC_SHA,
			},
		}
	} else {
		tlsCfg = nil
	}
	srv := &http.Server{
		Addr:           addr,
		Handler:        router,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
		TLSConfig:      tlsCfg,
		//TLSNextProto: make(map[string]func(*http.Server, *tls.Conn, http.Handler), 0),
	}
	return srv
}

// Doesnt seem to work? Causes duplicate port use??
func StartListeners(s []*http.Server) {
	var g errgroup.Group
	for _, server := range s {
		g.Go(func() error {
			err := server.ListenAndServe()
			if err != nil && err != http.ErrServerClosed {
				log.Fatal(err)
			}
			return err
		})
	}
	if err := g.Wait(); err != nil {
		log.Fatal(err)
	}
}
