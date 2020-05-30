package tracker

import (
	"bytes"
	"github.com/chihaya/bencode"
	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/mika/store"
	log "github.com/sirupsen/logrus"
	"net/http"
)

// scrape handles the bittorrent scrape protocol for
func (h *BitTorrentHandler) scrape(c *gin.Context) {
	var user store.User
	if !preFlightChecks(&user, c.Param("passkey"), c, h.tracker) {
		return
	}
	q, err := queryStringParser(c.Request.URL.RawQuery)
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
	var ih store.InfoHash
	for _, ihStr := range q.InfoHashes {
		if err := store.InfoHashFromString(&ih, ihStr); err != nil {
			log.Errorf("Failed to decode info hash in scrape: %s", ihStr)
			continue
		}
		var torrent store.Torrent
		if err := h.tracker.Torrents.Get(&torrent, ih, false); err != nil {
			log.Debugf("Scrape request for invalid torrent: %s", ih)
			continue
		}
		resp[ih.String()] = bencode.Dict{
			"complete":   torrent.Seeders,
			"downloaded": torrent.TotalCompleted,
			"incomplete": torrent.Leechers,
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
