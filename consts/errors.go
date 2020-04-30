package consts

import "github.com/pkg/errors"

var (
	// General request error for invalid inputs that dont fall under other categories
	ErrMalformedRequest    = errors.New("Malformed request")
	ErrInvalidMapKey       = errors.New("Invalid map key specified")
	ErrDuplicate           = errors.New("Duplicate entry")
	ErrInvalidInfoHash     = errors.New("Info hash not supplied")
	ErrInvalidIP           = errors.New("invalid client ip")
	ErrInvalidTorrentID    = errors.New("Invalid torrent_id")
	ErrInvalidDriver       = errors.New("invalid driver")
	ErrInvalidConfig       = errors.New("invalid configuration")
	ErrInvalidResponseCode = errors.New("invalid response code")
)
