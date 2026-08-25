package admin

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"internalauth/internal/domain"
	"internalauth/internal/plugin"
)

type Clock interface{ Now() time.Time }
type IDs interface{ Next(string) string }

type Service struct {
	repo  Repository
	clock Clock
	ids   IDs
}

func NewService(repo Repository, clock Clock, ids IDs) (*Service, error) {
	if repo == nil || clock == nil || ids == nil {
		return nil, errors.New("admin service dependencies are required")
	}
	return &Service{repo: repo, clock: clock, ids: ids}, nil
}

type ProvisionRequest struct {
	Redis   domain.CreateRedisConfig `json:"redis"`
	Policy  domain.CreateRoutePolicy `json:"policy"`
	Version string                   `json:"version"`
}
type ProvisionResult struct {
	Redis      domain.RedisConfig `json:"redis"`
	Policy     domain.RoutePolicy `json:"policy"`
	Deployment domain.Deployment  `json:"deployment"`
}

func (s *Service) Provision(request ProvisionRequest) (ProvisionResult, error) {
	now := s.clock.Now()
	config, err := domain.NewRedisConfig(request.Redis, now)
	if err != nil {
		return ProvisionResult{}, fmt.Errorf("prepare redis config: %w", err)
	}
	if err := s.repo.CreateRedisConfig(config); err != nil {
		return ProvisionResult{}, fmt.Errorf("persist redis config: %w", err)
	}
	policy, err := domain.NewRoutePolicy(request.Policy, now)
	if err != nil {
		return ProvisionResult{}, fmt.Errorf("prepare route policy: %w", err)
	}
	err = s.repo.CreatePolicy(policy)
	if err != nil {
		_, err := s.repo.GetRedisConfig(config.ID)
		if err != nil {
			return ProvisionResult{}, fmt.Errorf("verify redis config: %w", err)
		}
	}
	routeConfig, err := plugin.BuildRouteConfig(policy, config)
	if err != nil {
		return ProvisionResult{}, fmt.Errorf("build route configuration: %w", err)
	}
	luaJSON, err := plugin.RenderJSON(routeConfig)
	if err != nil {
		return ProvisionResult{}, err
	}
	deployment := domain.Deployment{ID: s.ids.Next("deployment"), RouteID: policy.ID, Revision: policy.Revision, LuaConfig: luaJSON, Status: "published", CreatedAt: now}
	if err := s.repo.SaveDeployment(deployment); err != nil {
		return ProvisionResult{}, fmt.Errorf("save deployment: %w", err)
	}
	return ProvisionResult{Redis: config, Policy: policy, Deployment: deployment}, nil
}

func (s *Service) CreateRedis(command domain.CreateRedisConfig) (domain.RedisConfig, error) {
	item, err := domain.NewRedisConfig(command, s.clock.Now())
	if err != nil {
		return domain.RedisConfig{}, err
	}
	if err := s.repo.CreateRedisConfig(item); err != nil {
		return domain.RedisConfig{}, err
	}
	return item, nil
}

func (s *Service) UpdateRedis(id string, mutate func(*domain.RedisConfig) error) (domain.RedisConfig, error) {
	item, err := s.repo.GetRedisConfig(id)
	if err != nil {
		return domain.RedisConfig{}, err
	}
	if err := mutate(&item); err != nil {
		return domain.RedisConfig{}, err
	}
	item.UpdatedAt = s.clock.Now()
	if err := domain.ValidateRedisConfig(item); err != nil {
		return domain.RedisConfig{}, err
	}
	if err := s.repo.UpdateRedisConfig(item); err != nil {
		return domain.RedisConfig{}, err
	}
	return item, nil
}

func (s *Service) CreatePolicy(command domain.CreateRoutePolicy) (domain.RoutePolicy, error) {
	if _, err := s.repo.GetRedisConfig(command.RedisConfigID); err != nil {
		return domain.RoutePolicy{}, err
	}
	item, err := domain.NewRoutePolicy(command, s.clock.Now())
	if err != nil {
		return domain.RoutePolicy{}, err
	}
	if err := s.repo.CreatePolicy(item); err != nil {
		return domain.RoutePolicy{}, err
	}
	return item, nil
}

func (s *Service) SetPolicyEnabled(id string, enabled bool) (domain.RoutePolicy, error) {
	item, err := s.repo.GetPolicy(id)
	if err != nil {
		return domain.RoutePolicy{}, err
	}
	if item.Enabled == enabled {
		return item, nil
	}
	item.Enabled = enabled
	item.Revision++
	item.UpdatedAt = s.clock.Now()
	if err := s.repo.UpdatePolicy(item); err != nil {
		return domain.RoutePolicy{}, err
	}
	return item, nil
}

