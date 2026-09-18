package parser

import (
	"strings"
	"testing"
)

func TestParseTextExtractsKeyValueLines(t *testing.T) {
	input := strings.NewReader("Name: Test Shelter B\nHours: 6pm-8am\nEligibility: referral only\n")

	rec, err := ParseText(input)
	if err != nil {
		t.Fatalf("ParseText returned error: %v", err)
	}

	want := Record{
		"name":        "Test Shelter B",
		"hours":       "6pm-8am",
		"eligibility": "referral only",
	}
	for k, v := range want {
		if rec[k] != v {
			t.Errorf("field %q = %q, want %q", k, rec[k], v)
		}
	}
}

func TestParseTextStripsMarkdownAroundKeys(t *testing.T) {
	input := strings.NewReader("**Name:** Test Shelter B\n- **Hours:** 6pm-8am\n# Eligibility: referral only\n")

	rec, err := ParseText(input)
	if err != nil {
		t.Fatalf("ParseText returned error: %v", err)
	}

	want := Record{
		"name":        "Test Shelter B",
		"hours":       "6pm-8am",
		"eligibility": "referral only",
	}
	for k, v := range want {
		if rec[k] != v {
			t.Errorf("field %q = %q, want %q", k, rec[k], v)
		}
	}
}

func TestParseTextIgnoresLinesWithoutColon(t *testing.T) {
	input := strings.NewReader("Just a sentence with no colon\nName: Test Shelter B\n")

	rec, err := ParseText(input)
	if err != nil {
		t.Fatalf("ParseText returned error: %v", err)
	}

	if len(rec) != 1 {
		t.Fatalf("expected 1 field, got %d: %v", len(rec), rec)
	}
	if rec["name"] != "Test Shelter B" {
		t.Errorf("name = %q, want %q", rec["name"], "Test Shelter B")
	}
}
