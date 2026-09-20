package parser

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"io"
)

// findZipFile returns the named entry from an OOXML (zip) archive, or nil if
// absent.
func findZipFile(zr *zip.Reader, name string) *zip.File {
	for _, f := range zr.File {
		if f.Name == name {
			return f
		}
	}
	return nil
}

func readZipFile(f *zip.File) ([]byte, error) {
	rc, err := f.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	return io.ReadAll(rc)
}

// extractOOXMLText walks an OOXML part's XML and concatenates the character
// data of every "t" element (Word's w:t, PowerPoint's a:t — Go's namespace-
// aware decoder reports both as local name "t"), inserting a newline at each
// "p" (paragraph/slide-text-body) close.
func extractOOXMLText(data []byte) string {
	dec := xml.NewDecoder(bytes.NewReader(data))
	var out bytes.Buffer
	inText := false

	for {
		tok, err := dec.Token()
		if err != nil {
			break
		}
		switch t := tok.(type) {
		case xml.StartElement:
			if t.Name.Local == "t" {
				inText = true
			}
		case xml.EndElement:
			if t.Name.Local == "t" {
				inText = false
			}
			if t.Name.Local == "p" {
				out.WriteByte('\n')
			}
		case xml.CharData:
			if inText {
				out.Write(t)
			}
		}
	}
	return out.String()
}
