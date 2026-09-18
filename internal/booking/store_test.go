package booking

import (
	"testing"
	"time"
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
