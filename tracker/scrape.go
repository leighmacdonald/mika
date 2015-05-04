package tracker

import (
	"bytes"
	"git.totdev.in/totv/mika/db"
	"git.totdev.in/totv/mika/stats"
	"git.totdev.in/totv/mika/util"
	"github.com/chihaya/bencode"
	"github.com/labstack/echo"
	"log"
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
func (t *Tracker) HandleScrape(c *echo.Context) {
	stats.Counter <- stats.EV_SCRAPE
	r := db.Pool.Get()
	defer r.Close()
	if r.Err() != nil {
		util.CaptureMessage(r.Err().Error())
		log.Println("HandleScrape: Cannot connect to redis", r.Err().Error())
		oops(c, MSG_GENERIC_ERROR)
		stats.Counter <- stats.EV_SCRAPE_FAIL
		return
	}

	passkey := c.Param("passkey")

	user := t.GetUserByPasskey(r, passkey)
	if user == nil {
		log.Println("HandleScrape: Invalid passkey supplied:", passkey)
		oops(c, MSG_GENERIC_ERROR)
		stats.Counter <- stats.EV_INVALID_PASSKEY
		return
	}

	q, err := QueryStringParser(c.Request.RequestURI)
	if err != nil {
		util.CaptureMessage(err.Error())
		log.Println("HandleScrape: Failed to parse scrape qs:", err)
		oops(c, MSG_GENERIC_ERROR)
		stats.Counter <- stats.EV_SCRAPE_FAIL
		return
	}

	// Todo limit scrape to N torrents
	resp := make(bencode.Dict, len(q.InfoHashes))

	for _, info_hash := range q.InfoHashes {
		torrent := t.GetTorrentByInfoHash(r, info_hash, false)
		if torrent != nil {
			resp[info_hash] = bencode.Dict{
				"complete":   torrent.Seeders,
				"downloaded": torrent.Snatches,
				"incomplete": torrent.Leechers,
			}
		} else {
			util.Debug("Unknown hash:", info_hash)
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
		return
	}
	encoded := out_bytes.String()
	util.Debug(encoded)
	c.String(http.StatusOK, encoded)
}
