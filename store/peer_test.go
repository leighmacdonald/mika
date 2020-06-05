package store

import (
	"fmt"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestClientString(t *testing.T) {
	type client struct {
		peerID PeerID
		client string
	}
	clients := []client{
		{peerID: PeerIDFromString("S58B-----XXXXXXXXXXX"), client: fmt.Sprintf("%s 5.8.11.0", clientNames["S"])},
		{peerID: PeerIDFromString("-qB4170-u-rGseINmloG"), client: fmt.Sprintf("%s 4.1.7.0", clientNames["qB"])},
		{peerID: PeerIDFromString("--------u-rGseINmloG"), client: "Unknown 0.0.0.0"},
	}
	for _, c := range clients {
		require.Equal(t, c.client, ClientString(c.peerID).String())
	}
}
