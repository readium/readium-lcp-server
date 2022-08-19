// Copyright 2021 Readium Foundation. All rights reserved.
// Use of this source code is governed by a BSD-style license
// that can be found in the LICENSE file exposed on Github (readium) in the project repository.

package encrypt

import (
	"errors"
	"os"
	"strings"

	"github.com/readium/readium-lcp-server/storage"
)

// StoreS3Publication stores an encrypted file into its definitive storage.
// Only called for S3 buckets.
func StoreS3Publication(inputPath, storagePath, name string) error {

	s3Split := strings.Split(storagePath, ":")

	s3conf := storage.S3Config{}
	s3conf.Region = s3Split[1]
	s3conf.Bucket = s3Split[2]

	var store storage.Store
	// init the S3 storage
	store, err := storage.S3(s3conf)
	if err != nil {
		return errors.New("could not init the S3 storage")
	}

	// open the encrypted file, defer its deletion
	file, err := os.Open(inputPath)
	if err != nil {
		return err
	}
	defer cleanupTempFile(file)

	// add the file to the storage with the name passed as parameter
	_, err = store.Add(name, file)
	if err != nil {
		return err
	}
	return nil
}

// cleanupTempFile closes and deletes a temporary file
func cleanupTempFile(f *os.File) {
	if f == nil {
		return
	}
	f.Close()
	os.Remove(f.Name())
}
