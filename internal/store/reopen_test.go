package store

import (
	"path/filepath"
	"testing"
	"time"

	"internalauth/internal/domain"
)

func TestPersistenceSurvivesReopen(t *testing.T) {
	path := filepath.Join(t.TempDir(), "persistent.db")
	now := time.Date(2026, 2, 3, 4, 5, 6, 0, time.UTC)
	first, err := Open(path, Options{})
	if err != nil {
		t.Fatal(err)
	}
	config, _ := domain.NewRedisConfig(domain.CreateRedisConfig{ID: "redis-reopen", Name: "reopen", Address: "127.0.0.1:6379", PoolSize: 16, TimeoutMS: 400}, now)
	policy, _ := domain.NewRoutePolicy(domain.CreateRoutePolicy{ID: "policy-reopen", Name: "orders", RouteURI: "/orders", RedisConfigID: config.ID, Enabled: true}, now)
	token, _ := domain.NewTokenRecord(domain.IssueToken{ID: "token-reopen", Plaintext: "reopen-token-value", Subject: "orders", ExpiresAt: now.Add(time.Hour)}, now)
	if err := first.CreateRedisConfig(config); err != nil {
		t.Fatal(err)
	}
	if err := first.CreatePolicy(policy); err != nil {
		t.Fatal(err)
	}
	if err := first.CreateToken(token); err != nil {
		t.Fatal(err)
	}
	if err := first.AppendAudit(domain.AuditEvent{ID: "audit-reopen", RouteID: policy.ID, Outcome: "allowed", Reason: "accepted", StatusCode: 200, OccurredAt: now}); err != nil {
		t.Fatal(err)
	}
	if err := first.Close(); err != nil {
		t.Fatal(err)
	}
	second, err := Open(path, Options{})
	if err != nil {
		t.Fatal(err)
	}
	defer second.Close()
	if _, err := second.GetRedisConfig(config.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := second.GetPolicy(policy.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := second.GetToken(token.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := second.GetAudit("audit-reopen"); err != nil {
		t.Fatal(err)
	}
}
