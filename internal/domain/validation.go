package domain

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"
	"time"
)

var ErrNotFound = errors.New("not found")
var ErrConflict = errors.New("conflict")
var ErrInvalid = errors.New("invalid input")
var ErrUnavailable = errors.New("dependency unavailable")

type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

type ValidationErrors []FieldError

func (v ValidationErrors) Error() string {
	parts := make([]string, 0, len(v))
	for _, item := range v {
		parts = append(parts, item.Field+": "+item.Message)
	}
	return strings.Join(parts, "; ")
}

func ValidateRedisConfig(c RedisConfig) error {
	var out ValidationErrors
	if strings.TrimSpace(c.ID) == "" {
		out = append(out, FieldError{"id", "is required"})
	}
	if strings.TrimSpace(c.Name) == "" {
		out = append(out, FieldError{"name", "is required"})
	}
	if strings.TrimSpace(c.Address) == "" {
		out = append(out, FieldError{"address", "is required"})
	} else if _, _, err := net.SplitHostPort(c.Address); err != nil {
		out = append(out, FieldError{"address", "must include host and port"})
	}
	if c.Database < 0 || c.Database > 15 {
		out = append(out, FieldError{"database", "must be between 0 and 15"})
	}
	if c.PoolSize < 1 || c.PoolSize > 1024 {
		out = append(out, FieldError{"pool_size", "must be between 1 and 1024"})
	}
	if c.TimeoutMS < 10 || c.TimeoutMS > 30000 {
		out = append(out, FieldError{"timeout_ms", "must be between 10 and 30000"})
	}
	if len(out) > 0 {
		return out
	}
	return nil
}

func ValidateRoutePolicy(p RoutePolicy) error {
	var out ValidationErrors
	if strings.TrimSpace(p.ID) == "" {
		out = append(out, FieldError{"id", "is required"})
	}
	if strings.TrimSpace(p.Name) == "" {
		out = append(out, FieldError{"name", "is required"})
	}
	if strings.TrimSpace(p.RedisConfigID) == "" {
		out = append(out, FieldError{"redis_config_id", "is required"})
	}
	if !strings.HasPrefix(p.RouteURI, "/") {
		out = append(out, FieldError{"route_uri", "must begin with slash"})
	}
	if p.TokenHeader == "" {
		out = append(out, FieldError{"token_header", "is required"})
	}
	if strings.ToLower(p.TokenHeader) != p.TokenHeader {
		out = append(out, FieldError{"token_header", "must be lowercase"})
	}
	if p.TokenHeader != "x-internal-token" {
		out = append(out, FieldError{"token_header", "must be x-internal-token"})
	}
	if strings.ContainsAny(p.KeyPrefix, " \r\n\t") {
		out = append(out, FieldError{"key_prefix", "must not contain whitespace"})
	}
	if p.Revision < 1 {
		out = append(out, FieldError{"revision", "must be positive"})
	}
	if len(out) > 0 {
		return out
	}
	return nil
}

func ValidateTokenRecord(t TokenRecord) error {
	var out ValidationErrors
	if strings.TrimSpace(t.ID) == "" {
		out = append(out, FieldError{"id", "is required"})
	}
	if len(t.TokenHash) != 64 {
		out = append(out, FieldError{"token_hash", "must be a sha256 hex digest"})
	}
	for _, r := range t.TokenHash {
		if !strings.ContainsRune("0123456789abcdef", r) {
			out = append(out, FieldError{"token_hash", "must use lowercase hex"})
			break
		}
	}
	if strings.TrimSpace(t.Subject) == "" {
		out = append(out, FieldError{"subject", "is required"})
	}
	if t.ExpiresAt.IsZero() {
		out = append(out, FieldError{"expires_at", "is required"})
	}
	if len(out) > 0 {
		return out
	}
	return nil
}

func ValidateAuditEvent(e AuditEvent) error {
	if e.ID == "" || e.RouteID == "" {
		return fmt.Errorf("%w: audit identifiers are required", ErrInvalid)
	}
	switch e.Outcome {
	case "allowed", "denied", "unavailable":
	default:
		return fmt.Errorf("%w: invalid audit outcome", ErrInvalid)
	}
	if e.StatusCode != 200 && e.StatusCode != 403 && e.StatusCode != 503 {
		return fmt.Errorf("%w: invalid status code", ErrInvalid)
	}
	if e.OccurredAt.IsZero() {
		return fmt.Errorf("%w: occurred_at is required", ErrInvalid)
	}
	return nil
}

func ValidateDeployment(d Deployment) error {
	if d.ID == "" || d.RouteID == "" {
		return fmt.Errorf("%w: deployment identifiers are required", ErrInvalid)
	}
	if d.Revision < 1 {
		return fmt.Errorf("%w: revision must be positive", ErrInvalid)
	}
	if strings.TrimSpace(d.LuaConfig) == "" {
		return fmt.Errorf("%w: lua config is required", ErrInvalid)
	}
	switch d.Status {
	case "draft", "published", "retired":
	default:
		return fmt.Errorf("%w: invalid deployment status", ErrInvalid)
	}
	return nil
}

func NormalizeRouteURI(raw string) (string, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("%w: invalid route URI", ErrInvalid)
	}
	if u.IsAbs() || u.Host != "" || u.RawQuery != "" || u.Fragment != "" {
		return "", fmt.Errorf("%w: route URI must be a path", ErrInvalid)
	}
	path := strings.TrimSpace(u.Path)
	if path == "" || path[0] != '/' {
		return "", fmt.Errorf("%w: route URI must begin with slash", ErrInvalid)
	}
	for strings.Contains(path, "//") {
		path = strings.ReplaceAll(path, "//", "/")
	}
	if len(path) > 1 {
		path = strings.TrimSuffix(path, "/")
	}
	return path, nil
}

func TokenActive(t TokenRecord, at time.Time) bool { return !t.Revoked && at.Before(t.ExpiresAt) }
