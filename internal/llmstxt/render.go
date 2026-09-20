package llmstxt

import (
	"fmt"
	"sort"
	"strings"

	"github.com/usr-wwelsh/answerable/internal/facts"
)

func Render(p facts.Provider, jsonldURL, agentCardURL, mcpManifestURL, mcpURL, factsHTMLURL string) string {
	return fmt.Sprintf(
		"# %s\n\n> Live, agent-queryable availability and eligibility facts published by %s.\n\n"+
			"## Current facts\n\n%s\n"+
			"- [Facts (JSON-LD)](%s)\n"+
			"- [Agent Card (A2A)](%s)\n"+
			"- [MCP manifest](%s)\n"+
			"- [MCP endpoint](%s)\n"+
			"- [Facts (HTML fallback)](%s) — plain HTML, for fetchers that can't reach the routes above\n",
		p.Name, p.Name, renderProperties(p.Properties), jsonldURL, agentCardURL, mcpManifestURL, mcpURL, factsHTMLURL,
	)
}

// renderProperties inlines the provider's current facts as plain text so a
// fetcher that can only read llms.txt itself — and can't follow links to
// the JSON-LD, agent card, or MCP endpoints — still gets the live data.
func renderProperties(props map[string]string) string {
	keys := make([]string, 0, len(props))
	for k := range props {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var b strings.Builder
	for _, k := range keys {
		fmt.Fprintf(&b, "- %s: %s\n", k, props[k])
	}
	b.WriteString("\n")
	return b.String()
}
