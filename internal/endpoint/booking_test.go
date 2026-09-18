package endpoint

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/usr-wwelsh/answerable/internal/booking"
)

func openTestBookingStore(t *testing.T) *booking.Store {
	t.Helper()
	s, err := booking.Open(":memory:")
	if err != nil {
		t.Fatalf("booking.Open returned error: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func postBook(t *testing.T, srv *Server, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/book", strings.NewReader(body))
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	return rec
}

func TestBookWithoutWebhookQueuesRequestAndOmitsJargon(t *testing.T) {
	store := openTestBookingStore(t)
	srv := New(testProvider(), store, "")

	rec := postBook(t, srv, `{"name":"Jane Doe","contact":"555-0100","need":"bed for two tonight"}`)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want 202", rec.Code)
	}

	var resp map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}
	if strings.Contains(strings.ToLower(resp["message"]), "webhook") {
		t.Errorf("message leaks implementation jargon: %q", resp["message"])
	}
	if !strings.Contains(strings.ToLower(resp["message"]), "phone") {
		t.Errorf("message should suggest calling if urgent: %q", resp["message"])
	}
	if _, leaked := resp["confirmUrl"]; leaked {
		t.Error("response leaks confirmUrl to the requester — only the webhook should see it")
	}

	list, err := store.List()
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if len(list) != 1 || list[0].Name != "Jane Doe" {
		t.Errorf("expected queued request for Jane Doe, got %+v", list)
	}
}

func TestBookWithWebhookNotifiesIt(t *testing.T) {
	notified := make(chan string, 1)
	hookSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]string
		json.NewDecoder(r.Body).Decode(&body)
		notified <- body["content"]
		w.WriteHeader(http.StatusNoContent)
	}))
	defer hookSrv.Close()

	store := openTestBookingStore(t)
	srv := New(testProvider(), store, hookSrv.URL)

	rec := postBook(t, srv, `{"name":"Jane Doe","contact":"555-0100","need":"bed for two tonight"}`)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want 202", rec.Code)
	}

	select {
	case content := <-notified:
		if !strings.Contains(content, "Jane Doe") {
			t.Errorf("webhook content missing name: %q", content)
		}
		if !strings.Contains(content, "/confirm") || !strings.Contains(content, "/deny") {
			t.Errorf("webhook content missing confirm/deny links: %q", content)
		}
	default:
		t.Fatal("webhook was not notified")
	}
}

func TestBookRejectsMissingRequiredFields(t *testing.T) {
	store := openTestBookingStore(t)
	srv := New(testProvider(), store, "")

	rec := postBook(t, srv, `{"contact":"555-0100"}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestConfirmAndDenyRoutesResolveTokens(t *testing.T) {
	store := openTestBookingStore(t)
	srv := New(testProvider(), store, "")

	postBook(t, srv, `{"name":"Jane Doe","contact":"555-0100","need":"bed for two tonight"}`)

	list, _ := store.List()
	if len(list) != 1 {
		t.Fatalf("expected 1 queued request, got %d", len(list))
	}
	token := list[0].Token

	confirmReq := httptest.NewRequest(http.MethodGet, "/confirm?token="+token, nil)
	confirmRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(confirmRec, confirmReq)
	if confirmRec.Code != http.StatusOK {
		t.Fatalf("confirm status = %d, want 200, body=%s", confirmRec.Code, confirmRec.Body.String())
	}

	list, _ = store.List()
	if list[0].Status != booking.StatusConfirmed {
		t.Errorf("status = %q, want confirmed", list[0].Status)
	}
}

func TestConfirmRejectsUnknownTokenWithBadRequest(t *testing.T) {
	store := openTestBookingStore(t)
	srv := New(testProvider(), store, "")

	req := httptest.NewRequest(http.MethodGet, "/confirm?token=bogus", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}
