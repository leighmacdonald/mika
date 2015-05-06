package mika

import (
	"fmt"
	"github.com/kisielk/raven-go/raven"
)

var (
	Version   string
	StartTime int32

	RavenClient *raven.Client
)

func VersionStr() string {
	return fmt.Sprintf("mika/%s", Version)
}
