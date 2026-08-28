package store

import (
	"errors"
	"path/filepath"
	"testing"
	"time"

	"internalauth/internal/domain"
)

func TestStoreRelationsAndDashboard(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "store.db"), Options{})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	config, _ := domain.NewRedisConfig(domain.CreateRedisConfig{ID: "redis-1", Name: "primary", Address: "127.0.0.1:6379", PoolSize: 20, TimeoutMS: 500}, now)
	policy, _ := domain.NewRoutePolicy(domain.CreateRoutePolicy{ID: "route-1", Name: "billing", RouteURI: "/billing", RedisConfigID: config.ID, Enabled: true}, now)
	if err := s.CreateRedisConfig(config); err != nil {
		t.Fatal(err)
	}
	if err := s.CreatePolicy(policy); err != nil {
		t.Fatal(err)
	}
	if err := s.DeleteRedisConfig(config.ID); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("got %v", err)
	}
	dashboard, err := s.Dashboard()
	if err != nil {
		t.Fatal(err)
	}
	if dashboard.Policies != 1 || dashboard.RedisConfigs != 1 {
		t.Fatalf("unexpected dashboard %#v", dashboard)
	}
}
