package httpapi

import (
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"internalauth/internal/admin"
	"internalauth/internal/auth"
	"internalauth/internal/store"
)

func TestHealthAndPluginEndpoints(t *testing.T) {
	repo, err := store.Open(filepath.Join(t.TempDir(), "http.db"), store.Options{})
	if err != nil {
		t.Fatal(err)
	}
	defer repo.Close()
	clock := auth.FixedClock{Value: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
	ids := &auth.SequenceIDs{}
	adminService, _ := admin.NewService(repo, clock, ids)
	authService, _ := auth.NewService(repo, auth.NewMemoryRedis(), clock, ids)
	server, _ := New(adminService, authService, 1<<20)
	for path, contentType := range map[string]string{"/healthz": "application/json", "/api/plugin.lua": "text/plain"} {
		request := httptest.NewRequest("GET", path, nil)
		response := httptest.NewRecorder()
		server.Handler().ServeHTTP(response, request)
		if response.Code != 200 {
			t.Fatalf("%s returned %d", path, response.Code)
		}
		if got := response.Header().Get("Content-Type"); len(got) < len(contentType) || got[:len(contentType)] != contentType {
			t.Fatalf("%s content type %s", path, got)
		}
	}
}
