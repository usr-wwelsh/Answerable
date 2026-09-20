package factshtml

import (
	"bytes"
	"html/template"
	"sort"

	"github.com/usr-wwelsh/answerable/internal/facts"
)

type property struct {
	Name  string
	Value string
}

type page struct {
	Name       string
	Properties []property
	BookingURL string
}

var tmpl = template.Must(template.New("facts").Parse(`<!DOCTYPE html>
<html lang="en"><head><meta charset="utf-8"><title>{{.Name}} — availability</title></head>
<body>
<h1>{{.Name}}</h1>
<ul>
{{range .Properties}}<li><strong>{{.Name}}:</strong> {{.Value}}</li>
{{end}}</ul>
<p><a href="{{.BookingURL}}">Request intake</a></p>
</body></html>
`))

// Render renders the provider's facts as a plain HTML page: a fallback for
// fetchers that only follow a GET and parse HTML, and can't reach the
// structured JSON-LD, A2A, or MCP endpoints.
func Render(p facts.Provider, bookingURL string) []byte {
	keys := make([]string, 0, len(p.Properties))
	for k := range p.Properties {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	props := make([]property, 0, len(keys))
	for _, k := range keys {
		props = append(props, property{Name: k, Value: p.Properties[k]})
	}

	var buf bytes.Buffer
	tmpl.Execute(&buf, page{Name: p.Name, Properties: props, BookingURL: bookingURL})
	return buf.Bytes()
}
