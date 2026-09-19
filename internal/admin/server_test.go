package admin

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/usr-wwelsh/answerable/internal/booking"
	"github.com/usr-wwelsh/answerable/internal/config"
	"github.com/usr-wwelsh/answerable/internal/facts"
)

type harness struct {
	srv       *Server
	updates   []facts.Provider
	webhooks  []string
	emails    []config.Email
	cfgStore  *config.Store
	bookStore *booking.Store
}

func newHarness(t *testing.T) *harness {
	t.Helper()

	cfgStore, err := config.Open(t.TempDir() + "/answerable.db")
	if err != nil {
		t.Fatalf("config.Open: %v", err)
	}
	t.Cleanup(func() { cfgStore.Close() })

	bookStore, err := booking.Open(":memory:")
	if err != nil {
		t.Fatalf("booking.Open: %v", err)
	}
	t.Cleanup(func() { bookStore.Close() })

	h := &harness{cfgStore: cfgStore, bookStore: bookStore}

	srv, err := New(Deps{
		ConfigStore:  cfgStore,
		BookingStore: bookStore,
		UploadDir:    t.TempDir(),
		OnProvider:   func(p facts.Provider) { h.updates = append(h.updates, p) },
		OnWebhook:    func(url string) { h.webhooks = append(h.webhooks, url) },
		OnEmail:      func(e config.Email) { h.emails = append(h.emails, e) },
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	h.srv = srv
	return h
}

func (h *harness) do(t *testing.T, method, path string, body []byte, contentType string) *httptest.ResponseRecorder {
	t.Helper()
	var reader *bytes.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	} else {
		reader = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, path, reader)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	rec := httptest.NewRecorder()
	h.srv.Handler().ServeHTTP(rec, req)
	return rec
}

func multipartUpload(t *testing.T, field, filename string, content []byte, extra map[string]string) ([]byte, string) {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	for k, v := range extra {
		if err := w.WriteField(k, v); err != nil {
			t.Fatalf("WriteField: %v", err)
		}
	}
	if filename != "" {
		fw, err := w.CreateFormFile(field, filename)
		if err != nil {
			t.Fatalf("CreateFormFile: %v", err)
		}
		if _, err := fw.Write(content); err != nil {
			t.Fatalf("write file: %v", err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}
	return buf.Bytes(), w.FormDataContentType()
}

func TestRootRedirectsToOnboardingWhenUnconfigured(t *testing.T) {
	h := newHarness(t)
	rec := h.do(t, http.MethodGet, "/", nil, "")
	if rec.Code != http.StatusFound {
		t.Fatalf("status = %d, want 302", rec.Code)
	}
	if loc := rec.Header().Get("Location"); loc != "/onboarding" {
		t.Errorf("Location = %q, want /onboarding", loc)
	}
}

func TestOnboardingIntroAvoidsJargon(t *testing.T) {
	h := newHarness(t)
	rec := h.do(t, http.MethodGet, "/onboarding", nil, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	for _, jargon := range []string{"JSON-LD", "MCP", "endpoint"} {
		if strings.Contains(body, jargon) {
			t.Errorf("onboarding copy contains jargon %q", jargon)
		}
	}
}

func TestSourceUploadSetsProviderAndAdvancesToWebhookStep(t *testing.T) {
	h := newHarness(t)
	body, ct := multipartUpload(t, "file", "shelter.csv", []byte("name,beds\nTest Shelter,12\n"), nil)

	rec := h.do(t, http.MethodPost, "/onboarding/source", body, ct)

	if rec.Code != http.StatusFound {
		t.Fatalf("status = %d, want 302, body=%s", rec.Code, rec.Body.String())
	}
	if loc := rec.Header().Get("Location"); loc != "/onboarding/delivery" {
		t.Errorf("Location = %q, want /onboarding/delivery", loc)
	}
	if len(h.updates) != 1 || h.updates[0].Name != "Test Shelter" {
		t.Fatalf("updates = %+v, want one update named Test Shelter", h.updates)
	}

	cfg, ok, err := h.cfgStore.Load()
	if err != nil || !ok {
		t.Fatalf("Load: ok=%v err=%v", ok, err)
	}
	if cfg.Source.Kind != config.SourceFile {
		t.Errorf("Source.Kind = %q, want file", cfg.Source.Kind)
	}
}

func TestSourceURLSetsProviderFromFetchedCSV(t *testing.T) {
	hookSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/csv")
		w.Write([]byte("name,beds\nRemote Shelter,7\n"))
	}))
	defer hookSrv.Close()

	h := newHarness(t)
	body, ct := multipartUpload(t, "file", "", nil, map[string]string{
		"url":             hookSrv.URL + "/sheet.csv",
		"refresh_minutes": "5",
	})

	rec := h.do(t, http.MethodPost, "/onboarding/source", body, ct)

	if rec.Code != http.StatusFound {
		t.Fatalf("status = %d, want 302, body=%s", rec.Code, rec.Body.String())
	}
	if len(h.updates) != 1 || h.updates[0].Name != "Remote Shelter" {
		t.Fatalf("updates = %+v, want one update named Remote Shelter", h.updates)
	}

	cfg, ok, err := h.cfgStore.Load()
	if err != nil || !ok {
		t.Fatalf("Load: ok=%v err=%v", ok, err)
	}
	if cfg.Source.Kind != config.SourceURL || cfg.Source.Value != hookSrv.URL+"/sheet.csv" {
		t.Errorf("Source = %+v, want kind url value %s", cfg.Source, hookSrv.URL+"/sheet.csv")
	}
}

func TestSourceRejectsWhenNeitherFileNorURLGiven(t *testing.T) {
	h := newHarness(t)
	body, ct := multipartUpload(t, "file", "", nil, nil)

	rec := h.do(t, http.MethodPost, "/onboarding/source", body, ct)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400, body=%s", rec.Code, rec.Body.String())
	}
	if len(h.updates) != 0 {
		t.Fatalf("expected no updates, got %+v", h.updates)
	}
}

