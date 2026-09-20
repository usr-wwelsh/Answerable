package parser

import (
	"archive/zip"
	"bytes"
	"testing"
)

func TestParseDOCXExtractsParagraphText(t *testing.T) {
	documentXML := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:body>
    <w:p><w:r><w:t>Name: Test Shelter</w:t></w:r></w:p>
    <w:p><w:r><w:t>Hours: </w:t></w:r><w:r><w:t>8pm-7am</w:t></w:r></w:p>
  </w:body>
</w:document>`

	docx := buildZip(t, map[string]string{"word/document.xml": documentXML})

	rec, err := ParseDOCX(bytes.NewReader(docx))
	if err != nil {
		t.Fatalf("ParseDOCX: %v", err)
	}
	if rec["name"] != "Test Shelter" {
		t.Errorf("name = %q, want %q", rec["name"], "Test Shelter")
	}
	if rec["hours"] != "8pm-7am" {
		t.Errorf("hours = %q, want %q", rec["hours"], "8pm-7am")
	}
}

func TestParseDOCXRejectsNonZip(t *testing.T) {
	if _, err := ParseDOCX(bytes.NewReader([]byte("not a zip"))); err == nil {
		t.Fatal("expected error for invalid docx")
	}
}

func buildZip(t *testing.T, files map[string]string) []byte {
	t.Helper()

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, content := range files {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatalf("zip create %s: %v", name, err)
		}
		if _, err := w.Write([]byte(content)); err != nil {
			t.Fatalf("zip write %s: %v", name, err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("zip close: %v", err)
	}
	return buf.Bytes()
}
