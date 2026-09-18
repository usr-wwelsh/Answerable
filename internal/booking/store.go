package booking

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"time"

	_ "modernc.org/sqlite"
)

type Status string

const (
	StatusPending   Status = "pending"
	StatusConfirmed Status = "confirmed"
	StatusDenied    Status = "denied"

	tokenTTL = 48 * time.Hour
)

var nowFunc = time.Now

type Request struct {
	ID          string
	Name        string
	Contact     string
	Need        string
	Status      Status
	Token       string
	StatusToken string
	CreatedAt   time.Time
	ExpiresAt   time.Time
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
		CREATE TABLE IF NOT EXISTS booking_requests (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			contact TEXT NOT NULL,
			need TEXT NOT NULL,
			status TEXT NOT NULL,
			token TEXT NOT NULL UNIQUE,
			created_at TIMESTAMP NOT NULL,
			expires_at TIMESTAMP NOT NULL
		)
	`)
	if err != nil {
		db.Close()
		return nil, err
	}

	if err := addColumnIfMissing(db, "booking_requests", "status_token", "TEXT NOT NULL DEFAULT ''"); err != nil {
		db.Close()
		return nil, err
	}

	return &Store{db: db}, nil
}

// addColumnIfMissing lets Open handle a database created before a column
// existed, by adding it in place instead of erroring on every query.
// table/column/decl are always internal constants, never user input, so
// building the DDL string is safe.
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

func randomToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func randomID() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func (s *Store) Enqueue(name, contact, need string) (Request, error) {
	id, err := randomID()
	if err != nil {
		return Request{}, err
	}
	token, err := randomToken()
	if err != nil {
		return Request{}, err
	}
	statusToken, err := randomToken()
	if err != nil {
		return Request{}, err
	}

	now := nowFunc()
	req := Request{
		ID:          id,
		Name:        name,
		Contact:     contact,
		Need:        need,
		Status:      StatusPending,
		Token:       token,
		StatusToken: statusToken,
		CreatedAt:   now,
		ExpiresAt:   now.Add(tokenTTL),
	}

	_, err = s.db.Exec(
		`INSERT INTO booking_requests (id, name, contact, need, status, token, status_token, created_at, expires_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		req.ID, req.Name, req.Contact, req.Need, string(req.Status), req.Token, req.StatusToken, req.CreatedAt, req.ExpiresAt,
	)
	if err != nil {
		return Request{}, err
	}

	return req, nil
}

func (s *Store) resolve(token string, next Status) (Request, error) {
	row := s.db.QueryRow(
		`SELECT id, name, contact, need, status, token, status_token, created_at, expires_at
		 FROM booking_requests WHERE token = ?`,
		token,
	)

	var req Request
	var status string
	if err := row.Scan(&req.ID, &req.Name, &req.Contact, &req.Need, &status, &req.Token, &req.StatusToken, &req.CreatedAt, &req.ExpiresAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Request{}, errors.New("unknown token")
		}
		return Request{}, err
	}
	req.Status = Status(status)

	if req.Status != StatusPending {
		return Request{}, errors.New("token already used")
	}
	if nowFunc().After(req.ExpiresAt) {
		return Request{}, errors.New("token expired")
	}

	if _, err := s.db.Exec(`UPDATE booking_requests SET status = ? WHERE token = ?`, string(next), token); err != nil {
		return Request{}, err
	}
	req.Status = next

	return req, nil
}

func (s *Store) Confirm(token string) (Request, error) {
	return s.resolve(token, StatusConfirmed)
}

func (s *Store) Deny(token string) (Request, error) {
	return s.resolve(token, StatusDenied)
}

// Status looks up a request by its status token — the read-only identifier
// handed back to whoever submitted the request, distinct from the
// confirm/deny Token so a requester can check in on their own request
// without ever being able to resolve it. Unlike Confirm/Deny, this never
// mutates state and can be called any number of times.
func (s *Store) Status(statusToken string) (Request, error) {
	if statusToken == "" {
		return Request{}, errors.New("unknown status token")
	}

	row := s.db.QueryRow(
		`SELECT id, name, contact, need, status, token, status_token, created_at, expires_at
		 FROM booking_requests WHERE status_token = ?`,
		statusToken,
	)

	var req Request
	var status string
	if err := row.Scan(&req.ID, &req.Name, &req.Contact, &req.Need, &status, &req.Token, &req.StatusToken, &req.CreatedAt, &req.ExpiresAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Request{}, errors.New("unknown status token")
		}
		return Request{}, err
	}
	req.Status = Status(status)

	return req, nil
}

func (s *Store) List() ([]Request, error) {
	rows, err := s.db.Query(
		`SELECT id, name, contact, need, status, token, status_token, created_at, expires_at
		 FROM booking_requests ORDER BY created_at ASC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []Request
	for rows.Next() {
		var req Request
		var status string
		if err := rows.Scan(&req.ID, &req.Name, &req.Contact, &req.Need, &status, &req.Token, &req.StatusToken, &req.CreatedAt, &req.ExpiresAt); err != nil {
			return nil, err
		}
		req.Status = Status(status)
		list = append(list, req)
	}

	return list, rows.Err()
}
