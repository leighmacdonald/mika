package config

import (
	"fmt"
	"github.com/mitchellh/go-homedir"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"os"
)

const (
	GeneralLogLevel            = "general_log_level"
	GeneralLogColour           = "general_log_colour"
	TrackerPublic              = "tracker_public"
	TrackerListen              = "tracker_listen"
	TrackerIPv6                = "tracker_ipv6"
	TrackerIPv6Only            = "tracker_ipv6_only"
	TrackerAnnounceInterval    = "tracker_announce_interval"
	TrackerAnnounceIntervalMin = "tracker_announce_interval_minimum"
	TrackerReapInterval        = "tracker_reap_internal"
	TrackerHNRThreshold        = "tracker_hnr_threshold"
	TrackerIndexInterval       = "tracker_index_interval"
	StoreType                  = "store_type"
	StoreHost                  = "store_host"
	StorePort                  = "store_port"
	StoreName                  = "store_name"
	StoreUser                  = "store_user"
	StorePassword              = "store_password"
	StoreProperties            = "store_properties"
	CacheType                  = "cache_type"
	CacheHost                  = "cache_host"
	CachePort                  = "cache_port"
	CachePassword              = "cache_password"
	CacheMaxIdle               = "cache_max_idle"
	CacheDB                    = "cache_db"
	GeodbPath                  = "geodb_path"
	GeodbApiKey                = "geodb_api_key"
	GeodbEnabled               = "geodb_enabled"
)

// Read reads in config file and ENV variables if set.
func Read(cfgFile string) {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else if os.Getenv("MIKA_CONFIG") != "" {
		viper.SetConfigFile(os.Getenv("MIKA_CONFIG"))
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		// Search config in home directory with name ".mika" (without extension).
		viper.AddConfigPath(home)
		viper.AddConfigPath(".")
		viper.AddConfigPath("../")
		viper.SetConfigName("mika")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		log.Debugf("Using config file: %s", viper.ConfigFileUsed())
		level := viper.GetString(GeneralLogLevel)
		colour := viper.GetBool(GeneralLogColour)
		SetupLogger(level, colour)
	}
}

func SetupLogger(levelStr string, colour bool) {
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
