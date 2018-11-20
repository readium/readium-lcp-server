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

package epub

import (
	"archive/zip"
	"fmt"
	"sort"
	"testing"
)

func TestEpubLoading(t *testing.T) {

	zr, err := zip.OpenReader("../test/samples/sample.epub")
	if err != nil {
		t.Fatal(err)
	}
	defer zr.Close()

	ep, err := Read(&zr.Reader)
	if err != nil {
		t.Fatal(err)
	}

	if len(ep.Resource) == 0 {
		t.Error("Expected some resources")
	}

	if len(ep.Package) != 1 {
		t.Errorf("Expected 1 opf, got %d", len(ep.Package))
	}

	expectedCleartext := []string{ContainerFile, "OPS/package.opf", "OPS/images/9780316000000.jpg", "OPS/toc.xhtml"}
	sort.Strings(expectedCleartext)
	if fmt.Sprintf("%v", ep.cleartextResources) != fmt.Sprintf("%v", expectedCleartext) {
		t.Errorf("Cleartext resources, expected %v, got %v", expectedCleartext, ep.cleartextResources)
	}

	if found, r := ep.Cover(); !found || r == nil {
		t.Error("Expected a cover to be found")
	}

	if expected := ContentType_XHTML; ep.Resource[2].ContentType != expected {
		t.Errorf("Content Type matching, expected %v, got %v", expected, ep.Resource[2].ContentType)
	}
}
