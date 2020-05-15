package util

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestNewPasskey(t *testing.T) {
	require.Equal(t, passkeyLen, len(NewPasskey()))
}
