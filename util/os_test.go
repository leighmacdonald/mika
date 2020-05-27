package util

import (
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"os"
	"testing"
)

func TestExists(t *testing.T) {
	require.False(t, Exists("1234567890.txt"))
	file, err := ioutil.TempFile("", "prefix")
	require.NoError(t, err)
	defer func() { _ = os.Remove(file.Name()) }()
	require.True(t, Exists(file.Name()))
}

func TestFindFile(t *testing.T) {
	require.Equal(t, FindFile("1234567890.txt"), "1234567890.txt")
	path := FindFile("README.md")
	fp, err := os.Open(path)
	require.NoError(t, err, "Failed to open file: %s", path)
	require.NoError(t, fp.Close())
}
