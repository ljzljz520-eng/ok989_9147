package report

import (
	"fmt"
	"sort"
	"strings"

	"internalauth/internal/domain"
	"internalauth/internal/plugin"
)

type Runbook struct {
	Title    string
	Sections []Section
}
type Section struct {
	Heading string
	Lines   []string
}

func BuildRunbook(bundle plugin.Bundle, assessment Assessment) Runbook {
	sections := []Section{
		{Heading: "Release", Lines: []string{"Bundle version: " + bundle.Version, fmt.Sprintf("Enabled routes: %d", len(bundle.Routes)), "Plugin file: apisix/plugins/internal-auth.lua"}},
		{Heading: "Preflight", Lines: preflightLines(bundle)},
		{Heading: "Rollout", Lines: rolloutLines(bundle)},
		{Heading: "Verification", Lines: verificationLines(bundle)},
		{Heading: "Rollback", Lines: rollbackLines(bundle)},
		{Heading: "Assessment", Lines: assessmentLines(assessment)},
	}
	return Runbook{Title: "APISIX internal-auth deployment runbook", Sections: sections}
}

func preflightLines(bundle plugin.Bundle) []string {
	out := []string{"Confirm lua-resty-redis is installed in the APISIX runtime.", "Confirm each Redis endpoint is reachable from every gateway worker.", "Confirm x-internal-token is removed from upstream logs."}
	seen := map[string]bool{}
	for _, route := range bundle.Routes {
		endpoint := fmt.Sprintf("%s:%d", route.Config.RedisHost, route.Config.RedisPort)
		if seen[endpoint] {
			continue
		}
		seen[endpoint] = true
		out = append(out, "Probe Redis endpoint "+endpoint+" with the configured database and credential.")
	}
	return out
}

func rolloutLines(bundle plugin.Bundle) []string {
	out := []string{"Copy the plugin module into the APISIX plugins directory.", "Add internal-auth to the APISIX plugin list.", "Reload one canary gateway instance."}
	for _, route := range bundle.Routes {
		out = append(out, "Apply route "+route.PolicyID+" for URI "+route.URI+" with pool size "+fmt.Sprintf("%d", route.Config.PoolSize)+".")
	}
	out = append(out, "Reload remaining gateway instances after canary verification.")
	return out
}

func verificationLines(bundle plugin.Bundle) []string {
	out := []string{"Send a request without x-internal-token and expect HTTP 403.", "Send a request with an unknown token and expect HTTP 403.", "Insert a known token key in Redis and expect the request to reach the upstream.", "Block Redis connectivity in a canary environment and expect HTTP 503."}
	for _, route := range bundle.Routes {
		out = append(out, "Inspect authorization audit events for route "+route.PolicyID+".")
	}
	return out
}

func rollbackLines(bundle plugin.Bundle) []string {
	out := []string{"Restore the previous route revision in APISIX.", "Reload the affected gateway instances.", "Retain audit records for incident review."}
	if len(bundle.Routes) > 1 {
		out = append(out, "Roll back routes independently to preserve healthy services.")
	}
	return out
}

func assessmentLines(assessment Assessment) []string {
	out := []string{Summarize(assessment)}
	for _, finding := range assessment.Findings {
		out = append(out, strings.ToUpper(string(finding.Severity))+" "+finding.Code+" "+finding.ResourceType+"/"+finding.ResourceID+": "+finding.Message+"; "+finding.Remediation)
	}
	if len(assessment.Findings) == 0 {
		out = append(out, "No operational findings were detected.")
	}
	return out
}

func RenderMarkdown(runbook Runbook) string {
	var out strings.Builder
	out.WriteString("# ")
	out.WriteString(runbook.Title)
	out.WriteString("\n\n")
	for _, section := range runbook.Sections {
		out.WriteString("## ")
		out.WriteString(section.Heading)
		out.WriteString("\n\n")
		for _, line := range section.Lines {
			out.WriteString("- ")
			out.WriteString(line)
			out.WriteByte('\n')
		}
		out.WriteByte('\n')
	}
	return out.String()
}

func RouteMatrix(policies []domain.RoutePolicy, configs []domain.RedisConfig) [][]string {
	byID := map[string]domain.RedisConfig{}
	for _, config := range configs {
		byID[config.ID] = config
	}
	out := [][]string{{"policy_id", "route_uri", "enabled", "redis_address", "database", "pool_size", "timeout_ms"}}
	sort.Slice(policies, func(i, j int) bool { return policies[i].ID < policies[j].ID })
	for _, policy := range policies {
		config, ok := byID[policy.RedisConfigID]
		address, database, pool, timeout := "missing", "", "", ""
		if ok {
			address = config.Address
			database = fmt.Sprintf("%d", config.Database)
			pool = fmt.Sprintf("%d", config.PoolSize)
			timeout = fmt.Sprintf("%d", config.TimeoutMS)
		}
		out = append(out, []string{policy.ID, policy.RouteURI, fmt.Sprintf("%t", policy.Enabled), address, database, pool, timeout})
	}
	return out
}

func RenderTable(rows [][]string) string {
	if len(rows) == 0 {
		return ""
	}
	widths := make([]int, len(rows[0]))
	for _, row := range rows {
		for index, value := range row {
			if index < len(widths) && len(value) > widths[index] {
				widths[index] = len(value)
			}
		}
	}
	var out strings.Builder
	for rowIndex, row := range rows {
		out.WriteString("| ")
		for index := range widths {
			value := ""
			if index < len(row) {
				value = row[index]
			}
			out.WriteString(value)
			out.WriteString(strings.Repeat(" ", widths[index]-len(value)))
			out.WriteString(" | ")
		}
		out.WriteByte('\n')
		if rowIndex == 0 {
			out.WriteString("| ")
			for _, width := range widths {
				out.WriteString(strings.Repeat("-", width))
				out.WriteString(" | ")
			}
			out.WriteByte('\n')
		}
	}
	return out.String()
}
