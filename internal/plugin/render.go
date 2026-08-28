package plugin

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"internalauth/internal/domain"
)

type RouteConfig struct {
	RedisHost     string `json:"redis_host"`
	RedisPort     int    `json:"redis_port"`
	RedisPassword string `json:"redis_password,omitempty"`
	RedisDatabase int    `json:"redis_database"`
	PoolSize      int    `json:"pool_size"`
	TimeoutMS     int    `json:"timeout_ms"`
	KeyPrefix     string `json:"key_prefix"`
	TokenHeader   string `json:"token_header"`
	TLS           bool   `json:"tls"`
}

func BuildRouteConfig(policy domain.RoutePolicy, redis domain.RedisConfig) (RouteConfig, error) {
	if err := domain.ValidateRoutePolicy(policy); err != nil {
		return RouteConfig{}, err
	}
	if err := domain.ValidateRedisConfig(redis); err != nil {
		return RouteConfig{}, err
	}
	if policy.RedisConfigID != redis.ID {
		return RouteConfig{}, fmt.Errorf("%w: policy references a different redis config", domain.ErrInvalid)
	}
	host, port, err := splitAddress(redis.Address)
	if err != nil {
		return RouteConfig{}, err
	}
	return RouteConfig{RedisHost: host, RedisPort: port, RedisPassword: redis.Password, RedisDatabase: redis.Database, PoolSize: redis.PoolSize, TimeoutMS: redis.TimeoutMS, KeyPrefix: policy.KeyPrefix, TokenHeader: policy.TokenHeader, TLS: redis.TLS}, nil
}

func splitAddress(address string) (string, int, error) {
	index := strings.LastIndex(address, ":")
	if index < 1 {
		return "", 0, fmt.Errorf("%w: redis address missing port", domain.ErrInvalid)
	}
	host := strings.Trim(address[:index], "[]")
	port, err := strconv.Atoi(address[index+1:])
	if err != nil || port < 1 || port > 65535 {
		return "", 0, fmt.Errorf("%w: invalid redis port", domain.ErrInvalid)
	}
	return host, port, nil
}

func RenderJSON(config RouteConfig) (string, error) {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return "", fmt.Errorf("render route config: %w", err)
	}
	return string(data), nil
}

func RenderRouteSnippet(policy domain.RoutePolicy, config RouteConfig) string {
	values := map[string]string{"uri": policy.RouteURI, "redis_host": config.RedisHost, "redis_port": strconv.Itoa(config.RedisPort), "redis_database": strconv.Itoa(config.RedisDatabase), "pool_size": strconv.Itoa(config.PoolSize), "timeout_ms": strconv.Itoa(config.TimeoutMS), "key_prefix": config.KeyPrefix, "token_header": config.TokenHeader, "tls": strconv.FormatBool(config.TLS)}
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var out strings.Builder
	out.WriteString("routes:\n  - id: ")
	out.WriteString(policy.ID)
	out.WriteString("\n")
	for _, key := range keys {
		out.WriteString("    ")
		out.WriteString(key)
		out.WriteString(": ")
		out.WriteString(strconv.Quote(values[key]))
		out.WriteString("\n")
	}
	return out.String()
}

func RenderLuaPlugin() string {
	return `local core = require("apisix.core")
local redis = require("resty.redis")
local plugin_name = "internal-auth"
local schema = {type = "object", properties = {
  redis_host = {type = "string"}, redis_port = {type = "integer"},
  redis_password = {type = "string"}, redis_database = {type = "integer", default = 0},
  pool_size = {type = "integer", default = 100}, timeout_ms = {type = "integer", default = 1000},
  key_prefix = {type = "string", default = "internal:token:"},
}, required = {"redis_host", "redis_port"}}
local _M = {version = 0.1, priority = 2500, name = plugin_name, schema = schema}
function _M.check_schema(conf) return core.schema.check(schema, conf) end
function _M.access(conf, ctx)
  local token = core.request.header(ctx, "x-internal-token")
  if not token or token == "" then return 403, {message = "forbidden"} end
  local client = redis:new()
  client:set_timeout(conf.timeout_ms)
  local ok, err = client:connect(conf.redis_host, conf.redis_port)
  if not ok then return 503, {message = "redis unavailable"} end
  if conf.redis_password and conf.redis_password ~= "" then
    local auth_ok = client:auth(conf.redis_password)
    if not auth_ok then return 503, {message = "redis unavailable"} end
  end
  if conf.redis_database > 0 then
    local select_ok = client:select(conf.redis_database)
    if not select_ok then return 503, {message = "redis unavailable"} end
  end
  local exists, lookup_err = client:exists(conf.key_prefix .. token)
  if lookup_err then return 503, {message = "redis unavailable"} end
  local keepalive_ok = client:set_keepalive(60000, conf.pool_size)
  if not keepalive_ok then core.log.warn("failed to retain redis connection") end
  if exists ~= 1 then return 403, {message = "forbidden"} end
end
return _M`
}

func ValidateLuaArtifact(lua string) error {
	required := []string{"x-internal-token", "client:exists", "return 403", "return 503", "set_keepalive", "pool_size"}
	for _, fragment := range required {
		if !strings.Contains(lua, fragment) {
			return fmt.Errorf("lua artifact missing %s", fragment)
		}
	}
	return nil
}