func TestWebhookStepSavesAndRedirectsHome(t *testing.T) {
	h := newHarness(t)
	form := strings.NewReader("webhook_url=https%3A%2F%2Fhooks.example%2Fabc")
	req := httptest.NewRequest(http.MethodPost, "/onboarding/webhook", form)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	h.srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("status = %d, want 302", rec.Code)
	}
	if loc := rec.Header().Get("Location"); loc != "/" {
		t.Errorf("Location = %q, want /", loc)
	}
	if len(h.webhooks) != 1 || h.webhooks[0] != "https://hooks.example/abc" {
		t.Fatalf("webhooks = %+v", h.webhooks)
	}
}

func TestWebhookStepSkipDoesNotCallOnWebhook(t *testing.T) {
	h := newHarness(t)
	rec := h.do(t, http.MethodPost, "/onboarding/webhook/skip", nil, "")

	if rec.Code != http.StatusFound {
		t.Fatalf("status = %d, want 302", rec.Code)
	}
	if len(h.webhooks) != 0 {
		t.Fatalf("expected skip to avoid calling OnWebhook, got %+v", h.webhooks)
	}
}

func TestOnboardingDeliveryOffersWebhookAndEmailChoices(t *testing.T) {
	h := newHarness(t)
	rec := h.do(t, http.MethodGet, "/onboarding/delivery", nil, "")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `href="/onboarding/webhook"`) {
		t.Errorf("delivery choice screen missing link to webhook step: %s", body)
	}
	if !strings.Contains(body, `href="/onboarding/email"`) {
		t.Errorf("delivery choice screen missing link to email step: %s", body)
	}
}

