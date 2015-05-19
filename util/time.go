package util

import "time"

// Unixtime Generates and returns a 32bit unix timestamp
func Unixtime() int32 {
	return int32(time.Now().Unix())
}
