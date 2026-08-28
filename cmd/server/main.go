package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"internalauth/internal/config"
	appRuntime "internalauth/internal/runtime"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.FromEnvironment(os.Getenv)
	if err != nil {
		return err
	}
	app, err := appRuntime.Build(cfg)
	if err != nil {
		return err
	}
	defer app.Close()
	serverErrors := make(chan error, 1)
	go func() { serverErrors <- app.Run() }()
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	select {
	case err := <-serverErrors:
		return err
	case <-signals:
		ctx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
		defer cancel()
		return app.Shutdown(ctx)
	case <-time.After(10 * time.Second):
		return nil
	}
}
