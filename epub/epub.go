package epub

import (
  "archive/zip"
  "github.com/jpbougie/lcpserve/epub/opf"
  "github.com/jpbougie/lcpserve/xmlenc"
  "bytes"
)

type Manifest struct {

}

type Signatures struct {

}

type Rights struct {

}

type Epub struct {
  Encryption *xmlenc.Manifest
  Manifest
  Package []opf.Package
  Resource []Resource
}

type Resource struct {
  File *zip.File
  Output *bytes.Buffer
}

