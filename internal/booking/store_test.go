package booking

import (
	"database/sql"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func openTestStore(t *testing.T) *Store {
	t.Helper()
	s, err := Open(":memory:")
	if err != nil {
		t.Fatalf("Open returned error: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestEnqueueGeneratesIDTokenAndPendingStatus(t *testing.T) {
	s := openTestStore(t)

	req, err := s.Enqueue("Jane Doe", "555-0100", "bed for two tonight")
	if err != nil {
		t.Fatalf("Enqueue returned error: %v", err)
	}

	if req.ID == "" {
		t.Error("ID is empty")
	}
	if req.Token == "" {
		t.Error("Token is empty")
	}
	if req.StatusToken == "" {
		t.Error("StatusToken is empty")
	}
	if req.StatusToken == req.Token {
		t.Error("StatusToken must differ from the confirm/deny Token — the requester should never be able to resolve their own request")
	}
	if req.Status != StatusPending {
		t.Errorf("Status = %q, want %q", req.Status, StatusPending)
	}
	if req.Name != "Jane Doe" || req.Contact != "555-0100" || req.Need != "bed for two tonight" {
		t.Errorf("fields not persisted correctly: %+v", req)
	}
	if !req.ExpiresAt.After(req.CreatedAt) {
		t.Errorf("ExpiresAt %v should be after CreatedAt %v", req.ExpiresAt, req.CreatedAt)
	}
}

func TestConfirmTransitionsPendingToConfirmed(t *testing.T) {
	s := openTestStore(t)
	req, _ := s.Enqueue("Jane Doe", "555-0100", "bed for two tonight")

	confirmed, err := s.Confirm(req.Token)
	if err != nil {
		t.Fatalf("Confirm returned error: %v", err)
	}
	if confirmed.Status != StatusConfirmed {
		t.Errorf("Status = %q, want %q", confirmed.Status, StatusConfirmed)
	}
}

func TestConfirmRejectsUnknownToken(t *testing.T) {
	s := openTestStore(t)

	if _, err := s.Confirm("not-a-real-token"); err == nil {
		t.Fatal("expected error for unknown token, got nil")
	}
}

func TestConfirmRejectsAlreadyUsedToken(t *testing.T) {
	s := openTestStore(t)
	req, _ := s.Enqueue("Jane Doe", "555-0100", "bed for two tonight")

	if _, err := s.Confirm(req.Token); err != nil {
		t.Fatalf("first confirm failed: %v", err)
	}
	if _, err := s.Confirm(req.Token); err == nil {
		t.Fatal("expected error re-using an already-confirmed token, got nil")
	}
}

func TestConfirmRejectsExpiredToken(t *testing.T) {
	s := openTestStore(t)
	realNow := nowFunc
	defer func() { nowFunc = realNow }()

	req, _ := s.Enqueue("Jane Doe", "555-0100", "bed for two tonight")

	nowFunc = func() time.Time { return req.ExpiresAt.Add(time.Second) }

	if _, err := s.Confirm(req.Token); err == nil {
		t.Fatal("expected error for expired token, got nil")
	}
}

func TestDenyTransitionsPendingToDenied(t *testing.T) {
	s := openTestStore(t)
	req, _ := s.Enqueue("Jane Doe", "555-0100", "bed for two tonight")

	denied, err := s.Deny(req.Token)
	if err != nil {
		t.Fatalf("Deny returned error: %v", err)
	}
	if denied.Status != StatusDenied {
		t.Errorf("Status = %q, want %q", denied.Status, StatusDenied)
	}
}

func TestStatusReturnsPendingBeforeResolution(t *testing.T) {
	s := openTestStore(t)
	req, _ := s.Enqueue("Jane Doe", "555-0100", "bed for two tonight")

	got, err := s.Status(req.StatusToken)
	if err != nil {
		t.Fatalf("Status returned error: %v", err)
	}
	if got.Status != StatusPending {
		t.Errorf("Status = %q, want %q", got.Status, StatusPending)
	}
}

func TestStatusReflectsConfirmedAfterConfirm(t *testing.T) {
	s := openTestStore(t)
	req, _ := s.Enqueue("Jane Doe", "555-0100", "bed for two tonight")
	if _, err := s.Confirm(req.Token); err != nil {
		t.Fatalf("Confirm returned error: %v", err)
	}

	got, err := s.Status(req.StatusToken)
	if err != nil {
		t.Fatalf("Status returned error: %v", err)
	}
	if got.Status != StatusConfirmed {
		t.Errorf("Status = %q, want %q", got.Status, StatusConfirmed)
	}
}

func TestStatusReflectsDeniedAfterDeny(t *testing.T) {
	s := openTestStore(t)
	req, _ := s.Enqueue("Jane Doe", "555-0100", "bed for two tonight")
	if _, err := s.Deny(req.Token); err != nil {
		t.Fatalf("Deny returned error: %v", err)
	}

	got, err := s.Status(req.StatusToken)
	if err != nil {
		t.Fatalf("Status returned error: %v", err)
	}
	if got.Status != StatusDenied {
		t.Errorf("Status = %q, want %q", got.Status, StatusDenied)
	}
}

func TestStatusRejectsUnknownToken(t *testing.T) {
	s := openTestStore(t)

	if _, err := s.Status("not-a-real-token"); err == nil {
		t.Fatal("expected error for unknown status token, got nil")
	}
}

func TestStatusDoesNotConsumeTheStatusToken(t *testing.T) {
	s := openTestStore(t)
	req, _ := s.Enqueue("Jane Doe", "555-0100", "bed for two tonight")

	if _, err := s.Status(req.StatusToken); err != nil {
		t.Fatalf("first Status call returned error: %v", err)
	}
	if _, err := s.Status(req.StatusToken); err != nil {
		t.Fatalf("second Status call returned error: %v — status checks must be repeatable", err)
	}
}

func TestOpenMigratesPreExistingSchemaMissingStatusTokenColumn(t *testing.T) {
	path := t.TempDir() + "/answerable.db"

	legacy, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	_, err = legacy.Exec(`
		CREATE TABLE booking_requests (
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
		t.Fatalf("create legacy table: %v", err)
	}
	_, err = legacy.Exec(
		`INSERT INTO booking_requests (id, name, contact, need, status, token, created_at, expires_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		"legacy-id", "Old Requester", "555-0199", "bed", string(StatusPending), "legacy-token",
		time.Now(), time.Now().Add(48*time.Hour),
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

	if _, err := s.Enqueue("New Requester", "555-0200", "bed"); err != nil {
		t.Fatalf("Enqueue on migrated schema: %v", err)
	}

	list, err := s.List()
	if err != nil {
		t.Fatalf("List on migrated schema: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 requests (legacy + new), got %d: %+v", len(list), list)
	}
}

func TestListReturnsAllRequestsInCreatedOrder(t *testing.T) {
	s := openTestStore(t)
	first, _ := s.Enqueue("Jane Doe", "555-0100", "bed for two tonight")
	second, _ := s.Enqueue("John Roe", "555-0101", "bed for one tonight")

	list, err := s.List()
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 requests, got %d", len(list))
	}
	if list[0].ID != first.ID || list[1].ID != second.ID {
		t.Errorf("List order = %v, want [%s, %s]", list, first.ID, second.ID)
	}
}
