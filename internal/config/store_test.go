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
	if got != want {
		t.Fatalf("Load = %+v, want %+v", got, want)
	}
}

func TestSaveOverwritesPreviousConfig(t *testing.T) {
	s := openTemp(t)

	first := Config{Source: Source{Kind: SourceFile, Value: "/data/a.csv"}}
	second := Config{Source: Source{Kind: SourceFile, Value: "/data/b.csv"}, WebhookURL: "https://hooks/b"}

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
	if got != second {
		t.Fatalf("Load = %+v, want %+v", got, second)
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
