package consts

import "github.com/pkg/errors"

var (
	ErrMalformedRequest = errors.New("Malformed request")
	ErrInvalidMapKey    = errors.New("Invalid map key specified")
	ErrDuplicate        = errors.New("Duplicate entry")
	ErrInvalidInfoHash  = errors.New("Info hash not supplied")
	ErrInvalidIP        = errors.New("invalid client ip")
	ErrInvalidTorrentID = errors.New("Invalid torrent_id")
	ErrInvalidDriver    = errors.New("invalid driver")
	ErrInvalidConfig    = errors.New("invalid configuration")
)
