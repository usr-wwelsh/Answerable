package factshtml

import (
	"strings"
	"testing"

	"github.com/usr-wwelsh/answerable/internal/facts"
)

func TestRenderIncludesNamePropertiesAndBookingLink(t *testing.T) {
	p := facts.Provider{
		Name: "Test Shelter A",
		Properties: map[string]string{
			"capacity_available": "12",
			"eligibility":        "walk-in",
		},
	}

	out := string(Render(p, "http://localhost:8080/book"))

	for _, want := range []string{"Test Shelter A", "capacity_available", "12", "eligibility", "walk-in", "http://localhost:8080/book"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q: %s", want, out)
		}
	}
}

func TestRenderEscapesUntrustedPropertyValues(t *testing.T) {
	p := facts.Provider{
		Name: "<script>alert(1)</script>",
		Properties: map[string]string{
			"note": "<script>alert(2)</script>",
		},
	}

	out := string(Render(p, "http://localhost:8080/book"))

	if strings.Contains(out, "<script>") {
		t.Errorf("output contains unescaped script tag: %s", out)
	}
}

func TestRenderIsStableContentType(t *testing.T) {
	p := facts.Provider{Name: "Test Shelter A"}

	out := string(Render(p, "http://localhost:8080/book"))

	if !strings.HasPrefix(out, "<!DOCTYPE html>") {
		t.Errorf("output missing doctype: %s", out)
	}
}
