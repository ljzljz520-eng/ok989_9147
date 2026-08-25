package auth

import (
	"context"
	"internalauth/internal/domain"
)

type TokenLookup interface {
	Exists(context.Context, domain.RedisConfig, string) (bool, error)
}
