package geo

import (
	"archive/tar"
	"compress/gzip"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"io"
	"strings"
)

func extractTarGz(gzipStream io.Reader, outStream io.Writer) error {
	uncompressedStream, err := gzip.NewReader(gzipStream)
	if err != nil {
		log.Fatal("extractTarGz: NewReader failed")
	}
	tarReader := tar.NewReader(uncompressedStream)
	foundFile := false
	for {
		header, err := tarReader.Next()
		if err == io.EOF || header == nil {
			break
		}
		if err != nil {
			log.Fatalf("extractTarGz: Next() failed: %s", err.Error())
		}
		switch header.Typeflag {
		case tar.TypeDir:
			//if err := os.Mkdir(header.Name, 0755); err != nil {
			//	log.Fatalf("extractTarGz: Mkdir() failed: %s", err.Error())
			//}
		case tar.TypeReg:
			if !strings.HasSuffix(header.Name, ".mmdb") {
				continue
			}
			foundFile = true
			if _, err := io.Copy(outStream, tarReader); err != nil {
				log.Fatalf("extractTarGz: Copy() failed: %s", err.Error())
			}
		default:
			log.Fatalf(
				"extractTarGz: unknown type: %v in %s",
				header.Typeflag,
				header.Name)
		}
	}
	if !foundFile {
		return errors.New("Archive contained no mmdb file")
	}
	return nil
}
