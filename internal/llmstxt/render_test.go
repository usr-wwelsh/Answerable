package llmstxt

import (
	"strings"
	"testing"

	"github.com/usr-wwelsh/answerable/internal/facts"
)

func TestRenderIncludesNameAndLinks(t *testing.T) {
	p := facts.Provider{Name: "Test Shelter A"}

	out := Render(p, "http://localhost:8080/facts.jsonld", "http://localhost:8080/.well-known/agent.json", "http://localhost:8080/mcp")

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
}
