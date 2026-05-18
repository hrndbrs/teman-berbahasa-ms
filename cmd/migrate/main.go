package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func main() {
	path := flag.String("path", "internal/db/migrations", "migrations directory")
	db := flag.String("database", "", "database URL")
	flag.Parse()

	if *db == "" {
		*db = os.Getenv("DATABASE_URL")
	}
	if *db == "" {
		slog.Error("database URL required: use -database or DATABASE_URL env var")
		os.Exit(1)
	}

	cmd := flag.Arg(0)
	if cmd == "" {
		fmt.Fprintln(os.Stderr, "usage: migrate [flags] <up|down|version>")
		os.Exit(1)
	}

	m, err := migrate.New("file://"+*path, *db)
	if err != nil {
		slog.Error("migrate init failed", "error", err)
		os.Exit(1)
	}
	defer m.Close()

	switch cmd {
	case "up":
		err = m.Up()
	case "down":
		err = m.Down()
	case "version":
		v, dirty, verr := m.Version()
		if verr != nil {
			slog.Error("version check failed", "error", verr)
			os.Exit(1)
		}
		fmt.Printf("version=%d dirty=%v\n", v, dirty)
		return
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", cmd)
		os.Exit(1)
	}

	if err != nil && err != migrate.ErrNoChange {
		slog.Error("migration failed", "error", err)
		os.Exit(1)
	}

	slog.Info("migration complete", "command", cmd)
}
