/*
 * Copyright (c) 2016-2018 Readium Foundation
 *
 * Redistribution and use in source and binary forms, with or without modification,
 * are permitted provided that the following conditions are met:
 *
 *  1. Redistributions of source code must retain the above copyright notice, this
 *     list of conditions and the following disclaimer.
 *  2. Redistributions in binary form must reproduce the above copyright notice,
 *     this list of conditions and the following disclaimer in the documentation and/or
 *     other materials provided with the distribution.
 *  3. Neither the name of the organization nor the names of its contributors may be
 *     used to endorse or promote products derived from this software without specific
 *     prior written permission
 *
 *  THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND
 *  ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED
 *  WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
 *  DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT OWNER OR CONTRIBUTORS BE LIABLE FOR
 *  ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
 *  (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
 *  LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND
 *  ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
 *  (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
 *  SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
 */

package epub

import (
	"archive/zip"
	"encoding/xml"
	"io"
)

const (
	ContainerFile    = "META-INF/container.xml"
	EncryptionFile   = "META-INF/encryption.xml"
	LicenseFile      = "META-INF/license.lcpl"
	ContentTypeXhtml = "application/xhtml+xml"
	ContentTypeHtml  = "text/html"
	ContentTypeNcx   = "application/x-dtbncx+xml"
	ContentTypeEpub  = "application/epub+zip"
	RootFileElement  = "rootfile"
)

type (
	Writer struct {
		w *zip.Writer
	}

	rootFile struct {
		FullPath  string `xml:"full-path,attr"`
		MediaType string `xml:"media-type,attr"`
	}
	Epub struct {
		Encryption         *XMLManifest
		Package            []Package
		Resource           []*Resource
		cleartextResources []string
	}
	Resource struct {
		Path          string
		ContentType   string
		OriginalSize  uint64
		ContentsSize  uint64
		Compressed    bool
		StorageMethod uint16
		Contents      io.Reader
	}
	Package struct {
		BasePath string   `xml:"-"`
		Metadata Metadata `xml:"http://www.idpf.org/2007/opf metadata"`
		Manifest Manifest `xml:"http://www.idpf.org/2007/opf manifest"`
	}
	Metadata struct {
		Author string `json:"author" xml:"http://purl.org/dc/elements/1.1/ creator"`
		Title  string `json:"title" xml:"http://purl.org/dc/elements/1.1/ title"`
		Isbn   string `json:"isbn" xml:"http://purl.org/dc/elements/1.1/ identifier"`
		Metas  []Meta `xml:"http://www.idpf.org/2007/opf meta"`
		Cover  string `json:"cover"`
	}

	Manifest struct {
		XMLName xml.Name
		Items   []Item `xml:"http://www.idpf.org/2007/opf item"`
	}

	Item struct {
		Id         string `xml:"id,attr"`
		Href       string `xml:"href,attr"`
		MediaType  string `xml:"media-type,attr"`
		Properties string `xml:"properties,attr"`
	}

	Meta struct {
		Name    string `xml:"name,attr"`
		Content string `xml:"content,attr"`
	}
)
