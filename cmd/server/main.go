package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"

	"github.com/hrndbrs/teman-berbahasa-ms/internal/config"
	interndb "github.com/hrndbrs/teman-berbahasa-ms/internal/db"
	"github.com/hrndbrs/teman-berbahasa-ms/internal/middleware"
	"github.com/hrndbrs/teman-berbahasa-ms/internal/module/auth"
	"github.com/hrndbrs/teman-berbahasa-ms/internal/module/health"
	"github.com/hrndbrs/teman-berbahasa-ms/internal/token"
	"github.com/hrndbrs/teman-berbahasa-ms/internal/worker"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	cfg, err := config.Load()
	if err != nil {
		slog.Error("config error", "error", err)
		os.Exit(1)
	}

	if cfg.SentryDSN != "" {
		if err := sentry.Init(sentry.ClientOptions{Dsn: cfg.SentryDSN}); err != nil {
			slog.Warn("sentry init failed", "error", err)
		}
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := interndb.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("db connect error", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	w := worker.New(100)
	w.Start(ctx, 4)

	tokenManager, err := token.NewManager(cfg.JWTPrivateKeyPath, cfg.JWTPublicKeyPath)
	if err != nil {
		slog.Error("token manager init failed", "error", err)
		os.Exit(1)
	}

	authRepo := auth.NewRepository(pool)
	emailSender := auth.NewEmailSender(cfg.ResendAPIKey, cfg.ResendFromEmail, cfg.FrontendURL)
	authSvc := auth.NewService(tokenManager, authRepo, emailSender)
	authHandler := auth.NewHandler(authSvc)

	r := chi.NewRouter()
	r.Use(middleware.Recovery)
	r.Use(middleware.Logger)
	r.Use(middleware.CORS(cfg.CORSAllowedOrigins))
	r.Use(chimw.StripSlashes)

	health.NewHandler(pool).Register(r)
	authHandler.Register(r)

	r.Group(func(r chi.Router) {
		r.Use(middleware.Auth(tokenManager))
		// future modules go here
	})

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.Port),
		Handler:      r,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		slog.Info("server starting", "port", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	slog.Info("shutting down")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("shutdown error", "error", err)
	}

	slog.Info("server stopped")
}
