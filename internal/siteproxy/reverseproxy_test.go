package siteproxy

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestReverseProxyInjectsLinksIntoUpstreamHTML(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte("<html><head><title>Existing Site</title></head><body>hi</body></html>"))
	}))
	defer upstream.Close()

	u, err := url.Parse(upstream.URL)
	if err != nil {
		t.Fatalf("url.Parse: %v", err)
	}

	proxy := ReverseProxy(u, []DiscoveryLink{{"llms-txt", "/llms.txt"}})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	proxy.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `<link rel="llms-txt" href="/llms.txt">`) {
		t.Errorf("discovery link not injected: %s", rec.Body.String())
	}
}

func TestReverseProxyLeavesNonHTMLUpstreamResponsesUntouched(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"ok":true}`))
	}))
	defer upstream.Close()

	u, _ := url.Parse(upstream.URL)
	proxy := ReverseProxy(u, DefaultLinks)

	req := httptest.NewRequest(http.MethodGet, "/api", nil)
	rec := httptest.NewRecorder()
	proxy.ServeHTTP(rec, req)

	if rec.Body.String() != `{"ok":true}` {
		t.Errorf("non-HTML body was modified: %s", rec.Body.String())
	}
}
