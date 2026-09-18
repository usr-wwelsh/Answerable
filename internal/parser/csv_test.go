package parser

import (
	"strings"
	"testing"
)

func TestParseCSVMapsHeaderToRowValues(t *testing.T) {
	input := strings.NewReader("name,capacity_total,capacity_available,eligibility,hours\n" +
		"Test Shelter A,40,12,walk-in,24/7\n")

	records, err := ParseCSV(input)
	if err != nil {
		t.Fatalf("ParseCSV returned error: %v", err)
	}

	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}

	got := records[0]
	want := Record{
		"name":               "Test Shelter A",
		"capacity_total":     "40",
		"capacity_available": "12",
		"eligibility":        "walk-in",
		"hours":              "24/7",
	}

	for k, v := range want {
		if got[k] != v {
			t.Errorf("field %q = %q, want %q", k, got[k], v)
		}
	}
}

func TestParseCSVRejectsMismatchedRowLength(t *testing.T) {
	input := strings.NewReader("name,hours\nTest Shelter A,24/7,extra\n")

	if _, err := ParseCSV(input); err == nil {
		t.Fatal("expected error for row with wrong column count, got nil")
	}
}
