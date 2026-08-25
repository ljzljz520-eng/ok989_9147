package domain

import (
	"testing"
	"time"
)

func TestNewTokenRecordHashesPlaintext(t *testing.T) {
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	item, err := NewTokenRecord(IssueToken{ID: "token-1", Plaintext: "internal-secret-0001", Subject: "billing", ExpiresAt: now.Add(time.Hour)}, now)
	if err != nil {
		t.Fatal(err)
	}
	if item.TokenHash == "internal-secret-0001" || len(item.TokenHash) != 64 {
		t.Fatalf("unexpected hash %q", item.TokenHash)
	}
	if !TokenActive(item, now) {
		t.Fatal("new token should be active")
	}
}

func TestFilterPolicies(t *testing.T) {
	yes := true
	items := []RoutePolicy{{ID: "2", Name: "Zulu", Enabled: false}, {ID: "1", Name: "Alpha", Enabled: true, RedisConfigID: "redis-1"}}
	got := FilterPolicies(items, PolicyFilter{Enabled: &yes, RedisConfigID: "redis-1", Query: "alp"})
	if len(got) != 1 || got[0].ID != "1" {
		t.Fatalf("unexpected policies %#v", got)
	}
}
