package tracker

import (
	"bytes"
	"git.totdev.in/totv/echo.git"
	"git.totdev.in/totv/mika/db"
	"git.totdev.in/totv/mika/stats"
	"git.totdev.in/totv/mika/util"
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

// Route handler for the /scrape requests
// /scrape?info_hash=f%5bs%de06%19%d3ET%cc%81%bd%e5%0dZ%84%7f%f3%da
func (t *Tracker) HandleScrape(c *echo.Context) *echo.HTTPError {
	stats.Counter <- stats.EV_SCRAPE
	r := db.Pool.Get()
	defer r.Close()
	if r.Err() != nil {
		util.CaptureMessage(r.Err().Error())
		log.Println("HandleScrape: Cannot connect to redis", r.Err().Error())
		oops(c, MSG_GENERIC_ERROR)
		stats.Counter <- stats.EV_SCRAPE_FAIL
		return nil
	}

	passkey := c.P(0)

	user_id := t.findUserID(passkey)

	if user_id == 0 {
		log.Println("HandleScrape: Invalid passkey supplied:", passkey)
		oops(c, MSG_GENERIC_ERROR)
		stats.Counter <- stats.EV_INVALID_PASSKEY
		return nil
	}
	user := t.FindUserByID(user_id)
	if !user.CanLeech {
		oopsStr(c, MSG_GENERIC_ERROR, "Leech not allowed")
		return nil
	}
	if !user.Enabled {
		oopsStr(c, MSG_INVALID_INFO_HASH, "User disabled")
		return nil
	}
	q, err := QueryStringParser(c.Request.RequestURI)
	if err != nil {
		util.CaptureMessage(err.Error())
		log.Println("HandleScrape: Failed to parse scrape qs:", err)
		oops(c, MSG_GENERIC_ERROR)
		stats.Counter <- stats.EV_SCRAPE_FAIL
		return nil
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
		util.CaptureMessage(err.Error())
		log.Println("HandleScrape: Failed to encode scrape response:", err)
		oops(c, MSG_GENERIC_ERROR)
		stats.Counter <- stats.EV_SCRAPE_FAIL
		return nil
	}
	encoded := out_bytes.String()
	log.Debug(encoded)
	c.String(http.StatusOK, encoded)
	return nil
}
