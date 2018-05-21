package crypto

import (
	"bytes"
	"io"
	"testing"
)

func TestOneBlock(t *testing.T) {
	buf := bytes.NewBufferString("4321")
	reader := PaddedReader(buf, 6, true)
	var out [12]byte
	n, err := reader.Read(out[:])
	if err != nil && err != io.EOF {
		t.Error(err)
	}
	if n != 6 {
		t.Errorf("should have read 6 bytes, read %d", n)
	}

	// PaddedReader constructor parameter "insertPadLengthAll" is true,
	// means all last bytes equate the padding length
	if out[4] != 2 || out[5] != 2 {
		t.Errorf("last values were expected to be 2, got [%x %x]", out[4], out[5])
	}
}

func TestFullPadding(t *testing.T) {
	buf := bytes.NewBufferString("1234")
	reader := PaddedReader(buf, 4, true)

	var out [8]byte
	n, err := io.ReadFull(reader, out[:])
	if err != nil {
		t.Error(err)
	}
	if n != 8 {
		t.Error("should have read 8 bytes, read %d", n)
	}

	// PaddedReader constructor parameter "insertPadLengthAll" is true,
	// means all last bytes equate the padding length
	if out[4] != 4 || out[5] != 4 || out[6] != 4 || out[7] != 4 {
		t.Errorf("last values were expected to be 4, got [%x %x %x %x]", out[4], out[5], out[6], out[7])
	}
}

func TestManyBlocks(t *testing.T) {
	buf := bytes.NewBufferString("1234")
	reader := PaddedReader(buf, 3, true)
	var out [3]byte
	n, err := io.ReadFull(reader, out[:])
	if err != nil {
		t.Error(err)
	}

	n, err = io.ReadFull(reader, out[:])
	if err != nil {
		t.Error(err)
	}

	if n != 3 {
		t.Errorf("should have read 3 bytes, read %d", n)
	}

	// PaddedReader constructor parameter "insertPadLengthAll" is true,
	// means all last bytes equate the padding length
	if out[1] != 2 || out[2] != 2 {
		t.Errorf("last values were expected to be 2, got [%x %x]", out[1], out[2])
	}
}

func TestOneBlock_Random(t *testing.T) {
	buf := bytes.NewBufferString("4321")
	reader := PaddedReader(buf, 6, false)
	var out [12]byte
	n, err := reader.Read(out[:])
	if err != nil && err != io.EOF {
		t.Error(err)
	}
	if n != 6 {
		t.Errorf("should have read 6 bytes, read %d", n)
	}

	// the PaddedReader constructor parameter "insertPadLengthAll" is false,
	// so only the last byte out[2] equates the padding length (the others are random)
	if out[4] == 2 || out[5] != 2 {
		t.Errorf("last values were expected to be [random, 2], got [%x %x]", out[4], out[5])
	}
}

func TestFullPadding_Random(t *testing.T) {
	buf := bytes.NewBufferString("1234")
	reader := PaddedReader(buf, 4, false)

	var out [8]byte
	n, err := io.ReadFull(reader, out[:])
	if err != nil {
		t.Error(err)
	}
	if n != 8 {
		t.Error("should have read 8 bytes, read %d", n)
	}

	// the PaddedReader constructor parameter "insertPadLengthAll" is false,
	// so only the last byte out[7] equates the padding length (the others are random)
	if out[4] == 4 || out[5] == 4 || out[6] == 4 || out[7] != 4 {
		t.Errorf("last values were expected to be [random, random, random, 4], got [%x %x %x %x]", out[4], out[5], out[6], out[7])
	}
}

func TestManyBlocks_Random(t *testing.T) {
	buf := bytes.NewBufferString("1234")
	reader := PaddedReader(buf, 3, false)
	var out [3]byte
	n, err := io.ReadFull(reader, out[:])
	if err != nil {
		t.Error(err)
	}

	n, err = io.ReadFull(reader, out[:])
	if err != nil {
		t.Error(err)
	}

	if n != 3 {
		t.Errorf("should have read 3 bytes, read %d", n)
	}

	// the PaddedReader constructor parameter "insertPadLengthAll" is false,
	// so only the last byte out[2] equates the padding length (the others are random)
	if out[1] == 2 || out[2] != 2 {
		t.Errorf("last values were expected to be [random 2], got [%x %x]", out[1], out[2])
	}
}

func TestPaddingInMultipleCalls(t *testing.T) {
	buf := bytes.NewBufferString("1")
	reader := PaddedReader(buf, 6, false)
	var out [3]byte
	n, err := io.ReadFull(reader, out[:])

	if err != nil {
		t.Error(err)
	}

	if n != 3 {
		t.Error("should have read 3 bytes, read %d", n)
	}

	n, err = io.ReadFull(reader, out[:])
	if err != nil {
		t.Error(err)
	}

	if n != 3 {
		t.Error("should have read 3 bytes, read %d", n)
	}

}

type failingReader struct {
}

func (r failingReader) Read(buf []byte) (int, error) {
	return 0, io.ErrShortBuffer
}

func TestFailingReader(t *testing.T) {
	reader := PaddedReader(failingReader{}, 8, false)
	var out [8]byte
	_, err := io.ReadFull(reader, out[:])

	if err != io.ErrShortBuffer {
		t.Error(err)
	}
}
