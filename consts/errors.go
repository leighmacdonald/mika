package consts

import "github.com/pkg/errors"

var (
	// ErrMalformedRequest is the general request error for invalid inputs that dont fall under other categories
	ErrMalformedRequest = errors.New("Malformed request")
	// ErrInvalidMapKey is for general map key lookup failure
	ErrInvalidMapKey = errors.New("Invalid map key specified")
	// ErrDuplicate duplicate entry error
	ErrDuplicate = errors.New("Duplicate entry")
	// ErrInvalidInfoHash returned to clients on unknown hash
	ErrInvalidInfoHash = errors.New("Info hash not supplied")
	// ErrInvalidTorrentID failure to find mapped torrent_id
	ErrInvalidTorrentID = errors.New("Invalid torrent_id")
	// ErrInvalidPeerID failure to find a peer by peer_id
	ErrInvalidPeerID = errors.New("Invalid peer_id")
	// ErrInvalidDriver is for when a unknown driver is used.
	// Either misspelled or using driver that wasn't built into the binary
	ErrInvalidDriver = errors.New("invalid driver")
	// ErrInvalidConfig is issued when a invalid config value is used
	ErrInvalidConfig = errors.New("invalid configuration")
	// ErrInvalidResponseCode is a generic error code representing a invalid response code
	// was received from the server
	ErrInvalidResponseCode = errors.New("invalid response code")
	// ErrUnauthorized is a general non-info disclosing auth error
	ErrUnauthorized = errors.New("not authorized")
	// ErrInvalidState is used when the state of the data returned is not what we expect or invalid
	// in any way.
	ErrInvalidState = errors.New("invalid struct state")
	// ErrInvalidUser is used when a user lookup fails
	ErrInvalidUser = errors.New("invalid user")

	// ErrInvalidClient is used when an invalid client is requested/used
	ErrInvalidClient = errors.New("invalid torrent client")
)
