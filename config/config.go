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
	// Torrent maps to store_torrent_* config options
	Torrent StoreType = iota
	// Peers maps to store_peers_* config options
	Peers
	// Users maps to store_users_* config options
	Users
)

const (
	// GeneralRunMode defines the application run mode.
	// debug|release
	GeneralRunMode = "general_run_mode"

	// GeneralLogLevel sets the logrus Logger level
	// info|warn|debug|trace
	GeneralLogLevel = "general_log_level"

	// GeneralLogColour toggles between colourised console output
	// true|false
	GeneralLogColour = "general_log_colour"

	// TrackerPublic enables/disables auto registration of torrents and users
	// true|false
	TrackerPublic = "tracker_public"
	// TrackerListen sets the host and port to listen on
	// hostname:port
	TrackerListen = "tracker_listen"
	// TrackerTLS enables TLS for the tracker component
	// true|false
	TrackerTLS = "tracker_tls"
	// TrackerIPv6 enables ipv6 peers
	// true|false
	TrackerIPv6 = "tracker_ipv6"
	// TrackerIPv6Only disables ipv4 peers
	// true|false
	TrackerIPv6Only = "tracker_ipv6_only"
	// TrackerAnnounceInterval defines how often peers should announce. The lower this is
	// the more load on your system you can expect
	// 60s|1m
	TrackerAnnounceInterval = "tracker_announce_interval"
	// TrackerAnnounceIntervalMin is the minimum interval a client is allowed
	// 60s|1m
	TrackerAnnounceIntervalMin = "tracker_announce_interval_minimum"
	// TrackerReapInterval defines how often we do a sweep of active swarms looking for stale
	// peers that can be removed.
	// 60s|1m
	TrackerReapInterval = "tracker_reap_internal"
	// TrackerHNRThreshold is how much time must pass before we mark a peer as Hit-N-Run
	// 1d|12h|60m
	TrackerHNRThreshold = "tracker_hnr_threshold"
	// TrackerIndexInterval is the amount of time between updating the torrent stats
	// 60s|1m
	TrackerIndexInterval = "tracker_index_interval"

	// APIListen sets the host and port that the admin API should bind to
	// localhost:34001
	APIListen = "api_listen"
	// APITLS enables TLS1.3 on the admin interface.
	// true|false
	APITLS = "api_tls"
	// APIIPv6 enables ipv6 for the admin API
	// true|false
	APIIPv6 = "api_ipv6"
	// APIIPv6Only disabled ipv4 to the admin interface
	// true|false
	APIIPv6Only = "api_ipv6_only"

	// StoreTorrentType sets the backing store type to be used for torrents
	// memory|redis|postgres|mysql|http
	StoreTorrentType = "store_torrent_type"
	// StoreTorrentHost is the host to connect to
	// localhost
	StoreTorrentHost = "store_torrent_host"
	// StoreTorrentPort is the port to connect to
	// 3306|6379|443
	StoreTorrentPort = "store_torrent_port"
	// StoreTorrentDatabase is the database / schema name to open on the backing store
	// Redis uses numeric values 0-16 by default
	// mika|0
	StoreTorrentDatabase = "store_torrent_database"
	// StoreTorrentUser user to connect with
	// mika
	StoreTorrentUser = "store_torrent_user"
	// StoreTorrentPassword password to connect with
	// mika
	StoreTorrentPassword = "store_torrent_password"
	// Additional properties passed to the backing store configuration
	StoreTorrentProperties = "store_torrent_properties"

	// StoreUsersType sets the backing store type to be used for users
	// memory|redis|postgres|mysql|http
	StoreUsersType = "store_users_type"
	// StoreUsersHost is the host to connect to
	// localhost
	StoreUsersHost = "store_users_host"
	// StoreUsersPort is the port to connect to
	// 3306|6379|443
	StoreUsersPort = "store_users_port"
	// StoreUsersDatabase is the database / schema name to open on the backing store
	// Redis uses numeric values 0-16 by default
	// mika|0
	StoreUsersDatabase = "store_users_database"
	// StoreUsersUser user to connect with
	// mika
	StoreUsersUser = "store_users_user"
	// StoreUsersPassword password to connect with
	// mika
	StoreUsersPassword = "store_users_password"
	// StoreUsersProperties sets additional properties passed to the backing store configuration
	StoreUsersProperties = "store_users_properties"

	// StorePeersType sets the backing store type to be used for peers
	// memory|redis|postgres|mysql|http
	StorePeersType = "store_peers_type"
	// StorePeersHost is the host to connect to
	// localhost
	StorePeersHost = "store_peers_host"
	// StorePeersPort is the port to connect to
	// 3306|6379|443
	StorePeersPort = "store_peers_port"
	// StorePeersDatabase is the database / schema name to open on the backing store
	// Redis uses numeric values 0-16 by default
	// mika|0
	StorePeersDatabase = "store_peers_database"
	// StorePeersUser user to connect with
	// mika
	StorePeersUser = "store_peers_user"
	// StorePeersPassword password to connect with
	// mika
	StorePeersPassword = "store_peers_password"
	// StorePeersProperties sets additional store specific properties passed to the backing store configuration
	StorePeersProperties = "store_peers_properties"

	// GeodbPath sets the path to use for downloading and loading the geo database. Relative to the binary's path.
	// ./path/to/file.mmdb
	GeodbPath = "geodb_path"
	// GeodbAPIKey is the MaxMind.com API key used to download the database
	// XXXXXXXXXXXXXXXX
	GeodbAPIKey = "geodb_api_key"
	// GeodbEnabled toggles use of the geo database
	// true|false
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
		setupLogger(level, colour)

		gin.SetMode(viper.GetString(GeneralRunMode))
	}
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
