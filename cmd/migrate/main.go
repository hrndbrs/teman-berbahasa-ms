package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

type migrationInfo struct {
	name    string
	version uint
}

func listMigrations(path string) ([]migrationInfo, map[uint]string, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, nil, fmt.Errorf("reading migrations directory: %w", err)
	}
	seen := map[string]bool{}
	var migrations []migrationInfo
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".up.sql") {
			continue
		}
		trimmed := strings.TrimSuffix(e.Name(), ".up.sql")
		if seen[trimmed] {
			continue
		}
		seen[trimmed] = true
		parts := strings.SplitN(trimmed, "_", 2)
		v, err := strconv.ParseUint(parts[0], 10, 64)
		if err != nil {
			continue
		}
		migrations = append(migrations, migrationInfo{name: trimmed, version: uint(v)})
	}
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].version < migrations[j].version
	})
	nameByVersion := make(map[uint]string, len(migrations))
	for _, m := range migrations {
		nameByVersion[m.version] = m.name
	}
	return migrations, nameByVersion, nil
}

func printStatus(migrations []migrationInfo, currentVersion uint, dirty bool) {
	for _, m := range migrations {
		applied := m.version <= currentVersion
		mark := " "
		if applied {
			mark = "x"
		}
		if dirty && m.version == currentVersion {
			fmt.Printf("  [!] %s  (dirty)\n", m.name)
		} else {
			fmt.Printf("  [%s] %s\n", mark, m.name)
		}
	}
	if len(migrations) == 0 {
		fmt.Println("  (no migration files found)")
	}
}

func runUp(m *migrate.Migrate, nameByVersion map[uint]string) error {
	for {
		v, dirty, err := m.Version()
		if err == migrate.ErrNilVersion {
			v = 0
		} else if err != nil {
			return fmt.Errorf("reading version: %w", err)
		}
		if dirty {
			slog.Warn("dirty migration detected", "version", v)
		}
		next := nameByVersion[v+1]
		if next == "" {
			return nil
		}
		slog.Info("applying migration", "migration", next, "version", v+1)
		if err := m.Steps(1); err != nil {
			if err == migrate.ErrNoChange {
				return nil
			}
			return fmt.Errorf("applying %s: %w", next, err)
		}
		slog.Info("migration applied", "migration", next)
	}
}

func runDown(m *migrate.Migrate, nameByVersion map[uint]string) error {
	for {
		v, dirty, err := m.Version()
		if err == migrate.ErrNilVersion {
			return nil
		} else if err != nil {
			return fmt.Errorf("reading version: %w", err)
		}
		if dirty {
			slog.Warn("dirty migration detected", "version", v)
		}
		name, ok := nameByVersion[v]
		if !ok {
			name = fmt.Sprintf("v%d", v)
		}
		slog.Info("reverting migration", "migration", name, "version", v)
		if err := m.Steps(-1); err != nil {
			if err == migrate.ErrNoChange {
				return nil
			}
			return fmt.Errorf("reverting %s: %w", name, err)
		}
		slog.Info("migration reverted", "migration", name)
	}
}

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
		fmt.Fprintln(os.Stderr, "usage: migrate [flags] <up|down|status|version>")
		os.Exit(1)
	}

	m, err := migrate.New("file://"+*path, *db)
	if err != nil {
		slog.Error("migrate init failed", "error", err)
		os.Exit(1)
	}
	defer m.Close()

	migrations, nameByVersion, err := listMigrations(*path)
	if err != nil {
		slog.Error("listing migrations failed", "error", err)
		os.Exit(1)
	}

	switch cmd {
	case "up":
		err = runUp(m, nameByVersion)
	case "down":
		err = runDown(m, nameByVersion)
	case "status":
		v, dirty, verr := m.Version()
		if verr == migrate.ErrNilVersion {
			v, dirty = 0, false
		} else if verr != nil {
			slog.Error("version check failed", "error", verr)
			os.Exit(1)
		}
		printStatus(migrations, v, dirty)
		return
	case "version":
		v, dirty, verr := m.Version()
		if verr == migrate.ErrNilVersion {
			fmt.Printf("version=0 dirty=false (no migrations applied)\n")
			return
		} else if verr != nil {
			slog.Error("version check failed", "error", verr)
			os.Exit(1)
		}
		name := nameByVersion[v]
		if name == "" {
			fmt.Printf("version=%d dirty=%v\n", v, dirty)
		} else {
			fmt.Printf("version=%d dirty=%v migration=%s\n", v, dirty, name)
		}
		return
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", cmd)
		os.Exit(1)
	}

	if err != nil && err != migrate.ErrNoChange {
		slog.Error("migration failed", "error", err)
		os.Exit(1)
	}

	slog.Info("migration complete")
}
