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

package xmlenc

import (
	"encoding/xml"
	"io"

	"golang.org/x/net/html/charset"
)

type Manifest struct {
	//Keys []Key
	Data    []Data   `xml:"http://www.w3.org/2001/04/xmlenc# EncryptedData"`
	XMLName struct{} `xml:"urn:oasis:names:tc:opendocument:xmlns:container encryption"`
}

// DataForFile returns the EncryptedData item corresponding to a given path
func (m Manifest) DataForFile(path string) (Data, bool) {
	uri := URI(path)
	for _, datum := range m.Data {
		if datum.CipherData.CipherReference.URI == uri {
			return datum, true
		}
	}

	return Data{}, false
}

// Write writes the encryption XML structure
func (m Manifest) Write(w io.Writer) error {
	w.Write([]byte(xml.Header))
	enc := xml.NewEncoder(w)
	enc.Indent("", "  ")
	return enc.Encode(m)
}

// Read parses the encryption XML structure
func Read(r io.Reader) (Manifest, error) {
	var m Manifest
	dec := xml.NewDecoder(r)
	// deal with non utf-8 xml files
	dec.CharsetReader = charset.NewReaderLabel
	err := dec.Decode(&m)

	return m, err
}

//<sequence>
//<element name="EncryptionMethod" type="xenc:EncryptionMethodType"
//minOccurs="0"/>
//<element ref="ds:KeyInfo" minOccurs="0"/>
//<element ref="xenc:CipherData"/>
//<element ref="xenc:EncryptionProperties" minOccurs="0"/>
//</sequence>
//<attribute name="Id" type="ID" use="optional"/>
//<attribute name="Type" type="anyURI" use="optional"/>
//<attribute name="MimeType" type="string" use="optional"/>
//<attribute name="Encoding" type="anyURI" use="optional"/>

type URI string

type Method struct {
	KeySize int `xml:"KeySize,omitempty"`
	//OAEPParams []byte `xml:"AOEParams,omitempty"`
	Algorithm URI `xml:"Algorithm,attr,omitempty"`
}

type CipherReference struct {
	URI URI `xml:"URI,attr"`
}

type CipherData struct {
	CipherReference CipherReference `xml:"http://www.w3.org/2001/04/xmlenc# CipherReference"`
	Value           []byte          `xml:"Value,omitempty"`
}

//type DSAKeyValue struct {
//P []byte
//Q []byte
//G []byte
//Y []byte
//J []byte
//Seed []byte
//PgenCounter []byte
//}

//type RSAKeyValue struct {
//Modulus []byte
//Exponent []byte
//}

//type KeyValue struct {
//DSAKeyValue
//RSAKeyValue
//}

type RetrievalMethod struct {
	URI  `xml:"URI,attr"`
	Type string `xml:"Type,attr"`
}

type KeyInfo struct {
	KeyName string `xml:"KeyName,attr,omitempty"`
	//KeyValue
	RetrievalMethod RetrievalMethod `xml:"http://www.w3.org/2000/09/xmldsig# RetrievalMethod"`
	//X509Data
	//PGPData
	//SPKIData
	//MgmtData
}

type encryptedType struct {
	Method     Method     `xml:"http://www.w3.org/2001/04/xmlenc# EncryptionMethod"`
	KeyInfo    *KeyInfo   `xml:"http://www.w3.org/2000/09/xmldsig# KeyInfo"`
	CipherData CipherData `xml:"http://www.w3.org/2001/04/xmlenc# CipherData"`
	Id         string     `xml:"Id,attr,omitempty"`
	Type       URI        `xml:"Type,attr,omitempty"`
	MimeType   string     `xml:"MimeType,omitempty"`
	Encoding   URI        `xml:"Encoding,omitempty"`
}

type ReferenceList struct {
	Key  []string
	Data []string
}

type Key struct {
	encryptedType
	References     ReferenceList
	CarriedKeyName string
	Recipient      string
}
type Compression struct {
	Method         int    `xml:"Method,attr"`
	OriginalLength uint64 `xml:"OriginalLength,attr"`
}

type EncryptionProperty struct {
	Compression Compression `xml:"http://www.idpf.org/2016/encryption#compression Compression"`
}

type EncryptionProperties struct {
	Properties []EncryptionProperty `xml:"http://www.w3.org/2001/04/xmlenc# EncryptionProperty"`
}

type Data struct {
	encryptedType
	Properties *EncryptionProperties `xml:"http://www.w3.org/2001/04/xmlenc# EncryptionProperties,omitempty"`
}
