package admin

import (
	"path/filepath"
	"testing"
	"time"

	"internalauth/internal/auth"
	"internalauth/internal/domain"
	"internalauth/internal/store"
)

func TestTokenLifecycle(t *testing.T) {
	now := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)
	repo, err := store.Open(filepath.Join(t.TempDir(), "admin.db"), store.Options{})
	if err != nil {
		t.Fatal(err)
	}
	defer repo.Close()
	service, _ := NewService(repo, auth.FixedClock{Value: now}, &auth.SequenceIDs{})
	item, err := service.IssueToken(domain.IssueToken{ID: "token-1", Plaintext: "long-internal-token", Subject: "inventory", ExpiresAt: now.Add(time.Hour)})
	if err != nil {
		t.Fatal(err)
	}
	status, err := service.TokenStatus(item.ID)
	if err != nil || status != "active" {
		t.Fatalf("status=%s err=%v", status, err)
	}
	if _, err := service.RevokeToken(item.ID); err != nil {
		t.Fatal(err)
	}
	status, _ = service.TokenStatus(item.ID)
	if status != "revoked" {
		t.Fatalf("status=%s", status)
	}
}
