package tracker

import (
	"bytes"
	"github.com/chihaya/bencode"
	"github.com/gin-gonic/gin"
	"log"
)

type ErrorResponse struct {
	FailReason string `bencode:"failure reason"`
}

const (
	MSG_INVALID_REQ_TYPE        int = 100
	MSG_MISSING_INFO_HASH       int = 101
	MSG_MISSING_PEER_ID         int = 102
	MSG_MISSING_PORT            int = 103
	MSG_INVALID_PORT            int = 104
	MSG_INVALID_INFO_HASH       int = 150
	MSG_INVALID_PEER_ID         int = 151
	MSG_INVALID_NUM_WANT        int = 152
	MSG_OK                      int = 200
	MSG_INFO_HASH_NOT_FOUND     int = 480
	MSG_INVALID_AUTH            int = 490
	MSG_CLIENT_REQUEST_TOO_FAST int = 500
	MSG_GENERIC_ERROR           int = 900
	MSG_MALFORMED_REQUEST       int = 901
	MSG_QUERY_PARSE_FAIL        int = 902
)

var (
	// Error code to message mappings
	resp_msg = map[int]string{
		MSG_INVALID_REQ_TYPE:        "Invalid request type",
		MSG_MISSING_INFO_HASH:       "info_hash missing from request",
		MSG_MISSING_PEER_ID:         "peer_id missing from request",
		MSG_MISSING_PORT:            "port missing from request",
		MSG_INVALID_PORT:            "Invalid port",
		MSG_INVALID_AUTH:            "Invalid passkey supplied",
		MSG_INVALID_INFO_HASH:       "Torrent info hash must be 20 characters",
		MSG_INVALID_PEER_ID:         "Peer ID Invalid",
		MSG_INVALID_NUM_WANT:        "num_want invalid",
		MSG_INFO_HASH_NOT_FOUND:     "Unknown infohash",
		MSG_CLIENT_REQUEST_TOO_FAST: "Slow down there jimmy.",
		MSG_MALFORMED_REQUEST:       "Malformed request",
		MSG_GENERIC_ERROR:           "Generic Error",
		MSG_QUERY_PARSE_FAIL:        "Could not parse request",
	}
)

// oops will output a bencoded error code to the torrent client using
// a preset message code constant
func oops(ctx *gin.Context, msg_code int) {
	msg, exists := resp_msg[msg_code]
	if !exists {
		msg = resp_msg[MSG_GENERIC_ERROR]
	}
	ctx.String(msg_code, responseError(msg))

	log.Println("Error in request (", msg_code, "):", msg)
	log.Println("From:", ctx.Request.RequestURI)
}

//
//// oopsStr will output a bencoded error code to the torrent client using
//// a supplied custom message string
//func oopsStr(ctx *gin.Context, msg_code int, msg string) {
//	ctx.String(msg_code, responseError(msg))
//}

// responseError generates a bencoded error response for the torrent client to
// parse and display to the user
//
// Note that this function does not generate or support a warning reason, which are rarely if
// ever used.
func responseError(message string) string {
	var out_bytes bytes.Buffer
	bencoder := bencode.NewEncoder(&out_bytes)
	bencoder.Encode(bencode.Dict{
		"failure reason": message,
	})
	return out_bytes.String()
}
