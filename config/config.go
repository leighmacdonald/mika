package config

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/mika/consts"
	"github.com/leighmacdonald/mika/util"
	"github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"net/url"
	"os"
	"strings"
	"time"
)

var (
	General generalConfig
	Tracker trackerConfig
	API     rpcConfig
	Store   StoreConfig
	GeoDB   geoDBConfig
)

type fullConfig struct {
	General generalConfig `mapstructure:"general"`
	Tracker trackerConfig `mapstructure:"tracker"`
	API     rpcConfig     `mapstructure:"api"`
	Store   StoreConfig   `mapstructure:"store"`
	GeoDB   geoDBConfig   `mapstructure:"geodb"`
}

type generalConfig struct {
	// RunMode defines the application run mode.
	// debug|release|testing
	RunMode string `mapstructure:"run_mode"`
	// LogLevel sets the logrus Logger level
	// info|warn|debug|trace
	LogLevel string `mapstructure:"log_level"`
	// LogColour toggles between colourised console output
	// true|false
	LogColour bool `mapstructure:"log_colour"`
}

type trackerConfig struct {
	// Public enables/disables auto registration of torrents and users
	// true|false
	Public bool `mapstructure:"public"`
	// Listen sets the host and port to listen on
	// hostname:port
	Listen string `mapstructure:"listen"`
	// TLS enables TLS for the tracker component
	// true|false
	TLS bool `mapstructure:"tls"`
	// IPv6 enables ipv6 peers
	// true|false
	IPv6 bool `mapstructure:"ipv6"`
	// IPv6Only disables ipv4 peers
	// true|false
	IPv6Only bool `mapstructure:"ipv6_only"`

	AutoRegister bool `mapstructure:"auto_register"`
	// ReaperInterval defines how often we do a sweep of active swarms looking for stale
	// peers that can be removed.
	// 60s|1m
	ReaperInterval       string `mapstructure:"reaper_interval"`
	ReaperIntervalParsed time.Duration
	// AnnounceInterval defines how often peers should announce. The lower this is
	// the more load on your system you can expect
	// 60s|1m
	AnnounceInterval       string `mapstructure:"announce_interval"`
	AnnounceIntervalParsed time.Duration
	// AnnounceIntervalMinimum is the minimum interval a client is allowed
	// 60s|1m
	AnnounceIntervalMinimum       string `mapstructure:"announce_interval_minimum"`
	AnnounceIntervalMinimumParsed time.Duration
	// TrackerHNRThreshold is how much time must pass before we mark a peer as Hit-N-Run
	// 1d|12h|60m
	HNRThreshold       string `mapstructure:"hnr_threshold"`
	HNRThresholdParsed time.Duration
	// TrackerBatchUpdateInterval defines how often we sync user stats to the back store
	BatchUpdateInterval       string `mapstructure:"batch_update_interval"`
	BatchUpdateIntervalParsed time.Duration
	// TrackerAllowNonRoutable defines whether we allow peers who are using non-public/routable addresses
	AllowNonRoutable bool `mapstructure:"allow_non_routable"`
	AllowClientIP    bool `mapstructure:"allow_client_ip"`

	MaxPeers int `mapstructure:"max_peers"`
}

type rpcConfig struct {
	// APIListen sets the host and port that the admin API should bind to
	// localhost:34001
	Listen string `mapstructure:"listen"`
	// APITLS enables TLS1.3 on the admin interface.
	// true|false
	TLS bool `mapstructure:"tls"`
	// APIKey Basic key authentication token for API calls
	Key string `mapstructure:"key"`
}

type StoreConfig struct {
	// Type sets the backing store type to be used
	// memory|redis|postgres|mysql|http
	Type string `mapstructure:"type"`
	// StoreTorrentHost is the host to connect to
	// localhost
	Host string `mapstructure:"host"`
	// StoreTorrentPort is the port to connect to
	// 3306|6379|443
	Port int `mapstructure:"port"`
	// User user to connect with
	// mika
	User string `mapstructure:"user"`
	// Password password to connect with
	// mika
	Password string `mapstructure:"password"`
	// Database is the database / schema name to open on the backing store
	// Redis uses numeric values 0-16 by default
	// mika|0
	Database string `mapstructure:"database"`
	// Properties will append a string of query args to the DSN
	// Format: arg1=foo&arg2=bar
	Properties string `mapstructure:"properties"`
}

