// Copyright 2020 Readium Foundation. All rights reserved.
// Use of this source code is governed by a BSD-style license
// that can be found in the LICENSE file exposed on Github (readium) in the project repository.

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

	"github.com/endigo/readium-lcp-server/crypto"
	"github.com/endigo/readium-lcp-server/epub"
	"github.com/endigo/readium-lcp-server/license"
	"github.com/endigo/readium-lcp-server/pack"
)

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

// EncryptPackage generates an encrypted output RWPP out of the input RWPP
// It is called from the test frontend server
func EncryptPackage(profile license.EncryptionProfile, inputPath string, outputPath string) (EncryptionArtifact, error) {

	// create an AES encrypter for publication resources
	encrypter := crypto.NewAESEncrypter_PUBLICATION_RESOURCES()

	// create a reader on the un-encrypted readium package
	reader, err := pack.OpenRWPP(inputPath)
	if err != nil {
		return encryptionError(err.Error())
	}
	// create the encrypted package file
	outputFile, err := os.Create(outputPath)
	if err != nil {
		return encryptionError("Unable to create output file")
	}
	defer outputFile.Close()
	// create a writer on the encrypted package
	writer, err := reader.NewWriter(outputFile)
	if err != nil {
		return encryptionError("Unable to create output writer")
	}
	// encrypt resources from the input package, return the encryption key
	encryptionKey, err := pack.Process(profile, encrypter, reader, writer)
	if err != nil {
		return encryptionError("Unable to encrypt file")
	}
	// calculate the output file size and checksum
	hasher := sha256.New()
	outputFile.Seek(0, 0)
	io.Copy(hasher, outputFile)

	stat, err := outputFile.Stat()
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

// EncryptEpub generates an encrypted output file out of the input file
// It is called from the test frontend server
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
	outputFile, err := os.Create(outputPath)
	if err != nil {
		return encryptionError("Unable to create output file")
	}

	// Pack / encrypt the epub content, fill the output file
	encrypter := crypto.NewAESEncrypter_PUBLICATION_RESOURCES()
	_, encryptionKey, err := pack.Do(encrypter, epubContent, outputFile)
	if err != nil {
		return encryptionError("Unable to encrypt file")
	}

	stats, err := outputFile.Stat()
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

	outputFile.Close()
	return EncryptionArtifact{outputPath, encryptionKey, stats.Size(), checksum}, nil
}
