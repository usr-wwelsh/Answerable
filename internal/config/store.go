package config

import (
	"database/sql"
	"time"

	_ "modernc.org/sqlite"
)

type SourceKind string

const (
	SourceFile SourceKind = "file"
	SourceURL  SourceKind = "url"
)

type Source struct {
	Kind  SourceKind
	Value string
}

type Config struct {
	Source          Source
	RefreshInterval time.Duration
	WebhookURL      string
}

type Store struct {
	db *sql.DB
}

func Open(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS provider_config (
			id INTEGER PRIMARY KEY CHECK (id = 1),
			source_kind TEXT NOT NULL,
			source_value TEXT NOT NULL,
			refresh_interval_seconds INTEGER NOT NULL,
			webhook_url TEXT NOT NULL
		)
	`)
	if err != nil {
		db.Close()
		return nil, err
	}

	return &Store{db: db}, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) Load() (Config, bool, error) {
	row := s.db.QueryRow(`
		SELECT source_kind, source_value, refresh_interval_seconds, webhook_url
		FROM provider_config WHERE id = 1
	`)

	var kind, value, webhookURL string
	var seconds int64
	if err := row.Scan(&kind, &value, &seconds, &webhookURL); err != nil {
		if err == sql.ErrNoRows {
			return Config{}, false, nil
		}
		return Config{}, false, err
	}

	cfg := Config{
		Source:          Source{Kind: SourceKind(kind), Value: value},
		RefreshInterval: time.Duration(seconds) * time.Second,
		WebhookURL:      webhookURL,
	}
	return cfg, true, nil
}

func (s *Store) Save(cfg Config) error {
	_, err := s.db.Exec(`
		INSERT INTO provider_config (id, source_kind, source_value, refresh_interval_seconds, webhook_url)
		VALUES (1, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			source_kind = excluded.source_kind,
			source_value = excluded.source_value,
			refresh_interval_seconds = excluded.refresh_interval_seconds,
			webhook_url = excluded.webhook_url
	`,
		string(cfg.Source.Kind), cfg.Source.Value,
		int64(cfg.RefreshInterval/time.Second), cfg.WebhookURL,
	)
	return err
}