func TestEmailStepSavesAndRedirectsHome(t *testing.T) {
	h := newHarness(t)
	form := "smtp_host=smtp.example.com&smtp_port=587&username=shelter%40example.com&password=app-password&from=shelter%40example.com&to=oncall%40example.com"
	req := httptest.NewRequest(http.MethodPost, "/onboarding/email", strings.NewReader(form))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	h.srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("status = %d, want 302, body=%s", rec.Code, rec.Body.String())
	}
	if loc := rec.Header().Get("Location"); loc != "/" {
		t.Errorf("Location = %q, want /", loc)
	}
	if len(h.emails) != 1 {
		t.Fatalf("emails = %+v, want one saved config", h.emails)
	}
	got := h.emails[0]
	want := config.Email{
		SMTPHost: "smtp.example.com",
		SMTPPort: 587,
		Username: "shelter@example.com",
		Password: "app-password",
		From:     "shelter@example.com",
		To:       "oncall@example.com",
	}
	if got != want {
		t.Errorf("saved email = %+v, want %+v", got, want)
	}

	cfg, ok, err := h.cfgStore.Load()
	if err != nil || !ok {
		t.Fatalf("Load: ok=%v err=%v", ok, err)
	}
	if cfg.Email != want {
		t.Errorf("persisted email = %+v, want %+v", cfg.Email, want)
	}
}

func TestEmailStepSkipDoesNotCallOnEmail(t *testing.T) {
	h := newHarness(t)
	rec := h.do(t, http.MethodPost, "/onboarding/email/skip", nil, "")

	if rec.Code != http.StatusFound {
		t.Fatalf("status = %d, want 302", rec.Code)
	}
	if len(h.emails) != 0 {
		t.Fatalf("expected skip to avoid calling OnEmail, got %+v", h.emails)
	}
}

func TestEmailStepRejectsInvalidAddress(t *testing.T) {
	h := newHarness(t)
	form := "smtp_host=smtp.example.com&smtp_port=587&from=not-an-email&to=oncall%40example.com"
	req := httptest.NewRequest(http.MethodPost, "/onboarding/email", strings.NewReader(form))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	h.srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400, body=%s", rec.Code, rec.Body.String())
	}
	if len(h.emails) != 0 {
		t.Fatalf("expected no saved config on validation error, got %+v", h.emails)
	}
}

func TestEmailStepRejectsInvalidPort(t *testing.T) {
	h := newHarness(t)
	form := "smtp_host=smtp.example.com&smtp_port=notanumber&from=shelter%40example.com&to=oncall%40example.com"
	req := httptest.NewRequest(http.MethodPost, "/onboarding/email", strings.NewReader(form))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	h.srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400, body=%s", rec.Code, rec.Body.String())
	}
}

func TestEmailStepPreservesPasswordWhenLeftBlankOnEdit(t *testing.T) {
	h := newHarness(t)
	first := "smtp_host=smtp.example.com&smtp_port=587&username=shelter%40example.com&password=app-password&from=shelter%40example.com&to=oncall%40example.com"
	req := httptest.NewRequest(http.MethodPost, "/onboarding/email", strings.NewReader(first))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	h.srv.Handler().ServeHTTP(httptest.NewRecorder(), req)

	second := "smtp_host=smtp.example.com&smtp_port=587&username=shelter%40example.com&password=&from=shelter%40example.com&to=oncall2%40example.com"
	req2 := httptest.NewRequest(http.MethodPost, "/onboarding/email", strings.NewReader(second))
	req2.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec2 := httptest.NewRecorder()
	h.srv.Handler().ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusFound {
		t.Fatalf("status = %d, want 302, body=%s", rec2.Code, rec2.Body.String())
	}
	if len(h.emails) != 2 {
		t.Fatalf("emails = %+v, want two saves", h.emails)
	}
	if h.emails[1].Password != "app-password" {
		t.Errorf("Password = %q, want preserved app-password when left blank", h.emails[1].Password)
	}
	if h.emails[1].To != "oncall2@example.com" {
		t.Errorf("To = %q, want updated address oncall2@example.com", h.emails[1].To)
	}
}

