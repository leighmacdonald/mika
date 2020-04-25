// Package util provides general utility functions shared by the other modules
package util

import (
	"fmt"
	"math"
)

// math.Max for uint64
func UMax64(a, b uint64) uint64 {
	if a > b {
		return a
	} else {
		return b
	}
}

// math.Max for uint64
func UMax(a, b uint) uint {
	if a > b {
		return a
	} else {
		return b
	}
}

// math.Min for uint64
func UMin64(a, b uint64) uint64 {
	if a < b {
		return a
	} else {
		return b
	}
}

// Estimate a peers speed using downloaded amount over time
func EstSpeed(startTime int32, lastTime int32, bytesSent uint64) float64 {
	if startTime <= 0 || lastTime <= 0 || bytesSent == 0 || lastTime < startTime {
		return 0.0
	}
	return RoundPlus(float64(bytesSent)/(float64(lastTime)-float64(startTime)), 2)
}

func LogN(n, b float64) float64 {
	return math.Log(n) / math.Log(b)
}

func humanizeBytes(s uint64, base float64, sizes []string) string {
	if s < 10 {
		return fmt.Sprintf("%dB", s)
	}
	e := math.Floor(LogN(float64(s), base))
	suffix := sizes[int(e)]
	val := math.Floor(float64(s)/math.Pow(base, e)*10+0.5) / 10
	f := "%.0f%s"
	if val < 10 {
		f = "%.1f%s"
	}

	return fmt.Sprintf(f, val, suffix)
}

// Bytes produces a human readable representation of an SI size.
//
// Bytes(82854982) -> 83MB
func Bytes(s uint64) string {
	sizes := []string{"B", "KB", "MB", "GB", "TB", "PB", "EB"}
	return humanizeBytes(s, 1000, sizes)
}

// IBytes produces a human readable representation of an IEC size.
//
// IBytes(82854982) -> 79MiB
func IBytes(s uint64) string {
	sizes := []string{"B", "KiB", "MiB", "GiB", "TiB", "PiB", "EiB"}
	return humanizeBytes(s, 1024, sizes)
}

func Round(f float64) float64 {
	return math.Floor(f + .5)
}

func RoundPlus(f float64, places int) float64 {
	shift := math.Pow(10, float64(places))
	return Round(f*shift) / shift
}
