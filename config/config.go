package config

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/mika/consts"
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

// Key represents a known configuration key
type Key string

const (
	// GeneralRunMode defines the application run mode.
	// debug|release|testing
	GeneralRunMode Key = "general_run_mode"

	// GeneralLogLevel sets the logrus Logger level
	// info|warn|debug|trace
	GeneralLogLevel Key = "general_log_level"

	// GeneralLogColour toggles between colourised console output
	// true|false
	GeneralLogColour Key = "general_log_colour"

	// TrackerPublic enables/disables auto registration of torrents and users
	// true|false
	TrackerPublic Key = "tracker_public"
	// TrackerListen sets the host and port to listen on
	// hostname:port
	TrackerListen Key = "tracker_listen"
	// TrackerTLS enables TLS for the tracker component
	// true|false
	TrackerTLS Key = "tracker_tls"
	// TrackerIPv6 enables ipv6 peers
	// true|false
	TrackerIPv6 Key = "tracker_ipv6"
	// TrackerIPv6Only disables ipv4 peers
	// true|false
	TrackerIPv6Only Key = "tracker_ipv6_only"
	// TrackerReaperInterval defines how often we do a sweep of active swarms looking for stale
	// peers that can be removed.
	// 60s|1m
	TrackerReaperInterval Key = "tracker_reaper_interval"
	// TrackerAnnounceInterval defines how often peers should announce. The lower this is
	// the more load on your system you can expect
	// 60s|1m
	TrackerAnnounceInterval Key = "tracker_announce_interval"
	// TrackerAnnounceIntervalMin is the minimum interval a client is allowed
	// 60s|1m
	TrackerAnnounceIntervalMin Key = "tracker_announce_interval_minimum"
	// TrackerHNRThreshold is how much time must pass before we mark a peer as Hit-N-Run
	// 1d|12h|60m
	TrackerHNRThreshold Key = "tracker_hnr_threshold"
	// TrackerBatchUpdateInterval defines how often we sync user stats to the back store
	TrackerBatchUpdateInterval Key = "tracker_batch_update_interval"

	// APIListen sets the host and port that the admin API should bind to
	// localhost:34001
	APIListen Key = "api_listen"
	// APITLS enables TLS1.3 on the admin interface.
	// true|false
	APITLS Key = "api_tls"
	// APIIPv6 enables ipv6 for the admin API
	// true|false
	APIIPv6 Key = "api_ipv6"
	// APIIPv6Only disabled ipv4 to the admin interface
	// true|false
	APIIPv6Only Key = "api_ipv6_only"

	// StoreTorrentType sets the backing store type to be used for torrents
	// memory|redis|postgres|mysql|http
	StoreTorrentType Key = "store_torrent_type"
	// StoreTorrentHost is the host to connect to
	// localhost
	StoreTorrentHost Key = "store_torrent_host"
	// StoreTorrentPort is the port to connect to
	// 3306|6379|443
	StoreTorrentPort Key = "store_torrent_port"
	// StoreTorrentDatabase is the database / schema name to open on the backing store
	// Redis uses numeric values 0-16 by default
	// mika|0
	StoreTorrentDatabase Key = "store_torrent_database"
	// StoreTorrentUser user to connect with
	// mika
	StoreTorrentUser Key = "store_torrent_user"
	// StoreTorrentPassword password to connect with
	// mika
	StoreTorrentPassword Key = "store_torrent_password"
	// StoreTorrentProperties sets additional properties passed to the backing store configuration
	StoreTorrentProperties Key = "store_torrent_properties"

	// StoreUsersType sets the backing store type to be used for users
	// memory|redis|postgres|mysql|http
	StoreUsersType Key = "store_users_type"
	// StoreUsersHost is the host to connect to
	// localhost
	StoreUsersHost Key = "store_users_host"
	// StoreUsersPort is the port to connect to
	// 3306|6379|443
	StoreUsersPort Key = "store_users_port"
	// StoreUsersDatabase is the database / schema name to open on the backing store
	// Redis uses numeric values 0-16 by default
	// mika|0
	StoreUsersDatabase Key = "store_users_database"
	// StoreUsersUser user to connect with
	// mika
	StoreUsersUser Key = "store_users_user"
	// StoreUsersPassword password to connect with
	// mika
	StoreUsersPassword Key = "store_users_password"
	// StoreUsersProperties sets additional properties passed to the backing store configuration
	StoreUsersProperties Key = "store_users_properties"

	// StorePeersType sets the backing store type to be used for peers
	// memory|redis|postgres|mysql|http
	StorePeersType Key = "store_peers_type"
	// StorePeersHost is the host to connect to
	// localhost
	StorePeersHost Key = "store_peers_host"
	// StorePeersPort is the port to connect to
	// 3306|6379|443
	StorePeersPort Key = "store_peers_port"
	// StorePeersDatabase is the database / schema name to open on the backing store
	// Redis uses numeric values 0-16 by default
	// mika|0
	StorePeersDatabase Key = "store_peers_database"
	// StorePeersUser user to connect with
	// mika
	StorePeersUser Key = "store_peers_user"
	// StorePeersPassword password to connect with
	// mika
	StorePeersPassword Key = "store_peers_password"
	// StorePeersProperties sets additional store specific properties passed to the backing store configuration
	StorePeersProperties Key = "store_peers_properties"

	// GeodbPath sets the path to use for downloading and loading the geo database. Relative to the binary's path.
	// ./path/to/file.mmdb
	GeodbPath Key = "geodb_path"
	// GeodbAPIKey is the MaxMind.com API key used to download the database
	// XXXXXXXXXXXXXXXX
	GeodbAPIKey Key = "geodb_api_key"
	// GeodbEnabled toggles use of the geo database
	// true|false
	GeodbEnabled Key = "geodb_enabled"
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
			Type:       viper.GetString(string(StoreUsersType)),
			Host:       viper.GetString(string(StoreUsersHost)),
			Port:       viper.GetInt(string(StoreUsersPort)),
			Username:   viper.GetString(string(StoreUsersUser)),
			Password:   viper.GetString(string(StoreUsersPassword)),
			Database:   viper.GetString(string(StoreUsersDatabase)),
			Properties: viper.GetString(string(StoreUsersProperties)),
		}
	case Torrent:
		return &StoreConfig{
			Type:       viper.GetString(string(StoreTorrentType)),
			Host:       viper.GetString(string(StoreTorrentHost)),
			Port:       viper.GetInt(string(StoreTorrentPort)),
			Username:   viper.GetString(string(StoreTorrentUser)),
			Password:   viper.GetString(string(StoreTorrentPassword)),
			Database:   viper.GetString(string(StoreTorrentDatabase)),
			Properties: viper.GetString(string(StoreTorrentProperties)),
		}
	case Peers:
		return &StoreConfig{
			Type:       viper.GetString(string(StorePeersType)),
			Host:       viper.GetString(string(StorePeersHost)),
			Port:       viper.GetInt(string(StorePeersPort)),
			Username:   viper.GetString(string(StorePeersUser)),
			Password:   viper.GetString(string(StorePeersPassword)),
			Database:   viper.GetString(string(StorePeersDatabase)),
			Properties: viper.GetString(string(StorePeersProperties)),
		}
	}
	return nil
}

// Read reads in config file and ENV variables if set.
func Read(cfgFile string) error {
	// Find home directory.
	home, err := homedir.Dir()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	viper.AddConfigPath(home)
	viper.AddConfigPath(".")
	viper.AddConfigPath("../")
	viper.AddConfigPath("../../")
	if os.Getenv("MIKA_CONFIG") != "" {
		viper.SetConfigFile(os.Getenv("MIKA_CONFIG"))
	} else if cfgFile != "" {
		viper.SetConfigName(cfgFile)
	} else {
		viper.SetConfigName("mika")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		log.Debugf("Using config file: %s", viper.ConfigFileUsed())
		level := viper.GetString(string(GeneralLogLevel))
		colour := viper.GetBool(string(GeneralLogColour))
		setupLogger(level, colour)
		gin.SetMode(viper.GetString(string(GeneralRunMode)))
		return nil
	}
	return consts.ErrInvalidConfig

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
