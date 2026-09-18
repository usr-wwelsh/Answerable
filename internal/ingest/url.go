package ingest

import (
	"fmt"
	"net/url"
	"path"
	"strings"
)

type Format string

const (
	FormatCSV  Format = "csv"
	FormatText Format = "text"
	FormatPDF  Format = "pdf"
)

var allowedSchemes = map[string]bool{
	"http":  true,
	"https": true,
}

// ValidateURL allows only http/https URLs with a host, by allowlist.
func ValidateURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}
	if !allowedSchemes[u.Scheme] {
		return fmt.Errorf("unsupported URL scheme %q (only http/https allowed)", u.Scheme)
	}
	if u.Host == "" {
		return fmt.Errorf("URL missing host")
	}
	return nil
}

// DetectFormat determines which parser to use, preferring signals in this
// order: file extension, query-string hints (for export links that carry no
// extension), then the HTTP response Content-Type.
func DetectFormat(raw, contentType string) Format {
	u, err := url.Parse(raw)
	if err == nil {
		switch strings.ToLower(path.Ext(u.Path)) {
		case ".csv":
			return FormatCSV
		case ".md", ".txt":
			return FormatText
		case ".pdf":
			return FormatPDF
		}

		q := u.Query()
		switch strings.ToLower(q.Get("format")) {
		case "csv":
			return FormatCSV
		case "txt", "text":
			return FormatText
		}
		if strings.ToLower(q.Get("output")) == "csv" {
			return FormatCSV
		}
	}

	mediaType := strings.ToLower(strings.TrimSpace(strings.SplitN(contentType, ";", 2)[0]))
	switch mediaType {
	case "text/csv":
		return FormatCSV
	case "application/pdf":
		return FormatPDF
	case "text/plain", "text/markdown":
		return FormatText
	}

	return ""
}
