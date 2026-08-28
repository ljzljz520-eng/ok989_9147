package plugin

import (
	"strings"
	"testing"
	"time"

	"internalauth/internal/domain"
)

func TestRenderLuaAndRouteConfig(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	config, _ := domain.NewRedisConfig(domain.CreateRedisConfig{ID: "redis", Name: "primary", Address: "redis.internal:6380", Password: "secret", Database: 2, PoolSize: 64, TimeoutMS: 800, TLS: true}, now)
	policy, _ := domain.NewRoutePolicy(domain.CreateRoutePolicy{ID: "route", Name: "billing", RouteURI: "/billing", RedisConfigID: config.ID, KeyPrefix: "billing:", Enabled: true}, now)
	route, err := BuildRouteConfig(policy, config)
	if err != nil {
		t.Fatal(err)
	}
	if route.RedisHost != "redis.internal" || route.RedisPort != 6380 {
		t.Fatalf("unexpected route config %#v", route)
	}
	lua := RenderLuaPlugin()
	if err := ValidateLuaArtifact(lua); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(lua, "set_keepalive") {
		t.Fatal("pool support absent")
	}
}
