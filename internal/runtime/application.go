package runtime

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"internalauth/internal/admin"
	"internalauth/internal/auth"
	"internalauth/internal/config"
	"internalauth/internal/httpapi"
	"internalauth/internal/store"
)

type Application struct {
	Config config.Config
	Store  *store.Store
	Redis  *auth.MemoryRedis
	IDs    *auth.SequenceIDs
	Clock  *Clock
	Admin  *admin.Service
	Auth   *auth.Service
	HTTP   *httpapi.Server
	server *http.Server
}

func Build(cfg config.Config) (*Application, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	database, err := store.Open(cfg.DatabasePath, store.Options{})
	if err != nil {
		return nil, err
	}
	clock := NewClock(time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC), time.Millisecond)
	ids := &auth.SequenceIDs{}
	redis := auth.NewMemoryRedis()
	adminService, err := admin.NewService(database, clock, ids)
	if err != nil {
		database.Close()
		return nil, err
	}
	authService, err := auth.NewService(database, redis, clock, ids)
	if err != nil {
		database.Close()
		return nil, err
	}
	httpService, err := httpapi.New(adminService, authService, cfg.MaxBodyBytes)
	if err != nil {
		database.Close()
		return nil, err
	}
	app := &Application{Config: cfg, Store: database, Redis: redis, IDs: ids, Clock: clock, Admin: adminService, Auth: authService, HTTP: httpService}
	app.server = &http.Server{Addr: cfg.ListenAddress, Handler: httpService.Handler(), ReadHeaderTimeout: 5 * time.Second, IdleTimeout: 60 * time.Second}
	return app, nil
}

func (a *Application) Run() error {
	if a == nil || a.server == nil {
		return errors.New("application is not built")
	}
	err := a.server.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

func (a *Application) Shutdown(ctx context.Context) error {
	if a == nil {
		return nil
	}
	var shutdownErr error
	if a.server != nil {
		shutdownErr = a.server.Shutdown(ctx)
	}
	closeErr := a.Store.Close()
	if shutdownErr != nil && closeErr != nil {
		return fmt.Errorf("shutdown HTTP: %v; close store: %w", shutdownErr, closeErr)
	}
	if shutdownErr != nil {
		return shutdownErr
	}
	return closeErr
}

func (a *Application) Close() error {
	if a == nil || a.Store == nil {
		return nil
	}
	return a.Store.Close()
}
