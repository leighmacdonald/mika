package http

import (
	"bytes"
	"github.com/chihaya/bencode"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"mika/model"
	"net/http"
)

type scrapeRequest struct {
	InfoHashes []string
}

type scrapeResponse struct {
	Files []bencode.Dict
}

// scrape handles the bittorrent scrape protocol for
func (h *BitTorrentHandler) scrape(c *gin.Context) {
	// Check that the user is valid before parsing anything
	pk := c.Param("passkey")
	if pk == "" {
		oops(c, msgInvalidAuth)
		return
	}
	usr, err := h.t.Users.GetUserByPasskey(pk)
	if err != nil || !usr.Valid() {
		oops(c, msgInvalidAuth)
		return
	}
	q, err := queryStringParser(c.Request.RequestURI)
	if err != nil {
		log.Errorf("Failed to parse request string")
		oops(c, msgMalformedRequest)
		return
	}
	// Technically no info hashes means we are supposed to send data for all known torrents.
	// This is something we do NOT want to do in a private tracker scenario (or really public for that matter)
	// TODO Add a config toggle for this?
	// TODO Its not technically malformed, should we return a empty file set instead?
	if len(q.InfoHashes) == 0 {
		log.Errorf("No infohash supplied")
		oops(c, msgMalformedRequest)
		return
	}
	// Todo limit scrape to N torrents
	resp := make(bencode.Dict, len(q.InfoHashes))
	for _, ihStr := range q.InfoHashes {
		ih := model.InfoHashFromString(ihStr)
		torrent, err := h.t.Torrents.GetTorrent(ih)
		if err != nil {
			log.Debugf("Scrape request for invalid torrent: %s", ih)
			continue
		}
		peers, err := h.t.Peers.GetPeers(ih, 100)
		if err != nil {
			log.Debugf("Failed to get peers for scrape: %s", ih)
			continue
		}
		seeders, leechers := peers.Counts()
		resp[ih.String()] = bencode.Dict{
			"complete":   seeders,
			"downloaded": torrent.TotalCompleted,
			"incomplete": leechers,
		}
	}
	var buf bytes.Buffer
	if err := bencode.NewEncoder(&buf).Encode(resp); err != nil {
		log.Errorf("Failed to encode scrape response")
		return
	}
	encoded := buf.String()
	log.Debug(encoded)
	c.String(http.StatusOK, encoded)
}
