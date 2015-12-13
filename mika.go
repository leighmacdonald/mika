// Package mika is a bittorrent tracker build using redis as a storage engine
package mika

import (
	"fmt"
	"github.com/leighmacdonald/mika/conf"
	log "github.com/Sirupsen/logrus"
	"github.com/getsentry/raven-go"
	"math/rand"
	"time"
)

var (
	// This is a special variable that is set by the go linker
	// If you do not build the project with make, or specify the linker settings
	// when building this will result in an empty version string
	Version = "master"

	// Timestamp of when the program first stared up
	StartTime int32

	SentryClient *raven.Client
)

// VersionStr returns the currently running version of the application.
// For this to function properly, the linker must set this value during
// build time. The makefile and build scripts will do this automatically.
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

func SetupSentry() {
	log.Info("Sentry configured for use @ ", conf.Config.SentryDSN)
	SentryClient, _ = raven.NewClient(conf.Config.SentryDSN, nil)
	if SentryClient == nil {
		log.Error("Could not initialize sentry")
	} else {
		SentryClient.SetRelease(VersionStr())
	}
}

func TestLog(args ...interface{}) {
	if conf.Config.Testing {
		log.Debug(args...)
	}
}

func init() {

	if Version == "" {
		log.Warn(`Build this binary with "make" or "./build.sh"", not "go build"`)
	}

	// Make sure we get random numbers in the application
	rand.Seed(time.Now().UTC().UnixNano())

	// Recorded to calculate app uptime
	StartTime = int32(time.Now().Unix())
}
