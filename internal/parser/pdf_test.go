package parser

import (
	"bytes"
	"compress/zlib"
	"strconv"
	"strings"
	"testing"
)

func TestParsePDFExtractsLiteralStringOperators(t *testing.T) {
	content := "BT /F1 12 Tf (Name: Test Shelter) Tj ET"
	pdf := wrapPDFStream(t, []byte(content), false)

	rec, err := ParsePDF(bytes.NewReader(pdf))
	if err != nil {
		t.Fatalf("ParsePDF: %v", err)
	}
	if got := rec["name"]; got != "Test Shelter" {
		t.Fatalf("name = %q, want %q", got, "Test Shelter")
	}
}

func TestParsePDFExtractsTJArrayOperators(t *testing.T) {
	content := "BT /F1 12 Tf [(Beds: )-20(12)] TJ ET"
	pdf := wrapPDFStream(t, []byte(content), false)

	rec, err := ParsePDF(bytes.NewReader(pdf))
	if err != nil {
		t.Fatalf("ParsePDF: %v", err)
	}
	if got := rec["beds"]; got != "12" {
		t.Fatalf("beds = %q, want %q", got, "12")
	}
}

func TestParsePDFHandlesEscapedParens(t *testing.T) {
	content := `BT (Notes: call \(555\) 0100) Tj ET`
	pdf := wrapPDFStream(t, []byte(content), false)

	rec, err := ParsePDF(bytes.NewReader(pdf))
	if err != nil {
		t.Fatalf("ParsePDF: %v", err)
	}
	if got := rec["notes"]; got != "call (555) 0100" {
		t.Fatalf("notes = %q, want %q", got, "call (555) 0100")
	}
}

func TestParsePDFDecompressesFlateStreams(t *testing.T) {
	content := "BT (Name: Compressed Shelter) Tj ET"
	pdf := wrapPDFStream(t, []byte(content), true)

	rec, err := ParsePDF(bytes.NewReader(pdf))
	if err != nil {
		t.Fatalf("ParsePDF: %v", err)
	}
	if got := rec["name"]; got != "Compressed Shelter" {
		t.Fatalf("name = %q, want %q", got, "Compressed Shelter")
	}
}

func wrapPDFStream(t *testing.T, content []byte, compress bool) []byte {
	t.Helper()

	body := content
	filter := ""
	if compress {
		var buf bytes.Buffer
		zw := zlib.NewWriter(&buf)
		if _, err := zw.Write(content); err != nil {
			t.Fatalf("zlib write: %v", err)
		}
		if err := zw.Close(); err != nil {
			t.Fatalf("zlib close: %v", err)
		}
		body = buf.Bytes()
		filter = " /Filter /FlateDecode"
	}

	var out strings.Builder
	out.WriteString("%PDF-1.4\n")
	out.WriteString("1 0 obj\n")
	out.WriteString("<< /Length " + strconv.Itoa(len(body)) + filter + " >>\n")
	out.WriteString("stream\n")
	return []byte(out.String() + string(body) + "\nendstream\nendobj\n")
}
