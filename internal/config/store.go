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

type Email struct {
	SMTPHost string
	SMTPPort int
	Username string
	Password string
	From     string
	To       string
}

type Config struct {
	Source          Source
	RefreshInterval time.Duration
	WebhookURL      string
	Email           Email
	SourceLabel     string
	SourceUpdatedAt time.Time
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
			webhook_url TEXT NOT NULL,
			source_label TEXT NOT NULL DEFAULT '',
			source_updated_at_unix INTEGER NOT NULL DEFAULT 0
		)
	`)
	if err != nil {
		db.Close()
		return nil, err
	}

	if err := addColumnIfMissing(db, "provider_config", "source_label", "TEXT NOT NULL DEFAULT ''"); err != nil {
		db.Close()
		return nil, err
	}
	if err := addColumnIfMissing(db, "provider_config", "source_updated_at_unix", "INTEGER NOT NULL DEFAULT 0"); err != nil {
		db.Close()
		return nil, err
	}
	for col, decl := range map[string]string{
		"email_smtp_host": "TEXT NOT NULL DEFAULT ''",
		"email_smtp_port": "INTEGER NOT NULL DEFAULT 0",
		"email_username":  "TEXT NOT NULL DEFAULT ''",
		"email_password":  "TEXT NOT NULL DEFAULT ''",
		"email_from":      "TEXT NOT NULL DEFAULT ''",
		"email_to":        "TEXT NOT NULL DEFAULT ''",
	} {
		if err := addColumnIfMissing(db, "provider_config", col, decl); err != nil {
			db.Close()
			return nil, err
		}
	}

	return &Store{db: db}, nil
}

// addColumnIfMissing lets Store.Open handle a database created before a
// column existed, by adding it in place instead of erroring on every
// Save/Load. table/column/decl are always internal constants, never
// user input, so building the DDL string is safe.
func addColumnIfMissing(db *sql.DB, table, column, decl string) error {
	rows, err := db.Query("PRAGMA table_info(" + table + ")")
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var cid, notnull, pk int
		var name, ctype string
		var dflt sql.NullString
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			return err
		}
		if name == column {
			return rows.Close()
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}

	_, err = db.Exec("ALTER TABLE " + table + " ADD COLUMN " + column + " " + decl)
	return err
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) Load() (Config, bool, error) {
	row := s.db.QueryRow(`
		SELECT source_kind, source_value, refresh_interval_seconds, webhook_url,
			email_smtp_host, email_smtp_port, email_username, email_password, email_from, email_to,
			source_label, source_updated_at_unix
		FROM provider_config WHERE id = 1
	`)

	var kind, value, webhookURL, label string
	var emailHost, emailUser, emailPass, emailFrom, emailTo string
	var emailPort int
	var seconds, updatedAtUnix int64
	if err := row.Scan(
		&kind, &value, &seconds, &webhookURL,
		&emailHost, &emailPort, &emailUser, &emailPass, &emailFrom, &emailTo,
		&label, &updatedAtUnix,
	); err != nil {
		if err == sql.ErrNoRows {
			return Config{}, false, nil
		}
		return Config{}, false, err
	}

	cfg := Config{
		Source:          Source{Kind: SourceKind(kind), Value: value},
		RefreshInterval: time.Duration(seconds) * time.Second,
		WebhookURL:      webhookURL,
		Email: Email{
			SMTPHost: emailHost,
			SMTPPort: emailPort,
			Username: emailUser,
			Password: emailPass,
			From:     emailFrom,
			To:       emailTo,
		},
		SourceLabel:     label,
		SourceUpdatedAt: unixToTime(updatedAtUnix),
	}
	return cfg, true, nil
}

func (s *Store) Save(cfg Config) error {
	_, err := s.db.Exec(`
		INSERT INTO provider_config (
			id, source_kind, source_value, refresh_interval_seconds, webhook_url,
			email_smtp_host, email_smtp_port, email_username, email_password, email_from, email_to,
			source_label, source_updated_at_unix
		)
		VALUES (1, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			source_kind = excluded.source_kind,
			source_value = excluded.source_value,
			refresh_interval_seconds = excluded.refresh_interval_seconds,
			webhook_url = excluded.webhook_url,
			email_smtp_host = excluded.email_smtp_host,
			email_smtp_port = excluded.email_smtp_port,
			email_username = excluded.email_username,
			email_password = excluded.email_password,
			email_from = excluded.email_from,
			email_to = excluded.email_to,
			source_label = excluded.source_label,
			source_updated_at_unix = excluded.source_updated_at_unix
	`,
		string(cfg.Source.Kind), cfg.Source.Value,
		int64(cfg.RefreshInterval/time.Second), cfg.WebhookURL,
		cfg.Email.SMTPHost, cfg.Email.SMTPPort, cfg.Email.Username, cfg.Email.Password, cfg.Email.From, cfg.Email.To,
		cfg.SourceLabel, timeToUnix(cfg.SourceUpdatedAt),
	)
	return err
}

func timeToUnix(t time.Time) int64 {
	if t.IsZero() {
		return 0
	}
	return t.Unix()
}

func unixToTime(sec int64) time.Time {
	if sec == 0 {
		return time.Time{}
	}
	return time.Unix(sec, 0).UTC()
}
