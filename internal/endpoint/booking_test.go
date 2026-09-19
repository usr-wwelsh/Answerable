package endpoint

import (
	"bufio"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/usr-wwelsh/answerable/internal/booking"
	"github.com/usr-wwelsh/answerable/internal/email"
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
	if resp["statusToken"] == "" {
		t.Error("response should include a statusToken so the requester can check back later")
	}

	list, err := store.List()
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if len(list) != 1 || list[0].Name != "Jane Doe" {
		t.Errorf("expected queued request for Jane Doe, got %+v", list)
	}
	if resp["statusToken"] != list[0].StatusToken {
		t.Errorf("returned statusToken %q does not match the stored request's %q", resp["statusToken"], list[0].StatusToken)
	}
}

func TestBookWithWebhookNotifiesItWithHyperlinkedConfirmDeny(t *testing.T) {
	notified := make(chan map[string]any, 1)
	hookSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		notified <- body
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
	case body := <-notified:
		content, _ := body["content"].(string)
		if !strings.Contains(content, "Jane Doe") {
			t.Errorf("webhook content missing name: %q", content)
		}
		if strings.Contains(content, "/confirm") || strings.Contains(content, "/deny") {
			t.Errorf("Discord content should carry hyperlinked confirm/deny via the embed, not raw links: %q", content)
		}

		text, _ := body["text"].(string)
		if !strings.Contains(text, "|Confirm>") || !strings.Contains(text, "|Deny>") {
			t.Errorf("Slack text missing masked confirm/deny links: %q", text)
		}

		embeds, _ := body["embeds"].([]any)
		if len(embeds) == 0 {
			t.Fatal("expected a Discord embed carrying the confirm/deny links")
		}
		embed, _ := embeds[0].(map[string]any)
		fields, _ := embed["fields"].([]any)
		if len(fields) == 0 {
			t.Fatal("expected an embed field with confirm/deny links")
		}
		field, _ := fields[0].(map[string]any)
		value, _ := field["value"].(string)
		if !strings.Contains(value, "/confirm") || !strings.Contains(value, "/deny") {
			t.Errorf("embed field missing confirm/deny links: %q", value)
		}
	default:
		t.Fatal("webhook was not notified")
	}
}

// fakeUnauthSMTPServer accepts a single connection and echoes back a
// no-auth-required conversation, capturing the DATA body. It's enough to
// prove bookIntake wires an email-configured Server through to a real SMTP
// send, without needing net/smtp AUTH plumbing.
func fakeUnauthSMTPServer(t *testing.T) (host string, port int, bodies chan string) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen: %v", err)
	}
	t.Cleanup(func() { ln.Close() })

	bodies = make(chan string, 1)
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		r := bufio.NewReader(conn)
		w := bufio.NewWriter(conn)
		writeLine := func(s string) { w.WriteString(s + "\r\n"); w.Flush() }
		writeLine("220 fake.local ESMTP ready")

		for {
			line, err := r.ReadString('\n')
			if err != nil {
				return
			}
			upper := strings.ToUpper(strings.TrimRight(line, "\r\n"))
			switch {
			case strings.HasPrefix(upper, "EHLO"):
				writeLine("250 fake.local")
			case strings.HasPrefix(upper, "MAIL FROM:"):
				writeLine("250 2.1.0 OK")
			case strings.HasPrefix(upper, "RCPT TO:"):
				writeLine("250 2.1.5 OK")
			case strings.HasPrefix(upper, "DATA"):
				writeLine("354 go ahead")
				var body strings.Builder
				for {
					dataLine, err := r.ReadString('\n')
					if err != nil {
						return
					}
					trimmed := strings.TrimRight(dataLine, "\r\n")
					if trimmed == "." {
						break
					}
					body.WriteString(strings.TrimPrefix(trimmed, "."))
					body.WriteString("\n")
				}
				bodies <- body.String()
				writeLine("250 2.0.0 OK")
			case strings.HasPrefix(upper, "QUIT"):
				writeLine("221 2.0.0 Bye")
				return
			default:
				writeLine("500 unrecognized command")
			}
		}
	}()

	addr := ln.Addr().(*net.TCPAddr)
	return addr.IP.String(), addr.Port, bodies
}

func TestBookWithEmailNotifiesIt(t *testing.T) {
	host, port, bodies := fakeUnauthSMTPServer(t)

	store := openTestBookingStore(t)
	srv := New(testProvider(), store, "")
	srv.UpdateEmail(email.Config{
		SMTPHost: host,
		SMTPPort: port,
		From:     "shelter@example.com",
		To:       "oncall@example.com",
	})

	rec := postBook(t, srv, `{"name":"Jane Doe","contact":"555-0100","need":"bed for two tonight"}`)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want 202", rec.Code)
	}

	select {
	case body := <-bodies:
		if !strings.Contains(body, "Jane Doe") {
			t.Errorf("email body missing name: %q", body)
		}
		if !strings.Contains(body, "/confirm") || !strings.Contains(body, "/deny") {
			t.Errorf("email body missing confirm/deny links: %q", body)
		}
	default:
		t.Fatal("email was not sent")
	}
}

