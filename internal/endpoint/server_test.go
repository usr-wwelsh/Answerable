package endpoint

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/usr-wwelsh/answerable/internal/facts"
)

func testProvider() facts.Provider {
	return facts.Provider{
		Name: "Test Shelter A",
		Properties: map[string]string{
			"capacity_available": "12",
			"eligibility":        "walk-in",
		},
	}
}

func testProviderNamed(name string) facts.Provider {
	return facts.Provider{Name: name}
}

func newTestServer(t *testing.T) *Server {
	t.Helper()
	return New(testProvider(), openTestBookingStore(t), "")
}

func TestFactsRouteServesJSONLD(t *testing.T) {
	srv := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/facts.jsonld", nil)
	rec := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "application/ld+json") {
		t.Errorf("Content-Type = %q, want application/ld+json", ct)
	}
	if !strings.Contains(rec.Body.String(), "Test Shelter A") {
		t.Errorf("body missing provider name: %s", rec.Body.String())
	}
}

func TestFactsRouteUsesHTTPSWhenForwardedByProxy(t *testing.T) {
	srv := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/facts.jsonld", nil)
	req.Header.Set("X-Forwarded-Proto", "https")
	rec := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec, req)

	if strings.Contains(rec.Body.String(), "http://") {
		t.Errorf("body contains insecure http:// link despite X-Forwarded-Proto: https: %s", rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "https://") {
		t.Errorf("body missing https:// link: %s", rec.Body.String())
	}
}

func TestFactsHTMLRouteServesHTML(t *testing.T) {
	srv := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/facts.html", nil)
	rec := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "text/html") {
		t.Errorf("Content-Type = %q, want text/html", ct)
	}
	if !strings.Contains(rec.Body.String(), "Test Shelter A") {
		t.Errorf("body missing provider name: %s", rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "capacity_available") {
		t.Errorf("body missing property name: %s", rec.Body.String())
	}
}

func TestAgentCardRouteServesJSON(t *testing.T) {
	srv := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/.well-known/agent.json", nil)
	rec := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "application/json") {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}
}

func TestUpdateProviderReflectsInFactsRoute(t *testing.T) {
	srv := newTestServer(t)

	srv.UpdateProvider(facts.Provider{Name: "Renamed Shelter"})

	req := httptest.NewRequest(http.MethodGet, "/facts.jsonld", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if !strings.Contains(rec.Body.String(), "Renamed Shelter") {
		t.Errorf("body missing updated provider name: %s", rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "Test Shelter A") {
		t.Errorf("body still contains stale provider name: %s", rec.Body.String())
	}
}

func TestUpdateWebhookChangesNotificationTarget(t *testing.T) {
	notified := make(chan struct{}, 1)
	hookSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		notified <- struct{}{}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer hookSrv.Close()

	srv := newTestServer(t)
	srv.UpdateWebhook(hookSrv.URL)

	postBook(t, srv, `{"name":"Jane Doe","contact":"555-0100","need":"bed for two tonight"}`)

	select {
	case <-notified:
	default:
		t.Fatal("updated webhook was not notified")
	}
}

func TestLLMsTxtRouteServesPlainText(t *testing.T) {
	srv := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/llms.txt", nil)
	rec := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "facts.jsonld") {
		t.Errorf("body missing facts link: %s", rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "facts.html") {
		t.Errorf("body missing HTML fallback link: %s", rec.Body.String())
	}
}

func TestRootRouteRedirectsToLLMsTxtByDefault(t *testing.T) {
	srv := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("status = %d, want 302", rec.Code)
	}
	if loc := rec.Header().Get("Location"); loc != "/llms.txt" {
		t.Errorf("Location = %q, want /llms.txt", loc)
	}
}

func TestAgentsRouteRedirectsToLLMsTxtByDefault(t *testing.T) {
	srv := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/agents", nil)
	rec := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("status = %d, want 302", rec.Code)
	}
	if loc := rec.Header().Get("Location"); loc != "/llms.txt" {
		t.Errorf("Location = %q, want /llms.txt", loc)
	}
}

func TestAgentsRouteServesJSONIndexWhenJSONRequested(t *testing.T) {
	srv := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/agents", nil)
	req.Header.Set("Accept", "application/json")
	rec := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []string{"llms_txt", "agent_card", "mcp_manifest", "\"mcp\"", "facts", "facts_html"} {
		if !strings.Contains(body, want) {
			t.Errorf("body missing %q: %s", want, body)
		}
	}
}

func TestRootRouteServesJSONIndexWhenJSONRequested(t *testing.T) {
	srv := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept", "application/json")
	rec := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "application/json") {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}
	body := rec.Body.String()
	for _, want := range []string{"llms_txt", "agent_card", "mcp_manifest", "\"mcp\"", "facts", "facts_html"} {
		if !strings.Contains(body, want) {
			t.Errorf("body missing %q: %s", want, body)
		}
	}
}

func TestMCPManifestRouteServesJSON(t *testing.T) {
	srv := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/.well-known/mcp.json", nil)
	rec := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "application/json") {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}
	body := rec.Body.String()
	for _, want := range []string{"/mcp", "streamable-http", "protocolVersion"} {
		if !strings.Contains(body, want) {
			t.Errorf("body missing %q: %s", want, body)
		}
	}
}

func TestRobotsRouteServesPlainTextWithLLMsTxtPointer(t *testing.T) {
	srv := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/robots.txt", nil)
	rec := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "llms.txt") {
		t.Errorf("body missing llms.txt pointer: %s", rec.Body.String())
	}
}

func TestUnknownPathStill404sInsteadOfClaimingRoot(t *testing.T) {
	srv := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/whatever-the-host-app-owns", nil)
	rec := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404 (root claim must not become a catch-all)", rec.Code)
	}
}

func TestNotFoundResponseIncludesDiscoveryHint(t *testing.T) {
	srv := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/api/beds", nil)
	rec := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "/llms.txt") {
		t.Errorf("404 body missing discovery hint: %s", rec.Body.String())
	}
}

func TestNotFoundResponseFromSiteIncludesDiscoveryHint(t *testing.T) {
	srv := newTestServer(t)
	srv.SetSite(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	req := httptest.NewRequest(http.MethodGet, "/api/beds", nil)
	rec := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "/llms.txt") {
		t.Errorf("404 body missing discovery hint: %s", rec.Body.String())
	}
}

func TestNotFoundHTMLPageFromSiteIsLeftIntact(t *testing.T) {
	srv := newTestServer(t)
	srv.SetSite(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`<html><head></head><body>Provider's own branded 404<a href="/llms.txt"></a></body></html>`))
	}))
	req := httptest.NewRequest(http.MethodGet, "/api/beds", nil)
	rec := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "Provider's own branded 404") {
		t.Errorf("provider's own HTML 404 page was clobbered: %s", rec.Body.String())
	}
}

func TestDiscoveryLinkHeaderPresentOnEveryResponse(t *testing.T) {
	srv := newTestServer(t)

	for _, path := range []string{"/facts.jsonld", "/nonexistent-path"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()

		srv.Handler().ServeHTTP(rec, req)

		link := rec.Header().Get("Link")
		for _, want := range []string{`rel="llms-txt"`, `rel="agent-card"`, `rel="mcp-manifest"`, `rel="mcp-server"`, `rel="facts-html"`} {
			if !strings.Contains(link, want) {
				t.Errorf("path %s: Link header missing %s: %s", path, want, link)
			}
		}
	}
}
