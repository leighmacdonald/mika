package util

import "time"

// Generate a 32bit unix timestamp
func Unixtime() int32 {
	return int32(time.Now().Unix())
}
