package store

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestInfoHash(t *testing.T) {
	hexEncoded := "ff503e9ca036f1647c2dfc1337b163e2c54f13f8"
	bytes := []byte{
		0xff, 0x50, 0x3e, 0x9c, 0xa0, 0x36, 0xf1, 0x64, 0x7c, 0x2d,
		0xfc, 0x13, 0x37, 0xb1, 0x63, 0xe2, 0xc5, 0x4f, 0x13, 0xf8}
	var ih1 InfoHash
	require.NoError(t, InfoHashFromHex(&ih1, hexEncoded))
	require.Equal(t, hexEncoded, ih1.String())
	require.Equal(t, bytes, ih1.Bytes())
}
