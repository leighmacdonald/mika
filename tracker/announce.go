package tracker

import (
	"bytes"
	"errors"
	"fmt"
	"git.totdev.in/totv/mika/conf"
	"git.totdev.in/totv/mika/db"
	"git.totdev.in/totv/mika/stats"
	"git.totdev.in/totv/mika/util"
	log "github.com/Sirupsen/logrus"
	"github.com/chihaya/bencode"
	"github.com/gin-gonic/gin"
	"net"
	"net/http"
	"strings"
)

// Announce types
const (
	STOPPED = iota
	STARTED
	COMPLETED
	ANNOUNCE
)

// Represents an announce received from the bittorrent client
type AnnounceRequest struct {
	Compact    bool
	Downloaded uint64
	Corrupt    uint64
	Event      int
	IPv4       net.IP
	InfoHash   string
	Left       uint64
	NumWant    int
	Passkey    string
	PeerID     string
	Port       uint64
	Uploaded   uint64
}

type AnnounceResponse struct {
	MinInterval int    `bencode:"min interval"`
	Complete    int    `bencode:"complete"`
	Incomplete  int    `bencode:"incomplete"`
	Interval    int    `bencode:"interval"`
	Peers       string `bencode:"peers"`
}

// getIP Parses and returns a IP from a string
func getIP(ip_str string) (net.IP, error) {
	ip := net.ParseIP(ip_str)
	if ip != nil {
		return ip.To4(), nil
	}
	return nil, errors.New("Failed to parse ip")
}

