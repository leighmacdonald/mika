// Copyright 2015 toor@titansof.tv
//
// A (currently) stateless torrent tracker using Redis as a backing store
//
// Performance tuning options for linux kernel
//
// Set in sysctl.conf
// fs.file-max=100000
// vm.overcommit_memory = 1
//
// echo 1 > /proc/sys/net/ipv4/tcp_tw_reuse
// echo 1 > /proc/sys/net/ipv4/tcp_tw_recycle
// echo 10000 > /proc/sys/net/core/somaxconn
//
// redis.conf
// maxmemory-policy noeviction
// notify-keyspace-events "KEx"

package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"github.com/chihaya/bencode"
	"github.com/garyburd/redigo/redis"
	"github.com/kisielk/raven-go/raven"
	"github.com/labstack/echo"
	"log"
	"os"
	"os/signal"
	"runtime"
	"runtime/pprof"
	"strings"
	"sync"
	"syscall"
	"time"
)

type ErrorResponse struct {
	FailReason string `bencode:"failure reason"`
}

type DBEntity interface {
	Sync(r redis.Conn) bool
}

type Queued struct {
	InQueue bool
}

const (
	MSG_OK                      int = 0
	MSG_INVALID_REQ_TYPE        int = 100
	MSG_MISSING_INFO_HASH       int = 101
	MSG_MISSING_PEER_ID         int = 102
	MSG_MISSING_PORT            int = 103
	MSG_INVALID_PORT            int = 104
	MSG_INVALID_INFO_HASH       int = 150
	MSG_INVALID_PEER_ID         int = 151
	MSG_INVALID_NUM_WANT        int = 152
	MSG_INFO_HASH_NOT_FOUND     int = 200
	MSG_CLIENT_REQUEST_TOO_FAST int = 500
	MSG_MALFORMED_REQUEST       int = 901
	MSG_GENERIC_ERROR           int = 900
)

var (
	cheese = `
                               ____________
                            __/ ///////// /|
                           /              ¯/|
                          /_______________/ |
    ________________      |  __________  |  |
   /               /|     | |          | |  |
  /               / |     | | > Mika   | |  |
 /_______________/  |/\   | | %s  | |  |
(_______________(   |  \  | |__________| | /
(_______________(   |   \ |______________|/ ___/\
(_______________(  /     |____>______<_____/     \
(_______________( /     / = ==== ==== ==== /|    _|_
(   RISC PC 600 (/     / ========= === == / /   ////
 ¯¯¯¯¯¯¯¯¯¯¯¯¯¯¯      / ========= === == / /   ////
                     <__________________<_/    ¯¯¯
`
	// Error code to message mappings
	resp_msg = map[int]string{
		MSG_INVALID_REQ_TYPE:        "Invalid request type",
		MSG_MISSING_INFO_HASH:       "info_hash missing from request",
		MSG_MISSING_PEER_ID:         "peer_id missing from request",
		MSG_MISSING_PORT:            "port missing from request",
		MSG_INVALID_PORT:            "Invalid port",
		MSG_INVALID_INFO_HASH:       "Torrent info hash must be 20 characters",
		MSG_INVALID_PEER_ID:         "Peer ID Invalid",
		MSG_INVALID_NUM_WANT:        "num_want invalid",
		MSG_INFO_HASH_NOT_FOUND:     "info_hash was not found, better luck next time",
		MSG_CLIENT_REQUEST_TOO_FAST: "Slot down there jimmy.",
		MSG_MALFORMED_REQUEST:       "Malformed request",
		MSG_GENERIC_ERROR:           "Generic Error :(",
	}

	mika *Tracker

	version string

	// Channels
	sync_user    = make(chan *User, 100)
	sync_peer    = make(chan *Peer, 1000)
	sync_torrent = make(chan *Torrent, 500)
	counter      = make(chan int)

	err_parse_reply = errors.New("Failed to parse reply")
	err_cast_reply  = errors.New("Failed to cast reply into type")

	stats *StatsCounter

	raven_client *raven.Client

	config     *Config
	configLock = new(sync.RWMutex)

	profile     = flag.String("profile", "", "write cpu profile to file")
	config_file = flag.String("config", "./config.json", "Config file path")
	num_procs   = flag.Int("procs", runtime.NumCPU()-1, "Number of CPU cores to use (default: ($num_cores-1))")
)

// math.Max for uint64
func UMax(a, b uint64) uint64 {
	if a > b {
		return a
	} else {
		return b
	}
}

// math.Min for uint64
func UMin(a, b uint64) uint64 {
	if a < b {
		return a
	} else {
		return b
	}
}

// Output a bencoded error code to the torrent client using
// a preset message code constant
func oops(c *echo.Context, msg_code int) {
	c.String(msg_code, responseError(resp_msg[msg_code]))
}

