package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

type seedUser struct {
	firstName string
	lastName  string
	email     string
	password  string
	role      string
	phone     *string
}

var seeds = []seedUser{
	{
		firstName: "Admin",
		lastName:  "Seed",
		email:     "admin@tb.local",
		password:  "P@ssw0rd",
		role:      "admin",
	},
	{
		firstName: "Teacher",
		lastName:  "Seed",
		email:     "teacher@tb.local",
		password:  "P@ssw0rd",
		role:      "teacher",
	},
	{
		firstName: "Staff",
		lastName:  "Seed",
		email:     "staff@tb.local",
		password:  "P@ssw0rd",
		role:      "staff",
	},
}

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, nil)))

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		slog.Error("DATABASE_URL not set")
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		slog.Error("db connect failed", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		slog.Error("db ping failed", "error", err)
		os.Exit(1)
	}

	inserted := 0
	skipped := 0

	for _, u := range seeds {
		hash, err := bcrypt.GenerateFromPassword([]byte(u.password), 12)
		if err != nil {
			slog.Error("bcrypt failed", "email", u.email, "error", err)
			os.Exit(1)
		}

		id, err := uuid.NewV7()
		if err != nil {
			slog.Error("uuid gen failed", "error", err)
			os.Exit(1)
		}

		tag, err := pool.Exec(ctx, `
			INSERT INTO users (id, first_name, last_name, email, password_hash, role, phone)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
			ON CONFLICT (email) DO NOTHING
		`, id, u.firstName, u.lastName, u.email, string(hash), u.role, u.phone)
		if err != nil {
			slog.Error("insert failed", "email", u.email, "error", err)
			os.Exit(1)
		}

		if tag.RowsAffected() == 0 {
			slog.Info("skipped (already exists)", "email", u.email)
			skipped++
		} else {
			slog.Info("inserted", "email", u.email, "role", u.role)
			inserted++
		}
	}

	fmt.Printf("\nseed done: %d inserted, %d skipped\n", inserted, skipped)
	fmt.Println("\ncredentials:")
	for _, u := range seeds {
		fmt.Printf("  %-30s  %-10s  %s\n", u.email, u.role, u.password)
	}
}