// HandleAnnounce is the handler for the /announce endpoint
// Here be dragons
func (t *Tracker) HandleAnnounce(c *gin.Context) {
	stats.Counter <- stats.EV_ANNOUNCE
	r := db.Pool.Get()
	defer r.Close()
	if r.Err() != nil {
		stats.Counter <- stats.EV_ANNOUNCE_FAIL
		c.Error(r.Err()).SetMeta(errMeta(
			MSG_GENERIC_ERROR,
			"Internal error, HALP",
			log.Fields{"fn": "HandleAnnounce"},
			log.ErrorLevel,
		))
		return
	}

	ann, err := NewAnnounce(c)
	if err != nil {
		stats.Counter <- stats.EV_ANNOUNCE_FAIL
		c.Error(err).SetMeta(errMeta(
			MSG_GENERIC_ERROR,
			"Failed to parse announce",
			log.Fields{
				"fn":        "HandleAnnounce",
				"remote_ip": c.Request.RemoteAddr,
				"uri":       c.Request.RequestURI,
			},
			log.ErrorLevel,
		))
		return
	}

	info_hash_hex := fmt.Sprintf("%x", ann.InfoHash)
	log.WithFields(log.Fields{
		"ih":    info_hash_hex[0:6],
		"ip":    ann.IPv4,
		"port":  ann.Port,
		"up":    util.Bytes(ann.Uploaded),
		"dn":    util.Bytes(ann.Downloaded),
		"left":  util.Bytes(ann.Left),
		"event": ann.Event,
	}).Debug("Announce event")

	passkey := c.Param("passkey") // eat a dick

	user_id := t.findUserID(passkey)

	if user_id == 0 {
		stats.Counter <- stats.EV_INVALID_PASSKEY
		c.Error(errors.New("Invalid passkey")).SetMeta(errMeta(
			MSG_GENERIC_ERROR,
			"Invalid passkey supplied",
			log.Fields{"fn": "HandleAnnounce", "passkey": passkey},
			log.ErrorLevel,
		))
		return
	}
	user := t.FindUserByID(user_id)
	if !user.CanLeech && ann.Left > 0 {
		c.Error(errors.New("Leech disabled for user")).SetMeta(errMeta(
			MSG_GENERIC_ERROR,
			"Leeching not allowed for user",
			log.Fields{"fn": "HandleAnnounce", "passkey": passkey},
			log.ErrorLevel,
		))
		return
	}
	if !user.Enabled {
		c.Error(errors.New("Disabled user")).SetMeta(errMeta(
			MSG_GENERIC_ERROR,
			"User disabled",
			log.Fields{"fn": "HandleAnnounce", "passkey": passkey},
			log.ErrorLevel,
		))
		return
	}

	if !t.IsValidClient(ann.PeerID) {
		stats.Counter <- stats.EV_INVALID_CLIENT
		c.Error(errors.New("Banned client")).SetMeta(errMeta(
			MSG_GENERIC_ERROR,
			"Banned client, check wiki for whitelisted clients",
			log.Fields{
				"fn":        "HandleAnnounce",
				"user_id":   user.UserID,
				"user_name": user.Username,
				"peer_id":   ann.PeerID[0:8],
			},
			log.ErrorLevel,
		))
		return
	}

	torrent := t.FindTorrentByInfoHash(info_hash_hex)
	if torrent == nil {
		stats.Counter <- stats.EV_INVALID_INFOHASH
		c.Error(errors.New("Invalid info hash")).SetMeta(errMeta(
			MSG_INFO_HASH_NOT_FOUND,
			"Torrent not found, try TPB",
			log.Fields{
				"fn":        "HandleAnnounce",
				"user_id":   user.UserID,
				"user_name": user.Username,
				"info_hash": info_hash_hex,
			},
			log.DebugLevel,
		))
	} else if !torrent.Enabled {
		stats.Counter <- stats.EV_INVALID_INFOHASH
		c.Error(errors.New("Torrent not enabled")).SetMeta(errMeta(
			MSG_INFO_HASH_NOT_FOUND,
			torrent.DelReason(),
			log.Fields{
				"fn":        "HandleAnnounce",
				"user_id":   user.UserID,
				"user_name": user.Username,
				"info_hash": info_hash_hex,
			},
			log.DebugLevel,
		))
	}
	peer := torrent.findPeer(ann.PeerID)
	if peer == nil {
		peer = NewPeer(ann.PeerID, ann.IPv4.String(), ann.Port, torrent, user)
		torrent.AddPeer(r, peer)
	}

	peer_diff := PeerDiff{User: user, Torrent: torrent}
	// user update MUST happen after peer update since we rely on the old dl/ul values
	peer.Update(ann, &peer_diff)
	torrent.Update(ann)
	user.Update(ann, &peer_diff, torrent.MultiUp, torrent.MultiDn)

	if ann.Event == STOPPED {
		torrent.DelPeer(r, peer)
	} else {
		if !torrent.HasPeer(peer) {
			torrent.AddPeer(r, peer)
		}
	}

	if ann.Event == STOPPED {
		// Remove from torrents active peer set
		r.Send("SREM", torrent.TorrentPeersKey, ann.PeerID)

		r.Send("SREM", user.KeyActive, torrent.TorrentID)

		// Mark the peer as inactive
		r.Send("HSET", peer.KeyPeer, "active", 0)

		r.Send("DEL", peer.KeyTimer)

		if peer.IsHNR() {
			user.AddHNR(r, torrent.TorrentID)
		}
	} else if ann.Event == COMPLETED {

		// Remove the torrent from the users incomplete set
		r.Send("SREM", user.KeyIncomplete, torrent.TorrentID)

		// Remove the torrent from the users incomplete set
		r.Send("SADD", user.KeyComplete, torrent.TorrentID)

		// Remove from the users hnr list if it exists
		r.Send("SREM", user.KeyHNR, torrent.TorrentID)

	} else if ann.Event == STARTED {
		// Make sure we account for a user completing a torrent outside of
		// our view, or resuming from previously completions
		if peer.IsSeeder() {
			r.Send("SREM", user.KeyHNR, torrent.TorrentID)
			r.Send("SREM", user.KeyIncomplete, torrent.TorrentID)
			r.Send("SADD", user.KeyComplete, torrent.TorrentID)
		} else {
			r.Send("SREM", user.KeyComplete, torrent.TorrentID)
			r.Send("SADD", user.KeyIncomplete, torrent.TorrentID)
		}
	}

	if ann.Event != STOPPED {

		// Add peer to torrent active peers
		r.Send("SADD", torrent.TorrentPeersKey, ann.PeerID)

		// Add to users active torrent set
		r.Send("SADD", user.KeyActive, torrent.TorrentID)

		// Refresh the peers expiration timer
		// If this expires, the peer reaper takes over and removes the
		// peer from torrents in the case of a non-clean client shutdown
		r.Send("SETEX", peer.KeyTimer, conf.Config.ReapInterval, 1)
	}
	r.Flush()

	if !torrent.InQueue() {
		torrent.SetInQueue(true)
		SyncEntityC <- torrent
	}
	if !user.InQueue() {
		user.SetInQueue(true)
		SyncEntityC <- user
	}

	dict := bencode.Dict{
		"complete":     torrent.Seeders,
		"incomplete":   torrent.Leechers,
		"interval":     conf.Config.AnnInterval,
		"min interval": conf.Config.AnnIntervalMin,
	}

	peers := torrent.GetPeers(r, ann.NumWant)
	if peers != nil {
		dict["peers"] = MakeCompactPeers(peers, ann.PeerID)
	} else {
		dict["peers"] = []byte{}
	}
	var out_bytes bytes.Buffer
	encoder := bencode.NewEncoder(&out_bytes)

	er_msg_encoded := encoder.Encode(dict)
	if er_msg_encoded != nil {
		stats.Counter <- stats.EV_ANNOUNCE_FAIL
		c.Error(er_msg_encoded).SetMeta(errMeta(
			MSG_GENERIC_ERROR,
			"Internal error",
			log.Fields{
				"fn":        "HandleAnnounce",
				"user_id":   user.UserID,
				"user_name": user.Username,
				"info_hash": info_hash_hex,
			},
			log.DebugLevel,
		))
		return
	}

	c.String(http.StatusOK, out_bytes.String())
}

