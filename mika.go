// Copyright 2015 toor@titansof.tv
//
// A (currently) stateless torrent tracker using Redis as a backing store
//
// Performance tuning options for linux kernel
//
// Set in sysctl.conf
// fs.file-max=100000
// vm.overcommit_memory = 1
// vm.swappiness=0
// net.ipv4.tcp_sack=1                   # enable selective acknowledgements
// net.ipv4.tcp_timestamps=1             # needed for selective acknowledgements
// net.ipv4.tcp_window_scaling=1         # scale the network window
// net.ipv4.tcp_congestion_control=cubic # better congestion algorythm
// net.ipv4.tcp_syncookies=1             # enable syn cookied
// net.ipv4.tcp_tw_recycle=1             # recycle sockets quickly
// net.ipv4.tcp_max_syn_backlog=NUMBER   # backlog setting
// net.core.somaxconn=10000              # up the number of connections per port
// #net.core.rmem_max=NUMBER              # up the receive buffer size
// #net.core.wmem_max=NUMBER              # up the buffer size for all connections
// echo 1 > /proc/sys/net/ipv4/tcp_tw_reuse
// echo 1 > /proc/sys/net/ipv4/tcp_tw_recycle
// echo 10000 > /proc/sys/net/core/somaxconn
// echo 'never' > /sys/kernel/mm/transparent_hugepage/enabled
// redis.conf
// maxmemory-policy noeviction
// notify-keyspace-events "KEx"
// tcp-backlog 65536
//

package main

import (
	"bytes"
	"crypto/tls"
	"errors"
	"flag"
	"fmt"
	"github.com/chihaya/bencode"
	"github.com/garyburd/redigo/redis"
	"github.com/goji/httpauth"
	"github.com/kisielk/raven-go/raven"
	"github.com/labstack/echo"
	"log"
	"net/http"
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
	InQueue bool `redis:"-" json:"-"`
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
		MSG_CLIENT_REQUEST_TOO_FAST: "Slow down there jimmy.",
		MSG_MALFORMED_REQUEST:       "Malformed request",
		MSG_GENERIC_ERROR:           "Generic Error :(",
	}

	mika *Tracker

	version string

	start_time int32

	// Channels
	sync_user    = make(chan *User, 100)
	sync_peer    = make(chan *Peer, 1000)
	sync_torrent = make(chan *Torrent, 500)
	sync_payload = make(chan Payload, 1000)
	counter      = make(chan int)

	err_parse_reply = errors.New("Failed to parse reply")
	err_cast_reply  = errors.New("Failed to cast reply into type")

	stats *StatsCounter

	whitelist    []string
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
	if start_time <= 0 || last_time <= 0 || bytes_sent == 0 || last_time < start_time {
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
		log.Println("CaptureMessage: Failed to send message:", err)
	}
}

func runTracker() {
	log.Println("Loading tracker router on:", config.ListenHost)
	// Initialize the router + middlewares
	e := echo.New()
	e.MaxParam(1)

	//	e.HTTPErrorHandler(func(code int, err error, c *echo.Context) {
	//		log.Println("--------")
	//		log.Println(err)
	//		c.JSON(code, ResponseErr{err.Error()})
	//		return
	//	})

	// Public tracker routes
	e.Get("/:passkey/announce", HandleAnnounce)
	e.Get("/:passkey/scrape", HandleScrape)

	e.Run(config.ListenHost)
}

func runAPI() {
	log.Println("Loading API router on:", config.ListenHostAPI, "(TLS)")
	e := echo.New()
	e.MaxParam(1)
	api := e.Group("/api")

	// Optionally enabled BasicAuth over the TLS only API
	if config.APIUsername == "" || config.APIPassword == "" {
		log.Println("[WARN] No credentials set for API. All users granted access.")
	} else {
		api.Use(httpauth.SimpleBasicAuth(config.APIUsername, config.APIPassword))
	}

	api.Get("/version", HandleVersion)
	api.Get("/uptime", HandleUptime)

	api.Get("/torrent/:info_hash", HandleTorrentGet)
	api.Post("/torrent", HandleTorrentAdd)
	api.Get("/torrent/:info_hash/peers", HandleGetTorrentPeers)
	api.Delete("/torrent/:info_hash", HandleTorrentDel)

	api.Post("/user", HandleUserCreate)
	api.Get("/user/:user_id", HandleUserGet)
	api.Post("/user/:user_id", HandleUserUpdate)
	api.Get("/user/:user_id/torrents", HandleUserTorrents)

	api.Post("/whitelist", HandleWhitelistAdd)
	api.Delete("/whitelist/:prefix", HandleWhitelistDel)
	tls_config := &tls.Config{
		MinVersion:               tls.VersionTLS12,
		PreferServerCipherSuites: true,
		CipherSuites: []uint16{tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
			tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA,
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA,
		},
	}
	srv := http.Server{TLSConfig: tls_config, Addr: config.ListenHostAPI, Handler: e}
	log.Fatal(srv.ListenAndServeTLS(config.SSLCert, config.SSLPrivateKey))
}

func sigHandler(s chan os.Signal) {
	for received_signal := range s {
		switch received_signal {
		case syscall.SIGINT:
			log.Println("")
			log.Println("CAUGHT SIGINT: Shutting down!")
			if *profile != "" {
				log.Println("> Writing out profile info")
				pprof.StopCPUProfile()
			}
			CaptureMessage("Stopped tracker")
			os.Exit(0)
		case syscall.SIGUSR2:
			log.Println("")
			log.Println("CAUGHT SIGUSR2: Reloading config")
			<-s
			loadConfig(false)
			log.Println("> Reloaded config")
			CaptureMessage("Reloaded configuration")
		}
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

	pool = &redis.Pool{
		MaxIdle:     0,
		IdleTimeout: 600 * time.Second,
		Wait:        true,
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
			if err != nil {
				// TODO remove me, temp hack to allow supervisord to reload process
				// since we currently don't actually handle graceful reconnects yet.
				log.Fatalln("Bad redis voodoo! exiting!", err)
			}
			return err
		},
	}

	// Pre load models into memory
	mika.Initialize()

	go dbStatIndexer()

	// Start watching for expiring peers
	go peerStalker()

	// Start writer channel
	go syncWriter()

	// Start server
	go runTracker()
	runAPI() // Block so we don't return
}

func init() {
	start_time = unixtime()
	if version == "" {
		log.Println(`[WARN] Build this binary with "make", not "go build"`)
	}
	whitelist = []string{}

	// Parse CLI args
	flag.Parse()

	loadConfig(true)

	// Start stat counter
	stats = NewStatCounter(counter)
	go stats.counter()
	go stats.statPrinter()

	// Alloc tracker
	mika = &Tracker{
		Torrents:      make(map[string]*Torrent),
		Users:         make(map[uint64]*User),
		TorrentsMutex: new(sync.RWMutex),
		UsersMutex:    new(sync.RWMutex),
	}

	s := make(chan os.Signal, 1)
	signal.Notify(s, syscall.SIGUSR2, syscall.SIGINT)
	go sigHandler(s)
}
