package config

import (
	"database/sql"
	"testing"
	"time"

	_ "modernc.org/sqlite"
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
		Email: Email{
			SMTPHost: "smtp.example.com",
			SMTPPort: 587,
			Username: "shelter@example.com",
			Password: "app-password",
			From:     "shelter@example.com",
			To:       "oncall@example.com",
		},
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

func TestOpenMigratesPreExistingSchemaMissingNewColumns(t *testing.T) {
	path := t.TempDir() + "/answerable.db"

	legacy, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	_, err = legacy.Exec(`
		CREATE TABLE provider_config (
			id INTEGER PRIMARY KEY CHECK (id = 1),
			source_kind TEXT NOT NULL,
			source_value TEXT NOT NULL,
			refresh_interval_seconds INTEGER NOT NULL,
			webhook_url TEXT NOT NULL
		)
	`)
	if err != nil {
		t.Fatalf("create legacy table: %v", err)
	}
	_, err = legacy.Exec(
		`INSERT INTO provider_config (id, source_kind, source_value, refresh_interval_seconds, webhook_url) VALUES (1, ?, ?, ?, ?)`,
		"file", "/data/old.csv", 0, "",
	)
	if err != nil {
		t.Fatalf("seed legacy row: %v", err)
	}
	if err := legacy.Close(); err != nil {
		t.Fatalf("close legacy db: %v", err)
	}

	s, err := Open(path)
	if err != nil {
		t.Fatalf("Open on legacy schema: %v", err)
	}
	defer s.Close()

	got, ok, err := s.Load()
	if err != nil {
		t.Fatalf("Load on legacy schema: %v", err)
	}
	if !ok || got.Source.Value != "/data/old.csv" {
		t.Fatalf("Load = %+v, ok=%v, want the pre-existing row", got, ok)
	}

	if err := s.Save(Config{
		Source:          Source{Kind: SourceFile, Value: "/data/new.csv"},
		SourceLabel:     "new.csv",
		SourceUpdatedAt: time.Now(),
	}); err != nil {
		t.Fatalf("Save after migration: %v", err)
	}
}

func TestOpenMigratesPreExistingSchemaMissingEmailColumns(t *testing.T) {
	path := t.TempDir() + "/answerable.db"

	legacy, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	_, err = legacy.Exec(`
		CREATE TABLE provider_config (
			id INTEGER PRIMARY KEY CHECK (id = 1),
			source_kind TEXT NOT NULL,
			source_value TEXT NOT NULL,
			refresh_interval_seconds INTEGER NOT NULL,
			webhook_url TEXT NOT NULL,
			source_label TEXT NOT NULL DEFAULT '',
			source_updated_at_unix INTEGER NOT NULL DEFAULT 0
		)
	`)
	if err != nil {
		t.Fatalf("create legacy table: %v", err)
	}
	_, err = legacy.Exec(
		`INSERT INTO provider_config (id, source_kind, source_value, refresh_interval_seconds, webhook_url, source_label, source_updated_at_unix) VALUES (1, ?, ?, ?, ?, ?, ?)`,
		"file", "/data/old.csv", 0, "", "old.csv", 0,
	)
	if err != nil {
		t.Fatalf("seed legacy row: %v", err)
	}
	if err := legacy.Close(); err != nil {
		t.Fatalf("close legacy db: %v", err)
	}

	s, err := Open(path)
	if err != nil {
		t.Fatalf("Open on legacy schema: %v", err)
	}
	defer s.Close()

	got, ok, err := s.Load()
	if err != nil {
		t.Fatalf("Load on legacy schema: %v", err)
	}
	if !ok || got.Source.Value != "/data/old.csv" {
		t.Fatalf("Load = %+v, ok=%v, want the pre-existing row", got, ok)
	}
	if got.Email != (Email{}) {
		t.Errorf("Email = %+v, want zero value on legacy row", got.Email)
	}

	if err := s.Save(Config{
		Source: Source{Kind: SourceFile, Value: "/data/new.csv"},
		Email:  Email{SMTPHost: "smtp.example.com", SMTPPort: 587, To: "oncall@example.com"},
	}); err != nil {
		t.Fatalf("Save after migration: %v", err)
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
	if got.Email != want.Email {
		t.Errorf("Email = %+v, want %+v", got.Email, want.Email)
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
