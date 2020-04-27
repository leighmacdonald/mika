package util

import (
	log "github.com/sirupsen/logrus"
	"strconv"
	"time"
)

func StringToInt16(s string, def int16) int16 {
	v, err := strconv.ParseInt(s, 10, 16)
	if err != nil {
		log.Warnf("failed to parse int16 value from redis: %s", s)
		return def
	}
	return int16(v)
}

func StringToUInt32(s string, def uint32) uint32 {
	v, err := strconv.ParseInt(s, 10, 32)
	if err != nil {
		log.Warnf("failed to parse uint32 value from redis: %s", s)
		return def
	}
	return uint32(v)
}

func StringToFloat64(s string, def float64) float64 {
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		log.Warnf("failed to parse float64 value from redis: %s", s)
		return def
	}
	return v
}

func StringToBool(s string, def bool) bool {
	v, err := strconv.ParseBool(s)
	if err != nil {
		log.Warnf("failed to parse bool value from redis: %s", s)
		return def
	}
	return v
}

func StringToTime(s string) time.Time {
	v, err := time.Parse(time.RFC1123Z, s)
	if err != nil {
		log.Warnf("failed to parse time value from redis: %s", s)
		return time.Now()
	}
	return v
}

func TimeToString(t time.Time) string {
	return t.Format(time.RFC1123Z)
}
