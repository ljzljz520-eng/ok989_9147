package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"time"
)

type CreateRedisConfig struct {
	ID, Name, Address, Password   string
	Database, PoolSize, TimeoutMS int
	TLS                           bool
}
type CreateRoutePolicy struct {
	ID, Name, RouteURI, RedisConfigID, KeyPrefix string
	Enabled                                      bool
}
type IssueToken struct {
	ID, Plaintext, Subject, Description string
	ExpiresAt                           time.Time
}
type AuditFilter struct {
	RouteID, Outcome string
	From, To         time.Time
	Limit            int
}
type PolicyFilter struct {
	Enabled              *bool
	RedisConfigID, Query string
}

func NewRedisConfig(c CreateRedisConfig, now time.Time) (RedisConfig, error) {
	out := RedisConfig{ID: strings.TrimSpace(c.ID), Name: strings.TrimSpace(c.Name), Address: strings.TrimSpace(c.Address), Password: c.Password, Database: c.Database, PoolSize: c.PoolSize, TimeoutMS: c.TimeoutMS, TLS: c.TLS, CreatedAt: now, UpdatedAt: now}
	if err := ValidateRedisConfig(out); err != nil {
		return RedisConfig{}, err
	}
	return out, nil
}

func NewRoutePolicy(c CreateRoutePolicy, now time.Time) (RoutePolicy, error) {
	uri, err := NormalizeRouteURI(c.RouteURI)
	if err != nil {
		return RoutePolicy{}, err
	}
	prefix := strings.TrimSpace(c.KeyPrefix)
	if prefix == "" {
		prefix = "internal:token:"
	}
	out := RoutePolicy{ID: strings.TrimSpace(c.ID), Name: strings.TrimSpace(c.Name), RouteURI: uri, RedisConfigID: strings.TrimSpace(c.RedisConfigID), TokenHeader: "x-internal-token", KeyPrefix: prefix, Enabled: c.Enabled, Revision: 1, CreatedAt: now, UpdatedAt: now}
	if err := ValidateRoutePolicy(out); err != nil {
		return RoutePolicy{}, err
	}
	return out, nil
}

func NewTokenRecord(c IssueToken, now time.Time) (TokenRecord, error) {
	if len(c.Plaintext) < 16 {
		return TokenRecord{}, fmt.Errorf("%w: token must contain at least 16 characters", ErrInvalid)
	}
	digest := sha256.Sum256([]byte(c.Plaintext))
	out := TokenRecord{ID: strings.TrimSpace(c.ID), TokenHash: hex.EncodeToString(digest[:]), Subject: strings.TrimSpace(c.Subject), Description: strings.TrimSpace(c.Description), ExpiresAt: c.ExpiresAt, CreatedAt: now, UpdatedAt: now}
	if err := ValidateTokenRecord(out); err != nil {
		return TokenRecord{}, err
	}
	return out, nil
}

func HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func FilterPolicies(items []RoutePolicy, filter PolicyFilter) []RoutePolicy {
	out := make([]RoutePolicy, 0)
	query := strings.ToLower(strings.TrimSpace(filter.Query))
	for _, item := range items {
		if filter.Enabled != nil && item.Enabled != *filter.Enabled {
			continue
		}
		if filter.RedisConfigID != "" && item.RedisConfigID != filter.RedisConfigID {
			continue
		}
		if query != "" && !strings.Contains(strings.ToLower(item.Name+" "+item.RouteURI), query) {
			continue
		}
		out = append(out, item)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func FilterAudits(items []AuditEvent, filter AuditFilter) []AuditEvent {
	out := make([]AuditEvent, 0)
	for _, item := range items {
		if filter.RouteID != "" && item.RouteID != filter.RouteID {
			continue
		}
		if filter.Outcome != "" && item.Outcome != filter.Outcome {
			continue
		}
		if !filter.From.IsZero() && item.OccurredAt.Before(filter.From) {
			continue
		}
		if !filter.To.IsZero() && item.OccurredAt.After(filter.To) {
			continue
		}
		out = append(out, item)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].OccurredAt.After(out[j].OccurredAt) })
	if filter.Limit > 0 && len(out) > filter.Limit {
		out = out[:filter.Limit]
	}
	return out
}
