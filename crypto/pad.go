package crypto

import (
	"io"
)

type paddedReader struct {
	io.Reader
	size  byte
	count byte
	left  byte
	done  bool
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
	for i = 0; capacity > 0 && r.left > 0; i++ {
		buf[i] = r.count
		capacity--
		r.left--
	}

	if r.left == 0 {
		err = io.EOF
	}

	return
}

func PaddedReader(r io.Reader, blockSize byte) io.Reader {
	return &paddedReader{Reader: r, size: blockSize, count: 0, left: 0, done: false}
}
