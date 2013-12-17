package main

import (
  _ "github.com/mattn/go-sqlite3"
  //"github.com/jpbougie/lcpserve/epub"
  //"github.com/jpbougie/lcpserve/crypto"
  //"github.com/jpbougie/lcpserve/pack"
  "github.com/jpbougie/lcpserve/storage"
  "github.com/jpbougie/lcpserve/index"
  "github.com/jpbougie/lcpserve/server"
  //"archive/zip"
  "os"
  //"fmt"
)

func main() {
  host := "localhost"
  if len(os.Args) > 1 {
    host = os.Args[1]
  }
  idx, err := index.Open("test.sqlite")
  store := storage.NewFileSystem("files", "http://" + host + ":8989/files")
  if err != nil {
    panic(err)
  }
  s := server.New(":8989", &idx, &store)
  s.ListenAndServe()
  //zipfile, err := zip.OpenReader("test/samples/sample.epub")
  //if err != nil {
    //panic(err)
  //}
  //ep, err := epub.Read(zipfile.Reader)
  //if err != nil {
    //panic(err)
  //}
  //fmt.Println(ep)

  //ep, k, err := pack.Do(ep)
  //fmt.Println(k)
  //w, err := os.Create("test/output.epub")
  //if err != nil {
    //panic(err)
  //}
  //err = ep.Write(w)
  //defer w.Close()
  //if err != nil {
    //panic(err)
  //}

  //zipfile, err = zip.OpenReader("test/output.epub")
  //if err != nil {
    //panic(err)
  //}
  //ep, err = epub.Read(zipfile.Reader)
  //if err != nil {
    //panic(err)
  //}
  //ep, err = pack.Undo(k, ep)
  //if err != nil {
    //panic(err)
  //}
  //w, err = os.Create("test/decrypted.epub")
  //if err != nil {
    //panic(err)
  //}
  //err = ep.Write(w)
  //defer w.Close()

  //log.Printf("Bytes read: %d", offset)
  //zipReader, err := zip.NewReader(bytes.NewReader(b), int64(len(b)))
  //if err != nil {
    //panic(err)
  //}


  
}
