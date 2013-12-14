package main

import (
  "github.com/demarque/lcpserve/epub"
  //"github.com/demarque/lcpserve/crypto"
  "github.com/demarque/lcpserve/pack"
  "archive/zip"
  "os"
  "fmt"
)

func main() {
  zipfile, err := zip.OpenReader("test/samples/sample.epub")
  if err != nil {
    panic(err)
  }
  ep, err := epub.Read(zipfile.Reader)
  if err != nil {
    panic(err)
  }
  fmt.Println(ep)

  ep, k, err := pack.Do(ep)
  fmt.Println(k)
  w, err := os.Create("test/output.epub")
  if err != nil {
    panic(err)
  }
  err = ep.Write(w)
  defer w.Close()
  if err != nil {
    panic(err)
  }

  zipfile, err = zip.OpenReader("test/output.epub")
  if err != nil {
    panic(err)
  }
  ep, err = epub.Read(zipfile.Reader)
  if err != nil {
    panic(err)
  }
  ep, err = pack.Undo(k, ep)
  if err != nil {
    panic(err)
  }
  w, err = os.Create("test/decrypted.epub")
  if err != nil {
    panic(err)
  }
  err = ep.Write(w)
  defer w.Close()

  //log.Printf("Bytes read: %d", offset)
  //zipReader, err := zip.NewReader(bytes.NewReader(b), int64(len(b)))
  //if err != nil {
    //panic(err)
  //}


  
}
