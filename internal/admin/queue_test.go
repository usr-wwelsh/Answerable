package admin

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/usr-wwelsh/answerable/internal/booking"
)

func TestQueueShowsConfirmAndDenyActionsForPendingRequests(t *testing.T) {
	h := newHarness(t)
	req, err := h.bookStore.Enqueue("Jane Doe", "555-0100", "bed for two")
	if err != nil {
		t.Fatalf("Enqueue: %v", err)
	}

	rec := h.do(t, http.MethodGet, "/queue", nil, "")
	body := rec.Body.String()
	if !strings.Contains(body, `action="/queue/confirm"`) || !strings.Contains(body, `action="/queue/deny"`) {
		t.Fatalf("queue view missing confirm/deny forms: %s", body)
	}
	if !strings.Contains(body, req.Token) {
		t.Fatalf("queue view missing token needed to act on the request: %s", body)
	}
}

func TestQueueConfirmMarksRequestConfirmed(t *testing.T) {
	h := newHarness(t)
	req, err := h.bookStore.Enqueue("Jane Doe", "555-0100", "bed for two")
	if err != nil {
		t.Fatalf("Enqueue: %v", err)
	}

	form := strings.NewReader("token=" + req.Token)
	httpReq := httptest.NewRequest(http.MethodPost, "/queue/confirm", form)
	httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	h.srv.Handler().ServeHTTP(rec, httpReq)

	if rec.Code != http.StatusFound {
		t.Fatalf("status = %d, want 302, body=%s", rec.Code, rec.Body.String())
	}

	list, _ := h.bookStore.List()
	if len(list) != 1 || list[0].Status != booking.StatusConfirmed {
		t.Fatalf("expected confirmed request, got %+v", list)
	}
}

func TestQueueDenyMarksRequestDenied(t *testing.T) {
	h := newHarness(t)
	req, err := h.bookStore.Enqueue("Jane Doe", "555-0100", "bed for two")
	if err != nil {
		t.Fatalf("Enqueue: %v", err)
	}

	form := strings.NewReader("token=" + req.Token)
	httpReq := httptest.NewRequest(http.MethodPost, "/queue/deny", form)
	httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	h.srv.Handler().ServeHTTP(rec, httpReq)

	if rec.Code != http.StatusFound {
		t.Fatalf("status = %d, want 302, body=%s", rec.Code, rec.Body.String())
	}

	list, _ := h.bookStore.List()
	if len(list) != 1 || list[0].Status != booking.StatusDenied {
		t.Fatalf("expected denied request, got %+v", list)
	}
}

func TestQueueConfirmRejectsUnknownToken(t *testing.T) {
	h := newHarness(t)

	form := strings.NewReader("token=bogus")
	httpReq := httptest.NewRequest(http.MethodPost, "/queue/confirm", form)
	httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	h.srv.Handler().ServeHTTP(rec, httpReq)

	if rec.Code != http.StatusFound {
		t.Fatalf("status = %d, want 302 (redirect back with error), body=%s", rec.Code, rec.Body.String())
	}

	followRec := h.do(t, http.MethodGet, rec.Header().Get("Location"), nil, "")
	if !strings.Contains(followRec.Body.String(), "unknown token") {
		t.Errorf("expected error surfaced on redirect target: %s", followRec.Body.String())
	}
}

func TestQueueDoesNotShowActionsForResolvedRequests(t *testing.T) {
	h := newHarness(t)
	req, err := h.bookStore.Enqueue("Jane Doe", "555-0100", "bed for two")
	if err != nil {
		t.Fatalf("Enqueue: %v", err)
	}
	if _, err := h.bookStore.Confirm(req.Token); err != nil {
		t.Fatalf("Confirm: %v", err)
	}

	rec := h.do(t, http.MethodGet, "/queue", nil, "")
	body := rec.Body.String()
	if strings.Contains(body, `action="/queue/confirm"`) {
		t.Errorf("resolved request should not show a confirm action: %s", body)
	}
}
