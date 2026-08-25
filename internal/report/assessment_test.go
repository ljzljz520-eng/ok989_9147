package report

import (
	"strings"
	"testing"
	"time"

	"internalauth/internal/domain"
)

func TestAssessmentAndRouteMatrix(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	config := domain.RedisConfig{ID: "redis", Name: "primary", Address: "redis:6379", PoolSize: 5, TimeoutMS: 4000}
	policy := domain.RoutePolicy{ID: "route", RouteURI: "/billing", RedisConfigID: config.ID, Enabled: true}
	audits := []domain.AuditEvent{{RouteID: policy.ID, Outcome: "unavailable"}, {RouteID: policy.ID, Outcome: "denied"}}
	assessment := Assess(Input{Policies: []domain.RoutePolicy{policy}, Configs: []domain.RedisConfig{config}, Audits: audits, Now: now})
	if assessment.Score >= 100 || len(assessment.Findings) == 0 {
		t.Fatalf("unexpected assessment %#v", assessment)
	}
	table := RenderTable(RouteMatrix([]domain.RoutePolicy{policy}, []domain.RedisConfig{config}))
	if !strings.Contains(table, "redis:6379") {
		t.Fatalf("unexpected table %s", table)
	}
}
