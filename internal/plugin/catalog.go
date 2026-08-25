package plugin

import (
	"fmt"
	"sort"
	"strings"

	"internalauth/internal/domain"
)

type Bundle struct {
	Lua     string          `json:"lua"`
	Routes  []RouteArtifact `json:"routes"`
	Version string          `json:"version"`
}
type RouteArtifact struct {
	PolicyID string      `json:"policy_id"`
	URI      string      `json:"uri"`
	Config   RouteConfig `json:"config"`
	Snippet  string      `json:"snippet"`
}

func BuildBundle(policies []domain.RoutePolicy, configs []domain.RedisConfig, version string) (Bundle, error) {
	if strings.TrimSpace(version) == "" {
		return Bundle{}, fmt.Errorf("%w: bundle version required", domain.ErrInvalid)
	}
	byID := make(map[string]domain.RedisConfig, len(configs))
	for _, config := range configs {
		if err := domain.ValidateRedisConfig(config); err != nil {
			return Bundle{}, err
		}
		byID[config.ID] = config
	}
	artifacts := make([]RouteArtifact, 0)
	for _, policy := range policies {
		if !policy.Enabled {
			continue
		}
		config, ok := byID[policy.RedisConfigID]
		if !ok {
			return Bundle{}, fmt.Errorf("%w: config %s for policy %s", domain.ErrNotFound, policy.RedisConfigID, policy.ID)
		}
		routeConfig, err := BuildRouteConfig(policy, config)
		if err != nil {
			return Bundle{}, err
		}
		artifacts = append(artifacts, RouteArtifact{PolicyID: policy.ID, URI: policy.RouteURI, Config: routeConfig, Snippet: RenderRouteSnippet(policy, routeConfig)})
	}
	sort.Slice(artifacts, func(i, j int) bool { return artifacts[i].PolicyID < artifacts[j].PolicyID })
	lua := RenderLuaPlugin()
	if err := ValidateLuaArtifact(lua); err != nil {
		return Bundle{}, err
	}
	return Bundle{Lua: lua, Routes: artifacts, Version: version}, nil
}

func BundleSummary(bundle Bundle) string {
	var out strings.Builder
	out.WriteString("bundle ")
	out.WriteString(bundle.Version)
	out.WriteString(" contains ")
	out.WriteString(fmt.Sprintf("%d enabled routes", len(bundle.Routes)))
	for _, route := range bundle.Routes {
		out.WriteString("\n- ")
		out.WriteString(route.PolicyID)
		out.WriteString(" ")
		out.WriteString(route.URI)
	}
	return out.String()
}
