package parser

import (
	"bytes"
	"testing"
)

func TestParsePPTXExtractsTextInSlideOrder(t *testing.T) {
	slide1 := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<p:sld xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main" xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main">
  <p:cSld><p:spTree><p:sp><p:txBody><a:p><a:r><a:t>Name: </a:t></a:r><a:r><a:t>Test Shelter</a:t></a:r></a:p></p:txBody></p:sp></p:spTree></p:cSld>
</p:sld>`
	slide2 := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<p:sld xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main" xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main">
  <p:cSld><p:spTree><p:sp><p:txBody><a:p><a:r><a:t>Hours: 8pm-7am</a:t></a:r></a:p></p:txBody></p:sp></p:spTree></p:cSld>
</p:sld>`

	pptx := buildZip(t, map[string]string{
		"ppt/slides/slide1.xml": slide1,
		"ppt/slides/slide2.xml": slide2,
	})

	rec, err := ParsePPTX(bytes.NewReader(pptx))
	if err != nil {
		t.Fatalf("ParsePPTX: %v", err)
	}
	if rec["name"] != "Test Shelter" {
		t.Errorf("name = %q, want %q", rec["name"], "Test Shelter")
	}
	if rec["hours"] != "8pm-7am" {
		t.Errorf("hours = %q, want %q", rec["hours"], "8pm-7am")
	}
}

func TestParsePPTXRejectsNoSlides(t *testing.T) {
	pptx := buildZip(t, map[string]string{"ppt/presentation.xml": "<p:presentation/>"})

	if _, err := ParsePPTX(bytes.NewReader(pptx)); err == nil {
		t.Fatal("expected error when no slides present")
	}
}
