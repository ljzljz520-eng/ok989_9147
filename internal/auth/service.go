package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"internalauth/internal/domain"
)

type Repository interface {
	GetPolicy(string) (domain.RoutePolicy, error)
	GetRedisConfig(string) (domain.RedisConfig, error)
	AppendAudit(domain.AuditEvent) error
}

type Clock interface{ Now() time.Time }
type FixedClock struct{ Value time.Time }

func (f FixedClock) Now() time.Time { return f.Value }

type IDSource interface{ Next(string) string }
type SequenceIDs struct {
	mu sync.Mutex
	n  int
}

func (s *SequenceIDs) Next(prefix string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.n++
	return fmt.Sprintf("%s-%06d", prefix, s.n)
}

type Service struct {
	repo  Repository
	redis TokenLookup
	clock Clock
	ids   IDSource
}

func NewService(repo Repository, redis TokenLookup, clock Clock, ids IDSource) (*Service, error) {
	if repo == nil || redis == nil || clock == nil || ids == nil {
		return nil, errors.New("auth service dependencies are required")
	}
	return &Service{repo: repo, redis: redis, clock: clock, ids: ids}, nil
}

func (s *Service) Authorize(ctx context.Context, request domain.AuthRequest) (domain.AuthResult, error) {
	result := domain.AuthResult{RouteID: request.RouteID, StatusCode: 403, Reason: "token rejected"}
	policy, err := s.repo.GetPolicy(request.RouteID)
	if err != nil {
		return result, fmt.Errorf("load route policy: %w", err)
	}
	if !policy.Enabled {
		result.Reason = "route policy disabled"
		return result, s.record(result, "")
	}
	token := headerValue(request.Headers, policy.TokenHeader)
	if token == "" {
		result.Reason = "missing x-internal-token"
		return result, s.record(result, "")
	}
	config, err := s.repo.GetRedisConfig(policy.RedisConfigID)
	if err != nil {
		result.StatusCode = 503
		result.Reason = "redis configuration unavailable"
		recordErr := s.record(result, "")
		if recordErr != nil {
			return result, recordErr
		}
		return result, fmt.Errorf("load redis config: %w", err)
	}
	key := policy.KeyPrefix + token
	exists, err := s.redis.Exists(ctx, config, key)
	if err != nil {
		result.StatusCode = 503
		result.Reason = "redis connection failed"
		recordErr := s.record(result, "")
		if recordErr != nil {
			return result, recordErr
		}
		return result, nil
	}
	if !exists {
		result.Reason = "token not found"
		return result, s.record(result, domain.HashToken(token))
	}
	result.Allowed = true
	result.StatusCode = 200
	result.Reason = "token accepted"
	return result, s.record(result, domain.HashToken(token))
}

func (s *Service) record(result domain.AuthResult, tokenID string) error {
	outcome := "denied"
	if result.StatusCode == 200 {
		outcome = "allowed"
	} else if result.StatusCode == 503 {
		outcome = "unavailable"
	}
	event := domain.AuditEvent{ID: s.ids.Next("audit"), RouteID: result.RouteID, TokenID: tokenID, Outcome: outcome, Reason: result.Reason, StatusCode: result.StatusCode, OccurredAt: s.clock.Now()}
	if err := s.repo.AppendAudit(event); err != nil {
		return fmt.Errorf("append auth audit: %w", err)
	}
	return nil
}

func headerValue(headers map[string]string, wanted string) string {
	for key, value := range headers {
		if strings.EqualFold(strings.TrimSpace(key), wanted) {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
