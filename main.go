package main

import (
    "net/http"
    "flag"
    "runtime"
    "time"
    "net"
    "bytes"
    "log"
    "github.com/labstack/echo"
    mw "github.com/labstack/echo/middleware"
    "github.com/thoas/stats"
    "github.com/garyburd/redigo/redis"
    "github.com/jackpal/bencode-go"
    "strings"
    "strconv"
    "errors"
    "net/url"
)

type (

    ScrapeResponse struct {}

    ErrorResponse struct {
        FailReason string `bencode:"failure reason"`
    }

    AnnounceResponse struct {
        FailReason  string `bencode:"failure reason"`
        WarnReason  string `bencode:"warning message"`
        MinInterval int    `bencode:"min interval"`
        Complete    int    `bencode:"complete"`
        Incomplete  int    `bencode:"incomplete"`
        Interval    int    `bencode:"interval"`
        Peers       []int  `bencode:"peers"`
    }

    // Peers
    Peer struct {
        PeerID       string
        Uploaded     uint64
        Downloaded   uint64
        IP           net.IP
        Port         uint64
        Left         uint64
        state        string
        last_request time.Time
    }

    AnnounceRequest struct {
        Compact    bool   `json:"compact"`
        Downloaded uint64 `json:"downloaded"`
        Event      string `json:"event"`
        IPv4       net.IP `json:"ipv4"`
        Infohash   string `json:"infohash"`
        Left       uint64 `json:"left"`
        NumWant    int    `json:"numwant"`
        Passkey    string `json:"passkey"`
        PeerID     string `json:"peer_id"`
        Port       uint64 `json:"port"`
        Uploaded   uint64 `json:"uploaded"`
    }

    ScrapeRequest struct {
        Passkey    string
        Infohashes []string
    }

    Query struct {
        Infohashes []string
        Params     map[string]string
    }
)

var (
    pool *redis.Pool

    listen_host = flag.String("listen", ":34000", "Host/port to bind to")
    redis_host = flag.String("redis", "localhost:6379", "Redis endpoint")
    redis_pass = flag.String("rpass", "", "Redis pasword")
    max_idle = flag.Int("max_idle", 50, "Max idle redis connections")
    num_procs = flag.Int("procs", runtime.NumCPU(), "Number of CPU cores to use (default: all)")
    debug = flag.Bool("debug", false, "Enable debugging output")
)


// New parses a raw url query.
func QueryStringParser(query string) (*Query, error) {
    var (
        keyStart, keyEnd int
        valStart, valEnd int
        firstInfohash    string

        onKey       = true
        hasInfohash = false

        q = &Query{
            Infohashes: nil,
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
                if hasInfohash {
                    // Multiple infohashes
                    if q.Infohashes == nil {
                        q.Infohashes = []string{firstInfohash}
                    }
                    q.Infohashes = append(q.Infohashes, valStr)
                } else {
                    firstInfohash = valStr
                    hasInfohash = true
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

func UMax(a, b uint64) uint64 {
    if a > b {
        return a
    }
    return b
}

func NewAnnounce(c *echo.Context) (*AnnounceRequest, error) {
    log.Println(c.Request.RequestURI)
    q, err := QueryStringParser(c.Request.RequestURI)
    if err != nil {
        return nil, err
    }

    s := strings.Split(c.Request.RemoteAddr, ":")
    ip_req, _ := s[0], s[1]

    compact := q.Params["compact"] != "0"
    event, _ := q.Params["event"]

    numWant := getNumWant(q, 30)

    infohash, exists := q.Params["info_hash"]
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
        left = UMax(0,  left)
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

    return &AnnounceRequest{
        Compact:    compact,
        Downloaded: downloaded,
        Event:      event,
        IPv4:       ipv4,
        Infohash:   infohash,
        Left:       left,
        NumWant:    numWant,
        PeerID:     peerID,
        Port:       port,
        Uploaded:   uploaded,
    }, nil
}

func getIP(ip_str string) (net.IP, error) {

    ip := net.ParseIP(ip_str)
    if ip != nil {
        return ip.To4(), nil
    }

    return nil, errors.New("Failed to parse ip")
}

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

func newPool(server, password string, max_idle int) *redis.Pool {
    return &redis.Pool{
        MaxIdle: max_idle,
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

func responseError(message string) string {
    var out_bytes bytes.Buffer;
    var er_msg = ErrorResponse{FailReason: message}
    er_msg_encoded := bencode.Marshal(&out_bytes, er_msg)
    if er_msg_encoded != nil {
        log.Println(er_msg_encoded)
        return "error"
    }
    return out_bytes.String()
}

func authenticate(r *redis.Conn, passkey string) bool {

    log.Println(passkey)
    return true
}

func handleAnnounce(c *echo.Context) {

    ann, err := NewAnnounce(c)

    if err != nil {
        log.Fatalln(err)
        c.String(http.StatusInternalServerError, responseError("Internal oopsie"))
        return
    }
    log.Println(ann.Infohash)

    r := pool.Get()
    defer r.Close()

    passkey := c.Param("passkey")
    if !authenticate(&r, passkey) {
        return
    }

    var ih = c.Request.RequestURI
    log.Println(ih)
    if ih != "" {
        info_hash := strings.Fields(ih)
        log.Println(info_hash)
    } else {
        c.String(http.StatusUnauthorized, responseError("Invalid infohash"))
        return
    }



    resp := responseError("hello!")
    log.Println(resp)
    c.String(http.StatusOK, resp)
}

func handleScrape(c *echo.Context) {
    c.String(http.StatusOK, "I like to scrape my ass")
}



func main() {
    // Parse CLI args
    flag.Parse()

    // Set max number of CPU cores to use
    log.Println("Num procs(s):", *num_procs)
    runtime.GOMAXPROCS(*num_procs)

    // Initialize the redis pool
    pool = newPool(*redis_host, *redis_pass, *max_idle)


    // Initialize the router + middlewares
    e := echo.New()
    e.MaxParam(1);

    if *debug {
        e.Use(mw.Logger)
    }

    // Third-party middleware
    s := stats.New()
    e.Use(s.Handler)
    // Route
    e.Get("/stats", func(c *echo.Context) {
        c.JSON(200, s.Data())
    })

    //************//
    //   Routes   //
    //************//
    e.Get("/:passkey/announce", handleAnnounce)
    e.Get("/:passkey/scrape", handleScrape)

    // Start server
    e.Run(*listen_host)
}

//func init() {
//    users = map[string]user{
//        "1": user{
//            ID:   "1",
//            Name: "Wreck-It Ralph",
//        },
//    }
//}