package llmstxt

import (
	"fmt"

	"github.com/usr-wwelsh/answerable/internal/facts"
)

func Render(p facts.Provider, jsonldURL, agentCardURL, mcpURL string) string {
	return fmt.Sprintf(
		"# %s\n\n> Live, agent-queryable availability and eligibility facts published by %s.\n\n"+
			"- [Facts (JSON-LD)](%s)\n"+
			"- [Agent Card (A2A)](%s)\n"+
			"- [MCP endpoint](%s)\n",
		p.Name, p.Name, jsonldURL, agentCardURL, mcpURL,
	)
}
