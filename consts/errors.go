package consts

import "github.com/pkg/errors"

var ErrMalformedRequest error
var ErrInvalidMapKey error
var ErrDuplicate error
var ErrInvalidInfoHash error
var ErrInvalidIP error

func init() {
	ErrMalformedRequest = errors.New("Malformed request")
	ErrInvalidMapKey = errors.New("Invalid map key specified")
	ErrDuplicate = errors.New("Duplicate entry")
	ErrInvalidInfoHash = errors.New("Info hash not supplied")
	ErrInvalidIP = errors.New("invalid client ip")
}
