package epub

import (
	"archive/zip"
	"io"
)

func findFileInZip(zr *zip.Reader, path string) (*zip.File, error) {
	for _, f := range zr.File {
		if f.Name == path {
			return f, nil
		}
	}
	return nil, io.EOF
}
