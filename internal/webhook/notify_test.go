package webhook

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNotifyPostsContentField(t *testing.T) {
	var gotBody map[string]string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	if err := Notify(srv.URL, "new intake request from Jane Doe"); err != nil {
		t.Fatalf("Notify returned error: %v", err)
	}

	if gotBody["content"] != "new intake request from Jane Doe" {
		t.Errorf("content = %q, want %q", gotBody["content"], "new intake request from Jane Doe")
	}
	if gotBody["text"] != "new intake request from Jane Doe" {
		t.Errorf("text = %q, want %q", gotBody["text"], "new intake request from Jane Doe")
	}
}

func TestNotifyErrorsOnNonSuccessStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	if err := Notify(srv.URL, "message"); err == nil {
		t.Fatal("expected error for non-success status, got nil")
	}
}
