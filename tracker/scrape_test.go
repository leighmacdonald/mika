package tracker

import (
	"fmt"
	"github.com/chihaya/bencode"
	"github.com/leighmacdonald/mika/store"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestBitTorrentHandler_Scrape(t *testing.T) {
	rh := NewBitTorrentHandler()
	scrapes := []sr{{
		req: scrapeReq{
			PK:         testUsers[0].Passkey,
			InfoHashes: []store.InfoHash{testTorrents[0].InfoHash}},
		exp: scrapeExpect{status: msgOk}},
	}

	for i, a := range scrapes {
		u := fmt.Sprintf("/scrape/%s?%s", a.req.PK, a.req.ToValues().Encode())
		w := performRequest(rh, "GET", u, nil, nil)
		require.EqualValues(t, a.exp.status, errCode(w.Code),
			fmt.Sprintf("%s (%d)", responseStringMap[errCode(w.Code)], i))

		v, err := bencode.NewDecoder(w.Body).Decode()
		require.NoError(t, err, "Failed to decode scrape: (%d)", i)
		d := v.(bencode.Dict)
		require.Equal(t, int64(1), d[testTorrents[0].InfoHash.String()].(bencode.Dict)["complete"].(int64))
		require.Equal(t, int64(1), d[testTorrents[0].InfoHash.String()].(bencode.Dict)["incomplete"].(int64))
		require.Equal(t, int64(2), d[testTorrents[0].InfoHash.String()].(bencode.Dict)["downloaded"].(int64))
	}
}
