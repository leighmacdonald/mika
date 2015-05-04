package mika

import (
	"github.com/kisielk/raven-go/raven"
)

var (
	Version string
	StartTime int32

	RavenClient *raven.Client
)
