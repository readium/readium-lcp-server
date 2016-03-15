package pack

import (
	"archive/zip"
	"bytes"
	"compress/flate"
	"io/ioutil"
	"testing"

	"github.com/readium/readium-lcp-server/crypto"
	"github.com/readium/readium-lcp-server/epub"
	"github.com/readium/readium-lcp-server/xmlenc"
)

func TestPacking(t *testing.T) {
	z, err := zip.OpenReader("../test/samples/sample.epub")
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	input, _ := epub.Read(&z.Reader)

	// keep a raw html file for future use
	htmlFilePath := "OPS/chapter_001.xhtml"
	inputRes, ok := findFile(htmlFilePath, input)
	if !ok {
		t.Fatalf("Could not find %s in input", htmlFilePath)
	}
	inputBytes, err := ioutil.ReadAll(inputRes.Contents)
	if err != nil {
		t.Fatalf("Could not find %s in input", htmlFilePath)
	}

	inputRes.Contents = bytes.NewReader(inputBytes)

	buf := new(bytes.Buffer)
	encryption, key, err := Do(input, buf)
	if err != nil {
		t.Fatal(err)
	}
	if encryption == nil {
		t.Fatal("Expected an xmlenc file")
	}

	if len(encryption.Data) == 0 {
		t.Error("Expected some encrypted data")
	}

	if key == nil {
		t.Error("expected a key")
	}

	for _, item := range encryption.Data {
		if !input.CanEncrypt(string(item.CipherData.CipherReference.URI)) {
			t.Errorf("Should not have encrypted %s\n", item.CipherData.CipherReference.URI)
		}
	}

	zr, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		t.Fatal(err)
	}
	output, _ := epub.Read(zr)

	if res, ok := findFile("OPS/images/Moby-Dick_FE_title_page.jpg", output); !ok {
		t.Errorf("Could not find image")
	} else {
		if res.Compressed {
			t.Errorf("Did not expect image to be compressed")
		}
	}

	//fmt.Printf("%#v\n", output.Encryption)
	var encryptionData xmlenc.Data
	for _, data := range output.Encryption.Data {
		if data.CipherData.CipherReference.URI == xmlenc.URI(htmlFilePath) {
			encryptionData = data
			break
		}
	}

	if l := encryptionData.Properties.Properties[0].Compression.OriginalLength; l != 13877 {
		t.Errorf("Expected %s to have an original length of %d, got %d", htmlFilePath, 13877, l)
	}

	if res, ok := findFile(htmlFilePath, output); !ok {
		t.Errorf("Could not find html file")
	} else {
		if !res.Compressed {
			t.Errorf("Expected html to be compressed")
		}

		var buf bytes.Buffer
		crypto.Decrypt(key, res.Contents, &buf)
		if outputBytes, err := ioutil.ReadAll(flate.NewReader(&buf)); err != nil {
			t.Fatalf("Could not decompress data from %s", htmlFilePath)
		} else {

			if !bytes.Equal(inputBytes, outputBytes) {
				t.Errorf("Expected the files to be equal before and after")
			}
		}
	}
}
