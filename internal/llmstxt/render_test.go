package llmstxt

import (
	"strings"
	"testing"

	"github.com/usr-wwelsh/answerable/internal/facts"
)

func TestRenderIncludesNameAndLinks(t *testing.T) {
	p := facts.Provider{Name: "Test Shelter A"}

	out := Render(p, "http://localhost:8080/facts.jsonld", "http://localhost:8080/.well-known/agent.json", "http://localhost:8080/mcp", "http://localhost:8080/facts.html")

	if !strings.Contains(out, "Test Shelter A") {
		t.Errorf("output missing provider name: %q", out)
	}
	if !strings.Contains(out, "http://localhost:8080/facts.jsonld") {
		t.Errorf("output missing JSON-LD link: %q", out)
	}
	if !strings.Contains(out, "http://localhost:8080/.well-known/agent.json") {
		t.Errorf("output missing agent card link: %q", out)
	}
	if !strings.Contains(out, "http://localhost:8080/mcp") {
		t.Errorf("output missing MCP endpoint link: %q", out)
	}
	if !strings.Contains(out, "http://localhost:8080/facts.html") {
		t.Errorf("output missing HTML fallback link: %q", out)
	}
}

func TestRenderInlinesCurrentFacts(t *testing.T) {
	p := facts.Provider{
		Name: "Test Shelter A",
		Properties: map[string]string{
			"capacity_available": "2",
			"capacity_total":     "40",
			"eligibility":        "walk-in",
		},
	}

	out := Render(p, "http://localhost:8080/facts.jsonld", "http://localhost:8080/.well-known/agent.json", "http://localhost:8080/mcp", "http://localhost:8080/facts.html")

	for _, want := range []string{"capacity_available: 2", "capacity_total: 40", "eligibility: walk-in"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing inlined fact %q: %q", want, out)
		}
	}
}
