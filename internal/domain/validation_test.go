package domain

import "testing"

func TestNormalizeRouteURI(t *testing.T) {
	got, err := NormalizeRouteURI("/internal//billing/")
	if err != nil {
		t.Fatal(err)
	}
	if got != "/internal/billing" {
		t.Fatalf("got %q", got)
	}
	if _, err := NormalizeRouteURI("https://example.test/internal"); err == nil {
		t.Fatal("absolute URI accepted")
	}
}

func TestValidateRedisConfigRejectsUnsafeValues(t *testing.T) {
	err := ValidateRedisConfig(RedisConfig{ID: "redis", Name: "primary", Address: "localhost", Database: 99, PoolSize: 0, TimeoutMS: 1})
	if err == nil {
		t.Fatal("invalid config accepted")
	}
	if got := err.Error(); got == "" {
		t.Fatal("empty validation error")
	}
}
