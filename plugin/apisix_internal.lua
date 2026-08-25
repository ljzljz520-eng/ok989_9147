local core = require("apisix.core")
local plugin_name = "internal-token-auth"
local _M = { version = 1, priority = 2500, name = plugin_name }
function _M.access(conf, ctx)
  local token = core.request.header(ctx, "x-internal-token")
  if not token or token == "" then return 403 end
  local ok, err = core.lrucache.global("redis:" .. token, conf.ttl or 60, function()
    local redis = require("resty.redis"):new()
    local connected, connect_err = redis:connect(conf.redis_host, conf.redis_port or 6379)
    if not connected then return nil, connect_err end
    local exists, exists_err = redis:exists(token)
    redis:set_keepalive(conf.pool_idle or 60000, conf.pool_size or 10)
    return exists, exists_err
  end)
  if err then return 503 end
  if not ok then return 403 end
end
return _M
