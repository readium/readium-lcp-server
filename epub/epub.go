package epub

import (
	"archive/zip"
	"bytes"
	"github.com/jpbougie/lcpserve/epub/opf"
	"github.com/jpbougie/lcpserve/xmlenc"
)

type Signatures struct {
}

type Rights struct {
}

type Epub struct {
	Encryption *xmlenc.Manifest
	Package    []opf.Package
	Resource   []Resource
}

func (ep Epub) Cover() (bool, *Resource) {
	for _, p := range ep.Package {
		for _, it := range p.Manifest.Items {
			if it.Id == "cover-image" {
				for _, r := range ep.Resource {
					if r.File.Name == it.Href {
						return true, &r
					}
				}
			}
		}
	}

	return false, nil
}

type Resource struct {
	File   *zip.File
	Output *bytes.Buffer
}
