package domain

import "time"

type RedisConfig struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Address   string    `json:"address"`
	Password  string    `json:"password,omitempty"`
	Database  int       `json:"database"`
	PoolSize  int       `json:"pool_size"`
	TimeoutMS int       `json:"timeout_ms"`
	TLS       bool      `json:"tls"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type RoutePolicy struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	RouteURI      string    `json:"route_uri"`
	RedisConfigID string    `json:"redis_config_id"`
	TokenHeader   string    `json:"token_header"`
	KeyPrefix     string    `json:"key_prefix"`
	Enabled       bool      `json:"enabled"`
	Revision      int       `json:"revision"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type TokenRecord struct {
	ID          string    `json:"id"`
	TokenHash   string    `json:"token_hash"`
	Subject     string    `json:"subject"`
	Description string    `json:"description"`
	ExpiresAt   time.Time `json:"expires_at"`
	Revoked     bool      `json:"revoked"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type AuditEvent struct {
	ID         string    `json:"id"`
	RouteID    string    `json:"route_id"`
	TokenID    string    `json:"token_id,omitempty"`
	Outcome    string    `json:"outcome"`
	Reason     string    `json:"reason"`
	StatusCode int       `json:"status_code"`
	OccurredAt time.Time `json:"occurred_at"`
}

type Deployment struct {
	ID        string    `json:"id"`
	RouteID   string    `json:"route_id"`
	Revision  int       `json:"revision"`
	LuaConfig string    `json:"lua_config"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

type Dashboard struct {
	Policies     int `json:"policies"`
	RedisConfigs int `json:"redis_configs"`
	ActiveTokens int `json:"active_tokens"`
	Allowed      int `json:"allowed"`
	Denied       int `json:"denied"`
	Unavailable  int `json:"unavailable"`
}

type AuthRequest struct {
	RouteID string            `json:"route_id"`
	Headers map[string]string `json:"headers"`
}

type AuthResult struct {
	Allowed    bool   `json:"allowed"`
	StatusCode int    `json:"status_code"`
	Reason     string `json:"reason"`
	RouteID    string `json:"route_id"`
}