func TestBookPrefersEmailOverWebhookWhenBothConfigured(t *testing.T) {
	host, port, bodies := fakeUnauthSMTPServer(t)

	webhookCalled := false
	hookSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		webhookCalled = true
		w.WriteHeader(http.StatusNoContent)
	}))
	defer hookSrv.Close()

	store := openTestBookingStore(t)
	srv := New(testProvider(), store, hookSrv.URL)
	srv.UpdateEmail(email.Config{
		SMTPHost: host,
		SMTPPort: port,
		From:     "shelter@example.com",
		To:       "oncall@example.com",
	})

	rec := postBook(t, srv, `{"name":"Jane Doe","contact":"555-0100","need":"bed for two tonight"}`)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want 202", rec.Code)
	}

	select {
	case <-bodies:
	default:
		t.Fatal("email was not sent")
	}
	if webhookCalled {
		t.Error("webhook was notified even though email is configured; expected email to take precedence")
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

func TestConfirmGetRendersPageWithoutMutating(t *testing.T) {
	store := openTestBookingStore(t)
	srv := New(testProvider(), store, "")

	postBook(t, srv, `{"name":"Jane Doe","contact":"555-0100","need":"bed for two tonight"}`)

	list, _ := store.List()
	if len(list) != 1 {
		t.Fatalf("expected 1 queued request, got %d", len(list))
	}
	token := list[0].Token

	// A GET must be safe to fetch without effect, since chat apps and
	// email clients auto-fetch links to build previews before any human
	// clicks them.
	confirmReq := httptest.NewRequest(http.MethodGet, "/confirm?token="+token, nil)
	confirmRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(confirmRec, confirmReq)
	if confirmRec.Code != http.StatusOK {
		t.Fatalf("GET /confirm status = %d, want 200, body=%s", confirmRec.Code, confirmRec.Body.String())
	}

	list, _ = store.List()
	if list[0].Status != booking.StatusPending {
		t.Errorf("GET must not mutate status; got %q, want pending", list[0].Status)
	}
}

func TestConfirmAndDenyPostResolveTokens(t *testing.T) {
	store := openTestBookingStore(t)
	srv := New(testProvider(), store, "")

	postBook(t, srv, `{"name":"Jane Doe","contact":"555-0100","need":"bed for two tonight"}`)

	list, _ := store.List()
	if len(list) != 1 {
		t.Fatalf("expected 1 queued request, got %d", len(list))
	}
	token := list[0].Token

	confirmReq := httptest.NewRequest(http.MethodPost, "/confirm?token="+token, nil)
	confirmRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(confirmRec, confirmReq)
	if confirmRec.Code != http.StatusOK {
		t.Fatalf("POST /confirm status = %d, want 200, body=%s", confirmRec.Code, confirmRec.Body.String())
	}

	list, _ = store.List()
	if list[0].Status != booking.StatusConfirmed {
		t.Errorf("status = %q, want confirmed", list[0].Status)
	}
}

func TestStatusRouteReflectsPendingThenConfirmed(t *testing.T) {
	store := openTestBookingStore(t)
	srv := New(testProvider(), store, "")

	rec := postBook(t, srv, `{"name":"Jane Doe","contact":"555-0100","need":"bed for two tonight"}`)
	var resp map[string]string
	json.Unmarshal(rec.Body.Bytes(), &resp)
	statusToken := resp["statusToken"]

	statusReq := httptest.NewRequest(http.MethodGet, "/status?token="+statusToken, nil)
	statusRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(statusRec, statusReq)

	if statusRec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", statusRec.Code, statusRec.Body.String())
	}
	var statusResp map[string]string
	json.Unmarshal(statusRec.Body.Bytes(), &statusResp)
	if statusResp["status"] != "pending" {
		t.Errorf("status = %q, want pending", statusResp["status"])
	}

	list, _ := store.List()
	if _, err := store.Confirm(list[0].Token); err != nil {
		t.Fatalf("Confirm returned error: %v", err)
	}

	statusRec2 := httptest.NewRecorder()
	srv.Handler().ServeHTTP(statusRec2, httptest.NewRequest(http.MethodGet, "/status?token="+statusToken, nil))
	json.Unmarshal(statusRec2.Body.Bytes(), &statusResp)
	if statusResp["status"] != "confirmed" {
		t.Errorf("status after confirm = %q, want confirmed", statusResp["status"])
	}
}

func TestStatusRouteCannotResolveTheRequestItself(t *testing.T) {
	store := openTestBookingStore(t)
	srv := New(testProvider(), store, "")

	rec := postBook(t, srv, `{"name":"Jane Doe","contact":"555-0100","need":"bed for two tonight"}`)
	var resp map[string]string
	json.Unmarshal(rec.Body.Bytes(), &resp)

	req := httptest.NewRequest(http.MethodPost, "/confirm?token="+resp["statusToken"], nil)
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)

	if rr.Code == http.StatusOK {
		t.Error("the status token must not be usable to confirm the request")
	}
}

func TestStatusRouteRejectsUnknownToken(t *testing.T) {
	store := openTestBookingStore(t)
	srv := New(testProvider(), store, "")

	req := httptest.NewRequest(http.MethodGet, "/status?token=bogus", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestConfirmRejectsUnknownTokenWithBadRequest(t *testing.T) {
	store := openTestBookingStore(t)
	srv := New(testProvider(), store, "")

	req := httptest.NewRequest(http.MethodPost, "/confirm?token=bogus", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}
