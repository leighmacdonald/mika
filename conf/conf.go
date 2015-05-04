package conf

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"sync"
)

var (
	// Global config instance & lock
	Config     *Configuration
	configLock = new(sync.RWMutex)
)

type Configuration struct {
	Debug          bool
	ListenHost     string
	ListenHostAPI  string
	APIUsername    string
	APIPassword    string
	RedisHost      string
	RedisPass      string
	RedisMaxIdle   int
	SSLPrivateKey  string
	SSLCert        string
	AnnInterval    int
	AnnIntervalMin int
	ReapInterval   int
	IndexInterval  int
	HNRThreshold   int32
	SentryDSN      string
	InfluxDSN      string
	InfluxDB       string
	InfluxUser     string
	InfluxPass     string
}

func LoadConfig(config_file string, fail bool) {
	log.Println("Loading config:", config_file)
	file, err := ioutil.ReadFile(config_file)
	if err != nil {
		log.Println("loadConfig: Failed to open config file:", err)
		if fail {
			os.Exit(1)
		}
	}

	temp := new(Configuration)
	if err = json.Unmarshal(file, temp); err != nil {
		log.Println("loadConfig: Failed to parse config: ", err)
		if fail {
			os.Exit(1)
		}
	}
	configLock.Lock()
	Config = temp
	configLock.Unlock()
}
