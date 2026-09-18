package ingest

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetcherFetchParsesCSV(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/csv")
		w.Write([]byte("name,beds\nTest Shelter,12\n"))
	}))
	defer srv.Close()

	rec, err := NewFetcher().Fetch(srv.URL + "/sheet.csv")
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if rec["name"] != "Test Shelter" || rec["beds"] != "12" {
		t.Fatalf("Fetch = %+v, want name=Test Shelter beds=12", rec)
	}
}

func TestFetcherFetchParsesTextByContentType(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte("Name: Test Shelter\nBeds: 12\n"))
	}))
	defer srv.Close()

	rec, err := NewFetcher().Fetch(srv.URL + "/export?format=txt")
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if rec["name"] != "Test Shelter" || rec["beds"] != "12" {
		t.Fatalf("Fetch = %+v, want name=Test Shelter beds=12", rec)
	}
}

func TestFetcherFetchRejectsDisallowedScheme(t *testing.T) {
	if _, err := NewFetcher().Fetch("file:///etc/passwd"); err == nil {
		t.Fatal("Fetch(file://...) = nil error, want rejection")
	}
}

func TestFetcherFetchRejectsUnknownFormat(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Write([]byte("binary junk"))
	}))
	defer srv.Close()

	if _, err := NewFetcher().Fetch(srv.URL + "/download"); err == nil {
		t.Fatal("Fetch(unknown format) = nil error, want error")
	}
}

func TestFetcherFetchPropagatesHTTPErrors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusNotFound)
	}))
	defer srv.Close()

	if _, err := NewFetcher().Fetch(srv.URL + "/sheet.csv"); err == nil {
		t.Fatal("Fetch(404) = nil error, want error")
	}
}
