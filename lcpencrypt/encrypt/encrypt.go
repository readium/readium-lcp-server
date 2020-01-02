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
	"io"
	"io/ioutil"
	"os"

	"github.com/readium/readium-lcp-server/crypto"
	"github.com/readium/readium-lcp-server/epub"
	"github.com/readium/readium-lcp-server/pack"
)

func EncryptWebPubPackage(profile pack.EncryptionProfile, inputPath string, outputPath string) (EncryptionArtifact, error) {
	reader, err := pack.OpenPackagedRWP(inputPath)
	if err != nil {
		return encryptionError(err.Error())
	}

	output, err := os.Create(outputPath)
	if err != nil {
		return encryptionError("Unable to create output file")
	}

	writer, err := reader.NewWriter(output)
	if err != nil {
		return encryptionError("Unable to create output writer")
	}

	encrypter := crypto.NewAESEncrypter_PUBLICATION_RESOURCES()

	encryptionKey, err := pack.Process(profile, encrypter, reader, writer)
	if err != nil {
		return encryptionError("Unable to encrypt file")
	}
	writer.Close()

	hasher := sha256.New()
	output.Seek(0, 0)
	io.Copy(hasher, output)

	stat, err := output.Stat()
	if err != nil {
		return encryptionError("Could not stat output file")
	}

	return EncryptionArtifact{
		Path:          outputPath,
		EncryptionKey: encryptionKey,
		Size:          stat.Size(),
		Checksum:      hex.EncodeToString(hasher.Sum(nil)),
	}, nil
}

// EncryptionArtifact is the result of a successful encryption process
type EncryptionArtifact struct {
	// The encryption process will have put the resulting encrypted file at this place.
	// It is the caller's responsibility to handle it afterwards
	Path string
	// The encryption key that was randomly generated to encrypt the file.
	EncryptionKey []byte
	// The size of the resulting file
	Size int64
	// A Hex-Encoded SHA256 checksum of the encrypted package
	Checksum string
}

func encryptionError(message string) (EncryptionArtifact, error) {
	return EncryptionArtifact{}, errors.New(message)
}

// EncryptEpub Encrypt input file to output file
func EncryptEpub(inputPath string, outputPath string) (EncryptionArtifact, error) {
	if _, err := os.Stat(inputPath); err != nil {
		return encryptionError("Input file does not exist")
	}

	// Read file
	buf, err := ioutil.ReadFile(inputPath)
	if err != nil {
		return encryptionError("Unable to read input file")
	}

	// Read the epub content from the zipped buffer
	zipReader, err := zip.NewReader(bytes.NewReader(buf), int64(len(buf)))
	if err != nil {
		return encryptionError("Invalid ZIP (EPUB) file")
	}

	epubContent, err := epub.Read(zipReader)
	if err != nil {
		return encryptionError("Invalid EPUB content")
	}

	// Create output file
	output, err := os.Create(outputPath)
	if err != nil {
		return encryptionError("Unable to create output file")
	}

	// Pack / encrypt the epub content, fill the output file
	encrypter := crypto.NewAESEncrypter_PUBLICATION_RESOURCES()
	_, encryptionKey, err := pack.Do(encrypter, epubContent, output)
	if err != nil {
		return encryptionError("Unable to encrypt file")
	}

	stats, err := output.Stat()
	if err != nil || (stats.Size() <= 0) {
		return encryptionError("Unable to stat output file")
	}

	hasher := sha256.New()
	s, err := ioutil.ReadFile(outputPath)
	_, err = hasher.Write(s)
	if err != nil {
		return encryptionError("Unable to build checksum")
	}

	checksum := hex.EncodeToString(hasher.Sum(nil))

	output.Close()
	return EncryptionArtifact{outputPath, encryptionKey, stats.Size(), checksum}, nil
}
