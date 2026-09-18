package ingest

import "testing"

func TestValidateURLAcceptsHTTPAndHTTPS(t *testing.T) {
	for _, u := range []string{"http://example.com/data.csv", "https://example.com/data.csv"} {
		if err := ValidateURL(u); err != nil {
			t.Errorf("ValidateURL(%q) = %v, want nil", u, err)
		}
	}
}

func TestValidateURLRejectsDisallowedSchemes(t *testing.T) {
	for _, u := range []string{
		"file:///etc/passwd",
		"ftp://example.com/data.csv",
		"javascript:alert(1)",
		"not a url at all",
		"example.com/data.csv",
	} {
		if err := ValidateURL(u); err == nil {
			t.Errorf("ValidateURL(%q) = nil, want error", u)
		}
	}
}

func TestDetectFormatByExtension(t *testing.T) {
	cases := map[string]Format{
		"https://example.com/sheet.csv":   FormatCSV,
		"https://example.com/blurb.md":    FormatText,
		"https://example.com/blurb.txt":   FormatText,
		"https://example.com/handout.pdf": FormatPDF,
	}
	for u, want := range cases {
		if got := DetectFormat(u, ""); got != want {
			t.Errorf("DetectFormat(%q) = %q, want %q", u, got, want)
		}
	}
}

func TestDetectFormatByQueryParam(t *testing.T) {
	cases := map[string]Format{
		"https://docs.google.com/spreadsheets/d/x/export?format=csv":  FormatCSV,
		"https://docs.google.com/spreadsheets/d/x/gviz/tq?output=csv": FormatCSV,
		"https://docs.google.com/document/d/x/export?format=txt":      FormatText,
	}
	for u, want := range cases {
		if got := DetectFormat(u, ""); got != want {
			t.Errorf("DetectFormat(%q) = %q, want %q", u, got, want)
		}
	}
}

func TestDetectFormatByContentType(t *testing.T) {
	cases := []struct {
		contentType string
		want        Format
	}{
		{"text/csv", FormatCSV},
		{"text/csv; charset=utf-8", FormatCSV},
		{"application/pdf", FormatPDF},
		{"text/plain", FormatText},
		{"text/markdown", FormatText},
	}
	for _, c := range cases {
		if got := DetectFormat("https://example.com/download", c.contentType); got != c.want {
			t.Errorf("DetectFormat(_, %q) = %q, want %q", c.contentType, got, c.want)
		}
	}
}

func TestDetectFormatUnknownReturnsEmpty(t *testing.T) {
	if got := DetectFormat("https://example.com/download", "application/octet-stream"); got != "" {
		t.Errorf("DetectFormat = %q, want empty", got)
	}
}
