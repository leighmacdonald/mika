package tracker

import (
	"crypto/tls"
	"fmt"
	"git.totdev.in/totv/mika/conf"
	"git.totdev.in/totv/mika/db"
	"git.totdev.in/totv/mika/stats"
	"git.totdev.in/totv/mika/util"
	"github.com/garyburd/redigo/redis"
	"github.com/goji/httpauth"
	"github.com/labstack/echo"
	"log"
	ghttp "net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	Mika *Tracker

	// Channels
	SyncUserC    = make(chan *User, 100)
	SyncPeerC    = make(chan *Peer, 1000)
	SyncTorrentC = make(chan *Torrent, 500)
)

func init() {

}

// Main track struct holding all the known models
type Tracker struct {
	TorrentsMutex *sync.RWMutex
	Torrents      map[string]*Torrent

	UsersMutex *sync.RWMutex
	Users      map[uint64]*User

	WhitelistMutex *sync.RWMutex
	Whitelist      []string
}

func NewTracker() *Tracker {
	// Alloc tracker
	tracker := &Tracker{
		Torrents:       make(map[string]*Torrent),
		Users:          make(map[uint64]*User),
		TorrentsMutex:  new(sync.RWMutex),
		UsersMutex:     new(sync.RWMutex),
		Whitelist:      []string{},
		WhitelistMutex: new(sync.RWMutex),
	}

	return tracker
}

// Load the models into memory from redis
func (t *Tracker) Initialize() error {
	log.Println("Initialize: Initializing models in memory...")
	r := db.Pool.Get()
	defer r.Close()

	t.initWhitelist(r)
	t.initUsers(r)
	t.initTorrents(r)

	return nil
}

func (t *Tracker) Run() {
	go t.dbStatIndexer()
	go t.syncWriter()
	go t.peerStalker()
	go t.listenTracker()
	t.listenAPI()
}

func (t *Tracker) listenTracker() {
	log.Println("Loading tracker router on:", conf.Config.ListenHost)
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
	e.Get("/:passkey/announce", t.HandleAnnounce)
	e.Get("/:passkey/scrape", t.HandleScrape)

	e.Run(conf.Config.ListenHost)
}

func (t *Tracker) listenAPI() {
	log.Println("Loading API router on:", conf.Config.ListenHostAPI, "(TLS)")
	e := echo.New()
	e.MaxParam(1)
	api := e.Group("/api")

	e.Use(stats.StatsMW)

	// Optionally enabled BasicAuth over the TLS only API
	if conf.Config.APIUsername == "" || conf.Config.APIPassword == "" {
		log.Println("[WARN] No credentials set for API. All users granted access.")
	} else {
		api.Use(httpauth.SimpleBasicAuth(conf.Config.APIUsername, conf.Config.APIPassword))
	}

	api.Get("/version", t.HandleVersion)
	api.Get("/uptime", t.HandleUptime)

	api.Get("/torrent/:info_hash", t.HandleTorrentGet)
	api.Post("/torrent", t.HandleTorrentAdd)
	api.Get("/torrent/:info_hash/peers", t.HandleGetTorrentPeers)
	api.Delete("/torrent/:info_hash", t.HandleTorrentDel)

	api.Post("/user", t.HandleUserCreate)
	api.Get("/user/:user_id", t.HandleUserGet)
	api.Post("/user/:user_id", t.HandleUserUpdate)
	api.Get("/user/:user_id/torrents", t.HandleUserTorrents)

	api.Post("/whitelist", t.HandleWhitelistAdd)
	api.Delete("/whitelist/:prefix", t.HandleWhitelistDel)
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
	if conf.Config.SSLCert == "" || conf.Config.SSLPrivateKey == "" {
		log.Fatalln("SSL config keys not set in config!")
	}
	srv := ghttp.Server{TLSConfig: tls_config, Addr: conf.Config.ListenHostAPI, Handler: e}
	log.Fatal(srv.ListenAndServeTLS(conf.Config.SSLCert, conf.Config.SSLPrivateKey))
}

// Fetch a torrents data from the database and return a Torrent struct.
// If the torrent doesn't exist in the database a new skeleton Torrent
// instance will be returned.
func (t *Tracker) GetTorrentByID(r redis.Conn, torrent_id uint64, make_new bool) *Torrent {
	t.TorrentsMutex.RLock()
	defer t.TorrentsMutex.RUnlock()
	for _, torrent := range t.Torrents {
		if torrent.TorrentID == torrent_id {
			return torrent
		}
	}
	return nil
}

