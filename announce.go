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

const (
	STOPPED   = iota
	STARTED   = iota
	COMPLETED = iota
	ANNOUNCE  = iota
)

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

	var torrent_id = mika.GetTorrentID(r, ann.InfoHash)
	if torrent_id <= 0 {
		oops(c, MSG_INFO_HASH_NOT_FOUND)
		return
	}
	torrent, err := mika.GetTorrent(r, torrent_id)
	if err != nil {
		oops(c, MSG_GENERIC_ERROR)
		return
	}

	Debug("TorrentID: ", torrent_id)

	peer, err := torrent.GetPeer(r, ann.PeerID)
	if err != nil {
		oops(c, MSG_GENERIC_ERROR)
		return
	}

	peer.Update(ann)
	peer.UserID = user_id //where to put this?

	torrent.Update(ann)

	if ann.Event == STOPPED {
		torrent.DelPeer(r, &peer)
	} else {
		torrent.AddPeer(r, &peer)
	}
	peers := torrent.GetPeers(r, ann.NumWant)

	// Define our keys
	users_active_key := fmt.Sprintf("t:u:%d:active", peer.UserID)
	users_incomplete_key := fmt.Sprintf("t:u:%d:incomplete", peer.UserID)
	users_complete_key := fmt.Sprintf("t:u:%d:complete", peer.UserID)
	users_hnr_key := fmt.Sprintf("t:u:%d:hnr", peer.UserID)

	// pipe.hset(peer_key, "completed", 0) ?"stopped"?

	if ann.Event == STOPPED {
		// Remove from torrents active peer set
		r.Send("SREM", torrent.TorrentPeersKey, ann.PeerID)

		r.Send("SREM", users_active_key, torrent_id)

		// Mark the peer as inactive
		r.Send("HSET", peer.KeyPeer, "active", 0)

		// Handle total changes if we were previously an active peer?

		if peer.Left > 0 {
			Debug("Torrent Leechers -1")
			torrent.Leechers--
		} else {
			Debug("Torrent Seeders  +1")
			torrent.Seeders--
		}

	} else if ann.Event == COMPLETED {
		if peer.Active {
			// If the user was previously an active peer and has data left
			// we assume he was leeching so we decrement it now
			torrent.Leechers--
			Debug("Torrent Leechers -1")

		}
		// Should we disallow peers being able to trigger this twice?
		// Forcing only 1 for now

		// Increment active seeders for the torrent
		torrent.Seeders++
		Debug("Torrent Seeders  +1")

		// Remove the torrent from the users incomplete set
		r.Send("SREM", users_incomplete_key, torrent_id)

		// Remove the torrent from the users incomplete set
		r.Send("SADD", users_complete_key, torrent_id)

		// Remove from the users hnr list if it exists
		r.Send("SREM", users_hnr_key, torrent_id)

	} else if ann.Event == STARTED {

		if ann.Left > 0 {
			// Add the torrent to the users incomplete set
			r.Send("SREM", users_incomplete_key, torrent_id)

			torrent.Leechers++
			Debug("Torrent Leechers +1")
		} else {
			torrent.Seeders++
			Debug("Torrent Seeders  +1")
		}
	}
	if ann.Event != STOPPED {
		// Add peer to torrent active peers
		r.Send("SADD", torrent.TorrentPeersKey, ann.PeerID)

		// Add to users active torrent set
		r.Send("SADD", users_active_key, torrent_id)

		// Refresh the peers expiration timer
		// If this expires, the peer reaper takes over and removes the
		// peer from torrents in the case of a non-clean client shutdown
		r.Send("SETEX", fmt.Sprintf("t:t:%d:%s:exp", torrent_id, ann.PeerID), config.ReapInterval, 1)
	}

	peer.AnnounceLast = unixtime()
	peer.Sync(r)
	torrent.Sync(r)
	r.Flush()

	dict := bencode.Dict{
		"complete":     torrent.Seeders,
		"incomplete":   torrent.Leechers,
		"interval":     config.AnnInterval,
		"min interval": config.AnnIntervalMin,
		"peers":        makeCompactPeers(peers, ann.PeerID),
	}

	var out_bytes bytes.Buffer
	encoder := bencode.NewEncoder(&out_bytes)

	er_msg_encoded := encoder.Encode(dict)
	if er_msg_encoded != nil {
		oops(c, MSG_GENERIC_ERROR)
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

	s := strings.Split(c.Request.RemoteAddr, ":")
	ip_req, _ := s[0], s[1]

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
