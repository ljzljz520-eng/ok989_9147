package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync/atomic"

	"internalauth/internal/admin"
	"internalauth/internal/auth"
	"internalauth/internal/domain"
	"internalauth/internal/plugin"
)

type Server struct {
	admin   *admin.Service
	auth    *auth.Service
	maxBody int64
	mux     *http.ServeMux
}

func New(adminService *admin.Service, authService *auth.Service, maxBody int64) (*Server, error) {
	if adminService == nil || authService == nil {
		return nil, errors.New("http services required")
	}
	if maxBody < 1024 {
		return nil, errors.New("max body must be at least 1024 bytes")
	}
	s := &Server{admin: adminService, auth: authService, maxBody: maxBody, mux: http.NewServeMux()}
	s.routes()
	return s, nil
}

func (s *Server) Handler() http.Handler { return requestID(recoverer(s.mux)) }

func (s *Server) routes() {
	s.mux.HandleFunc("GET /healthz", s.health)
	s.mux.HandleFunc("GET /api/dashboard", s.dashboard)
	s.mux.HandleFunc("GET /api/policies", s.listPolicies)
	s.mux.HandleFunc("POST /api/policies", s.createPolicy)
	s.mux.HandleFunc("POST /api/policies/{id}/enable", s.enablePolicy)
	s.mux.HandleFunc("POST /api/policies/{id}/disable", s.disablePolicy)
	s.mux.HandleFunc("POST /api/policies/{id}/publish", s.publishPolicy)
	s.mux.HandleFunc("GET /api/redis-configs", s.listRedis)
	s.mux.HandleFunc("POST /api/redis-configs", s.createRedis)
	s.mux.HandleFunc("POST /api/tokens", s.issueToken)
	s.mux.HandleFunc("POST /api/tokens/{id}/revoke", s.revokeToken)
	s.mux.HandleFunc("POST /api/authorize", s.authorize)
	s.mux.HandleFunc("GET /api/audits.csv", s.auditCSV)
	s.mux.HandleFunc("GET /api/plugin.lua", s.luaPlugin)
	s.mux.HandleFunc("GET /api/bundle", s.bundle)
	s.mux.HandleFunc("POST /api/provision", s.provision)
}

func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, 200, map[string]string{"status": "ok"})
}

func (s *Server) dashboard(w http.ResponseWriter, _ *http.Request) {
	value, err := s.admin.Dashboard()
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, 200, value)
}

func (s *Server) listPolicies(w http.ResponseWriter, r *http.Request) {
	filter := domain.PolicyFilter{RedisConfigID: r.URL.Query().Get("redis_config_id"), Query: r.URL.Query().Get("q")}
	if raw := r.URL.Query().Get("enabled"); raw != "" {
		enabled, err := strconv.ParseBool(raw)
		if err != nil {
			writeJSON(w, 400, errorBody{Error: "enabled must be true or false"})
			return
		}
		filter.Enabled = &enabled
	}
	items, err := s.admin.ListPolicies(filter)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, 200, items)
}

func (s *Server) createPolicy(w http.ResponseWriter, r *http.Request) {
	var command domain.CreateRoutePolicy
	if !s.decode(w, r, &command) {
		return
	}
	item, err := s.admin.CreatePolicy(command)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, 201, item)
}

func (s *Server) enablePolicy(w http.ResponseWriter, r *http.Request) { s.setPolicyEnabled(w, r, true) }
func (s *Server) disablePolicy(w http.ResponseWriter, r *http.Request) {
	s.setPolicyEnabled(w, r, false)
}

func (s *Server) setPolicyEnabled(w http.ResponseWriter, r *http.Request, enabled bool) {
	item, err := s.admin.SetPolicyEnabled(r.PathValue("id"), enabled)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, 200, item)
}

func (s *Server) publishPolicy(w http.ResponseWriter, r *http.Request) {
	item, err := s.admin.PublishPolicy(r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, 201, item)
}

func (s *Server) listRedis(w http.ResponseWriter, _ *http.Request) {
	items, err := s.admin.ExportBundle("preview")
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, 200, items.Routes)
}

func (s *Server) createRedis(w http.ResponseWriter, r *http.Request) {
	var command domain.CreateRedisConfig
	if !s.decode(w, r, &command) {
		return
	}
	item, err := s.admin.CreateRedis(command)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, 201, item)
}

func (s *Server) issueToken(w http.ResponseWriter, r *http.Request) {
	var command domain.IssueToken
	if !s.decode(w, r, &command) {
		return
	}
	item, err := s.admin.IssueToken(command)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, 201, item)
}

func (s *Server) revokeToken(w http.ResponseWriter, r *http.Request) {
	item, err := s.admin.RevokeToken(r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, 200, item)
}

func (s *Server) authorize(w http.ResponseWriter, r *http.Request) {
	var request domain.AuthRequest
	if !s.decode(w, r, &request) {
		return
	}
	result, err := s.auth.Authorize(r.Context(), request)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, result.StatusCode, result)
}

func (s *Server) auditCSV(w http.ResponseWriter, r *http.Request) {
	filter := domain.AuditFilter{RouteID: r.URL.Query().Get("route_id"), Outcome: r.URL.Query().Get("outcome"), Limit: 1000}
	value, err := s.admin.AuditReport(filter)
	if err != nil {
		writeError(w, err)
		return
	}
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.WriteHeader(200)
	_, _ = io.WriteString(w, value)
}

func (s *Server) luaPlugin(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = io.WriteString(w, plugin.RenderLuaPlugin())
}

func (s *Server) bundle(w http.ResponseWriter, r *http.Request) {
	version := r.URL.Query().Get("version")
	if version == "" {
		version = "preview"
	}
	value, err := s.admin.ExportBundle(version)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, 200, value)
}

func (s *Server) provision(w http.ResponseWriter, r *http.Request) {
	var request admin.ProvisionRequest
	if !s.decode(w, r, &request) {
		return
	}
	value, err := s.admin.Provision(request)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, 201, value)
}

func (s *Server) decode(w http.ResponseWriter, r *http.Request, target any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, s.maxBody)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		writeJSON(w, 400, errorBody{Error: "invalid JSON: " + err.Error()})
		return false
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		writeJSON(w, 400, errorBody{Error: "request must contain one JSON value"})
		return false
	}
	return true
}

type errorBody struct {
	Error string `json:"error"`
}

func writeError(w http.ResponseWriter, err error) {
	status := 500
	if errors.Is(err, domain.ErrNotFound) {
		status = 404
	} else if errors.Is(err, domain.ErrConflict) {
		status = 409
	} else if errors.Is(err, domain.ErrInvalid) {
		status = 422
	} else {
		var validation domain.ValidationErrors
		if errors.As(err, &validation) {
			status = 422
		}
	}
	writeJSON(w, status, errorBody{Error: err.Error()})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		http.Error(w, "encode response", 500)
	}
}

func requestID(next http.Handler) http.Handler {
	var sequence uint64
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimSpace(r.Header.Get("X-Request-ID"))
		if id == "" {
			id = fmt.Sprintf("req-%d", atomic.AddUint64(&sequence, 1))
		}
		w.Header().Set("X-Request-ID", id)
		next.ServeHTTP(w, r)
	})
}

func recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if recover() != nil {
				writeJSON(w, 500, errorBody{Error: "internal server error"})
			}
		}()
		next.ServeHTTP(w, r)
	})
}
