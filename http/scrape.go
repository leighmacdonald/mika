package http

import (
	"github.com/chihaya/bencode"
	"github.com/gin-gonic/gin"
)

type scrapeRequest struct {
	InfoHashes []string
}

type scrapeResponse struct {
	Files []bencode.Dict
}

func (h *BitTorrentHandler) scrape(c *gin.Context) {
	//q, err := queryStringParser(c.Request.RequestURI)
	//if err != nil {
	//	return
	//}
	//
	//// Todo limit scrape to N torrents
	//resp := make(bencode.Dict, len(q.InfoHashes))
	//
	//for _, info_hash := range q.InfoHashes {
	//	torrent := h.t.FindTorrentByInfoHash(info_hash)
	//	if torrent != nil {
	//		resp[info_hash] = bencode.Dict{
	//			"complete":   torrent.Seeders,
	//			"downloaded": torrent.Snatches,
	//			"incomplete": torrent.Leechers,
	//		}
	//	} else {
	//		log.Debug("Unknown hash:", info_hash)
	//	}
	//}
	//
	//var out_bytes bytes.Buffer
	//encoder := bencode.NewEncoder(&out_bytes)
	//err = encoder.Encode(resp)
	//if err != nil {
	//	stats.RegisterEvent(stats.EV_SCRAPE_FAIL)
	//	ctx.Error(err).SetMeta(errMeta(
	//		MSG_GENERIC_ERROR,
	//		"Failed to encode scrape response",
	//		log.Fields{"fn": "HandleScrape"},
	//		log.ErrorLevel,
	//	))
	//	return
	//}
	//encoded := out_bytes.String()
	//log.Debug(encoded)
	//ctx.String(http.StatusOK, encoded)
}
