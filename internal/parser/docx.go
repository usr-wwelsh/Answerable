package parser

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"strings"
)

// ParseDOCX does best-effort text extraction from a Word document: it reads
// word/document.xml out of the zip and concatenates paragraph text runs, then
// feeds the result through ParseText's "Key: value" line scanner. It does not
// understand tables, headers/footers, or embedded objects.
func ParseDOCX(r io.Reader) (Record, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("open docx: %w", err)
	}

	f := findZipFile(zr, "word/document.xml")
	if f == nil {
		return nil, fmt.Errorf("docx: word/document.xml not found")
	}

	xmlData, err := readZipFile(f)
	if err != nil {
		return nil, err
	}

	return ParseText(strings.NewReader(extractOOXMLText(xmlData)))
}
