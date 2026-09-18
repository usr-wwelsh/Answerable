package config

import (
	"testing"
	"time"
)

func TestLoadOnEmptyStoreReportsNotFound(t *testing.T) {
	s := openTemp(t)

	_, ok, err := s.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if ok {
		t.Fatal("expected ok=false on empty store")
	}
}

func TestSaveThenLoadRoundTrips(t *testing.T) {
	s := openTemp(t)

	want := Config{
		Source:          Source{Kind: SourceURL, Value: "https://example.com/sheet.csv"},
		RefreshInterval: 5 * time.Minute,
		WebhookURL:      "https://discord.com/api/webhooks/x",
		SourceLabel:     "sheet.csv",
		SourceUpdatedAt: time.Now(),
	}
	if err := s.Save(want); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, ok, err := s.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !ok {
		t.Fatal("expected ok=true after Save")
	}
	assertConfigsEqual(t, got, want)
}

func TestSaveOverwritesPreviousConfig(t *testing.T) {
	s := openTemp(t)

	first := Config{Source: Source{Kind: SourceFile, Value: "/data/a.csv"}}
	second := Config{
		Source:          Source{Kind: SourceFile, Value: "/data/b.csv"},
		WebhookURL:      "https://hooks/b",
		SourceLabel:     "b.csv",
		SourceUpdatedAt: time.Now(),
	}

	if err := s.Save(first); err != nil {
		t.Fatalf("Save first: %v", err)
	}
	if err := s.Save(second); err != nil {
		t.Fatalf("Save second: %v", err)
	}

	got, ok, err := s.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !ok {
		t.Fatal("expected ok=true")
	}
	assertConfigsEqual(t, got, second)
}

func TestSaveWithZeroUpdatedAtRoundTripsAsZero(t *testing.T) {
	s := openTemp(t)

	want := Config{Source: Source{Kind: SourceFile, Value: "/data/a.csv"}}
	if err := s.Save(want); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, _, err := s.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !got.SourceUpdatedAt.IsZero() {
		t.Errorf("SourceUpdatedAt = %v, want zero", got.SourceUpdatedAt)
	}
}

func assertConfigsEqual(t *testing.T, got, want Config) {
	t.Helper()
	if got.Source != want.Source {
		t.Errorf("Source = %+v, want %+v", got.Source, want.Source)
	}
	if got.RefreshInterval != want.RefreshInterval {
		t.Errorf("RefreshInterval = %v, want %v", got.RefreshInterval, want.RefreshInterval)
	}
	if got.WebhookURL != want.WebhookURL {
		t.Errorf("WebhookURL = %q, want %q", got.WebhookURL, want.WebhookURL)
	}
	if got.SourceLabel != want.SourceLabel {
		t.Errorf("SourceLabel = %q, want %q", got.SourceLabel, want.SourceLabel)
	}
	if !got.SourceUpdatedAt.Truncate(time.Second).Equal(want.SourceUpdatedAt.Truncate(time.Second)) {
		t.Errorf("SourceUpdatedAt = %v, want %v", got.SourceUpdatedAt, want.SourceUpdatedAt)
	}
}

func openTemp(t *testing.T) *Store {
	t.Helper()
	dir := t.TempDir()
	s, err := Open(dir + "/answerable.db")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}
