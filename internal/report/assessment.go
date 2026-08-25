package report

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"internalauth/internal/domain"
)

type Severity string

const (
	SeverityInfo     Severity = "info"
	SeverityWarning  Severity = "warning"
	SeverityCritical Severity = "critical"
)

type Finding struct {
	Code         string   `json:"code"`
	Severity     Severity `json:"severity"`
	ResourceType string   `json:"resource_type"`
	ResourceID   string   `json:"resource_id"`
	Message      string   `json:"message"`
	Remediation  string   `json:"remediation"`
}
type Assessment struct {
	GeneratedAt time.Time `json:"generated_at"`
	Score       int       `json:"score"`
	Findings    []Finding `json:"findings"`
	PolicyCount int       `json:"policy_count"`
	ConfigCount int       `json:"config_count"`
	TokenCount  int       `json:"token_count"`
	AuditCount  int       `json:"audit_count"`
}

type Input struct {
	Policies []domain.RoutePolicy
	Configs  []domain.RedisConfig
	Tokens   []domain.TokenRecord
	Audits   []domain.AuditEvent
	Now      time.Time
}

func Assess(input Input) Assessment {
	out := Assessment{GeneratedAt: input.Now, Score: 100, PolicyCount: len(input.Policies), ConfigCount: len(input.Configs), TokenCount: len(input.Tokens), AuditCount: len(input.Audits), Findings: make([]Finding, 0)}
	configs := make(map[string]domain.RedisConfig, len(input.Configs))
	for _, config := range input.Configs {
		configs[config.ID] = config
		out.Findings = append(out.Findings, assessConfig(config)...)
	}
	for _, policy := range input.Policies {
		out.Findings = append(out.Findings, assessPolicy(policy, configs)...)
	}
	for _, token := range input.Tokens {
		out.Findings = append(out.Findings, assessToken(token, input.Now)...)
	}
	out.Findings = append(out.Findings, assessAudits(input.Audits)...)
	for _, finding := range out.Findings {
		switch finding.Severity {
		case SeverityCritical:
			out.Score -= 20
		case SeverityWarning:
			out.Score -= 8
		case SeverityInfo:
			out.Score -= 1
		}
	}
	if out.Score < 0 {
		out.Score = 0
	}
	sort.Slice(out.Findings, func(i, j int) bool {
		left, right := severityRank(out.Findings[i].Severity), severityRank(out.Findings[j].Severity)
		if left == right {
			if out.Findings[i].ResourceType == out.Findings[j].ResourceType {
				return out.Findings[i].ResourceID < out.Findings[j].ResourceID
			}
			return out.Findings[i].ResourceType < out.Findings[j].ResourceType
		}
		return left > right
	})
	return out
}

func assessConfig(config domain.RedisConfig) []Finding {
	out := make([]Finding, 0)
	if config.PoolSize < 10 {
		out = append(out, Finding{"REDIS_POOL_SMALL", SeverityWarning, "redis_config", config.ID, "connection pool has fewer than ten slots", "raise pool_size after measuring gateway concurrency"})
	}
	if config.PoolSize > 500 {
		out = append(out, Finding{"REDIS_POOL_LARGE", SeverityInfo, "redis_config", config.ID, "connection pool can create more than five hundred connections", "confirm Redis maxclients and APISIX worker count"})
	}
	if config.TimeoutMS > 3000 {
		out = append(out, Finding{"REDIS_TIMEOUT_HIGH", SeverityWarning, "redis_config", config.ID, "Redis timeout can hold a gateway request for more than three seconds", "use a bounded timeout below the upstream request budget"})
	}
	if config.TimeoutMS < 50 {
		out = append(out, Finding{"REDIS_TIMEOUT_LOW", SeverityInfo, "redis_config", config.ID, "Redis timeout is below fifty milliseconds", "confirm network latency leaves adequate headroom"})
	}
	if config.Password == "" {
		out = append(out, Finding{"REDIS_NO_PASSWORD", SeverityWarning, "redis_config", config.ID, "Redis authentication is not configured", "set a route-scoped Redis credential"})
	}
	if !config.TLS {
		out = append(out, Finding{"REDIS_NO_TLS", SeverityInfo, "redis_config", config.ID, "Redis transport encryption is disabled", "enable TLS when traffic crosses a trust boundary"})
	}
	return out
}

