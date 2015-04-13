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
	"fmt"
	"github.com/garyburd/redigo/redis"
	"github.com/jackpal/bencode-go"
	"github.com/labstack/echo"
	mw "github.com/labstack/echo/middleware"
	"github.com/thoas/stats"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

type (

Config struct {
	Debug      bool
	ListenHost string
	RedisHost string
	RedisPass string
	RedisMaxIdle int
}

ScrapeResponse struct {}

ErrorResponse struct {
	FailReason string `bencode:"failure reason"`
}

AnnounceResponse struct {
	MinInterval int    `bencode:"min interval"`
	Complete    int    `bencode:"complete"`
	Incomplete  int    `bencode:"incomplete"`
	Interval    int    `bencode:"interval"`
	Peers       string `bencode:"peers"`
}

// Peers
Peer struct {
	SpeedUP       float64 `redis:"speed_up"`
	SpeedDN       float64 `redis:"speed_dj"`
	Uploaded      uint64  `redis:"uploaded"`
	Downloaded    uint64  `redis:"downloaded"`
	Corrupt       uint64  `redis:"corrupt"`
	IP            net.IP  `redis:"ip"`
	Port          uint64  `redis:"port"`
	Left          uint64  `redis:"left"`
	Announces     uint64  `redis:"announces"`
	TotalTime     uint64  `redis:"total_time"`
	AnnounceLast  int32   `redis:"last_announce"`
	AnnounceFirst int32   `redis:"first_announce"`
	New           bool    `redis:"new"`
	Active        bool    `redis:"active"`
	UserID        uint64  `redis:"user_id"`
}

AnnounceRequest struct {
	Compact    bool
	Downloaded uint64
	Corrupt    uint64
	Event      string
	IPv4       net.IP
	InfoHash   string
	Left       uint64
	NumWant    int
	Passkey    string
	PeerID     string
	Port       uint64
	Uploaded   uint64
}

ScrapeRequest struct {
	InfoHashes []string
}

Query struct {
	InfoHashes []string
	Params     map[string]string
}
)

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
	ANNOUNCE_INTERVAL           int = 600
	PEER_EXPIRY                 float32 = float32(ANNOUNCE_INTERVAL) * 1.25
)

var (

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

	err_parse_reply = errors.New("Failed to parse reply")
	err_cast_reply = errors.New("Failed to cast reply into type")

	config     *Config
	configLock = new(sync.RWMutex)

	pool *redis.Pool

	config_file = flag.String("config", "./config.json", "Config file path")
	num_procs = flag.Int("procs", runtime.NumCPU()-1, "Number of CPU cores to use (default: ($num_cores-1))")
)

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

func GetConfig() *Config {
	configLock.RLock()
	defer configLock.RUnlock()
	return config
}

// Fetch the stores torrent_id that corresponds to the info_hash supplied
// as a GET value. If the info_hash doesnt return an id we consider the torrent
// either soft-deleted or non-existent
func GetTorrentID(r redis.Conn, info_hash string) uint64 {
	torrent_id_reply, err := r.Do("GET", fmt.Sprintf("t:info_hash:%x", info_hash))
	if err != nil {
		log.Println("Failed to execute torrent_id query", err)
		return 0
	}
	if torrent_id_reply == nil {
		log.Println("Invalid info hash")
		return 0
	}
	torrent_id, tid_err := redis.Uint64(torrent_id_reply, nil)
	if tid_err != nil {
		log.Println("Failed to parse torrent_id reply", tid_err)
		return 0
	}
	return torrent_id
}

func makeCompactPeers(peers []Peer) string {
	var out_buf bytes.Buffer
	for _, peer := range peers {
		out_buf.Write(peer.IP.To4())
		out_buf.Write([]byte{byte(peer.Port >> 8), byte(peer.Port & 0xff)})
	}
	return string(out_buf.Bytes()[:])
}

// Fetch a torrents data from the database and return a Torrent struct.
// If the torrent doesnt exist in the database a new skeleton Torrent
// instance will be returned.
func GetTorrent(r redis.Conn, torrent_id uint64) map[string]string {
	torrent_reply, err := r.Do("HGETALL", fmt.Sprintf("t:t:%d", torrent_id))
	if err != nil {
		log.Println("Error executing torrent fetch query: ", err)
	}
	if torrent_reply == nil {
		log.Println("Making new torrent struct")
		torrent := make(map[string]string)
		return torrent
	} else {
		torrent, err := redis.StringMap(torrent_reply, nil)
		if err != nil {
			log.Println("Failed to fetch torrent: ", err)
		}
		return torrent
	}
}

