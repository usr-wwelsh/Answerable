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
		"https://example.com/data.json":   FormatJSON,
		"https://example.com/page.html":   FormatHTML,
		"https://example.com/page.htm":    FormatHTML,
		"https://example.com/feed.xml":    FormatXML,
		"https://example.com/doc.docx":    FormatDOCX,
		"https://example.com/deck.pptx":   FormatPPTX,
		"https://example.com/book.xlsx":   FormatXLSX,
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
		{"application/json", FormatJSON},
		{"text/html", FormatHTML},
		{"application/xml", FormatXML},
		{"text/xml", FormatXML},
		{"application/vnd.openxmlformats-officedocument.wordprocessingml.document", FormatDOCX},
		{"application/vnd.openxmlformats-officedocument.presentationml.presentation", FormatPPTX},
		{"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", FormatXLSX},
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

func TestNormalizeSourceURLRewritesGoogleSheetsEditLink(t *testing.T) {
	cases := map[string]string{
		"https://docs.google.com/spreadsheets/d/1zP_rbjm1eIZAdSd8a7dksWpWfrBCCoOU72YD0eEs9cc/edit?usp=sharing": "https://docs.google.com/spreadsheets/d/1zP_rbjm1eIZAdSd8a7dksWpWfrBCCoOU72YD0eEs9cc/export?format=csv",
		"https://docs.google.com/spreadsheets/d/abc123/edit":                                                   "https://docs.google.com/spreadsheets/d/abc123/export?format=csv",
		"https://docs.google.com/spreadsheets/d/abc123":                                                        "https://docs.google.com/spreadsheets/d/abc123/export?format=csv",
	}
	for in, want := range cases {
		if got := NormalizeSourceURL(in); got != want {
			t.Errorf("NormalizeSourceURL(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestNormalizeSourceURLPreservesSheetTabViaGID(t *testing.T) {
	cases := map[string]string{
		"https://docs.google.com/spreadsheets/d/abc123/edit#gid=98765":             "https://docs.google.com/spreadsheets/d/abc123/export?format=csv&gid=98765",
		"https://docs.google.com/spreadsheets/d/abc123/edit?usp=sharing#gid=98765": "https://docs.google.com/spreadsheets/d/abc123/export?format=csv&gid=98765",
		"https://docs.google.com/spreadsheets/d/abc123/edit?gid=98765&usp=sharing": "https://docs.google.com/spreadsheets/d/abc123/export?format=csv&gid=98765",
	}
	for in, want := range cases {
		if got := NormalizeSourceURL(in); got != want {
			t.Errorf("NormalizeSourceURL(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestNormalizeSourceURLLeavesNonSheetsURLsUnchanged(t *testing.T) {
	for _, u := range []string{
		"https://docs.google.com/spreadsheets/d/abc123/export?format=csv",
		"https://docs.google.com/spreadsheets/d/abc123/gviz/tq?output=csv",
		"https://docs.google.com/document/d/abc123/edit",
		"https://example.com/data.csv",
		"not a url at all",
	} {
		if got := NormalizeSourceURL(u); got != u {
			t.Errorf("NormalizeSourceURL(%q) = %q, want unchanged", u, got)
		}
	}
}
