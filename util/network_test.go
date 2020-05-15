package util

import (
	"github.com/stretchr/testify/require"
	"net"
	"testing"
)

func TestIsPrivateIP(t *testing.T) {
	ips := []struct {
		ip      string
		private bool
	}{
		{"127.0.0.1", true},
		{"172.16.100.200", true},
		{"10.10.10.10", true},
		{"192.168.100.100", true},
		{"127.0.0.1", true},
		{"::1", true},
		{"8.8.8.8", false},
	}
	for _, i := range ips {
		require.Equal(t, i.private, IsPrivateIP(net.ParseIP(i.ip)))
	}
}
