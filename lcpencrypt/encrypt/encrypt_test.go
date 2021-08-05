// Copyright 2020 Readium Foundation. All rights reserved.
// Use of this source code is governed by a BSD-style license
// that can be found in the LICENSE file exposed on Github (readium) in the project repository.

package encrypt

import (
	"fmt"
	"testing"
)

func TestEncrypt(t *testing.T) {

	inputPath := "../../test/samples/sample.epub"
	outputPath := "../../test/samples/sample-encrypted.epub"
	result, err := EncryptEpub(inputPath, outputPath)
	if err != nil {
		t.Error(err.Error())
	}

	fmt.Printf("output: %s size %d\n", result.Path, result.Size)

}
