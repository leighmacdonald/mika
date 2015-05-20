package tracker

import (
	"bytes"
	"git.totdev.in/totv/echo.git"
	"git.totdev.in/totv/mika/db"
	"git.totdev.in/totv/mika/stats"
	log "github.com/Sirupsen/logrus"
	"github.com/chihaya/bencode"
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
func (t *Tracker) HandleScrape(c *echo.Context) *echo.HTTPError {
	stats.Counter <- stats.EV_SCRAPE
	r := db.Pool.Get()
	defer r.Close()
	if r.Err() != nil {
		stats.Counter <- stats.EV_SCRAPE_FAIL
		return &echo.HTTPError{
			Code:    MSG_GENERIC_ERROR,
			Fields:  log.Fields{"fn": "HandleScrape"},
			Message: "Internal error :(",
			Error:   r.Err(),
		}
	}

	passkey := c.P(0)

	user_id := t.findUserID(passkey)

	if user_id == 0 {
		stats.Counter <- stats.EV_INVALID_PASSKEY
		return &echo.HTTPError{
			Code:    MSG_GENERIC_ERROR,
			Fields:  log.Fields{"fn": "HandleScrape"},
			Message: "Invalid passkey",
		}
	}
	user := t.FindUserByID(user_id)
	if !user.CanLeech {
		return &echo.HTTPError{
			Code:    MSG_GENERIC_ERROR,
			Fields:  log.Fields{"fn": "HandleScrape"},
			Message: "Leech not allowed",
			Level:   log.DebugLevel,
		}
	}
	if !user.Enabled {
		return &echo.HTTPError{
			Code:    MSG_GENERIC_ERROR,
			Fields:  log.Fields{"fn": "HandleScrape"},
			Message: "User disabled",
			Level:   log.DebugLevel,
		}
	}
	q, err := QueryStringParser(c.Request.RequestURI)
	if err != nil {
		stats.Counter <- stats.EV_SCRAPE_FAIL
		return &echo.HTTPError{
			Code:    MSG_GENERIC_ERROR,
			Fields:  log.Fields{"fn": "HandleScrape"},
			Message: "Could not parse request",
			Level:   log.ErrorLevel,
			Error:   err,
		}
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
		return &echo.HTTPError{
			Code:    MSG_GENERIC_ERROR,
			Fields:  log.Fields{"fn": "HandleScrape"},
			Message: "Failed to encode scrape response",
			Level:   log.ErrorLevel,
			Error:   err,
		}
	}
	encoded := out_bytes.String()
	log.Debug(encoded)
	return c.String(http.StatusOK, encoded)
}
