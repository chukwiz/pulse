package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/chukwiz/pulse/internal/config"
	"github.com/chukwiz/pulse/internal/gateway"
	"github.com/chukwiz/pulse/internal/memstore"
	"github.com/chukwiz/pulse/internal/middleware"
	"github.com/chukwiz/pulse/internal/ratelimit"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	if err := run(logger); err != nil {
		logger.Error("gateway exited", "err", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	eventStore := memstore.New()
	gw := gateway.New(eventStore, logger)
	rateLimiter := ratelimit.New(cfg.RateLimit.RPS, cfg.RateLimit.Burst)

	// /v1 routes live on their own mux so only they require an API key.
	v1 := http.NewServeMux()
	gw.RegisterRoutes(v1)

	root := http.NewServeMux()
	gw.RegisterHealth(root)

	var api http.Handler = v1
	api = middleware.RateLimit(rateLimiter)(api)
	api = middleware.Auth(cfg.APIKeys)(api)
	root.Handle("/v1/", api)

	var handler http.Handler = root
	handler = middleware.Recover(handler)
	handler = middleware.ContextLogger(logger)(handler)
	handler = middleware.RequestID(handler)

	httpServer := &http.Server{
		Addr:              cfg.Server.Addr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	serverErrors := make(chan error, 1)
	go func() {
		logger.Info("server starting", "addr", httpServer.Addr)

		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErrors <- err
		}
	}()

	select {
	case err := <-serverErrors:
		return fmt.Errorf("server failed: %w", err)

	case <-ctx.Done():
		logger.Info("server shutting down", "cause", context.Cause(ctx))
		stop()

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			closeErr := httpServer.Close()
			return fmt.Errorf("graceful shutdown: %w", errors.Join(err, closeErr))
		}
	}

	logger.Info("server is stopped")
	return nil
}
