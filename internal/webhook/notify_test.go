package webhook

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func testNotification() Notification {
	return Notification{
		Name:       "Jane Doe",
		Contact:    "555-0100",
		Need:       "bed for two tonight",
		ConfirmURL: "https://example.com/confirm?token=abc",
		DenyURL:    "https://example.com/deny?token=abc",
	}
}

func TestNotifyPostsPlainSummaryInContent(t *testing.T) {
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	if err := Notify(srv.URL, testNotification()); err != nil {
		t.Fatalf("Notify returned error: %v", err)
	}

	content, _ := gotBody["content"].(string)
	if !strings.Contains(content, "Jane Doe") {
		t.Errorf("content = %q, want it to mention the requester", content)
	}
	if strings.Contains(content, "http") {
		t.Errorf("content should not carry raw links: %q", content)
	}
}

func TestNotifyPostsSlackMaskedLinksInText(t *testing.T) {
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&gotBody)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	n := testNotification()
	if err := Notify(srv.URL, n); err != nil {
		t.Fatalf("Notify returned error: %v", err)
	}

	text, _ := gotBody["text"].(string)
	if !strings.Contains(text, "<"+n.ConfirmURL+"|Confirm>") {
		t.Errorf("text missing masked confirm link: %q", text)
	}
	if !strings.Contains(text, "<"+n.DenyURL+"|Deny>") {
		t.Errorf("text missing masked deny link: %q", text)
	}
}

func TestNotifyPostsDiscordEmbedWithLinkedConfirmDeny(t *testing.T) {
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&gotBody)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	n := testNotification()
	if err := Notify(srv.URL, n); err != nil {
		t.Fatalf("Notify returned error: %v", err)
	}

	embeds, _ := gotBody["embeds"].([]any)
	if len(embeds) != 1 {
		t.Fatalf("expected 1 embed, got %d", len(embeds))
	}
	embed, _ := embeds[0].(map[string]any)
	fields, _ := embed["fields"].([]any)
	if len(fields) != 1 {
		t.Fatalf("expected 1 embed field, got %d", len(fields))
	}
	field, _ := fields[0].(map[string]any)
	value, _ := field["value"].(string)
	if !strings.Contains(value, "]("+n.ConfirmURL+")") {
		t.Errorf("embed field missing masked confirm link: %q", value)
	}
	if !strings.Contains(value, "]("+n.DenyURL+")") {
		t.Errorf("embed field missing masked deny link: %q", value)
	}
}

func TestNotifyErrorsOnNonSuccessStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	if err := Notify(srv.URL, testNotification()); err == nil {
		t.Fatal("expected error for non-success status, got nil")
	}
}