// Fetch a user_id from the supplied passkey. A return value
// of 0 denotes a non-existing or disabled user_id
func GetUserID(r redis.Conn, passkey string) uint64 {
	Debug(fmt.Println("Fetching passkey", passkey))
	user_id_reply, err := r.Do("GET", fmt.Sprintf("t:user:%s", passkey))
	if err != nil {
		log.Println(err)
		return 0
	}
	user_id, err_b := redis.Uint64(user_id_reply, nil)
	if err_b != nil {
		log.Println("Failed to find user", err_b)
		return 0
	}
	return user_id
}

// Checked if the clients peer_id prefix matches the client prefixes
// stored in the white lists
func IsValidClient(r redis.Conn, peer_id string) bool {
	a, err := r.Do("HKEYS", "t:whitelist")

	if err != nil {
		log.Println(err)
		return false
	}
	clients, err := redis.Strings(a, nil)
	for _, client_prefix := range clients {
		if strings.HasPrefix(peer_id, client_prefix) {
			return true
		}
	}
	return false
}

// Get an array of peers for a supplied torrent_id
func GetPeers(r redis.Conn, torrent_id uint64, max_peers int) []Peer {
	peers_reply, err := r.Do("SMEMBERS", fmt.Sprintf("t:t:%d:p", torrent_id))
	if err != nil || peers_reply == nil {
		log.Println("Error fetching peers_resply", err)
		return nil
	}
	peer_ids, err := redis.Strings(peers_reply, nil)
	if err != nil {
		log.Println("Error parsing peers_resply", err)
		return nil
	}
	var peer_size = Min(len(peer_ids), max_peers)
	for _, peer_id := range peer_ids[0:peer_size] {
		r.Send("HGETALL", fmt.Sprintf("t:t:%d:%s", torrent_id, peer_id))
	}
	r.Flush()
	peers := make([]Peer, peer_size)

	for range peers[0:peer_size] {
		peer_reply, err := r.Receive()
		if err != nil {
			log.Println(err)
		} else {
			peer, err := makePeer(peer_reply)
			if err != nil {

			} else {
				peers = append(peers, peer)
			}
		}
	}
	return peers
}

func makePeer(redis_reply interface{}) (Peer, error) {
	peer := Peer{
		Active:        false,
		Announces:     0,
		SpeedUP:       0,
		SpeedDN:       0,
		Uploaded:      0,
		Downloaded:    0,
		Left:          0,
		Corrupt:       0,
		IP:            net.ParseIP("127.0.0.1"),
		Port:          0,
		AnnounceFirst: unixtime(),
		AnnounceLast:  unixtime(),
		TotalTime:     0,
		UserID:        0,
		New:           true,
	}

	values, err := redis.Values(redis_reply, nil)
	if err != nil {
		log.Println("Failed to parse peer reply: ", err)
		return peer, err_parse_reply
	}
	if values == nil {
		log.Println("Making new peer struct")
	} else {
		err := redis.ScanStruct(values, &peer)
		if err != nil {
			log.Println("Failed to fetch peer: ", err)
			return peer, err_cast_reply
		} else {
			peer.Announces += 1
			peer.New = false
		}
	}
	Debug("Peer: ", peer)
	return peer, nil

}

// Fetch an existing peers data if it exists, other wise generate a
// new peer with default data values. The data is parsed into a Peer
// struct and returned.
func GetPeer(r redis.Conn, torrent_id uint64, peer_id string) (Peer, error) {
	peer_reply, err := r.Do("HGETALL", fmt.Sprintf("t:t:%d:%s", torrent_id, peer_id))
	if err != nil {
		log.Println("Error executing peer fetch query: ", err)
	}
	return makePeer(peer_reply)
}

