package crypto

import (
  "testing"
  "bytes"
  "crypto/aes"
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

  if len(bytes) != aes.BlockSize * 2 {
    t.Errorf("Expected %d bytes, got %d", aes.BlockSize * 2, len(bytes))
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

func TestFailingReaderForEncryption(t * testing.T) {
  var output bytes.Buffer
  var key [32]byte //not a safe key to have
  err := Encrypt(key[:], failingReader{}, &output)

  if err == nil {
    t.Error("expected an error from the reader")
  }
}
