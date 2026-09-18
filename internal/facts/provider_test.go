package facts

import (
	"reflect"
	"testing"

	"github.com/usr-wwelsh/answerable/internal/parser"
)

func TestFromRecordSplitsNameFromProperties(t *testing.T) {
	rec := parser.Record{
		"name":               "Test Shelter A",
		"capacity_total":     "40",
		"capacity_available": "12",
		"eligibility":        "walk-in",
	}

	p, err := FromRecord(rec)
	if err != nil {
		t.Fatalf("FromRecord returned error: %v", err)
	}

	if p.Name != "Test Shelter A" {
		t.Errorf("Name = %q, want %q", p.Name, "Test Shelter A")
	}

	want := map[string]string{
		"capacity_total":     "40",
		"capacity_available": "12",
		"eligibility":        "walk-in",
	}
	if !reflect.DeepEqual(p.Properties, want) {
		t.Errorf("Properties = %v, want %v", p.Properties, want)
	}
}

func TestFromRecordErrorsWithoutName(t *testing.T) {
	rec := parser.Record{"hours": "24/7"}

	if _, err := FromRecord(rec); err == nil {
		t.Fatal("expected error for record missing name, got nil")
	}
}
