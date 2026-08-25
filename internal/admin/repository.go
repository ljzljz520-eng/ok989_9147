package admin

import (
	"internalauth/internal/domain"
	"time"
)

type Repository interface {
	CreateRedisConfig(domain.RedisConfig) error
	UpdateRedisConfig(domain.RedisConfig) error
	GetRedisConfig(string) (domain.RedisConfig, error)
	ListRedisConfigs() ([]domain.RedisConfig, error)
	DeleteRedisConfig(string) error
	CreatePolicy(domain.RoutePolicy) error
	UpdatePolicy(domain.RoutePolicy) error
	GetPolicy(string) (domain.RoutePolicy, error)
	ListPolicies() ([]domain.RoutePolicy, error)
	DeletePolicy(string) error
	CreateToken(domain.TokenRecord) error
	UpdateToken(domain.TokenRecord) error
	GetToken(string) (domain.TokenRecord, error)
	FindTokenByHash(string) (domain.TokenRecord, error)
	ListTokens() ([]domain.TokenRecord, error)
	RevokeToken(string, time.Time) (domain.TokenRecord, error)
	AppendAudit(domain.AuditEvent) error
	ListAudits(domain.AuditFilter) ([]domain.AuditEvent, error)
	SaveDeployment(domain.Deployment) error
	ListDeployments(string) ([]domain.Deployment, error)
	Dashboard() (domain.Dashboard, error)
}