// Add a peer to a torrents active peer_id list
func AddPeer(r redis.Conn, torrent_id uint64, peer_id string) bool {
	v, err := r.Do("SADD", fmt.Sprintf("t:t:%d:p", torrent_id), peer_id)
	if err != nil {
		log.Println("Error executing peer fetch query: ", err)
		return false
	}
	if v == "0" {
		log.Println("Tried to add peer to set with existing element")
	}
	return true
}

// Remove a peer from a torrents active peer_id list
func DelPeer(r redis.Conn, torrent_id uint64, peer_id string) bool {
	_, err := r.Do("SREM", fmt.Sprintf("t:t:%s:p", torrent_id), peer_id)
	if err != nil {
		log.Println("Error executing peer fetch query: ", err)
		return false
	}
	// Mark inactive?
	//r.Do("DEL", fmt.Sprintf("t:t:%d:p:%s", torrent_id, peer_id))
	return true
}

// Parses a raw url query into a Query struct
func QueryStringParser(query string) (*Query, error) {
	var (
		keyStart, keyEnd int
		valStart, valEnd int
		firstInfoHash string

		onKey = true
		hasInfoHash = false

		q = &Query{
			InfoHashes: nil,
			Params:     make(map[string]string),
		}
	)

	for i, length := 0, len(query); i < length; i++ {
		separator := query[i] == '&' || query[i] == ';' || query[i] == '?'
		if separator || i == length-1 {
			if onKey {
				keyStart = i + 1
				continue
			}
			if i == length-1 && !separator {
				if query[i] == '=' {
					continue
				}
				valEnd = i
			}
			keyStr, err := url.QueryUnescape(query[keyStart : keyEnd+1])
			if err != nil {
				return nil, err
			}
			valStr, err := url.QueryUnescape(query[valStart : valEnd+1])
			if err != nil {
				return nil, err
			}
			q.Params[strings.ToLower(keyStr)] = valStr

			if keyStr == "info_hash" {
				if hasInfoHash {
					// Multiple info hashes
					if q.InfoHashes == nil {
						q.InfoHashes = []string{firstInfoHash}
					}
					q.InfoHashes = append(q.InfoHashes, valStr)
				} else {
					firstInfoHash = valStr
					hasInfoHash = true
				}
			}
			onKey = true
			keyStart = i + 1
		} else if query[i] == '=' {
			onKey = false
			valStart = i + 1
		} else if onKey {
			keyEnd = i
		} else {
			valEnd = i
		}
	}

	return q, nil
}

// Uint64 is a helper to obtain a uint of any length from a Query. After being
// called, you can safely cast the uint64 to your desired length.
func (q *Query) Uint64(key string) (uint64, error) {
	str, exists := q.Params[key]
	if !exists {
		return 0, errors.New("value does not exist for key: " + key)
	}

	val, err := strconv.ParseUint(str, 10, 64)
	if err != nil {
		return 0, err
	}
	return val, nil
}

// math.Max for uint64
func UMax(a, b uint64) uint64 {
	if a > b {
		return a
	} else {
		return b
	}
}

// math.Max for uint64
func Min(a, b int) int {
	if a < b {
		return a
	} else {
		return b
	}
}

// Parse the query string into an AnnounceRequest struct
func NewAnnounce(c *echo.Context) (*AnnounceRequest, error) {
	q, err := QueryStringParser(c.Request.RequestURI)
	if err != nil {
		return nil, err
	}

	s := strings.Split(c.Request.RemoteAddr, ":")
	ip_req, _ := s[0], s[1]

	compact := q.Params["compact"] != "0"
	event, _ := q.Params["event"]

	numWant := getNumWant(q, 30)

	info_hash, exists := q.Params["info_hash"]
	if !exists {
		return nil, errors.New("Invalid info hash")
	}

	peerID, exists := q.Params["peer_id"]
	if !exists {
		return nil, errors.New("Invalid peer_id")
	}

	ipv4, err := getIP(q.Params["ip"])
	if err != nil {
		ipv4_new, err := getIP(ip_req)
		if err != nil {
			log.Println(err)
			return nil, errors.New("Invalid ip hash")
		}
		ipv4 = ipv4_new
	}

	port, err := q.Uint64("port")
	if err != nil || port < 1024 || port > 65535 {
		return nil, errors.New("Invalid port")
	}

	left, err := q.Uint64("left")
	if err != nil {
		return nil, errors.New("No left value")
	} else {
		left = UMax(0, left)
	}

	downloaded, err := q.Uint64("downloaded")
	if err != nil {
		return nil, errors.New("Invalid downloaded value")
	} else {
		downloaded = UMax(0, downloaded)
	}

	uploaded, err := q.Uint64("uploaded")
	if err != nil {
		return nil, errors.New("Invalid uploaded value")
	} else {
		uploaded = UMax(0, uploaded)
	}

	corrupt, err := q.Uint64("corrupt")
	if err != nil {
		// Assume we just don't have the param
		corrupt = 0
	} else {
		corrupt = UMax(0, corrupt)
	}

	return &AnnounceRequest{
		Compact:    compact,
		Corrupt:    corrupt,
		Downloaded: downloaded,
		Event:      event,
		IPv4:       ipv4,
		InfoHash:   info_hash,
		Left:       left,
		NumWant:    numWant,
		PeerID:     peerID,
		Port:       port,
		Uploaded:   uploaded,
	}, nil
}

