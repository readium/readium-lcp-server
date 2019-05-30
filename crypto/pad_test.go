// Copyright (c) 2016 Readium Foundation
//
// Redistribution and use in source and binary forms, with or without modification,
// are permitted provided that the following conditions are met:
//
// 1. Redistributions of source code must retain the above copyright notice, this
//    list of conditions and the following disclaimer.
// 2. Redistributions in binary form must reproduce the above copyright notice,
//    this list of conditions and the following disclaimer in the documentation and/or
//    other materials provided with the distribution.
// 3. Neither the name of the organization nor the names of its contributors may be
//    used to endorse or promote products derived from this software without specific
//    prior written permission
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND
// ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED
// WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
// DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT OWNER OR CONTRIBUTORS BE LIABLE FOR
// ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
// (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
// LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND
// ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
// (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
// SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE. 

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
		t.Errorf("should have read 8 bytes, read %d", n)
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
		t.Errorf("should have read 8 bytes, read %d", n)
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
		t.Errorf("should have read 3 bytes, read %d", n)
	}

	n, err = io.ReadFull(reader, out[:])
	if err != nil {
		t.Error(err)
	}

	if n != 3 {
		t.Errorf("should have read 3 bytes, read %d", n)
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
