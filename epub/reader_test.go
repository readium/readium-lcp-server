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
		t.Error("Expected 1 opf, got %d", len(ep.Package))
	}

	expectedCleartext := []string{"META-INF/container.xml", "OPS/package.opf", "OPS/images/9780316000000.jpg", "OPS/toc.xhtml"}
	sort.Strings(expectedCleartext)
	if fmt.Sprintf("%v", ep.cleartextResources) != fmt.Sprintf("%v", expectedCleartext) {
		t.Errorf("Cleartext resources, expected %v, got %v", expectedCleartext, ep.cleartextResources)
	}

	if found, r := ep.Cover(); !found || r == nil {
		t.Error("Expected a cover to be found")
	}

	if expected := "application/xhtml+xml"; ep.Resource[2].ContentType != expected {
		t.Errorf("Content Type matching, expected %v, got %v", expected, ep.Resource[2].ContentType)
	}
}
