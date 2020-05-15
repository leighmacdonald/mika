package util

import (
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"os"
	"strings"
	"testing"
)

func TestExists(t *testing.T) {
	require.False(t, Exists("1234567890.txt"))
	file, err := ioutil.TempFile("dir", "prefix")
	require.NoError(t, err)
	defer func() { _ = os.Remove(file.Name()) }()
	require.True(t, Exists(file.Name()))
}

func TestFindFile(t *testing.T) {
	require.Equal(t, FindFile("1234567890.txt"), "1234567890.txt")
	require.True(t, strings.HasPrefix(FindFile("README.md"), "/"))
}
