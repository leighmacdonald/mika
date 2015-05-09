// Package util provides general utility functions shared by the other modules
package util

// math.Max for uint64
func UMax(a, b uint64) uint64 {
	if a > b {
		return a
	} else {
		return b
	}
}

// math.Min for uint64
func UMin(a, b uint64) uint64 {
	if a < b {
		return a
	} else {
		return b
	}
}

// Estimate a peers speed using downloaded amount over time
func EstSpeed(start_time int32, last_time int32, bytes_sent uint64) float64 {
	if start_time <= 0 || last_time <= 0 || bytes_sent == 0 || last_time < start_time {
		return 0.0
	}
	return float64(bytes_sent) / (float64(last_time) - float64(start_time))
}
