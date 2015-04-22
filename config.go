package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
)

type Config struct {
	Debug          bool
	ListenHost     string
	RedisHost      string
	RedisPass      string
	RedisMaxIdle   int
	AnnInterval    int
	AnnIntervalMin int
	ReapInterval   int
	IndexInterval  int
	HNRThreshold   uint32
	SentryDSN      string
	InfluxDSN      string
	InfluxDB       string
	InfluxUser     string
	InfluxPass     string
}

func loadConfig(fail bool) {
	file, err := ioutil.ReadFile(*config_file)
	if err != nil {
		log.Println("open config: ", err)
		if fail {
			os.Exit(1)
		}
	}

	temp := new(Config)
	if err = json.Unmarshal(file, temp); err != nil {
		log.Println("! Parse config error: ", err)
		if fail {
			os.Exit(1)
		}
	}
	configLock.Lock()
	config = temp
	configLock.Unlock()
}
