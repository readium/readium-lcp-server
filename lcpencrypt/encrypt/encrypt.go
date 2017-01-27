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

package encrypt

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io/ioutil"
	"os"

	"github.com/readium/readium-lcp-server/crypto"
	"github.com/readium/readium-lcp-server/epub"
	"github.com/readium/readium-lcp-server/pack"
)

// EncryptedEpub Encrypted epub
type EncryptedEpub struct {
	Path          string
	EncryptionKey []byte
	Size          int64
	Checksum      string
}

// EncryptEpub Encrypt input file to output file
func EncryptEpub(inputPath string, outputPath string) (EncryptedEpub, error) {
	if _, err := os.Stat(inputPath); err != nil {
		return EncryptedEpub{}, errors.New("Input file does not exists")
	}

	// Read file
	buf, err := ioutil.ReadFile(inputPath)
	if err != nil {
		return EncryptedEpub{}, errors.New("Unable to read input file")
	}

	// Read the epub content from the zipped buffer
	zipReader, err := zip.NewReader(bytes.NewReader(buf), int64(len(buf)))
	if err != nil {
		return EncryptedEpub{}, errors.New("Invalid zip (epub) file")
	}

	epubContent, err := epub.Read(zipReader)
	if err != nil {
		return EncryptedEpub{}, errors.New("Invalid epub content")
	}

	// Create output file
	output, err := os.Create(outputPath)
	if err != nil {
		return EncryptedEpub{}, errors.New("Unable to create output file")
	}

	// Pack / encrypt the epub content, fill the output file
	encrypter := crypto.NewAESEncrypter_PUBLICATION_RESOURCES()
	_, encryptionKey, err := pack.Do(encrypter, epubContent, output)
	if err != nil {
		return EncryptedEpub{}, errors.New("Unable to encrypt file")
	}

	stats, err := output.Stat()
	if err != nil && (stats.Size() > 0) {
		return EncryptedEpub{}, errors.New("Unable to output file")
	}

	hasher := sha256.New()
	s, err := ioutil.ReadFile(outputPath)
	_, err = hasher.Write(s)
	if err != nil {
		return EncryptedEpub{}, errors.New("Unable to build checksum")
	}

	checksum := hex.EncodeToString(hasher.Sum(nil))

	output.Close()
	return EncryptedEpub{outputPath, encryptionKey, stats.Size(), checksum}, nil
}
