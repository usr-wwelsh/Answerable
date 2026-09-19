package siteproxy

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func writeTestFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
}

func TestFileServerInjectsLinksIntoHTML(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, dir, "index.html", "<html><head><title>Harbor House</title></head><body>hi</body></html>")

	h := FileServer(dir, []DiscoveryLink{{"llms-txt", "/llms.txt"}})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `<link rel="llms-txt" href="/llms.txt">`) {
		t.Errorf("discovery link not injected: %s", rec.Body.String())
	}
}

func TestFileServerLeavesNonHTMLUntouched(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, dir, "style.css", "body { color: red; }")

	h := FileServer(dir, DefaultLinks)

	req := httptest.NewRequest(http.MethodGet, "/style.css", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if rec.Body.String() != "body { color: red; }" {
		t.Errorf("non-HTML body was modified: %s", rec.Body.String())
	}
}

func TestFileServerSetsCorrectContentLengthAfterInjection(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, dir, "index.html", "<html><head></head><body></body></html>")

	h := FileServer(dir, []DiscoveryLink{{"llms-txt", "/llms.txt"}})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	wantLen := len(rec.Body.String())
	gotLen := rec.Header().Get("Content-Length")
	if gotLen != strconv.Itoa(wantLen) {
		t.Errorf("Content-Length = %q, want %d", gotLen, wantLen)
	}
}
