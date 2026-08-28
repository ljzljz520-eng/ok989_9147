package auth

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"internalauth/internal/domain"
	"internalauth/internal/store"
)

func TestAuthorizeOutcomes(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	repo, err := store.Open(filepath.Join(t.TempDir(), "auth.db"), store.Options{})
	if err != nil {
		t.Fatal(err)
	}
	defer repo.Close()
	config, _ := domain.NewRedisConfig(domain.CreateRedisConfig{ID: "redis-1", Name: "primary", Address: "127.0.0.1:6379", PoolSize: 10, TimeoutMS: 100}, now)
	policy, _ := domain.NewRoutePolicy(domain.CreateRoutePolicy{ID: "route-1", Name: "billing", RouteURI: "/billing", RedisConfigID: config.ID, Enabled: true}, now)
	repo.CreateRedisConfig(config)
	repo.CreatePolicy(policy)
	redis := NewMemoryRedis()
	redis.Put(config.ID, policy.KeyPrefix+"valid-token")
	service, _ := NewService(repo, redis, FixedClock{Value: now}, &SequenceIDs{})
	allowed, err := service.Authorize(context.Background(), domain.AuthRequest{RouteID: policy.ID, Headers: map[string]string{"X-Internal-Token": "valid-token"}})
	if err != nil || allowed.StatusCode != 200 {
		t.Fatalf("allowed=%#v err=%v", allowed, err)
	}
	denied, err := service.Authorize(context.Background(), domain.AuthRequest{RouteID: policy.ID, Headers: map[string]string{"x-internal-token": "unknown"}})
	if err != nil || denied.StatusCode != 403 {
		t.Fatalf("denied=%#v err=%v", denied, err)
	}
	redis.SetFailure(config.ID, errors.New("offline"))
	unavailable, err := service.Authorize(context.Background(), domain.AuthRequest{RouteID: policy.ID, Headers: map[string]string{"x-internal-token": "valid-token"}})
	if err != nil || unavailable.StatusCode != 503 {
		t.Fatalf("unavailable=%#v err=%v", unavailable, err)
	}
}
