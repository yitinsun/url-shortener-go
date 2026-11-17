package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"url-shortener/config"
	"url-shortener/repo"
	"url-shortener/schedule"
	"url-shortener/service"
	"url-shortener/web"
)

// @title URL Shortener API
// @version 0.1.0
// @description Create a short code from a long HTTP(S) URL. Click analytics, auth, and custom aliases are out of v1. The redirect route is not part of this document.
// @BasePath /

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, nil)))
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := run(ctx); err != nil {
		slog.Error("url-shortener failed", "error", err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	cfg, err := config.FromEnv()
	if err != nil {
		return err
	}
	db, err := repo.Open(ctx, cfg)
	if err != nil {
		return err
	}
	defer db.Close()

	store := repo.NewMySQL(db)
	schedule.Start(ctx, store, schedule.CleanupInterval)
	handler := web.NewRouter(service.New(store, cfg.BaseURL))
	if cfg.OpenAPIEnabled {
		web.MountOpenAPI(handler, cfg.BaseURL)
	}
	server := &http.Server{
		Addr:              cfg.BindAddr,
		Handler:           web.LegalPath(web.AllowPublic(web.HtmlSanitize(handler))),
		ReadHeaderTimeout: 5 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		slog.Info("url shortener listening", "addr", cfg.BindAddr, "env", cfg.Env)
		errCh <- server.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return server.Shutdown(shutdownCtx)
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}
