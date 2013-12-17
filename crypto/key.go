package crypto

import (
  "crypto/rand"
)

const (
  keyLength = 32 // 256 bits
)

func GenerateKey() ([]byte, error) {
  k := make([]byte, keyLength)

  _, err := rand.Read(k)
  if err != nil {
    return nil, err
  }

  return k, nil
}
