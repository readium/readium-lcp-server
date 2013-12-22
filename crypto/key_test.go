package crypto

import (
	"testing"
)

func TestGenerateKey(t *testing.T) {
	buf, err := GenerateKey()

	if err != nil {
		t.Error(err)
	}

	if len(buf) != 32 {
		t.Error("it should be a 32-byte long buffer")
	}
}
