package crypto

import (
	"crypto/rand"
)

type ContentKey []byte

func GenerateKey(size int) ([]byte, error) {
	k := make([]byte, size)

	_, err := rand.Read(k)
	if err != nil {
		return nil, err
	}

	return k, nil
}
