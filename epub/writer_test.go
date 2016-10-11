package epub

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io/ioutil"
	"strings"
	"testing"
)

const containerSpec = `<?xml version="1.0" encoding="UTF-8"?><container xmlns="urn:oasis:names:tc:opendocument:xmlns:container" version="1.0">
<rootfiles>
<rootfile full-path="EPUB/package.opf" media-type="application/oebps-package+xml"/>
</rootfiles>
</container>`

const basicOpf = `
<?xml version="1.0" encoding="UTF-8"?>
<package xmlns="http://www.idpf.org/2007/opf" prefix="rendition: http://www.idpf.org/vocab/rendition/# cc: http://creativecommons.org/ns#" version="3.0" unique-identifier="bookid" xml:lang="fr" dir="ltr" id="package">
	<metadata xmlns:dc="http://purl.org/dc/elements/1.1/">
		<dc:identifier id="bookid">test-basic-epub</dc:identifier>
		<dc:language>fr</dc:language>
	</metadata>
	<manifest>
		<item id="page" href="page.xhtml" media-type="application/xhtml+xml"/>
	</manifest>
	<spine>
		<itemref idref="page"/>
	</spine>
</package>`

const basicPage = `
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE html>
<html>
<body>Hello</body>
</htmL>`

func createBasicEpub() Epub {
	var ep Epub
	ep.Add(ContainerFile, strings.NewReader(containerSpec), uint64(len(containerSpec)))

	ep.Add("EPUB/package.opf", strings.NewReader(basicOpf), uint64(len(basicOpf)))

	ep.Add("EPUB/page.xhtml", strings.NewReader(basicPage), uint64(len(basicPage)))

	return ep
}

func TestWriteBasicEpub(t *testing.T) {
	ep := createBasicEpub()

	// Mark the page as already-compressed
	ep.Resource[len(ep.Resource)-1].Compressed = true

	var buf bytes.Buffer
	ep.Write(&buf)

	r := bytes.NewReader(buf.Bytes())
	zr, err := zip.NewReader(r, int64(buf.Len()))

	if err != nil {
		t.Fatal("Could not read zip", err)
	}

	out, err := Read(zr)
	if err != nil {
		t.Fatal("Could not construct epub from zip", err)
	}

	if l := len(out.Resource); l != 3 {
		t.Fatalf("Expected 3 resources, got %d", l)
	}

	for i, r := range ep.Resource {
		if equiv := out.Resource[i]; r.Path != equiv.Path {
			t.Errorf("Expected %s, got %s", r.Path, equiv.Path)
		}
	}

	testContentsOfFileInZip(t, zr, zip.Store, "mimetype", ContentType_EPUB)
	testContentsOfFileInZip(t, zr, zip.Deflate, ContainerFile, containerSpec)
	testContentsOfFileInZip(t, zr, zip.Deflate, "EPUB/package.opf", basicOpf)
	testContentsOfFileInZip(t, zr, zip.Deflate, "EPUB/page.xhtml", basicPage)
}

func testContentsOfFileInZip(t *testing.T, zr *zip.Reader, m uint16, path, expected string) {
	for _, f := range zr.File {
		fmt.Println(f.Name)
	}

	if f, err := findFileInZip(zr, path); err != nil {
		t.Fatalf("Could not find %s in file", path)
	} else {
		if meth := f.FileHeader.Method; meth != m {
			t.Errorf("Expected %s to have method %d, got %d", path, m, meth)
		}

		r, err := f.Open()
		if err != nil {
			t.Fatalf("Could not open %s", path)
		}
		defer r.Close()

		if bb, err := ioutil.ReadAll(r); err != nil {
			t.Fatalf("Could not read %s", path)
		} else {
			if string(bb) != expected {
				t.Errorf("Expected %s to contain %s, got %s", path, expected, string(bb))
			}
		}
	}
}
