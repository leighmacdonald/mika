package geo

import (
	"bytes"
	"github.com/leighmacdonald/mika/util"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
)

func TestExtract(t *testing.T) {
	archivePath := util.FindFile("examples/data/archive.tar.gz")
	archiveReader, err := os.Open(archivePath)
	outBuf := bytes.NewBufferString("")
	require.NoError(t, err, "Failed to open file reader path")
	require.NoError(t, extractTarGz(archiveReader, outBuf), "Failed to extract file")
	require.Equal(t, "file contents\n", outBuf.String())
}