// Fetch the stores torrent_id that corresponds to the info_hash supplied
// as a GET value. If the info_hash doesn't return an id we consider the torrent
// either soft-deleted or non-existent
func (t *Tracker) GetTorrentByInfoHash(r redis.Conn, info_hash string, make_new bool) *Torrent {
	info_hash = strings.ToLower(info_hash)
	t.TorrentsMutex.RLock()
	torrent, exists := t.Torrents[info_hash]
	t.TorrentsMutex.RUnlock()
	if exists {
		return torrent
	}
	if make_new {
		torrent = t.FetchTorrent(r, info_hash)
		if torrent == nil {
			return nil
		}
		t.TorrentsMutex.Lock()
		t.Torrents[info_hash] = torrent
		t.TorrentsMutex.Unlock()
		util.Debug("GetTorrentByInfoHash: Added new torrent to in-memory cache:", info_hash)
	}
	return nil
}

// Create a new torrent, fetching info from redis if exist data
// exists.
func (t *Tracker) FetchTorrent(r redis.Conn, info_hash string) *Torrent {
	// Make new struct to use for cache
	torrent := &Torrent{
		InfoHash: info_hash,
		Name:     "",
		Enabled:  true,
		Peers:    []*Peer{},
		MultiUp:  1.0,
		MultiDn:  1.0,
	}
	key := fmt.Sprintf("t:t:%s", info_hash)
	exists_reply, err := r.Do("EXISTS", key)
	exists, err := redis.Bool(exists_reply, err)
	if err != nil {
		exists = false
	}
	if exists {
		torrent_reply, err := r.Do("HGETALL", key)
		if err != nil {
			log.Println(fmt.Sprintf("FetchTorrent: Failed to get torrent from redis [%s]", key), err)
			return nil
		}

		values, err := redis.Values(torrent_reply, nil)
		if err != nil {
			log.Println("FetchTorrent: Failed to parse torrent reply: ", err)
			return nil
		}

		err = redis.ScanStruct(values, torrent)
		if err != nil {
			log.Println("FetchTorrent: Torrent scanstruct failure", err)
			return nil
		}

		if torrent.TorrentID == 0 {
			util.Debug("FetchTorrent: Trying to fetch info hash without valid key:", info_hash)
			r.Do("DEL", fmt.Sprintf("t:t:%s", torrent.InfoHash))
			return nil
		}
	}
	// Reset counts since we cant guarantee the accuracy after restart
	// TODO allow reloading of peer/seed counts if a maximum time has not elapsed
	// since the last startup.
	torrent.Seeders = 0
	torrent.Leechers = 0

	// Make these once and save the results in mem
	torrent.TorrentKey = fmt.Sprintf("t:t:%s", info_hash)
	torrent.TorrentPeersKey = fmt.Sprintf("t:tpeers:%s", info_hash)

	return torrent
}

// Fetch the client whitelist from redis and load it into memory
func (t *Tracker) initWhitelist(r redis.Conn) {
	t.Whitelist = []string{}
	a, err := r.Do("HKEYS", "t:whitelist")

	if err != nil {
		log.Println("initWhitelist: Failed to fetch whitelist", err)
		return
	}
	t.Whitelist, err = redis.Strings(a, nil)
	log.Println(fmt.Sprintf("initWhitelist: Loaded %d whitelist clients", len(t.Whitelist)))
}

// Fetch the torrents stored in redis and load them into active memory as models
func (t *Tracker) initTorrents(r redis.Conn) {
	torrent_keys_reply, err := r.Do("KEYS", "t:t:*")
	if err != nil {
		log.Println("initTorrents: Failed to get torrent from redis", err)
		return
	}
	torrent_keys, err := redis.Strings(torrent_keys_reply, nil)
	if err != nil {
		log.Println("initTorrents: Failed to parse torrent keys reply: ", err)
		return
	}
	torrents := 0

	for _, torrent_key := range torrent_keys {
		pcs := strings.SplitN(torrent_key, ":", 3)
		// Skip malformed keys and peer suffixed keys
		if len(pcs) != 3 || len(pcs[2]) != 40 {
			continue
		}
		torrent := t.FetchTorrent(r, pcs[2])
		if torrent != nil {
			t.TorrentsMutex.Lock()
			t.Torrents[torrent.InfoHash] = torrent
			t.TorrentsMutex.Unlock()
			torrents++
		} else {
			// Drop keys we don't have valid id's'for
			r.Do("DEL", torrent_key)
		}
	}

	log.Println(fmt.Sprintf("initTorrents: Loaded %d torrents into memory", torrents))
}