func (s *Service) MovePolicy(id, redisConfigID string) (domain.RoutePolicy, error) {
	if _, err := s.repo.GetRedisConfig(redisConfigID); err != nil {
		return domain.RoutePolicy{}, err
	}
	item, err := s.repo.GetPolicy(id)
	if err != nil {
		return domain.RoutePolicy{}, err
	}
	if item.RedisConfigID == redisConfigID {
		return item, nil
	}
	item.RedisConfigID = redisConfigID
	item.Revision++
	item.UpdatedAt = s.clock.Now()
	if err := s.repo.UpdatePolicy(item); err != nil {
		return domain.RoutePolicy{}, err
	}
	return item, nil
}

func (s *Service) ListPolicies(filter domain.PolicyFilter) ([]domain.RoutePolicy, error) {
	items, err := s.repo.ListPolicies()
	if err != nil {
		return nil, err
	}
	return domain.FilterPolicies(items, filter), nil
}

func (s *Service) IssueToken(command domain.IssueToken) (domain.TokenRecord, error) {
	item, err := domain.NewTokenRecord(command, s.clock.Now())
	if err != nil {
		return domain.TokenRecord{}, err
	}
	if existing, lookupErr := s.repo.FindTokenByHash(item.TokenHash); lookupErr == nil {
		return domain.TokenRecord{}, fmt.Errorf("%w: plaintext already belongs to %s", domain.ErrConflict, existing.ID)
	} else if !errors.Is(lookupErr, domain.ErrNotFound) {
		return domain.TokenRecord{}, lookupErr
	}
	if err := s.repo.CreateToken(item); err != nil {
		return domain.TokenRecord{}, err
	}
	return item, nil
}

func (s *Service) RevokeToken(id string) (domain.TokenRecord, error) {
	return s.repo.RevokeToken(id, s.clock.Now())
}

func (s *Service) TokenStatus(id string) (string, error) {
	item, err := s.repo.GetToken(id)
	if err != nil {
		return "", err
	}
	if item.Revoked {
		return "revoked", nil
	}
	if !s.clock.Now().Before(item.ExpiresAt) {
		return "expired", nil
	}
	return "active", nil
}

func (s *Service) PublishPolicy(id string) (domain.Deployment, error) {
	policy, err := s.repo.GetPolicy(id)
	if err != nil {
		return domain.Deployment{}, err
	}
	if !policy.Enabled {
		return domain.Deployment{}, fmt.Errorf("%w: disabled policy cannot be published", domain.ErrInvalid)
	}
	config, err := s.repo.GetRedisConfig(policy.RedisConfigID)
	if err != nil {
		return domain.Deployment{}, err
	}
	routeConfig, err := plugin.BuildRouteConfig(policy, config)
	if err != nil {
		return domain.Deployment{}, err
	}
	artifact, err := plugin.RenderJSON(routeConfig)
	if err != nil {
		return domain.Deployment{}, err
	}
	item := domain.Deployment{ID: s.ids.Next("deployment"), RouteID: id, Revision: policy.Revision, LuaConfig: artifact, Status: "published", CreatedAt: s.clock.Now()}
	if err := s.repo.SaveDeployment(item); err != nil {
		return domain.Deployment{}, err
	}
	return item, nil
}

func (s *Service) ExportBundle(version string) (plugin.Bundle, error) {
	policies, err := s.repo.ListPolicies()
	if err != nil {
		return plugin.Bundle{}, err
	}
	configs, err := s.repo.ListRedisConfigs()
	if err != nil {
		return plugin.Bundle{}, err
	}
	return plugin.BuildBundle(policies, configs, version)
}

func (s *Service) Dashboard() (domain.Dashboard, error) { return s.repo.Dashboard() }

func (s *Service) AuditReport(filter domain.AuditFilter) (string, error) {
	items, err := s.repo.ListAudits(filter)
	if err != nil {
		return "", err
	}
	var out strings.Builder
	out.WriteString("occurred_at,route_id,outcome,status_code,reason\n")
	sort.Slice(items, func(i, j int) bool { return items[i].OccurredAt.Before(items[j].OccurredAt) })
	for _, item := range items {
		out.WriteString(item.OccurredAt.UTC().Format(time.RFC3339))
		out.WriteByte(',')
		out.WriteString(csv(item.RouteID))
		out.WriteByte(',')
		out.WriteString(item.Outcome)
		out.WriteByte(',')
		out.WriteString(fmt.Sprintf("%d", item.StatusCode))
		out.WriteByte(',')
		out.WriteString(csv(item.Reason))
		out.WriteByte('\n')
	}
	return out.String(), nil
}

func csv(value string) string {
	if !strings.ContainsAny(value, ",\"\r\n") {
		return value
	}
	return "\"" + strings.ReplaceAll(value, "\"", "\"\"") + "\""
}
