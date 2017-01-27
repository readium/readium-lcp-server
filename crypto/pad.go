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
	"io"
	"math/rand"
	"time"
)

type paddedReader struct {
	io.Reader
	size  byte
	count byte
	left  byte
	done  bool
	insertPadLengthAll bool
}

func (r *paddedReader) Read(buf []byte) (int, error) {
	// we're still reading stuff
	if !r.done {
		n, err := r.Reader.Read(buf)

		if err != nil && err != io.EOF {
			return n, err
		}

		//update counter
		r.count = byte((n + int(r.count)) % int(r.size))

		// try to read more
		var nn int
		for n < len(buf) && err == nil {
			nn, err = r.Reader.Read(buf[n:])
			n += nn
			r.count = byte((nn + int(r.count)) % int(r.size))
		}

		if err == io.EOF {
			r.done = true
			r.left = r.size - r.count
			r.count = r.left
			paddingAdded, err := r.pad(buf[n:])
			return n + paddingAdded, err
		}

		return n, err
	}

	return r.pad(buf)
}

func (r *paddedReader) pad(buf []byte) (i int, err error) {
	capacity := cap(buf)

	src := rand.New(rand.NewSource(time.Now().UnixNano()))

	for i = 0; capacity > 0 && r.left > 0; i++ {

		if (r.insertPadLengthAll) {
			buf[i] = r.count
		} else {
			if r.left == 1 { //capacity == 1 && 
				buf[i] = r.count
			} else {
				buf[i] = byte(src.Intn(254) + 1)
			}
		}

		capacity--
		r.left--
	}

	if r.left == 0 {
		err = io.EOF
	}

	return
}


// insertPadLengthAll = true means PKCS#7 (padding length inserted in each padding slot),
// otherwise false means padding length inserted only in the last slot (the rest is random bytes)
func PaddedReader(r io.Reader, blockSize byte, insertPadLengthAll bool) io.Reader {
	return &paddedReader{Reader: r, size: blockSize, count: 0, left: 0, done: false, insertPadLengthAll: insertPadLengthAll}
}
