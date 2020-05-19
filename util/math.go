// Package util provides general utility functions shared by the other modules
package util

import (
	"crypto/rand"
	"fmt"
	"math"
)

// GenRandomBytes returns N bytes of random data
func GenRandomBytes(size int) (blk []byte, err error) {
	blk = make([]byte, size)
	_, err = rand.Read(blk)
	return
}

// MinInt returns the smallest of two int values
func MinInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// UMax64 is math.Max for uint64
func UMax64(a, b uint64) uint64 {
	if a > b {
		return a
	}
	return b
}

// UMax32 is math.Max for uint64
func UMax32(a, b uint32) uint32 {
	if a > b {
		return a
	}
	return b
}

// UMax16 is math.Max for uint16
func UMax16(a, b uint16) uint16 {
	if a > b {
		return a
	}
	return b
}

// Max is math.Max for int
//func Max(a, b int) int {
//	if a > b {
//		return a
//	}
//	return b
//}

// UMax is math.Max for uint
func UMax(a, b uint) uint {
	if a > b {
		return a
	}
	return b
}

// UMin64 is math.Min for uint64
func UMin64(a, b uint64) uint64 {
	if a < b {
		return a
	}
	return b
}

// EstSpeed will estimate a peers speed using downloaded amount over time
func EstSpeed(startTime int32, lastTime int32, bytesSent uint64) float64 {
	if startTime <= 0 || lastTime <= 0 || bytesSent == 0 || lastTime < startTime {
		return 0.0
	}
	return round64Plus(float64(bytesSent)/(float64(lastTime)-float64(startTime)), 2)
}

func logN(n, b float64) float64 {
	return math.Log(n) / math.Log(b)
}

func humanizeBytes(s uint64, base float64, sizes []string) string {
	if s < 10 {
		return fmt.Sprintf("%dB", s)
	}
	e := math.Floor(logN(float64(s), base))
	suffix := sizes[int(e)]
	val := math.Floor(float64(s)/math.Pow(base, e)*10+0.5) / 10
	f := "%.0f%s"
	if val < 10 {
		f = "%.1f%s"
	}

	return fmt.Sprintf(f, val, suffix)
}

// HumanBytesString produces a human readable representation of an SI size.
//
// Bytes(82854982) -> 83MB
//noinspection GoUnusedExportedFunction
func HumanBytesString(s uint64) string {
	sizes := []string{"B", "KB", "MB", "GB", "TB", "PB", "EB"}
	return humanizeBytes(s, 1000, sizes)
}

// HumanIBytesString produces a human readable representation of an IEC size.
//
// IBytes(82854982) -> 79MiB
//noinspection GoUnusedExportedFunction
func HumanIBytesString(s uint64) string {
	sizes := []string{"B", "KiB", "MiB", "GiB", "TiB", "PiB", "EiB"}
	return humanizeBytes(s, 1024, sizes)
}

func roundF64(f float64) float64 {
	return math.Floor(f + .5)
}

func round64Plus(f float64, places int) float64 {
	shift := math.Pow(10, float64(places))
	return roundF64(f*shift) / shift
}
