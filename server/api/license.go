package api

import (
  "net/http"
  "github.com/gorilla/mux"
  "encoding/json"
  "io"
  "time"
  "bytes"
  "crypto/sha256"
  "github.com/jpbougie/lcpserve/crypto"
)

//{
    //"content_key": "12345",
    //"date": "2013-11-04T01:08:15+01:00",
    //"hint": "Enter your email address",
    //"hint_url": "http://www.imaginaryebookretailer.com/lcp"
//}
type License struct {
  ContentKey []byte `json:"content_key"`
  Date time.Time `json:"date"`
  Hint string `json:"hint"`
  HintUrl string `json:"hint_url"`
  FetchUrl string `json:"fetch_url"`
}

func GrantLicense(w http.ResponseWriter, r *http.Request, s Server) {
  vars := mux.Vars(r)
  err := grantLicense(vars["key"], vars["passphrase"], s, w)
  if err != nil {
    w.WriteHeader(500)
    w.Write([]byte(err.Error()))
  }
}


func grantLicense(key, passphrase string, s Server, w io.Writer) error {
  p, err := s.Index().Get(key)
  if err != nil {
    return err
  }
  
  item, err := s.Store().Get(key)
  if err != nil {
    return err
  }

  l := License{
    ContentKey: encryptKey(p.EncryptionKey, passphrase),
    FetchUrl: item.PublicUrl(),
    Hint: "passphrase",
    HintUrl: "http://readium.org/lcp/hint",
    Date: time.Now(),
    }

  enc := json.NewEncoder(w)
  enc.Encode(l)

  return nil
}

func encryptKey(key []byte, passphrase string) []byte {
  kek := sha256.Sum256([]byte(passphrase))
  var out bytes.Buffer
  in := bytes.NewReader(key)
  crypto.Encrypt(kek[:], in, &out)
  return out.Bytes()
}

