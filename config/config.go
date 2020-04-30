package config

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/mitchellh/go-homedir"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"net/url"
	"os"
)

// StoreType is a mapping to the backing store types used
type StoreType int

const (
	Torrent StoreType = iota
	Peers
	Users
)

const (
	GeneralRunMode   = "general_run_mode"
	GeneralLogLevel  = "general_log_level"
	GeneralLogColour = "general_log_colour"

	TrackerPublic              = "tracker_public"
	TrackerListen              = "tracker_listen"
	TrackerTLS                 = "tracker_tls"
	TrackerIPv6                = "tracker_ipv6"
	TrackerIPv6Only            = "tracker_ipv6_only"
	TrackerAnnounceInterval    = "tracker_announce_interval"
	TrackerAnnounceIntervalMin = "tracker_announce_interval_minimum"
	TrackerReapInterval        = "tracker_reap_internal"
	TrackerHNRThreshold        = "tracker_hnr_threshold"
	TrackerIndexInterval       = "tracker_index_interval"

	APIListen   = "api_listen"
	APITLS      = "api_tls"
	APIIPv6     = "api_ipv6"
	APIIPv6Only = "api_ipv6_only"

	StoreTorrentType       = "store_torrent_type"
	StoreTorrentHost       = "store_torrent_host"
	StoreTorrentPort       = "store_torrent_port"
	StoreTorrentDatabase   = "store_torrent_database"
	StoreTorrentUser       = "store_torrent_user"
	StoreTorrentPassword   = "store_torrent_password"
	StoreTorrentProperties = "store_torrent_properties"

	StoreUsersType       = "store_users_type"
	StoreUsersHost       = "store_users_host"
	StoreUsersPort       = "store_users_port"
	StoreUsersDatabase   = "store_users_database"
	StoreUsersUser       = "store_users_user"
	StoreUsersPassword   = "store_users_password"
	StoreUsersProperties = "store_users_properties"

	StorePeersType       = "store_peers_type"
	StorePeersHost       = "store_peers_host"
	StorePeersPort       = "store_peers_port"
	StorePeersDatabase   = "store_peers_database"
	StorePeersUser       = "store_peers_user"
	StorePeersPassword   = "store_peers_password"
	StorePeersProperties = "store_peers_properties"

	GeodbPath    = "geodb_path"
	GeodbApiKey  = "geodb_api_key"
	GeodbEnabled = "geodb_enabled"
)

// StoreConfig provides a common config struct for backing stores
type StoreConfig struct {
	Type       string
	Host       string
	Port       int
	Username   string
	Password   string
	Database   string
	Properties string
}

// DSN constructs a URI for database connection strings
//
// protocol//[user]:[password]@tcp([host]:[port])[/database][?properties]
func (c StoreConfig) DSN() string {
	props := c.Properties
	if props != "" {
		props = "?" + props
	}
	s := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s%s",
		c.Username, c.Password, c.Host, c.Port, c.Database, props)
	u, err := url.Parse(s)
	if err != nil {
		log.Fatalf("Failed to construct valid database DSN: %s", err.Error())
		return ""
	}
	return u.String()
}

// GetStoreConfig returns the config options for the store type provided
func GetStoreConfig(storeType StoreType) *StoreConfig {
	switch storeType {
	case Users:
		return &StoreConfig{
			Type:       viper.GetString(StoreUsersType),
			Host:       viper.GetString(StoreUsersHost),
			Port:       viper.GetInt(StoreUsersPort),
			Username:   viper.GetString(StoreUsersUser),
			Password:   viper.GetString(StoreUsersPassword),
			Database:   viper.GetString(StoreUsersDatabase),
			Properties: viper.GetString(StoreUsersProperties),
		}
	case Torrent:
		return &StoreConfig{
			Type:       viper.GetString(StoreTorrentType),
			Host:       viper.GetString(StoreTorrentHost),
			Port:       viper.GetInt(StoreTorrentPort),
			Username:   viper.GetString(StoreTorrentUser),
			Password:   viper.GetString(StoreTorrentPassword),
			Database:   viper.GetString(StoreTorrentDatabase),
			Properties: viper.GetString(StoreTorrentProperties),
		}
	case Peers:
		return &StoreConfig{
			Type:       viper.GetString(StorePeersType),
			Host:       viper.GetString(StorePeersHost),
			Port:       viper.GetInt(StorePeersPort),
			Username:   viper.GetString(StorePeersUser),
			Password:   viper.GetString(StorePeersPassword),
			Database:   viper.GetString(StorePeersDatabase),
			Properties: viper.GetString(StorePeersProperties),
		}
	}
	return nil
}

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

		gin.SetMode(viper.GetString(GeneralRunMode))
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
