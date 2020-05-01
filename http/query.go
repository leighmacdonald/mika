package http

import (
	"mika/consts"
	"net/url"
	"strconv"
	"strings"
)

type announceParam string

const (
	paramInfoHash   announceParam = "info_hash"
	paramPeerID     announceParam = "peer_id"
	paramIP         announceParam = "ip"
	paramPort       announceParam = "port"
	paramLeft       announceParam = "left"
	paramDownloaded announceParam = "downloaded"
	paramUploaded   announceParam = "uploaded"
	paramCorrupt    announceParam = "corrupt"
	paramNumWant    announceParam = "numwant"
	paramCompact    announceParam = "compact"
)

type query struct {
	InfoHashes []string
	Params     map[announceParam]string
}

// queryStringParser transforms a raw url query into a Query struct
// This is used to avoid reflection calls used in the underlying gin.Bind functionality that
// would normally be used to parse the query params
func queryStringParser(qStr string) (*query, error) {
	var (
		keyStart, keyEnd int
		valStart, valEnd int
		firstInfoHash    string
		onKey            = true
		hasInfoHash      = false
		q                = &query{
			InfoHashes: nil,
			Params:     make(map[announceParam]string),
		}
	)

	for i, length := 0, len(qStr); i < length; i++ {
		separator := qStr[i] == '&' || qStr[i] == ';' || qStr[i] == '?'
		if separator || i == length-1 {
			if onKey {
				keyStart = i + 1
				continue
			}
			if i == length-1 && !separator {
				if qStr[i] == '=' {
					continue
				}
				valEnd = i
			}
			keyStr, err := url.QueryUnescape(qStr[keyStart : keyEnd+1])
			if err != nil {
				return nil, err
			}
			// The start can be greater than the end when the query contains an invalid
			// empty query value
			if valStart > valEnd {
				return nil, consts.ErrMalformedRequest
			}

			valStr, err := url.QueryUnescape(qStr[valStart : valEnd+1])
			if err != nil {
				return nil, err
			}
			q.Params[announceParam(strings.ToLower(keyStr))] = valStr

			if keyStr == "info_hash" {
				if hasInfoHash {
					// Multiple info hashes
					if q.InfoHashes == nil {
						q.InfoHashes = []string{firstInfoHash}
					}

					q.InfoHashes = append(q.InfoHashes, valStr)
				} else {
					firstInfoHash = valStr
					hasInfoHash = true
				}
			}
			onKey = true
			keyStart = i + 1
		} else if qStr[i] == '=' {
			onKey = false
			valStart = i + 1
		} else if onKey {
			keyEnd = i
		} else {
			valEnd = i
		}
	}

	return q, nil
}

// Uint64 is a helper to obtain a uint64 of any length from a Query. After being
// called, you can safely cast the uint64 to your desired length.
func (q *query) Uint64(key announceParam) (uint64, error) {
	str, exists := q.Params[key]
	if !exists {
		return 0, consts.ErrInvalidMapKey
	}
	return strconv.ParseUint(str, 10, 64)
}

// Uint32 is a helper to obtain a uint32 of any length from a Query. After being
// called, you can safely cast the uint32 to your desired length.
func (q *query) Uint32key(key announceParam) (uint32, error) {
	str, exists := q.Params[key]
	if !exists {
		return 0, consts.ErrInvalidMapKey
	}
	v, err := strconv.ParseUint(str, 10, 32)
	if err != nil {
		return 0, err
	}
	return uint32(v), nil
}

// Uint16 is a helper to obtain a uint16 of any length from a Query. After being
// called, you can safely cast the uint16 to your desired length.
func (q *query) Uint16(key announceParam) (uint16, error) {
	str, exists := q.Params[key]
	if !exists {
		return 0, consts.ErrInvalidMapKey
	}
	v, err := strconv.ParseUint(str, 10, 16)
	if err != nil {
		return 0, err
	}
	return uint16(v), nil
}

// Uint is a helper to obtain a uint of any length from a Query. After being
// called, you can safely cast the uint64 to your desired length.
func (q *query) Uint(key announceParam) (uint, error) {
	str, exists := q.Params[key]
	if !exists {
		return 0, consts.ErrInvalidMapKey
	}
	v, err := strconv.ParseUint(str, 10, 32)
	if err != nil {
		return 0, err
	}
	return uint(v), nil
}
