package crypto

import (
	"bytes"
	"crypto/aes"
	"testing"
)

func TestSimpleEncrypt(t *testing.T) {
	input := bytes.NewBufferString("1234")
	var output bytes.Buffer
	var key [32]byte //not a safe key to have

	err := Encrypt(key[:], input, &output)

	if err != nil {
		t.Log(err)
		t.FailNow()
	}

	bytes := output.Bytes()

	if len(bytes) != aes.BlockSize*2 {
		t.Errorf("Expected %d bytes, got %d", aes.BlockSize*2, len(bytes))
	}
}

func TestConsecutiveEncrypts(t *testing.T) {
	input := bytes.NewBufferString("1234")
	var output bytes.Buffer
	var key [32]byte //not a safe key to have

	err := Encrypt(key[:], input, &output)

	if err != nil {
		t.Log(err)
		t.FailNow()
	}

	input = bytes.NewBufferString("1234")
	var output2 bytes.Buffer

	err = Encrypt(key[:], input, &output2)

	if err != nil {
		t.Log(err)
		t.FailNow()
	}

	if bytes.Equal(output.Bytes(), output2.Bytes()) {
		t.Error("2 calls with the same key should still result in different encryptions")
	}
}

func TestFailingReaderForEncryption(t *testing.T) {
	var output bytes.Buffer
	var key [32]byte //not a safe key to have
	err := Encrypt(key[:], failingReader{}, &output)

	if err == nil {
		t.Error("expected an error from the reader")
	}
}

func TestKeyWrap(t *testing.T) {
	key := []byte{0x00, 0x01, 0x02, 0x03,
		0x04, 0x05, 0x06, 0x07,
		0x08, 0x09, 0x0A, 0x0B,
		0x0C, 0x0D, 0x0E, 0x0F}
	plain := []byte{0x00, 0x11, 0x22, 0x33,
		0x44, 0x55, 0x66, 0x77,
		0x88, 0x99, 0xAA, 0xBB,
		0xCC, 0xDD, 0xEE, 0xFF}
	expected := []byte{0x1F, 0xA6, 0x8B, 0x0A,
		0x81, 0x12, 0xB4, 0x47,
		0xAE, 0xF3, 0x4B, 0xD8,
		0xFB, 0x5A, 0x7B, 0x82,
		0x9D, 0x3E, 0x86, 0x23,
		0x71, 0xD2, 0xCF, 0xE5}

	out := KeyWrap(key, plain)
	if !bytes.Equal(out, expected) {
		t.Errorf("Expected %x, got %x", expected, out)
	}
}
