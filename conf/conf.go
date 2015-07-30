// Package conf provides handling of loading and reading of JSON based config files
package conf

import (
	"encoding/json"
	log "github.com/Sirupsen/logrus"
	"io/ioutil"
	"os"
	"sync"
)

var (
	// Global config instance & lock
	Config     *Configuration
	configLock = new(sync.RWMutex)
)

type Configuration struct {
	// Enabled debug functions
	Debug bool

	// Enable testing mode which bypasses some functionality
	// Do not set this to true in production ever!
	Testing bool

	// A loglevel to use "info", "error", "warn", "debug", "fatal", "panic"
	LogLevel string

	// URI for the tracker listen host :34000
	ListenHost string

	// URI for the api listen host :34001
	ListenHostAPI string

	// Enable IPv6 stack.
	IPV6Enabled bool

	// Force only using ipv6
	IPV6Only bool

	// Username required to use api, empty string for none
	APIUsername string

	// Password required to use api, empty string for none
	APIPassword string

	// Redis hostname
	RedisHost string

	// Redis password, empty string for none
	RedisPass string

	// Maximum amount of idle redis connection to allow to idle
	RedisMaxIdle int

	// Redis database number to use
	RedisDB int

	// Path to the SSL private key
	SSLPrivateKey string

	// Path to the SSL CA cert
	SSLCert string

	// Announce interval sent to clients
	AnnInterval int

	// Minimum announce interval sent to clients
	AnnIntervalMin int

	// How long to wait until reaping a pear after an announce
	ReapInterval int

	// How often to index the torrent seeder/leecher counts
	IndexInterval int

	// How much seeding time is required to remove hnr
	HNRThreshold int32

	// Minimum amount of bytes need to allow HNR to occur
	HNRMinBytes uint64

	// Full DSN for Sentry
	SentryDSN string

	// Full DSN for graphite metric reporting eg: "localhost:2003"
	MetricsDSN string

	GeoEnabled bool

	GeoDBPath string

	// Wait time before writing the data to graphite
	//	MetricsWriteTimer timemake make sed.Duration

	// Use colours log output
	ColourLogs bool
}

// LoadConfig reads in a json based config file from the path provided and updated
// the currently active application configuration
func LoadConfig(config_file string, fail bool) {
	log.WithFields(log.Fields{
		"config_file": config_file,
	}).Info("Loading config")

	file, err := ioutil.ReadFile(config_file)
	if err != nil {
		log.WithFields(log.Fields{
			"fn":          "LoadConfig",
			"err":         err.Error(),
			"config_file": config_file,
		}).Fatal("Failed to open config file")
		if fail {
			os.Exit(1)
		}
	}

	temp := new(Configuration)
	if err = json.Unmarshal(file, temp); err != nil {
		log.WithFields(log.Fields{
			"fn":          "LoadConfig",
			"err":         err.Error(),
			"config_file": config_file,
		}).Error("Failed to parse config file, cannot continue")
		if fail {
			os.Exit(1)
		}
	}
	configLock.Lock()
	Config = temp
	configLock.Unlock()

	if Config.ReapInterval <= Config.AnnIntervalMin {
		log.Warn("ReapInterval less than AnnInterval (here be dragons!)")
		log.Warn("This is almost certainly not what you want, fix required.")
	}
}
