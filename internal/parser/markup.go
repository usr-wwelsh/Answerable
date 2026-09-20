package parser

import (
	"html"
	"io"
	"regexp"
	"strings"
)

var (
	scriptStyleRe = regexp.MustCompile(`(?is)<(script|style)\b[^>]*>.*?</(script|style)>`)
	tagRe         = regexp.MustCompile(`(?s)<[^>]*>`)
)

// ParseHTML and ParseXML do best-effort text extraction: tags are stripped
// (script/style contents dropped) and entities unescaped, then the result is
// fed through ParseText's "Key: value" line scanner. Neither understands
// layout, attributes, or malformed markup beyond what a regex tag-strip
// tolerates.
func ParseHTML(r io.Reader) (Record, error) {
	return parseMarkup(r)
}

func ParseXML(r io.Reader) (Record, error) {
	return parseMarkup(r)
}

func parseMarkup(r io.Reader) (Record, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	text := scriptStyleRe.ReplaceAllString(string(data), "\n")
	text = tagRe.ReplaceAllString(text, "\n")
	text = html.UnescapeString(text)

	return ParseText(strings.NewReader(text))
}
