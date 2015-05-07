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

// SetupLogger will configure logrus to use our config
// force_colour will enable colour codes to be used even if there is no TTY detected
func SetupLogger(log_level string, force_colour bool) {
	log.SetFormatter(&log.TextFormatter{
		ForceColors:    force_colour,
		DisableSorting: true,
	})
	switch log_level {
	case "fatal":
		log.SetLevel(log.FatalLevel)
	case "panic":
		log.SetLevel(log.PanicLevel)
	case "debug":
		log.SetLevel(log.DebugLevel)
	case "error":
		log.SetLevel(log.ErrorLevel)
	case "warn":
		log.SetLevel(log.WarnLevel)
	default:
		log.SetLevel(log.InfoLevel)
	}
}
