package mika

import (
	"fmt"
	log "github.com/Sirupsen/logrus"
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

func SetupLogger(log_level log.Level) {
	log.SetFormatter(&log.TextFormatter{})
	log.SetLevel(log_level)
}
