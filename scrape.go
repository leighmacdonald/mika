package main

import (
	"bytes"
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
func HandleScrape(c *echo.Context) {
	r := pool.Get()
	defer r.Close()
	Debug("Got scrape")
	Debug(c.Request.RequestURI)

	q, err := QueryStringParser(c.Request.RequestURI)
	if err != nil {
		log.Println(err)
		oops(c, MSG_GENERIC_ERROR)
		return
	}

	// Todo limit scrape to N torrents

	log.Println("Infohashes in scrape:", q.InfoHashes)

	resp := make(bencode.Dict, len(q.InfoHashes))

	for _, info_hash := range q.InfoHashes {
		torrent := mika.GetTorrentByInfoHash(r, info_hash)
		if torrent != nil {
			resp[info_hash] = bencode.Dict{
				"complete":   torrent.Seeders,
				"downloaded": torrent.Snatches,
				"incomplete": torrent.Leechers,
			}
		} else {
			Debug("Unknown hash:", info_hash)
		}
	}

	var out_bytes bytes.Buffer
	encoder := bencode.NewEncoder(&out_bytes)
	err = encoder.Encode(resp)
	if err != nil {
		log.Println("Failedto encode scrape response:", err)
		oops(c, MSG_GENERIC_ERROR)
	}
	encoded := out_bytes.String()
	log.Println(encoded)
	c.String(http.StatusOK, encoded)
}
