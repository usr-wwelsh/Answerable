package parser

import (
	"strings"
	"testing"
)

func TestParseByExtensionDispatchesCSV(t *testing.T) {
	rec, err := ParseByExtension(".csv", strings.NewReader("name,beds\nTest Shelter,12\n"))
	if err != nil {
		t.Fatalf("ParseByExtension: %v", err)
	}
	if rec["name"] != "Test Shelter" || rec["beds"] != "12" {
		t.Fatalf("rec = %+v", rec)
	}
}

func TestParseByExtensionDispatchesText(t *testing.T) {
	for _, ext := range []string{".md", ".txt"} {
		rec, err := ParseByExtension(ext, strings.NewReader("Name: Test Shelter\n"))
		if err != nil {
			t.Fatalf("ParseByExtension(%q): %v", ext, err)
		}
		if rec["name"] != "Test Shelter" {
			t.Fatalf("ext %q rec = %+v", ext, rec)
		}
	}
}

func TestParseByExtensionRejectsUnsupported(t *testing.T) {
	if _, err := ParseByExtension(".rtf", strings.NewReader("")); err == nil {
		t.Fatal("expected error for unsupported extension")
	}
}

func TestParseByExtensionDispatchesNewFormats(t *testing.T) {
	rec, err := ParseByExtension(".json", strings.NewReader(`{"name": "Test Shelter"}`))
	if err != nil {
		t.Fatalf("ParseByExtension(.json): %v", err)
	}
	if rec["name"] != "Test Shelter" {
		t.Fatalf("rec = %+v", rec)
	}

	rec, err = ParseByExtension(".html", strings.NewReader(`<p>Name: Test Shelter</p>`))
	if err != nil {
		t.Fatalf("ParseByExtension(.html): %v", err)
	}
	if rec["name"] != "Test Shelter" {
		t.Fatalf("rec = %+v", rec)
	}

	rec, err = ParseByExtension(".xml", strings.NewReader(`<s>Name: Test Shelter</s>`))
	if err != nil {
		t.Fatalf("ParseByExtension(.xml): %v", err)
	}
	if rec["name"] != "Test Shelter" {
		t.Fatalf("rec = %+v", rec)
	}
}

func TestParseByExtensionRejectsEmptyCSV(t *testing.T) {
	if _, err := ParseByExtension(".csv", strings.NewReader("name,beds\n")); err == nil {
		t.Fatal("expected error for CSV with no data rows")
	}
}
