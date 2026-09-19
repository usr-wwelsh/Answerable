// Package siteproxy serves or proxies to the provider's existing site —
// the one Answerable is bolted onto — and injects agent-discovery <link>
// tags into every HTML page's <head> on the way out. This is what makes a
// provider's whole site agent-discoverable automatically: an agent landing
// on the homepage finds llms.txt, the agent card, and the MCP endpoint
// without the provider hand-editing their template or an agent being told
// where to look first.
package siteproxy

import (
	"bytes"
	"fmt"
	"regexp"
)

type DiscoveryLink struct {
	Rel  string
	Href string
}

// DefaultLinks are root-relative, so they resolve correctly regardless of
// host or scheme and need no request-time information to build.
var DefaultLinks = []DiscoveryLink{
	{"llms-txt", "/llms.txt"},
	{"agent-card", "/.well-known/agent.json"},
	{"mcp-manifest", "/.well-known/mcp.json"},
	{"mcp-server", "/mcp"},
}

var headOpenTag = regexp.MustCompile(`(?i)<head[^>]*>`)
var bodyCloseTag = regexp.MustCompile(`(?i)</body\s*>`)

// InjectLinks inserts a <link> tag for each discovery link immediately
// after the page's opening <head> tag, and a small visible <a> link to
// llms.txt immediately before the closing </body> tag. The <head> links
// are the machine-readable signal; the visible link matters because most
// LLM browsing tools extract only a page's readable text and body links,
// not <head> metadata, so a crawler that never inspects <head> still has
// something to click. It leaves body untouched if the relevant tag isn't
// found, so a non-HTML or malformed body passes through rather than
// getting corrupted.
func InjectLinks(body []byte, links []DiscoveryLink) []byte {
	body = injectHeadLinks(body, links)
	body = injectVisibleFooterLink(body, links)
	return body
}

func injectHeadLinks(body []byte, links []DiscoveryLink) []byte {
	loc := headOpenTag.FindIndex(body)
	if loc == nil {
		return body
	}

	var tags bytes.Buffer
	for _, l := range links {
		fmt.Fprintf(&tags, `<link rel="%s" href="%s">`, l.Rel, l.Href)
	}

	out := make([]byte, 0, len(body)+tags.Len())
	out = append(out, body[:loc[1]]...)
	out = append(out, tags.Bytes()...)
	out = append(out, body[loc[1]:]...)
	return out
}

func injectVisibleFooterLink(body []byte, links []DiscoveryLink) []byte {
	loc := bodyCloseTag.FindIndex(body)
	if loc == nil {
		return body
	}

	var href string
	for _, l := range links {
		if l.Rel == "llms-txt" {
			href = l.Href
			break
		}
	}
	if href == "" {
		return body
	}

	footer := fmt.Sprintf(`<p style="font-size:0.75em;opacity:0.6"><a href="%s">Agent/API data</a></p>`, href)

	out := make([]byte, 0, len(body)+len(footer))
	out = append(out, body[:loc[0]]...)
	out = append(out, footer...)
	out = append(out, body[loc[0]:]...)
	return out
}