func assessPolicy(policy domain.RoutePolicy, configs map[string]domain.RedisConfig) []Finding {
	out := make([]Finding, 0)
	if _, ok := configs[policy.RedisConfigID]; !ok {
		out = append(out, Finding{"POLICY_CONFIG_MISSING", SeverityCritical, "route_policy", policy.ID, "policy references a missing Redis configuration", "bind the route to an existing Redis configuration"})
	}
	if !policy.Enabled {
		out = append(out, Finding{"POLICY_DISABLED", SeverityInfo, "route_policy", policy.ID, "route authentication is disabled", "enable the policy before publishing the route"})
	}
	if policy.Revision > 50 {
		out = append(out, Finding{"POLICY_REVISION_HIGH", SeverityInfo, "route_policy", policy.ID, "policy has been revised more than fifty times", "review change history and retire obsolete deployments"})
	}
	if policy.KeyPrefix == "internal:token:" {
		out = append(out, Finding{"POLICY_SHARED_PREFIX", SeverityInfo, "route_policy", policy.ID, "policy uses the global default key prefix", "consider a service-specific prefix to constrain token scope"})
	}
	if strings.Contains(policy.RouteURI, "*") {
		out = append(out, Finding{"POLICY_WILDCARD_ROUTE", SeverityWarning, "route_policy", policy.ID, "policy URI contains a wildcard", "confirm every matched internal endpoint needs the same token scope"})
	}
	return out
}

func assessToken(token domain.TokenRecord, now time.Time) []Finding {
	out := make([]Finding, 0)
	if token.Revoked {
		return out
	}
	remaining := token.ExpiresAt.Sub(now)
	if remaining <= 0 {
		out = append(out, Finding{"TOKEN_EXPIRED", SeverityWarning, "token_record", token.ID, "token record is expired but not revoked", "revoke or purge the expired record"})
	} else if remaining < 24*time.Hour {
		out = append(out, Finding{"TOKEN_EXPIRING", SeverityInfo, "token_record", token.ID, "token expires within twenty-four hours", "rotate the token before expiration"})
	}
	if strings.TrimSpace(token.Description) == "" {
		out = append(out, Finding{"TOKEN_UNDOCUMENTED", SeverityInfo, "token_record", token.ID, "token has no operational description", "record its owning workload and purpose"})
	}
	return out
}

func assessAudits(items []domain.AuditEvent) []Finding {
	if len(items) == 0 {
		return []Finding{{"AUDIT_EMPTY", SeverityWarning, "audit", "global", "no authorization events are available", "verify that gateway requests reach the plugin"}}
	}
	counts := map[string]int{}
	byRoute := map[string]map[string]int{}
	for _, item := range items {
		counts[item.Outcome]++
		route := byRoute[item.RouteID]
		if route == nil {
			route = map[string]int{}
			byRoute[item.RouteID] = route
		}
		route[item.Outcome]++
	}
	out := make([]Finding, 0)
	if counts["unavailable"]*10 > len(items) {
		out = append(out, Finding{"AUDIT_REDIS_UNAVAILABLE", SeverityCritical, "audit", "global", "more than ten percent of authorization attempts report Redis unavailability", "inspect Redis reachability, pool saturation, and timeout settings"})
	}
	if counts["denied"]*2 > len(items) {
		out = append(out, Finding{"AUDIT_DENIAL_HIGH", SeverityWarning, "audit", "global", "more than half of authorization attempts are denied", "check token distribution and route key prefixes"})
	}
	for routeID, routeCounts := range byRoute {
		total := routeCounts["allowed"] + routeCounts["denied"] + routeCounts["unavailable"]
		if total >= 5 && routeCounts["unavailable"]*4 >= total {
			out = append(out, Finding{"ROUTE_UNAVAILABLE_HIGH", SeverityCritical, "route_policy", routeID, fmt.Sprintf("%d of %d recent attempts were unavailable", routeCounts["unavailable"], total), "validate the Redis configuration assigned to this route"})
		}
	}
	return out
}

func severityRank(value Severity) int {
	switch value {
	case SeverityCritical:
		return 3
	case SeverityWarning:
		return 2
	case SeverityInfo:
		return 1
	default:
		return 0
	}
}

func Summarize(assessment Assessment) string {
	counts := map[Severity]int{}
	for _, finding := range assessment.Findings {
		counts[finding.Severity]++
	}
	return fmt.Sprintf("score=%d critical=%d warning=%d info=%d policies=%d configs=%d tokens=%d audits=%d", assessment.Score, counts[SeverityCritical], counts[SeverityWarning], counts[SeverityInfo], assessment.PolicyCount, assessment.ConfigCount, assessment.TokenCount, assessment.AuditCount)
}