func TestDashboardShowsProviderOnceConfigured(t *testing.T) {
	h := newHarness(t)
	body, ct := multipartUpload(t, "file", "shelter.csv", []byte("name,beds\nTest Shelter,12\n"), nil)
	h.do(t, http.MethodPost, "/onboarding/source", body, ct)

	rec := h.do(t, http.MethodGet, "/", nil, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "Test Shelter") {
		t.Errorf("dashboard missing provider name: %s", rec.Body.String())
	}
}

func TestDashboardShowsUploadedFilenameAndTimestamp(t *testing.T) {
	h := newHarness(t)
	body, ct := multipartUpload(t, "file", "shelter-hours.csv", []byte("name,beds\nTest Shelter,12\n"), nil)
	h.do(t, http.MethodPost, "/onboarding/source", body, ct)

	cfg, ok, err := h.cfgStore.Load()
	if err != nil || !ok {
		t.Fatalf("Load: ok=%v err=%v", ok, err)
	}
	if cfg.SourceLabel != "shelter-hours.csv" {
		t.Errorf("SourceLabel = %q, want shelter-hours.csv", cfg.SourceLabel)
	}
	if cfg.SourceUpdatedAt.IsZero() {
		t.Error("SourceUpdatedAt is zero, want it set on upload")
	}

	rec := h.do(t, http.MethodGet, "/", nil, "")
	body2 := rec.Body.String()
	if !strings.Contains(body2, "shelter-hours.csv") {
		t.Errorf("dashboard missing uploaded filename: %s", body2)
	}
	if !strings.Contains(body2, "data-ts=") {
		t.Errorf("dashboard missing machine-readable timestamp for client-side timezone rendering: %s", body2)
	}
}

func TestRefreshUpdatesSourceTimestamp(t *testing.T) {
	calls := 0
	hookSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Content-Type", "text/csv")
		w.Write([]byte("name,beds\nRemote Shelter,7\n"))
	}))
	defer hookSrv.Close()

	t0 := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	restore := nowFunc
	nowFunc = func() time.Time { return t0 }
	defer func() { nowFunc = restore }()

	h := newHarness(t)
	body, ct := multipartUpload(t, "file", "", nil, map[string]string{"url": hookSrv.URL + "/sheet.csv"})
	h.do(t, http.MethodPost, "/onboarding/source", body, ct)

	firstCfg, _, _ := h.cfgStore.Load()

	nowFunc = func() time.Time { return t0.Add(time.Hour) }
	h.do(t, http.MethodPost, "/refresh", nil, "")

	secondCfg, _, _ := h.cfgStore.Load()
	if !secondCfg.SourceUpdatedAt.After(firstCfg.SourceUpdatedAt) {
		t.Errorf("SourceUpdatedAt did not advance after refresh: first=%v second=%v", firstCfg.SourceUpdatedAt, secondCfg.SourceUpdatedAt)
	}
	if calls < 2 {
		t.Errorf("expected at least 2 fetches (initial + refresh), got %d", calls)
	}
}

func TestQueueListsBookingRequestsEscaped(t *testing.T) {
	h := newHarness(t)
	if _, err := h.bookStore.Enqueue("Jane <script>alert(1)</script> Doe", "555-0100", "bed for two"); err != nil {
		t.Fatalf("Enqueue: %v", err)
	}

	rec := h.do(t, http.MethodGet, "/queue", nil, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	if strings.Contains(body, "<script>") {
		t.Errorf("queue view did not escape submitted PII: %s", body)
	}
	if !strings.Contains(body, "555-0100") {
		t.Errorf("queue view missing contact: %s", body)
	}
}

func TestStaticAssetsServed(t *testing.T) {
	h := newHarness(t)
	rec := h.do(t, http.MethodGet, "/static/style.css", nil, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "css") {
		t.Errorf("Content-Type = %q, want css", ct)
	}
}
