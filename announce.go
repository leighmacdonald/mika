package main

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/chihaya/bencode"
	"github.com/labstack/echo"
	"log"
	"net"
	"net/http"
	"strings"
)

// Announce types
const (
	STOPPED   = iota
	STARTED   = iota
	COMPLETED = iota
	ANNOUNCE  = iota
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

// Parse and return a IP from a string
func getIP(ip_str string) (net.IP, error) {
	ip := net.ParseIP(ip_str)
	if ip != nil {
		return ip.To4(), nil
	}
	return nil, errors.New("Failed to parse ip")
}

// Route handler for the /announce endpoint
// Here be dragons
func HandleAnnounce(c *echo.Context) {
	counter <- EV_ANNOUNCE
	r := pool.Get()
	defer r.Close()
	if r.Err() != nil {
		log.Println("HandleAnnounce: Failed to get redis conn:", r.Err().Error())
		oops(c, MSG_GENERIC_ERROR)
		counter <- EV_ANNOUNCE_FAIL
		return
	}

	ann, err := NewAnnounce(c)
	if err != nil {
		log.Println("HandleAnnounce: Failed to parse announce:", err)
		oops(c, MSG_GENERIC_ERROR)
		counter <- EV_ANNOUNCE_FAIL
		return
	}

	passkey := c.P(0) // eat a dick

	user := GetUserByPasskey(r, passkey)
	if user == nil {
		log.Println("HandleAnnounce: Invalid passkey", passkey)
		oopsStr(c, MSG_GENERIC_ERROR, "Invalid passkey")
		counter <- EV_INVALID_PASSKEY
		return
	}
	if !user.CanLeech && ann.Left > 0 {
		oopsStr(c, MSG_GENERIC_ERROR, "Leech Disabled")
		return
	}
	if !IsValidClient(r, ann.PeerID) {
		log.Println("HandleAnnounce:", fmt.Sprintf("Invalid Client %s [%d/%s]", ann.PeerID[0:6], user.UserID, user.Username))
		oops(c, MSG_INVALID_PEER_ID)
		counter <- EV_INVALID_CLIENT
		return
	}

	torrent := mika.GetTorrentByInfoHash(r, fmt.Sprintf("%x", ann.InfoHash), true)
	if torrent == nil {
		log.Println("HandleAnnounce:", fmt.Sprintf("Torrent not found: %x [%d/%s]", ann.InfoHash, user.UserID, user.Username))
		oops(c, MSG_INFO_HASH_NOT_FOUND)
		counter <- EV_INVALID_INFOHASH
		return
	} else if !torrent.Enabled {
		Debug("HandleAnnounce:", fmt.Sprintf("Disabled torrent: %x [%d/%s]", ann.InfoHash, user.UserID, user.Username))
		oopsStr(c, MSG_INFO_HASH_NOT_FOUND, torrent.DelReason())
		counter <- EV_INVALID_INFOHASH
		return
	}
	peer, err := torrent.GetPeer(r, ann.PeerID)
	if err != nil {
		log.Println("HandleAnnounce: Failed to fetch/create peer:", err.Error())
		oops(c, MSG_GENERIC_ERROR)
		counter <- EV_ANNOUNCE_FAIL
		return
	}
	peer.SetUserID(user.UserID, user.Username) //where to put this/handle this cleaner?

	// user update MUST happen after peer update since we rely on the old dl/ul values
	ul, dl := peer.Update(ann)
	torrent.Update(ann)
	user.Update(ann, ul, dl, torrent.MultiUp, torrent.MultiDn)

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
		if peer.IsHNR() {
			peer.AddHNR(r, torrent.TorrentID)
		}
	} else if ann.Event == COMPLETED {

		// Remove the torrent from the users incomplete set
		r.Send("SREM", user.KeyIncomplete, torrent.TorrentID)

		// Remove the torrent from the users incomplete set
		r.Send("SADD", user.KeyComplete, torrent.TorrentID)

		// Remove from the users hnr list if it exists
		r.Send("SREM", user.KeyHNR, torrent.TorrentID)

	} else if ann.Event == STARTED {
		// Ignore start event from active peers to prevent stat skew potential
		if !peer.IsSeeder() {
			r.Send("SADD", user.KeyIncomplete, torrent.TorrentID)
		}
	}

	if ann.Event != STOPPED {

		peer.Active = true

		// Add peer to torrent active peers
		r.Send("SADD", torrent.TorrentPeersKey, ann.PeerID)

		// Add to users active torrent set
		r.Send("SADD", user.KeyActive, torrent.TorrentID)

		// Refresh the peers expiration timer
		// If this expires, the peer reaper takes over and removes the
		// peer from torrents in the case of a non-clean client shutdown
		r.Send("SETEX", fmt.Sprintf("t:t:%s:%s:exp", torrent.InfoHash, ann.PeerID), config.ReapInterval, 1)
	}
	r.Flush()

	peer.AnnounceLast = unixtime()
	if !peer.InQueue {
		peer.InQueue = true
		sync_peer <- peer
	}
	if !torrent.InQueue {
		torrent.InQueue = true
		sync_torrent <- torrent
	}
	if !user.InQueue {
		user.InQueue = true
		sync_user <- user
	}

	dict := bencode.Dict{
		"complete":     torrent.Seeders,
		"incomplete":   torrent.Leechers,
		"interval":     config.AnnInterval,
		"min interval": config.AnnIntervalMin,
	}

	peers := torrent.GetPeers(r, ann.NumWant)
	if peers != nil {
		dict["peers"] = makeCompactPeers(peers, ann.PeerID)
	} else {
		dict["peers"] = []byte{}
	}
	var out_bytes bytes.Buffer
	encoder := bencode.NewEncoder(&out_bytes)

	er_msg_encoded := encoder.Encode(dict)
	if er_msg_encoded != nil {
		log.Println("HandleAnnounce:", fmt.Sprintf("Failed to encode response %s [%d/%s]", ann.InfoHash, user.UserID, user.Username))
		oops(c, MSG_GENERIC_ERROR)
		counter <- EV_ANNOUNCE_FAIL
		return
	}

	c.String(http.StatusOK, out_bytes.String())

}

// Parse the query string into an AnnounceRequest struct
func NewAnnounce(c *echo.Context) (*AnnounceRequest, error) {
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
	case "complete":
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
				log.Println("NewAnnounce: Failed to parse header supplied IP", err)
				return nil, errors.New("Invalid ip header")
			}
			ipv4 = ipv4_new
		} else {
			s := strings.Split(c.Request.RemoteAddr, ":")
			ip_req, _ := s[0], s[1]
			ipv4_new, err := getIP(ip_req)
			if err != nil {
				log.Println("NewAnnounce: Failed to parse detected IP", err)
				return nil, errors.New("Invalid ip hash")
			}
			ipv4 = ipv4_new
		}
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
		downloaded = 0
	} else {
		downloaded = UMax(0, downloaded)
	}

	uploaded, err := q.Uint64("uploaded")
	if err != nil {
		uploaded = 0
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
