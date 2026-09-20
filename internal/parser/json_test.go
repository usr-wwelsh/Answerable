package parser

import (
	"strings"
	"testing"
)

func TestParseJSONExtractsFlatObject(t *testing.T) {
	input := strings.NewReader(`{"Name": "Test Shelter", "Capacity Total": 40}`)

	rec, err := ParseJSON(input)
	if err != nil {
		t.Fatalf("ParseJSON: %v", err)
	}
	if rec["name"] != "Test Shelter" {
		t.Errorf("name = %q, want %q", rec["name"], "Test Shelter")
	}
	if rec["capacity_total"] != "40" {
		t.Errorf("capacity_total = %q, want %q", rec["capacity_total"], "40")
	}
}

func TestParseJSONFlattensNestedObjects(t *testing.T) {
	input := strings.NewReader(`{"name": "Test Shelter", "hours": {"open": "8pm", "close": "7am"}}`)

	rec, err := ParseJSON(input)
	if err != nil {
		t.Fatalf("ParseJSON: %v", err)
	}
	if rec["hours.open"] != "8pm" || rec["hours.close"] != "7am" {
		t.Fatalf("rec = %+v", rec)
	}
}

func TestParseJSONTakesFirstElementOfArray(t *testing.T) {
	input := strings.NewReader(`[{"name": "Test Shelter A"}, {"name": "Test Shelter B"}]`)

	rec, err := ParseJSON(input)
	if err != nil {
		t.Fatalf("ParseJSON: %v", err)
	}
	if rec["name"] != "Test Shelter A" {
		t.Fatalf("name = %q, want %q", rec["name"], "Test Shelter A")
	}
}

func TestParseJSONRejectsEmptyArray(t *testing.T) {
	if _, err := ParseJSON(strings.NewReader(`[]`)); err == nil {
		t.Fatal("expected error for empty array")
	}
}

func TestParseJSONRejectsInvalidJSON(t *testing.T) {
	if _, err := ParseJSON(strings.NewReader(`not json`)); err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}
