package util

import (
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestStringToInt16(t *testing.T) {
	require.Equal(t, int16(10), StringToInt16("10", 1))
	require.Equal(t, int16(-10), StringToInt16("-10", 1))
	require.Equal(t, int16(10), StringToInt16("", 10))
}

func TestStringToUInt16(t *testing.T) {
	require.Equal(t, uint16(10), StringToUInt16("10", 1))
	require.Equal(t, uint16(1), StringToUInt16("1000000000000", 1))
	require.Equal(t, uint16(10), StringToUInt16("", 10))
}

func TestStringToUInt32(t *testing.T) {
	require.Equal(t, uint32(10), StringToUInt32("10", 1))
	require.Equal(t, uint32(1), StringToUInt32("1000000000000", 1))
	require.Equal(t, uint32(10), StringToUInt32("", 10))
}

func TestStringToUInt64(t *testing.T) {
	require.Equal(t, uint64(10), StringToUInt64("10", 1))
	require.Equal(t, uint64(1000000000000), StringToUInt64("1000000000000", 1))
	require.Equal(t, uint64(10), StringToUInt64("", 10))
}

func TestStringToFloat64(t *testing.T) {
	require.Equal(t, 10.500, StringToFloat64("10.500", 1))
	require.Equal(t, 10.0, StringToFloat64("", 10))
}

func TestStringToBool(t *testing.T) {
	require.Equal(t, true, StringToBool("", true))
	require.Equal(t, true, StringToBool("true", false))
}

func TestStringToTime(t *testing.T) {
	t0 := time.Now()
	require.Equal(t, t0.Unix(), StringToTime(TimeToString(t0)).Unix())
}
