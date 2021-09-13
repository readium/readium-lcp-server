// Copyright 2020 Readium Foundation. All rights reserved.
// Use of this source code is governed by a BSD-style license
// that can be found in the LICENSE file exposed on Github (readium) in the project repository.

package encrypt

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"

	"github.com/readium/readium-lcp-server/crypto"
	"github.com/readium/readium-lcp-server/epub"
	"github.com/readium/readium-lcp-server/license"
	"github.com/readium/readium-lcp-server/pack"
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

// EncryptPackage generates an encrypted output RPF out of the input RPF
// It is called from the test frontend server
func EncryptPackage(profile license.EncryptionProfile, inputPath string, outputPath string) (EncryptionArtifact, error) {

	// create an AES encrypter for publication resources
	encrypter := crypto.NewAESEncrypter_PUBLICATION_RESOURCES()

	// create a reader on the un-encrypted readium package
	reader, err := pack.OpenRPF(inputPath)
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

	err = writer.Close()
	if err != nil {
		return encryptionError("Unable to close the writer")
	}

	// calculate the output file size and checksum
	hasher := sha256.New()
	outputFile.Seek(0, 0)
	size, err := io.Copy(hasher, outputFile)
	if err != nil {
		return encryptionError("Could not generate a hash for the file")
	}

	return EncryptionArtifact{
		Path:          outputPath,
		EncryptionKey: encryptionKey,
		Size:          size,
		Checksum:      hex.EncodeToString(hasher.Sum(nil)),
	}, nil
}

// EncryptEpub generates an encrypted output file out of the input file
// It is called from the test frontend server; inputPath is therefore a file path.
func EncryptEpub(inputPath string, outputPath string) (EncryptionArtifact, error) {

	if _, err := os.Stat(inputPath); err != nil {
		return encryptionError("Input file does not exist")
	}

	// create a zip reader from the input path
	zr, err := zip.OpenReader(inputPath)
	if err != nil {
		return encryptionError("Unable to open the input file")
	}
	defer zr.Close()

	// parse the source epub
	epubContent, err := epub.Read(&zr.Reader)
	if err != nil {
		return encryptionError("Error reading epub content")
	}

	// create the output file
	outputFile, err := os.Create(outputPath)
	if err != nil {
		return encryptionError("Unable to create output file")
	}
	defer outputFile.Close()

	// pack / encrypt the epub content, fill the output file
	encrypter := crypto.NewAESEncrypter_PUBLICATION_RESOURCES()
	_, encryptionKey, err := pack.Do(encrypter, epubContent, outputFile)
	if err != nil {
		return encryptionError("Unable to encrypt file")
	}

	// calculate the output file size and checksum
	hasher := sha256.New()
	outputFile.Seek(0, 0)
	size, err := io.Copy(hasher, outputFile)
	if err != nil {
		return encryptionError("Could not generate a hash for the file")
	}

	return EncryptionArtifact{
		Path:          outputPath,
		EncryptionKey: encryptionKey,
		Size:          size,
		Checksum:      hex.EncodeToString(hasher.Sum(nil)),
	}, nil
}
