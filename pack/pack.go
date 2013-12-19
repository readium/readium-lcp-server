package pack

import (
  "github.com/jpbougie/lcpserve/epub"
  "github.com/jpbougie/lcpserve/crypto"
  "github.com/jpbougie/lcpserve/xmlenc"
  "strings"
)

func Do(ep epub.Epub) (epub.Epub, []byte, error) {
  k, err := crypto.GenerateKey()
  if err != nil {
    return ep, nil, err
  }

  ep.Encryption = &xmlenc.Manifest{}
  for _, res := range ep.Resource {
    if canEncrypt(res, ep) {
      encryptFile(k, ep.Encryption, res)
    }
  }
  return ep, k, nil
}

func canEncrypt(file epub.Resource, ep epub.Epub) bool {
  n := file.File.Name
  hasCover, cover := ep.Cover()
  return (n != "mimetype" &&
  n != "META-INF/container.xml" &&
  n != "META-INF/encryption.xml" &&
  n != "META-INF/manifest.xml" &&
  n != "META-INF/metadata.xml" &&
  n != "META-INF/rights.xml" &&
  n != "META-INF/signatures.xml" &&
  n != "META-INF/license.json") &&
  !strings.HasSuffix(n, ".opf") &&
  !strings.HasSuffix(n, ".ncx") &&
  (!hasCover || n != cover.File.Name)
}

func encryptFile(key []byte, m *xmlenc.Manifest, file epub.Resource) error {
  data := xmlenc.Data{}
  data.Method.Algorithm = "http://www.w3.org/2001/04/xmlenc#aes256-cbc"
  data.KeyInfo.RetrievalMethod.URI = "license.json#/content_key"
  data.KeyInfo.RetrievalMethod.Type = "http://readium.org/2014/01/lcp#EncryptedContentKey"
  data.CipherData.CipherReference.URI = xmlenc.URI(file.File.Name)
  m.Data = append(m.Data, data)

  r, err := file.File.Open()
  if err != nil {
    return err
  }
  defer r.Close()

  err = crypto.Encrypt(key, r, file.Output)

  return err
}

func Undo(key []byte, ep epub.Epub) (epub.Epub, error) {
  for _, data := range ep.Encryption.Data {
    ok, res := findFile(string(data.CipherData.CipherReference.URI), ep)
    if ok {
      r, err := res.File.Open()
      if err != nil {
        return ep, err
      }

      crypto.Decrypt(key, r, res.Output)
    }
  }

  ep.Encryption = nil

  return ep, nil
}

func findFile(name string, ep epub.Epub) (bool, epub.Resource) {
  for _, res := range ep.Resource {
    if res.File.Name == name {
      return true, res
    }
  }

  return false, epub.Resource{}
}
