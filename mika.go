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
// echo never > /sys/kernel/mm/transparent_hugepage/enabled
// echo 10000 > /proc/sys/net/core/somaxconn

package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"github.com/chihaya/bencode"
	"github.com/garyburd/redigo/redis"
	"github.com/kisielk/raven-go/raven"
	"github.com/labstack/echo"
//	"github.com/thoas/stats"
	"log"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"runtime/pprof"
	"strconv"
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
  /               / |     | |  Mika    | |  |
 /_______________/  |/\   | |  v1.0    | |  |
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

	sync_user = make(chan *User, 100)
	sync_peer = make(chan *Peer, 1000)
	sync_torrent = make(chan *Torrent, 500)

	err_parse_reply = errors.New("Failed to parse reply")
	err_cast_reply = errors.New("Failed to cast reply into type")

	raven_client *raven.Client

	config     *Config
	configLock = new(sync.RWMutex)

//	pool *redis.Pool

	profile = flag.String("profile", "", "write cpu profile to file")
	config_file = flag.String("config", "./config.json", "Config file path")
	num_procs = flag.Int("procs", runtime.NumCPU()-1, "Number of CPU cores to use (default: ($num_cores-1))")
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

func RedisConn() (redis.Conn, error) {
	conn, err := redis.Dial("tcp", config.RedisHost)
	if err != nil {
		return nil, err
	}
	if config.RedisPass != "" {
		if _, err := conn.Do("AUTH", config.RedisPass); err != nil {
			conn.Close()
			return nil, err
		}
	}
	return conn, nil
}

// Create a new redis pool
func newPool(server, password string, max_idle int) *redis.Pool {
	return &redis.Pool{
		MaxIdle:     max_idle,
		IdleTimeout: 10 * time.Second,
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", server)
			if err != nil {
				return nil, err
			}
			if password != "" {
				if _, err := c.Do("AUTH", password); err != nil {
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
		Wait: true,
	}
}

// Output a bencoded error code to the torrent client using
// a preset message code constant
func oops(c *echo.Context, msg_code int) {
	c.String(msg_code, responseError(resp_msg[msg_code]))
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

func jsonString(obj interface{}) string {
	var b bytes.Buffer
	json.NewEncoder(&b).Encode(obj)
	return strings.Replace(b.String(), "\n", "", -1)
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

func HandleTorrentInfo(c *echo.Context) {
	r, redis_err := RedisConn()
	if redis_err != nil {
		CaptureMessage(redis_err.Error())
		log.Println("TorrentInfo redis conn:", redis_err.Error())
		return
	}
	defer r.Close()

	torrent_id_str := c.Param("torrent_id")
	torrent_id, err := strconv.ParseUint(torrent_id_str, 10, 64)
	if err != nil {
		log.Println(err)
		c.String(http.StatusNotFound, err.Error())
		return
	}
	torrent := mika.GetTorrentByID(r, torrent_id)

	//	for _, peer := range torrent.Peers {
	//		log.Println(peer)
	//	}
	c.JSON(http.StatusOK, torrent)
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
	//Debug("Event Registered:", id)
}

func syncWriter() {
	r, redis_err := RedisConn()
	if redis_err != nil {
		CaptureMessage(redis_err.Error())
		log.Println("SyncWriter redis conn:", redis_err.Error())
		return
	}
	defer r.Close()
	for {
		select {
		case user := <-sync_user:
			Debug("sync user")
			user.Sync(r)
			user.InQueue = false
		case torrent := <-sync_torrent:
			Debug("sync torrent")
			torrent.Sync(r)
			torrent.InQueue = false
		case peer := <-sync_peer:
			Debug("sync peer")
			peer.Sync(r)
			peer.InQueue = false
		}
		err := r.Flush()
		if err != nil {
			log.Println(err)
		}
	}
}

// Do it
func main() {
	log.Println(cheese)

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

	// Initialize the redis pool
	//	pool = newPool(config.RedisHost, config.RedisPass, config.RedisMaxIdle)

	// Initialize the router + middlewares
	e := echo.New()

	// Passkey is the only param we use, so only allocate for 1
	e.MaxParam(1)

	// Third-party middleware
	//	s := stats.New()
	//	e.Use(s.Handler)

	// Stats route
	//	e.Get("/stats", func(c *echo.Context) {
	//		c.JSON(200, s.Data())
	//	})

	// Public tracker routes
	e.Get("/:passkey/announce", HandleAnnounce)
	e.Get("/:passkey/scrape", HandleScrape)

	e.Get("/torrent/:torrent_id", HandleTorrentInfo)

	// Start watching for expiring peers
	go peerStalker()

	// Start writer channel
	go syncWriter()

	// Start server
	e.Run(config.ListenHost)
}

func init() {
	// Parse CLI args
	flag.Parse()

	mika = &Tracker{
		Torrents: make(map[uint64]*Torrent),
		Users:    make(map[uint64]*User),
	}

	loadConfig(true)
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
