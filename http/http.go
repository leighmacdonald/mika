package http

import (
	"bytes"
	"github.com/chihaya/bencode"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"net"
	"strings"
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

// handleTrackerErrors is used as the default error handler for tracker requests
// the error is returned to the client as a bencoded error string as defined in the
// bittorrent specs.
//func handleTrackerErrors(ctx *gin.Context) {
//	// Run request handler
//	ctx.Next()
//
//	// Handle any errors recorded
//	error_returned := ctx.Errors.Last()
//	if error_returned != nil {
//		meta := error_returned.JSON().(gin.H)
//
//		status := MsgGenericError
//		custom_status, found := meta["status"]
//		if found {
//			status = custom_status.(int)
//		}
//
//		// TODO handle private/public errors separately, like sentry output for priv errors
//		oops(ctx, status)
//	}
//}

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

type RequestHandlers struct {
	Api
	BitTorrentHandler
}

func NewHandler() *gin.Engine {
	r := newRouter()
	//r.Use(handleTrackerErrors)
	rh := RequestHandlers{}
	r.GET("/:passkey/announce", rh.Announce)
	r.GET("/:passkey/scrape", rh.Scrape)
	return r
}