// Parse the query string into an AnnounceRequest struct
func NewAnnounce(c *gin.Context) (*AnnounceRequest, error) {
	q, err := QueryStringParser(c.Request.RequestURI)
	if err != nil {
		return nil, err
	}

	compact := q.Params["compact"] != "0"

	event := ANNOUNCE
	event_name, _ := q.Params["event"]
	switch event_name {
	case "started":
		event = STARTED
	case "stopped":
		event = STOPPED
	case "completed":
		event = COMPLETED
	}

	numWant := getNumWant(q, 30)

	info_hash, exists := q.Params["info_hash"]
	if !exists {
		return nil, errors.New("Info hash not supplied")
	}

	peerID, exists := q.Params["peer_id"]
	if !exists {
		return nil, errors.New("Peer id not supplied")
	}

	ipv4, err := getIP(q.Params["ip"])
	if err != nil {
		// Look for forwarded ip in header then default to remote address
		forwarded_ip := c.Request.Header.Get("X-Forwarded-For")
		if forwarded_ip != "" {
			ipv4_new, err := getIP(forwarded_ip)
			if err != nil {
				log.Error("NewAnnounce: Failed to parse header supplied IP", err)
				return nil, errors.New("Invalid ip header")
			}
			ipv4 = ipv4_new
		} else {
			s := strings.Split(c.Request.RemoteAddr, ":")
			ip_req, _ := s[0], s[1]
			ipv4_new, err := getIP(ip_req)
			if err != nil {
				log.Error("NewAnnounce: Failed to parse detected IP", err)
				return nil, errors.New("Invalid ip hash")
			}
			ipv4 = ipv4_new
		}
	}

	port, err := q.Uint64("port")
	if err != nil || port < 1024 || port > 65535 {
		return nil, errors.New("Invalid port, must be between 1024 and 65535")
	}

	left, err := q.Uint64("left")
	if err != nil {
		return nil, errors.New("No left value")
	} else {
		left = util.UMax(0, left)
	}

	downloaded, err := q.Uint64("downloaded")
	if err != nil {
		downloaded = 0
	} else {
		downloaded = util.UMax(0, downloaded)
	}

	uploaded, err := q.Uint64("uploaded")
	if err != nil {
		uploaded = 0
	} else {
		uploaded = util.UMax(0, uploaded)
	}

	corrupt, err := q.Uint64("corrupt")
	if err != nil {
		// Assume we just don't have the param
		corrupt = 0
	} else {
		corrupt = util.UMax(0, corrupt)
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
