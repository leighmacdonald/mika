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

	// A loglevel to use "info", "error", "warn", "debug", "fatal", "panic"
	LogLevel string

	// URI for the tracker listen host :34000
	ListenHost string

	// URI for the api listen host :34001
	ListenHostAPI string

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

	// Full DSN for InfluxDB metric reporting
	InfluxDSN string

	// Influx database to write points to
	InfluxDB string

	// Influx user
	InfluxUser string

	// Influx password
	InfluxPass string

	// Number of points to buffer before writing the data to influxdb
	InfluxWriteBuffer int

	// Use colours log output
	ColourLogs bool
}

// LoadConfig reads in a json based config file from the path provided and updated
// the currently active application configuration
func LoadConfig(config_file string, fail bool) {
	log.Info("Loading config:", config_file)
	file, err := ioutil.ReadFile(config_file)
	if err != nil {
		log.Error("loadConfig: Failed to open config file:", err)
		if fail {
			os.Exit(1)
		}
	}

	temp := new(Configuration)
	if err = json.Unmarshal(file, temp); err != nil {
		log.Error("loadConfig: Failed to parse config: ", err)
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
