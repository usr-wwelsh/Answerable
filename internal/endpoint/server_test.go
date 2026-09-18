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
}
