package pack

import (
  "github.com/jpbougie/lcpserve/epub"
  "testing"
  "archive/zip"
)

func TestPacking(t *testing.T) {
  z, err := zip.OpenReader("../test/samples/sample.epub")
  if err != nil {
    t.Error(err)
    t.FailNow()
  }

  input, _ := epub.Read(z.Reader)
  output, key, err := Do(input)
  if err != nil {
    t.Error(err)
    t.FailNow()
  }

  if output.Encryption == nil {
    t.Error("Expected an xmlenc file")
  }

  if len(output.Encryption.Data) == 0 {
    t.Error("Expected some encrypted data")
  }

  if key == nil {
    t.Error("expected a key")
  }
}
