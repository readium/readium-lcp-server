package crypto

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/hex"
	"testing"
)

func TestEncryptGCM(t *testing.T) {
	key, _ := hex.DecodeString("11754cd72aec309bf52f7687212e8957")

	encrypter := NewAESGCMEncrypter()

	data := []byte("The quick brown fox jumps over the lazy dog")

	r := bytes.NewReader(data)
	w := new(bytes.Buffer)

	if err := encrypter.Encrypt(ContentKey(key), r, w); err != nil {
		t.Fatal("Encryption failed", err)
	}

	block, _ := aes.NewCipher(key)
	gcm, _ := cipher.NewGCM(block)

	out := w.Bytes()
	t.Logf("nonce size: %#v", gcm.NonceSize())
	t.Logf("nonce: %#v", out[0:gcm.NonceSize()])
	t.Logf("ciphertext: %#v", out[gcm.NonceSize():])
	clear := make([]byte, 0)
	clear, err := gcm.Open(clear, out[0:gcm.NonceSize()], out[gcm.NonceSize():], nil)

	if err != nil {
		t.Fatal("Decryption failed", err)
	}

	if diff := bytes.Compare(data, clear); diff != 0 {
		t.Logf("Original: %#v", data)
		t.Logf("After cycle: %#v", clear)
		t.Errorf("Expected encryption-decryption to return original")
	}
}
