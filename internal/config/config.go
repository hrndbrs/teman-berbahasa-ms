package config

import (
	"fmt"
	"os"
	"strings"
)

type Config struct {
	DatabaseURL        string
	JWTPrivateKeyPath  string
	JWTPublicKeyPath   string
	SentryDSN          string
	CORSAllowedOrigins []string
	Port               string
	ResendAPIKey       string
	ResendFromEmail    string
	FrontendURL        string
}

func Load() (*Config, error) {
	cfg := &Config{
		DatabaseURL:       os.Getenv("DATABASE_URL"),
		JWTPrivateKeyPath: os.Getenv("JWT_PRIVATE_KEY_PATH"),
		JWTPublicKeyPath:  os.Getenv("JWT_PUBLIC_KEY_PATH"),
		SentryDSN:         os.Getenv("SENTRY_DSN"),
		Port:              os.Getenv("PORT"),
		ResendAPIKey:      os.Getenv("RESEND_API_KEY"),
		ResendFromEmail:   os.Getenv("RESEND_FROM_EMAIL"),
		FrontendURL:       os.Getenv("FRONTEND_URL"),
	}

	if cfg.Port == "" {
		cfg.Port = "8080"
	}

	if origins := os.Getenv("CORS_ALLOWED_ORIGINS"); origins != "" {
		for _, o := range strings.Split(origins, ",") {
			cfg.CORSAllowedOrigins = append(cfg.CORSAllowedOrigins, strings.TrimSpace(o))
		}
	}

	var missing []string
	if cfg.DatabaseURL == "" {
		missing = append(missing, "DATABASE_URL")
	}
	if cfg.JWTPrivateKeyPath == "" {
		missing = append(missing, "JWT_PRIVATE_KEY_PATH")
	}
	if cfg.JWTPublicKeyPath == "" {
		missing = append(missing, "JWT_PUBLIC_KEY_PATH")
	}
	if len(cfg.CORSAllowedOrigins) == 0 {
		missing = append(missing, "CORS_ALLOWED_ORIGINS")
	}
	if cfg.ResendAPIKey == "" {
		missing = append(missing, "RESEND_API_KEY")
	}
	if cfg.ResendFromEmail == "" {
		missing = append(missing, "RESEND_FROM_EMAIL")
	}
	if cfg.FrontendURL == "" {
		missing = append(missing, "FRONTEND_URL")
	}

	if len(missing) > 0 {
		return nil, fmt.Errorf("missing required environment variables: %s", strings.Join(missing, ", "))
	}

	return cfg, nil
}
