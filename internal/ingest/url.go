package ingest

import (
	"fmt"
	"net/url"
	"path"
	"regexp"
	"strings"
)

type Format string

const (
	FormatCSV  Format = "csv"
	FormatText Format = "text"
	FormatPDF  Format = "pdf"
	FormatJSON Format = "json"
	FormatHTML Format = "html"
	FormatXML  Format = "xml"
	FormatDOCX Format = "docx"
	FormatPPTX Format = "pptx"
	FormatXLSX Format = "xlsx"
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
		case ".json":
			return FormatJSON
		case ".html", ".htm":
			return FormatHTML
		case ".xml":
			return FormatXML
		case ".docx":
			return FormatDOCX
		case ".pptx":
			return FormatPPTX
		case ".xlsx":
			return FormatXLSX
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
	case "application/json":
		return FormatJSON
	case "text/html":
		return FormatHTML
	case "text/xml", "application/xml":
		return FormatXML
	case "application/vnd.openxmlformats-officedocument.wordprocessingml.document":
		return FormatDOCX
	case "application/vnd.openxmlformats-officedocument.presentationml.presentation":
		return FormatPPTX
	case "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet":
		return FormatXLSX
	}

	return ""
}

var googleSheetsPath = regexp.MustCompile(`^/spreadsheets/d/([a-zA-Z0-9_-]+)(?:/edit)?/?$`)

// NormalizeSourceURL rewrites a Google Sheets "share" or "edit" link (the
// kind Google's UI hands you: .../spreadsheets/d/<id>/edit?usp=sharing) into
// its CSV export link, so onboarding accepts the URL users actually copy out
// of Sheets and periodic refresh keeps pulling the live sheet. Links that
// already point at a specific export/gviz endpoint, or aren't Google Sheets
// links at all, are returned unchanged.
func NormalizeSourceURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || u.Host != "docs.google.com" {
		return raw
	}

	m := googleSheetsPath.FindStringSubmatch(u.Path)
	if m == nil {
		return raw
	}
	id := m[1]

	gid := u.Query().Get("gid")
	if gid == "" {
		if frag, err := url.ParseQuery(u.Fragment); err == nil {
			gid = frag.Get("gid")
		}
	}

	out := fmt.Sprintf("https://docs.google.com/spreadsheets/d/%s/export?format=csv", id)
	if gid != "" {
		out += "&gid=" + gid
	}
	return out
}
