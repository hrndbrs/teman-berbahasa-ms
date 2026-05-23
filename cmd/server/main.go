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
	"github.com/hrndbrs/teman-berbahasa-ms/internal/module/batch"
	"github.com/hrndbrs/teman-berbahasa-ms/internal/module/course"
	"github.com/hrndbrs/teman-berbahasa-ms/internal/module/enrollment"
	"github.com/hrndbrs/teman-berbahasa-ms/internal/module/health"
	"github.com/hrndbrs/teman-berbahasa-ms/internal/module/student"
	"github.com/hrndbrs/teman-berbahasa-ms/internal/module/user"
	"github.com/hrndbrs/teman-berbahasa-ms/internal/token"
	"github.com/hrndbrs/teman-berbahasa-ms/internal/worker"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		// config not loaded yet — plain logger is fine for this fatal error
		slog.New(slog.NewJSONHandler(os.Stdout, nil)).Error("config error", "error", err)
		os.Exit(1)
	}

	var logLevel slog.LevelVar
	if err := logLevel.UnmarshalText([]byte(cfg.LogLevel)); err != nil {
		logLevel.Set(slog.LevelInfo)
	}
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: &logLevel})))

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

	w := worker.New(100)
	w.Start(ctx, 4)

	tokenManager, err := token.NewManager(cfg.JWTPrivateKeyPath, cfg.JWTPublicKeyPath)
	if err != nil {
		slog.Error("token manager init failed", "error", err)
		os.Exit(1)
	}

	authRepo := auth.NewRepository(pool)
	emailSender := auth.NewEmailSender(cfg.ResendAPIKey, cfg.ResendFromEmail, cfg.FrontendURL)
	authSvc := auth.NewService(tokenManager, authRepo, pool, emailSender)
	authHandler := auth.NewHandler(authSvc)

	userRepo := user.NewRepository(pool)
	userEmailSender := user.NewEmailSender(cfg.ResendAPIKey, cfg.ResendFromEmail, cfg.FrontendURL)
	userSvc := user.NewService(userRepo, userEmailSender)
	userHandler := user.NewHandler(userSvc)

	studentRepo := student.NewRepository(pool)
	studentSvc := student.NewService(studentRepo)
	studentHandler := student.NewHandler(studentSvc)

	courseRepo := course.NewRepository(pool)
	courseSvc := course.NewService(courseRepo)
	courseHandler := course.NewHandler(courseSvc)

	batchRepo := batch.NewRepository(pool)
	batchSvc := batch.NewService(batchRepo)
	batchHandler := batch.NewHandler(batchSvc)

	enrollmentRepo := enrollment.NewRepository(pool)
	enrollmentSvc := enrollment.NewService(enrollmentRepo)
	enrollmentHandler := enrollment.NewHandler(enrollmentSvc)

	r := chi.NewRouter()
	r.Use(middleware.Recovery)
	r.Use(middleware.Logger)
	r.Use(middleware.CORS(cfg.CORSAllowedOrigins))
	r.Use(chimw.StripSlashes)

	r.Options("/*", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	health.NewHandler(pool).Register(r)
	authHandler.Register(r)

	r.Group(func(r chi.Router) {
		r.Use(middleware.Auth(tokenManager))
		authHandler.RegisterProtected(r)
		userHandler.Register(r)
		studentHandler.Register(r)
		courseHandler.Register(r)
		batchHandler.Register(r)
		enrollmentHandler.Register(r)
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
		slog.Error("http shutdown error", "error", err)
	}

	w.Drain(10 * time.Second)

	if cfg.SentryDSN != "" {
		sentry.Flush(2 * time.Second)
	}

	pool.Close()

	slog.Info("server stopped")
}
