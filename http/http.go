package http

import (
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"mika/http/api"
	"mika/http/tracker"
)

type AnnounceType string

// Announce types
const (
	STARTED   AnnounceType = "started"
	STOPPED   AnnounceType = "stopped"
	COMPLETED AnnounceType = "completed"
	ANNOUNCE  AnnounceType = ""
)

type ErrorResponse struct {
	FailReason string `bencode:"failure reason"`
}

type TrackerErrCode int

const (
	MsgInvalidReqType       TrackerErrCode = 100
	MsgMissingInfoHash      TrackerErrCode = 101
	MsgMissingPeerId        TrackerErrCode = 102
	MsgMissingPort          TrackerErrCode = 103
	MsgInvalidPort          TrackerErrCode = 104
	MsgInvalidInfoHash      TrackerErrCode = 150
	MsgInvalidPeerId        TrackerErrCode = 151
	MsgInvalidNumWant       TrackerErrCode = 152
	MsgOk                   TrackerErrCode = 200
	MsgInfoHashNotFound     TrackerErrCode = 480
	MsgInvalidAuth          TrackerErrCode = 490
	MsgClientRequestTooFast TrackerErrCode = 500
	MsgGenericError         TrackerErrCode = 900
	MsgMalformedRequest     TrackerErrCode = 901
	MsgQueryParseFail       TrackerErrCode = 902
)

var (
	// Error code to message mappings
	responseStringMap = map[TrackerErrCode]error{
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

func TrackerErr(code TrackerErrCode) error {
	return responseStringMap[code]
}

// handleTrackerErrors is used as the default error handler for tracker requests
// the error is returned to the client as a bencoded error string as defined in the
// bittorrent specs.
func handleTrackerErrors(ctx *gin.Context) {
	// Run request handler
	ctx.Next()

	// Handle any errors recorded
	error_returned := ctx.Errors.Last()
	if error_returned != nil {
		meta := error_returned.JSON().(gin.H)

		status := MsgGenericError
		custom_status, found := meta["status"]
		if found {
			status = custom_status.(int)
		}

		// TODO handle private/public errors separately, like sentry output for priv errors
		oops(ctx, status)
	}
}

// NewRouter creates and returns a newly configured router instance using
// the default middleware handlers.
func newRouter() *gin.Engine {
	router := gin.New()
	router.Use(gin.Recovery())
	return router
}

type RequestHandlers struct {
	api.Api
	tracker.BitTorrentHandler
}

func NewHandler() *gin.Engine {
	r := newRouter()
	r.Use(handleTrackerErrors)
	rh := RequestHandlers{}
	r.GET("/:passkey/announce", rh.Announce)
	r.GET("/:passkey/scrape", rh.Scrape)
	return r
}