// Load all the users into memory
func (t *Tracker) initUsers(r redis.Conn) {
	user_keys_reply, err := r.Do("KEYS", "t:u:*")
	if err != nil {
		log.Println("initUsers: Failed to get torrent from redis", err)
		return
	}
	user_keys, err := redis.Strings(user_keys_reply, nil)
	if err != nil {
		log.Println("initUsers: Failed to parse peer reply: ", err)
		return
	}
	users := 0

	for _, user_key := range user_keys {
		pcs := strings.SplitN(user_key, ":", 3)
		if len(pcs) != 3 {
			continue
		}
		user_id, err := strconv.ParseUint(pcs[2], 10, 64)
		if err != nil {
			// Other key type probably
			continue
		}
		user := fetchUser(r, user_id)
		if user != nil {
			t.UsersMutex.Lock()
			t.Users[user_id] = user
			t.UsersMutex.Unlock()
			users++
		}
	}

	log.Println(fmt.Sprintf("initUsers: Loaded %d users into memory", users))
}

// This function will periodically update the torrent sort indexes
func (t *Tracker) dbStatIndexer() {
	log.Println("dbStatIndexer: Background indexer started")
	r := db.Pool.Get()
	defer r.Close()

	key_leechers := "t:i:leechers"
	key_seeders := "t:i:seeders"
	key_snatches := "t:i:snatches"

	count := 0

	leecher_args := []uint64{}
	seeder_args := []uint64{}
	snatch_args := []uint64{}

	for {
		time.Sleep(time.Duration(conf.Config.IndexInterval) * time.Second)
		t.TorrentsMutex.RLock()
		for _, torrent := range t.Torrents {
			t.TorrentsMutex.RLock()
			leecher_args = append(leecher_args, uint64(torrent.Leechers), torrent.TorrentID)
			seeder_args = append(seeder_args, uint64(torrent.Seeders), torrent.TorrentID)
			snatch_args = append(snatch_args, uint64(torrent.Snatches), torrent.TorrentID)
			t.TorrentsMutex.RUnlock()
			count++
		}
		t.TorrentsMutex.RUnlock()
		if count > 0 {
			r.Send("ZADD", key_leechers, leecher_args)
			r.Send("ZADD", key_seeders, seeder_args)
			r.Send("ZADD", key_snatches, snatch_args)
			r.Flush()
			leecher_args = leecher_args[:0]
			seeder_args = seeder_args[:0]
			snatch_args = snatch_args[:0]
		}
		count = 0
	}
}

// Handle writing out new data to the redis db in a queued manner
// Only items with the .InQueue flag set to false should be added.
// TODO channel as param
func (t *Tracker) syncWriter() {
	r := db.Pool.Get()
	defer r.Close()
	if r.Err() != nil {
		util.CaptureMessage(r.Err().Error())
		log.Println("SyncWriter: Failed to get redis conn:", r.Err().Error())
		return
	}
	for {
		select {
		case payload := <-db.SyncPayloadC:
			util.Debug("Sync payload")
			r.Do(payload.Command, payload.Args...)
		case user := <-SyncUserC:
			util.Debug("sync user")
			user.Sync(r)
			user.Lock()
			user.InQueue = false
			user.Unlock()
		case torrent := <-SyncTorrentC:
			util.Debug("sync torrent")
			torrent.Sync(r)
			torrent.Lock()
			torrent.InQueue = false
			torrent.Unlock()
		case peer := <-SyncPeerC:
			util.Debug("sync peer")
			peer.Sync(r)
			peer.Lock()
			peer.InQueue = false
			peer.Unlock()
		}
		err := r.Flush()
		if err != nil {
			log.Println("syncWriter: Failed to flush connection:", err)
		}
	}
}
