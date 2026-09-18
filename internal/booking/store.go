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
	ID        string
	Name      string
	Contact   string
	Need      string
	Status    Status
	Token     string
	CreatedAt time.Time
	ExpiresAt time.Time
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

	return &Store{db: db}, nil
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

	now := nowFunc()
	req := Request{
		ID:        id,
		Name:      name,
		Contact:   contact,
		Need:      need,
		Status:    StatusPending,
		Token:     token,
		CreatedAt: now,
		ExpiresAt: now.Add(tokenTTL),
	}

	_, err = s.db.Exec(
		`INSERT INTO booking_requests (id, name, contact, need, status, token, created_at, expires_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		req.ID, req.Name, req.Contact, req.Need, string(req.Status), req.Token, req.CreatedAt, req.ExpiresAt,
	)
	if err != nil {
		return Request{}, err
	}

	return req, nil
}

func (s *Store) resolve(token string, next Status) (Request, error) {
	row := s.db.QueryRow(
		`SELECT id, name, contact, need, status, token, created_at, expires_at
		 FROM booking_requests WHERE token = ?`,
		token,
	)

	var req Request
	var status string
	if err := row.Scan(&req.ID, &req.Name, &req.Contact, &req.Need, &status, &req.Token, &req.CreatedAt, &req.ExpiresAt); err != nil {
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

func (s *Store) List() ([]Request, error) {
	rows, err := s.db.Query(
		`SELECT id, name, contact, need, status, token, created_at, expires_at
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
		if err := rows.Scan(&req.ID, &req.Name, &req.Contact, &req.Need, &status, &req.Token, &req.CreatedAt, &req.ExpiresAt); err != nil {
			return nil, err
		}
		req.Status = Status(status)
		list = append(list, req)
	}

	return list, rows.Err()
}
