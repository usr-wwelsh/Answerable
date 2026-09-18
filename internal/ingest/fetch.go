package ingest

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/usr-wwelsh/answerable/internal/parser"
)

const (
	fetchTimeout  = 15 * time.Second
	maxFetchBytes = 16 << 20
)

type Fetcher struct {
	client *http.Client
}

func NewFetcher() *Fetcher {
	return &Fetcher{client: &http.Client{Timeout: fetchTimeout}}
}

// Fetch retrieves rawURL and parses it into a Record using the same parsers
// as file upload. It only allows http/https URLs (ValidateURL).
func (f *Fetcher) Fetch(rawURL string) (parser.Record, error) {
	if err := ValidateURL(rawURL); err != nil {
		return nil, err
	}

	resp, err := f.client.Get(rawURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("fetch %s: status %d", rawURL, resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxFetchBytes))
	if err != nil {
		return nil, err
	}

	format := DetectFormat(rawURL, resp.Header.Get("Content-Type"))
	switch format {
	case FormatCSV:
		records, err := parser.ParseCSV(bytes.NewReader(body))
		if err != nil {
			return nil, err
		}
		if len(records) == 0 {
			return nil, fmt.Errorf("fetch %s: no rows found", rawURL)
		}
		return records[0], nil
	case FormatText:
		return parser.ParseText(bytes.NewReader(body))
	case FormatPDF:
		return parser.ParsePDF(bytes.NewReader(body))
	default:
		return nil, fmt.Errorf("fetch %s: could not determine format (expected CSV, text/markdown, or PDF)", rawURL)
	}
}