// Parse and return a IP from a string
func getIP(ip_str string) (net.IP, error) {
	ip := net.ParseIP(ip_str)
	if ip != nil {
		return ip.To4(), nil
	}
	return nil, errors.New("Failed to parse ip")
}

// Parse the num want from the announce request, replacing with our
// own default value if the supplied value is missing or deemed invalid
func getNumWant(q *Query, fallback int) int {
	if numWantStr, exists := q.Params["numwant"]; exists {
		numWant, err := strconv.Atoi(numWantStr)
		if err != nil {
			return fallback
		}
		return numWant
	}

	return fallback
}

// Create a new redis pool
func newPool(server, password string, max_idle int) *redis.Pool {
	return &redis.Pool{
		MaxIdle:     max_idle,
		IdleTimeout: 240 * time.Second,
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
	var er_msg = ErrorResponse{FailReason: message}
	er_msg_encoded := bencode.Marshal(&out_bytes, er_msg)
	if er_msg_encoded != nil {
		return "."
	}
	return out_bytes.String()
}

// Estimate a peers speed using downloaded amount over time
func estSpeed(start_time int32, last_time int32, bytes_sent uint64) float64 {
	if last_time < start_time {
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

// Route handler for the /announce endpoint
// Here be dragons
func handleAnnounce(c *echo.Context) {
	cur_time := unixtime()
	r := pool.Get()
	defer r.Close()

	ann, err := NewAnnounce(c)
	if err != nil {
		log.Println(err)
		oops(c, MSG_GENERIC_ERROR)
		return
	}

	passkey := c.Param("passkey")

	var user_id = GetUserID(r, passkey)
	if user_id <= 0 {
		oops(c, MSG_GENERIC_ERROR)
		return
	}
	Debug("UserID: ", user_id)

	if !IsValidClient(r, ann.PeerID) {
		oops(c, MSG_INVALID_PEER_ID)
		return
	}

	var torrent_id = GetTorrentID(r, ann.InfoHash)
	if torrent_id <= 0 {
		oops(c, MSG_INFO_HASH_NOT_FOUND)
		return
	}
	Debug("TorrentID: ", torrent_id)


	peer, err := GetPeer(r, torrent_id, ann.PeerID)
	if err != nil {
		oops(c, MSG_GENERIC_ERROR)
		return
	}
	peer.UserID = user_id

	torrent := GetTorrent(r, torrent_id)
	Debug(torrent)

	peer.IP = ann.IPv4
	peer.Corrupt += ann.Corrupt
	peer.Left = ann.Left
	peer.SpeedUP = estSpeed(peer.AnnounceLast, cur_time, ann.Uploaded)
	peer.SpeedDN = estSpeed(peer.AnnounceLast, cur_time, ann.Downloaded)

	if ann.Event == "stopped" {
		peer.Active = false
		DelPeer(r, torrent_id, ann.PeerID)
	} else {
		peer.Active = true
		AddPeer(r, torrent_id, ann.PeerID)
	}
	peers := GetPeers(r, torrent_id, ann.NumWant)
	for _, p := range peers {
		Debug(p)
	}
	// compact_peers := makeCompactPeers(peers)

	// Define our keys
	// TODO memoization function?
	torrent_key := fmt.Sprintf("t:t:%d", torrent_id)
	peer_key := fmt.Sprintf("t:t:%d:%s", torrent_id, ann.PeerID)
	torrent_peers_set := fmt.Sprintf("t:t:%d:p", torrent_id)
	users_active_key := fmt.Sprintf("t:u:%d:active", peer.UserID)
	users_incomplete_key := fmt.Sprintf("t:u:%d:incomplete", peer.UserID)
	users_complete_key := fmt.Sprintf("t:u:%d:complete", peer.UserID)
	users_hnr_key := fmt.Sprintf("t:u:%d:hnr", peer.UserID)

	r.Send("HINCRBY", torrent_key, "announces", 1)
	r.Send("HINCRBY", peer_key, "announces", 1)

	// pipe.hset(peer_key, "completed", 0) ??

	if ann.Event == "stopped" {
		// Remove from torrents active peer set
		r.Send("SREM", torrent_peers_set, ann.PeerID)

		// Mark the peer as inactive
		r.Send("HSET", peer_key, "active", 0)

		// Handle total changes if we were previously an active peer
		if peer.Active {
			if peer.Left > 0 {
				r.Send("HINCRBY", torrent_key, "leechers", -1)
			} else {
				// For sanity, maybe probably removable once stable
				r.Send("HSET", peer_key, "completed", 1)

				// Remove active seeder
				r.Send("HINCRBY", torrent_key, "seeders", -1)
			}
		}
	} else if ann.Event == "completed" {
		if peer.Left > 0 {
			if !peer.New && peer.Active {
				// If the user was previously an active peer and has data left
				// we assume he was leeching so we decrement it now
				r.Send("HINCRBY", torrent_key, "leechers", -1)
				Debug("Torrent Leecher -1")
			}
		}
		// Should we disallow peers being able to trigger this twice?
		// Forcing only 1 for now
		r.Send("HSET", peer_key, "completed", 1)

		// Increment active seeders for the torrent
		r.Send("HINCRBY", torrent_key, "seeders", 1)

		// Remove the torrent from the users incomplete set
		r.Send("SREM", users_incomplete_key, torrent_id)

		// Remove the torrent from the users incomplete set
		r.Send("SADD", users_complete_key, torrent_id)

		// Remove from the users hnr list if it exists
		r.Send("SREM", users_hnr_key, torrent_id)

	} else if ann.Event == "started" {

		if ann.Left > 0 {
			// Add the torrent to the users incomplete set
			r.Send("SREM", users_incomplete_key, torrent_id)

			r.Send("HINCRBY", torrent_key, "leechers", 1)
		} else {
			r.Send("HINCRBY", torrent_key, "seeders", 1)
		}
	}
	if ann.Event == "stopped" {
		// Remove from the users active torrents set
		r.Send("SREM", users_active_key, torrent_id)
	} else {
		// Add peer to torrent active peers
		r.Send("SADD", torrent_peers_set, ann.PeerID)

		// Add to users active torrent set
		r.Send("SADD", users_active_key, torrent_id)

		// Refresh the peers expiration timer
		r.Send("SETEX", fmt.Sprintf("t:t:%d:%s:exp", torrent_id, ann.PeerID), 4, 1)
	}

	// Update tracker totals
	r.Send("HINCRBY", torrent_key, "uploaded", ann.Uploaded)
	r.Send("HINCRBY", torrent_key, "downloaded", ann.Downloaded)

	// Update peer transfer stats
	r.Send("HINCRBY", peer_key, "uploaded", ann.Uploaded)
	r.Send("HINCRBY", peer_key, "downloaded", ann.Downloaded)
	r.Send("HINCRBY", peer_key, "corrupt", ann.Corrupt)

	// Must be active to have a real time delta
	if peer.Active {
		ann_diff := uint64(unixtime() - peer.AnnounceLast)
		// Ignore long periods of inactivity
		if ann_diff < 1500 {
			r.Send("HINCRBY", peer_key, "total_time", ann_diff)
		}
	}

	r.Send("HMSET", peer_key, "ip", ann.IPv4.String(), "port", ann.Port, "left", ann.Left, "first_announce", peer.AnnounceFirst, "last_announce", peer.AnnounceLast, "speed_up", peer.SpeedUP, "speed_dn", peer.SpeedDN)
	r.Flush()
	resp := AnnounceResponse{
		Complete:    1,
		Incomplete:  1,
		Interval:    900,
		MinInterval: 300,
		Peers:       makeCompactPeers(peers),
	}
	var out_bytes bytes.Buffer
	er_msg_encoded := bencode.Marshal(&out_bytes, resp)
	if er_msg_encoded != nil {
		oops(c, MSG_GENERIC_ERROR)
		log.Println(er_msg_encoded)
		return
	}
	encoded := out_bytes.String()
	log.Println(encoded)
	c.String(http.StatusOK, encoded)
}

// Will mark a torrent peer as inactive and remove them
// from the torrents active peer_id set
func reapPeer(torrent_id, peer_id string) {
	r := pool.Get()
	defer r.Close()
	Debug("Reaping peer:", torrent_id, peer_id)

	torrent_id_uint, err := strconv.ParseUint(torrent_id, 10, 64)
	if err != nil {
		log.Println("Failed to parse torrent id into uint64", err)
		return
	}

	// Fetch before we set active to 0
	peer, err := GetPeer(r, torrent_id_uint, peer_id)
	if err != nil {
		log.Println("Failed to fetch peer while reaping")
	}
	queued := 2
	r.Send("SREM", fmt.Sprintf("t:t:%s:p", torrent_id), peer_id)
	r.Send("HSET", fmt.Sprintf("t:t:%s:p:%s", torrent_id, peer_id), "active", 0)
	if peer.Active {
		if peer.Left > 0 {
			r.Send("HINCRBY", fmt.Sprintf("t:t:%s", torrent_id), "leechers", -1)
		} else {
			r.Send("HINCRBY", fmt.Sprintf("t:t:%s", torrent_id), "seeders", -1)
		}
		queued += 1
	}

	r.Flush()
	v, err := r.Receive()
	queued -= 1
	if err != nil {
		log.Println("Tried to remove non-existant peer: ", torrent_id, peer_id)
	}
	if v == "1" {
		Debug("Reaped peer successfully: ", peer_id)
	}

	// all needed i think, must match r.Send count?
	for i := 0; i < queued; i++ {
		r.Receive()
	}

}

// This is a goroutine that will watch for peer key expiry events and
// act on them, removing them from the active peer lists
func peer_stalker() {
	r := pool.Get()
	defer r.Close()

	psc := redis.PubSubConn{r}
	psc.Subscribe("__keyevent@0__:expired")
	for {
		switch v := psc.Receive().(type) {
			case redis.Message:
			Debug(fmt.Sprintf("Key Expiry: %s\n", v.Data))
			p := strings.SplitN(string(v.Data[:]), ":", 5)
			reapPeer(p[2], p[3])
			case redis.Subscription:
			log.Printf("%s: %s %d\n", v.Channel, v.Kind, v.Count)
			case error:
			log.Println("Subscriber error: ", v.Error())
		}
	}
}

// Route handler for the /scrape requests
func handleScrape(c *echo.Context) {
	c.String(http.StatusOK, "I like to scrape my ass")
}

// Do it
func main() {


	// Set max number of CPU cores to use
	log.Println("Num procs(s):", *num_procs)
	runtime.GOMAXPROCS(*num_procs)

	// Initialize the redis pool
	pool = newPool(config.RedisHost, config.RedisPass, config.RedisMaxIdle)

	// Initialize the router + middlewares
	e := echo.New()
	e.MaxParam(1)

	if config.Debug {
		e.Use(mw.Logger)
	}

	// Third-party middleware
	s := stats.New()
	e.Use(s.Handler)
	// Stats route
	e.Get("/stats", func(c *echo.Context) {
		c.JSON(200, s.Data())
	})

	// Public tracker routes
	e.Get("/:passkey/announce", handleAnnounce)
	e.Get("/:passkey/scrape", handleScrape)

	// Start watching for expiring peers
	go peer_stalker()

	// Start server
	log.Println(config)
	e.Run(config.ListenHost)
}

func init() {
	// Parse CLI args
	flag.Parse()

	loadConfig(true)
	s := make(chan os.Signal, 1)
	signal.Notify(s, syscall.SIGUSR2)
	go func() {
		for {
			<-s
			loadConfig(false)
			log.Println("> Reloaded config")
		}
	}()
}
