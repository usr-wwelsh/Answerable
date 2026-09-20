package parser

import (
	"strings"
	"testing"
)

func TestParseHTMLExtractsTextFromTags(t *testing.T) {
	input := strings.NewReader(`<html><body><p>Name: Test Shelter</p><p>Hours: 8pm-7am</p></body></html>`)

	rec, err := ParseHTML(input)
	if err != nil {
		t.Fatalf("ParseHTML: %v", err)
	}
	if rec["name"] != "Test Shelter" || rec["hours"] != "8pm-7am" {
		t.Fatalf("rec = %+v", rec)
	}
}

func TestParseHTMLSkipsScriptAndStyle(t *testing.T) {
	input := strings.NewReader(`<html><style>.x{color:Red:1}</style><script>var x = "Ignore: this";</script><p>Name: Test Shelter</p></html>`)

	rec, err := ParseHTML(input)
	if err != nil {
		t.Fatalf("ParseHTML: %v", err)
	}
	if rec["name"] != "Test Shelter" {
		t.Fatalf("rec = %+v", rec)
	}
	if _, ok := rec["ignore"]; ok {
		t.Fatalf("script content leaked into rec: %+v", rec)
	}
}

func TestParseHTMLUnescapesEntities(t *testing.T) {
	input := strings.NewReader(`<p>Name: Fish &amp; Chips Shelter</p>`)

	rec, err := ParseHTML(input)
	if err != nil {
		t.Fatalf("ParseHTML: %v", err)
	}
	if rec["name"] != "Fish & Chips Shelter" {
		t.Fatalf("name = %q", rec["name"])
	}
}

func TestParseXMLExtractsTextFromElements(t *testing.T) {
	input := strings.NewReader(`<shelter><name>Name: Test Shelter</name><hours>Hours: 8pm-7am</hours></shelter>`)

	rec, err := ParseXML(input)
	if err != nil {
		t.Fatalf("ParseXML: %v", err)
	}
	if rec["name"] != "Test Shelter" || rec["hours"] != "8pm-7am" {
		t.Fatalf("rec = %+v", rec)
	}
}