type geoDBConfig struct {
	// GeodbPath sets the path to use for downloading and loading the geo database. Relative to the binary's path.
	// ./path/to/file.mmdb
	Path string `mapstructure:"path"`
	// GeodbAPIKey is the MaxMind.com API key used to download the database
	// XXXXXXXXXXXXXXXX
	APIKey string `mapstructure:"api_key"`
	// GeodbEnabled toggles use of the geo database
	// true|false
	Enabled bool `mapstructure:"enabled"`
}

// DSN constructs a URI for database connection strings
//
// protocol//[user]:[password]@tcp([host]:[port])[/database][?properties]
func (c StoreConfig) DSN() string {
	props := c.Properties
	if props != "" && !strings.HasPrefix(props, "?") {
		props = "?" + props
	}
	s := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s%s",
		c.User, c.Password, c.Host, c.Port, c.Database, props)
	u, err := url.Parse(s)
	if err != nil {
		log.Fatalf("Failed to construct valid database DSN: %s", err.Error())
		return ""
	}
	return u.String()
}

// Read reads in config file and ENV variables if set.
func Read(cfgFile string) error {
	// Find home directory.
	home, _ := homedir.Dir()
	viper.AddConfigPath(home)
	viper.AddConfigPath(".")
	viper.AddConfigPath("../")
	viper.AddConfigPath("../../")
	viper.SetConfigName("mika")
	if os.Getenv("MIKA_CONFIG") != "" {
		viper.SetConfigFile(os.Getenv("MIKA_CONFIG"))
	} else if cfgFile != "" {
		viper.SetConfigName(cfgFile)
	}
	setDuration := func(target *time.Duration, value string) error {
		d, err := util.ParseDuration(value)
		if err != nil {
			return err
		}
		*target = d
		return nil
	}
	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err != nil {
		return errors.Wrap(err, consts.ErrInvalidConfig.Error())
	}
	log.Debugf("Using config file: %s", viper.ConfigFileUsed())
	full := fullConfig{}
	if err := viper.Unmarshal(&full); err != nil {
		return errors.Wrapf(err, "Failed to parse config")
	}
	durations := []struct {
		target *time.Duration
		value  string
	}{
		{&full.Tracker.AnnounceIntervalMinimumParsed, full.Tracker.AnnounceIntervalMinimum},
		{&full.Tracker.AnnounceIntervalParsed, full.Tracker.AnnounceInterval},
		{&full.Tracker.BatchUpdateIntervalParsed, full.Tracker.BatchUpdateInterval},
		{&full.Tracker.HNRThresholdParsed, full.Tracker.HNRThreshold},
		{&full.Tracker.ReaperIntervalParsed, full.Tracker.ReaperInterval},
	}
	for _, dur := range durations {
		if err := setDuration(dur.target, dur.value); err != nil {
			return errors.Wrapf(err, "Failed to parse time duration")
		}
	}
	if full.API.Key == "" {
		return errors.New("api.key cannot be empty")
	}
	General = full.General
	Tracker = full.Tracker
	API = full.API
	GeoDB = full.GeoDB
	Store = full.Store

	setupLogger(General.LogLevel, General.LogColour)
	gin.SetMode(General.RunMode)
	return nil
}

func setupLogger(levelStr string, colour bool) {
	log.SetFormatter(&log.TextFormatter{
		ForceColors:      colour,
		DisableTimestamp: true,
	})
	log.SetOutput(os.Stdout)
	level, err := log.ParseLevel(levelStr)
	if err != nil {
		log.Panicln("Invalid log level defined")
	}
	log.SetLevel(level)
}

func Save() error {
	return viper.WriteConfig()
}