// Output a bencoded error code to the torrent client using
// a preset message code constant
func oopsStr(c *echo.Context, msg_code int, msg string) {
	c.String(msg_code, responseError(msg))
}

// Generate a bencoded error response for the torrent client to
// parse and display to the user
func responseError(message string) string {
	var out_bytes bytes.Buffer
	//	var er_msg = ErrorResponse{FailReason: message}
	//	er_msg_encoded := bencode.Marshal(&out_bytes)
	//	if er_msg_encoded != nil {
	//		return "."
	//	}
	bencoder := bencode.NewEncoder(&out_bytes)
	bencoder.Encode(bencode.Dict{
		"failure reason": message,
	})
	return out_bytes.String()
}

// Estimate a peers speed using downloaded amount over time
func estSpeed(start_time int32, last_time int32, bytes_sent uint64) float64 {
	if start_time <= 0 || last_time <= 0 || bytes_sent <= 0 || last_time < start_time {
		return 0.0
	}
	return float64(bytes_sent) / (float64(last_time) - float64(start_time))
}

// Generate a 32bit unix timestamp
func unixtime() int32 {
	return int32(time.Now().Unix())
}

func Debug(msg ...interface{}) {
	if config.Debug {
		log.Println(msg...)
	}
}

func CaptureMessage(message ...string) {
	if config.SentryDSN == "" {
		return
	}
	msg := strings.Join(message, "")
	if msg == "" {
		return
	}
	_, err := raven_client.CaptureMessage()
	if err != nil {
		log.Println(err)
	}
}

// Do it
func main() {
	log.Println(fmt.Sprintf(cheese, version))

	log.Println("Process ID:", os.Getpid())

	// Set max number of CPU cores to use
	log.Println("Num procs(s):", *num_procs)
	runtime.GOMAXPROCS(*num_procs)

	if *profile != "" {
		f, err := os.Create(*profile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}
	var err error
	raven_client, err = raven.NewClient(config.SentryDSN)
	if err != nil {
		log.Println("Could not connect to sentry")
	}
	CaptureMessage("Started tracker")

	connRequest = make(chan bool, 200)
	connResponse = make(chan redis.Conn, 200)
	connDone = make(chan redis.Conn, 200)
	pool = &redis.Pool{
		MaxIdle: 20,
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", config.RedisHost)
			if err != nil {
				return nil, err
			}
			if config.RedisPass != "" {
				if _, err := c.Do("AUTH", config.RedisPass); err != nil {
					c.Close()
					return nil, err
				}
			}
			return c, err
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			_, err := c.Do("PING")
			return err
		},
	}

	// Initialize the redis pool manager
	go redisPoolManager(pool)

	// Pre load models into memory
	mika.Initialize()

	go dbStatIndexer()

	// Initialize the router + middlewares
	e := echo.New()

	e.MaxParam(1)

	// Public tracker routes
	e.Get("/:passkey/announce", HandleAnnounce)
	e.Get("/:passkey/scrape", HandleScrape)

	api_grp := e.Group("/api")
	api_grp.Get("/version", HandleVersion)
	api_grp.Get("/torrent/:torrent_id", HandleTorrentGet)
	api_grp.Post("/torrent", HandleTorrentAdd)
	api_grp.Get("/test", HandleGetTorrentPeer)
	api_grp.Get("/user/:user_id", HandleUserGet)
	api_grp.Post("/user/:user_id", HandleUserUpdate)

	// Start watching for expiring peers
	go peerStalker()

	// Start writer channel
	go syncWriter()

	// Start server
	e.Run(config.ListenHost)
}

func init() {
	if version == "" {
		log.Fatalln(`Build this binary with "make", not "go build"`)
	}
	// Parse CLI args
	flag.Parse()

	loadConfig(true)

	// Start stat counter
	stats = NewStatCounter(counter)
	go stats.counter()
	go stats.statPrinter()

	// Alloc tracker
	mika = &Tracker{
		Torrents: make(map[uint64]*Torrent),
		Users:    make(map[uint64]*User),
	}

	s := make(chan os.Signal, 1)
	signal.Notify(s, syscall.SIGUSR2, syscall.SIGINT)
	go func() {
		for received_signal := range s {
			switch received_signal {
			case syscall.SIGINT:
				log.Println("\nShutting down!")
				if *profile != "" {
					log.Println("> Writing out profile info")
					pprof.StopCPUProfile()
				}
				CaptureMessage("Stopped tracker")
				os.Exit(0)
			case syscall.SIGUSR2:
				log.Println("SIGUSR2")
				<-s
				loadConfig(false)
				log.Println("> Reloaded config")
				CaptureMessage("Reloaded configuration")
			}
		}
	}()
}
