package tracker

import (
	"errors"
	"net/url"
	"strconv"
	"strings"
)

type Query struct {
	InfoHashes []string
	Params     map[string]string
}

// Parses a raw url query into a Query struct
// Taken from Chihaya
func QueryStringParser(query string) (*Query, error) {
	var (
		keyStart, keyEnd int
		valStart, valEnd int
		firstInfoHash    string

		onKey       = true
		hasInfoHash = false

		q = &Query{
			InfoHashes: nil,
			Params:     make(map[string]string),
		}
	)

	for i, length := 0, len(query); i < length; i++ {
		separator := query[i] == '&' || query[i] == ';' || query[i] == '?'
		if separator || i == length-1 {
			if onKey {
				keyStart = i + 1
				continue
			}
			if i == length-1 && !separator {
				if query[i] == '=' {
					continue
				}
				valEnd = i
			}
			keyStr, err := url.QueryUnescape(query[keyStart : keyEnd+1])
			if err != nil {
				return nil, err
			}
			valStr, err := url.QueryUnescape(query[valStart : valEnd+1])
			if err != nil {
				return nil, err
			}
			q.Params[strings.ToLower(keyStr)] = valStr

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
		} else if query[i] == '=' {
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

// Uint64 is a helper to obtain a uint of any length from a Query. After being
// called, you can safely cast the uint64 to your desired length.
func (q *Query) Uint64(key string) (uint64, error) {
	str, exists := q.Params[key]
	if !exists {
		return 0, errors.New("value does not exist for key: " + key)
	}
	return strconv.ParseUint(str, 10, 64)
}

// Parse the num want from the announce request, replacing with our
// own default value if the supplied value is missing or deemed invalid
func getNumWant(q *Query, fallback int) int {
	if numWantStr, exists := q.Params["numwant"]; exists {
		numWant, err := strconv.Atoi(numWantStr)
		if err != nil {
			return fallback
		}
		return numWant
	}

	return fallback
}
