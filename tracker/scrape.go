package tracker

import (
	"bytes"
	"errors"
	"git.totdev.in/totv/mika/db"
	"git.totdev.in/totv/mika/stats"
	log "github.com/Sirupsen/logrus"
	"github.com/chihaya/bencode"
	"github.com/gin-gonic/gin"
	"net/http"
)

type ScrapeRequest struct {
	InfoHashes []string
}

type ScrapeResponse struct {
	Files []bencode.Dict
}

// HandleScrape is the route handler for the /scrape requests
// /scrape?info_hash=f%5bs%de06%19%d3ET%cc%81%bd%e5%0dZ%84%7f%f3%da
func (t *Tracker) HandleScrape(c *gin.Context) {
	stats.Counter <- stats.EV_SCRAPE
	r := db.Pool.Get()
	defer r.Close()
	if r.Err() != nil {
		stats.Counter <- stats.EV_SCRAPE_FAIL
		c.Error(r.Err()).SetMeta(errMeta(
			MSG_GENERIC_ERROR,
			"Internal error :(",
			log.Fields{"fn": "HandleScrape"},
			log.ErrorLevel,
		))
		return
	}

	passkey := c.Param("passkey")

	user_id := t.findUserID(passkey)

	if user_id == 0 {
		stats.Counter <- stats.EV_INVALID_PASSKEY
		c.Error(errors.New("Invalid torrent")).SetMeta(errMeta(
			MSG_GENERIC_ERROR,
			"Invalid passkey",
			log.Fields{"fn": "HandleScrape"},
			log.ErrorLevel,
		))
		return
	}
	user := t.FindUserByID(user_id)
	if !user.CanLeech {
		c.Error(errors.New("Leech not allowed")).SetMeta(errMeta(
			MSG_GENERIC_ERROR,
			"Leech not allowed",
			log.Fields{"fn": "HandleScrape"},
			log.DebugLevel,
		))
		return
	}
	if !user.Enabled {
		c.Error(errors.New("User disabled")).SetMeta(errMeta(
			MSG_GENERIC_ERROR,
			"User disabled",
			log.Fields{"fn": "HandleScrape"},
			log.DebugLevel,
		))
		return
	}
	q, err := QueryStringParser(c.Request.RequestURI)
	if err != nil {
		stats.Counter <- stats.EV_SCRAPE_FAIL
		c.Error(err).SetMeta(errMeta(
			MSG_GENERIC_ERROR,
			"Could not parse request",
			log.Fields{"fn": "HandleScrape"},
			log.ErrorLevel,
		))
		return
	}

	// Todo limit scrape to N torrents
	resp := make(bencode.Dict, len(q.InfoHashes))

	for _, info_hash := range q.InfoHashes {
		torrent := t.FindTorrentByInfoHash(info_hash)
		if torrent != nil {
			resp[info_hash] = bencode.Dict{
				"complete":   torrent.Seeders,
				"downloaded": torrent.Snatches,
				"incomplete": torrent.Leechers,
			}
		} else {
			log.Debug("Unknown hash:", info_hash)
		}
	}

	var out_bytes bytes.Buffer
	encoder := bencode.NewEncoder(&out_bytes)
	err = encoder.Encode(resp)
	if err != nil {
		stats.Counter <- stats.EV_SCRAPE_FAIL
		c.Error(err).SetMeta(errMeta(
			MSG_GENERIC_ERROR,
			"Failed to encode scrape response",
			log.Fields{"fn": "HandleScrape"},
			log.ErrorLevel,
		))
		return
	}
	encoded := out_bytes.String()
	log.Debug(encoded)
	c.String(http.StatusOK, encoded)
}
