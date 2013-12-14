package epub

import (
	"archive/zip"
  "io"
  "fmt"
)


func findFileInZip(zr zip.Reader, path string) (*zip.File, error) {
	for _, f := range zr.File {
    fmt.Println(f.Name)
		if f.Name == path {
      fmt.Printf("%s FOUND!\n", path)
			return f, nil
		}
	}
	return nil, io.EOF
}
