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

func TestFactsRouteServesJSONLD(t *testing.T) {
	srv := New(testProvider())
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
	srv := New(testProvider())
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

func TestLLMsTxtRouteServesPlainText(t *testing.T) {
	srv := New(testProvider())
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
