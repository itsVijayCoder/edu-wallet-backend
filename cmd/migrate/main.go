// Command migrate runs database migrations as a one-off command, replacing the
// manual `migrate -database ... -path migrations <cmd>` shell steps.
//
// It reuses the application configuration (config.Load) so the database DSN is
// resolved the same way as the API server: DATABASE_URL takes precedence,
// otherwise the individual DB_* env vars are combined. The .env file is loaded
// automatically.
//
// Usage:
//
//	eduwallet-migrate up                 # apply all pending migrations
//	eduwallet-migrate down [N]           # roll back N migrations (default 1)
//	eduwallet-migrate goto V             # migrate to version V
//	eduwallet-migrate force V            # force schema version V (un-dirty)
//	eduwallet-migrate version            # print current version + dirty state
//	eduwallet-migrate create NAME        # create empty up/down migration pair
//	eduwallet-migrate drop               # drop everything in the database (DANGEROUS)
//
// Flags:
//
//	-path string   migrations directory (default "migrations")
//	-ext  string   file extension for created migrations (default "sql")
package main

import (
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"

	"github.com/itsVijayCoder/edu-wallet-backend/internal/config"
	"github.com/itsVijayCoder/edu-wallet-backend/pkg/logger"
)

const usage = `eduwallet-migrate - database migrations

Usage:
  eduwallet-migrate [flags] <command> [args]

Commands:
  up                 Apply all pending migrations
  down [N]           Roll back N migrations (default 1)
  goto V             Migrate to version V
  force V            Force schema version V (clears dirty state)
  version            Print current migration version and dirty state
  create NAME        Create a new sequential up/down migration pair named NAME
  drop               Drop all database objects (DANGEROUS, requires confirmation)

Flags:
  -path string   migrations directory (default "migrations")
  -ext  string   file extension for created migrations (default "sql")
`

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "migrate: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	fs := flag.NewFlagSet("eduwallet-migrate", flag.ContinueOnError)
	fs.Usage = func() { fmt.Fprint(os.Stderr, usage) }

	migrationsPath := fs.String("path", "migrations", "migrations directory")
	migrationsExt := fs.String("ext", "sql", "file extension for created migrations")
	if err := fs.Parse(args); err != nil {
		return err
	}

	rest := fs.Args()
	if len(rest) == 0 {
		fs.Usage()
		return errors.New("no command provided")
	}
	cmd := rest[0]
	cmdArgs := rest[1:]

	// `create` only touches the filesystem; it does not need a database
	// connection, so skip config loading for it.
	if cmd == "create" {
		return createMigration(*migrationsPath, *migrationsExt, cmdArgs)
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	log := logger.New(cfg.App.Env)

	dsn := cfg.DB.DSN()
	migrationsDir, err := filepath.Abs(*migrationsPath)
	if err != nil {
		return fmt.Errorf("resolve migrations path: %w", err)
	}

	m, err := migrate.New("file://"+migrationsDir, dsn)
	if err != nil {
		return fmt.Errorf("init migrate: %w", err)
	}
	defer func() {
		if srcErr, dbErr := m.Close(); srcErr != nil {
			log.Error("migrate source close error", slog.String("error", srcErr.Error()))
		} else if dbErr != nil {
			log.Error("migrate database close error", slog.String("error", dbErr.Error()))
		}
	}()

	switch cmd {
	case "up":
		return runUp(m, log)
	case "down":
		return runDown(m, log, cmdArgs)
	case "goto":
		return runGoto(m, log, cmdArgs)
	case "force":
		return runForce(m, log, cmdArgs)
	case "version":
		return runVersion(m, log)
	case "drop":
		return runDrop(m, log, cmdArgs)
	default:
		fs.Usage()
		return fmt.Errorf("unknown command: %s", cmd)
	}
}

// runUp applies all pending migrations. ErrNoChange is treated as success.
func runUp(m *migrate.Migrate, log *slog.Logger) error {
	if err := m.Up(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			log.Info("no new migrations to apply")
			return nil
		}
		return fmt.Errorf("apply migrations: %w", err)
	}
	log.Info("migrations applied successfully")
	return printVersion(m, log)
}

// runDown rolls back N migrations (default 1). ErrNoChange is treated as success.
func runDown(m *migrate.Migrate, log *slog.Logger, args []string) error {
	n := 1
	if len(args) > 0 {
		parsed, err := strconv.Atoi(args[0])
		if err != nil || parsed < 1 {
			return fmt.Errorf("invalid step count %q: must be a positive integer", args[0])
		}
		n = parsed
	}
	if err := m.Steps(-n); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			log.Info("no migrations to roll back")
			return nil
		}
		return fmt.Errorf("roll back migrations: %w", err)
	}
	log.Info("migrations rolled back", slog.Int("steps", n))
	return printVersion(m, log)
}

// runGoto migrates to an exact version.
func runGoto(m *migrate.Migrate, log *slog.Logger, args []string) error {
	if len(args) != 1 {
		return errors.New("goto requires a version argument: goto V")
	}
	v, err := strconv.ParseUint(args[0], 10, 64)
	if err != nil {
		return fmt.Errorf("invalid version %q: %w", args[0], err)
	}
	if err := m.Migrate(uint(v)); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			log.Info("already at target version", slog.Uint64("version", v))
			return nil
		}
		return fmt.Errorf("migrate to version %d: %w", v, err)
	}
	log.Info("migrated to version", slog.Uint64("version", v))
	return printVersion(m, log)
}

// runForce forces a specific schema version, clearing the dirty flag.
func runForce(m *migrate.Migrate, log *slog.Logger, args []string) error {
	if len(args) != 1 {
		return errors.New("force requires a version argument: force V")
	}
	v, err := strconv.Atoi(args[0])
	if err != nil || v < 0 {
		return fmt.Errorf("invalid version %q: must be a non-negative integer", args[0])
	}
	if err := m.Force(v); err != nil {
		return fmt.Errorf("force version %d: %w", v, err)
	}
	log.Info("forced schema version", slog.Int("version", v))
	return printVersion(m, log)
}

// runVersion prints the current migration version and dirty state.
func runVersion(m *migrate.Migrate, log *slog.Logger) error {
	return printVersion(m, log)
}

// runDrop drops all database objects. Requires explicit confirmation.
func runDrop(m *migrate.Migrate, log *slog.Logger, args []string) error {
	confirmed := len(args) > 0 && (args[0] == "--yes" || args[0] == "-y")
	if !confirmed {
		return errors.New("drop is destructive; confirm with --yes")
	}
	if err := m.Drop(); err != nil {
		return fmt.Errorf("drop database: %w", err)
	}
	log.Info("database dropped")
	return nil
}

// printVersion logs the current schema version and dirty state.
func printVersion(m *migrate.Migrate, log *slog.Logger) error {
	version, dirty, err := m.Version()
	if err != nil {
		if errors.Is(err, migrate.ErrNilVersion) {
			log.Info("no migrations have been applied (version 0)")
			return nil
		}
		return fmt.Errorf("read version: %w", err)
	}
	log.Info("current schema version",
		slog.Uint64("version", uint64(version)),
		slog.Bool("dirty", dirty),
	)
	if dirty {
		return errors.New("schema is dirty; run `force V` to set a clean version")
	}
	return nil
}

// createMigration writes a new sequential up/down migration pair, matching the
// existing NNNNNN_name.{up,down}.sql convention (6-digit zero-padded sequence).
func createMigration(dir, ext string, args []string) error {
	if len(args) != 1 || strings.TrimSpace(args[0]) == "" {
		return errors.New("create requires a migration name: create NAME")
	}
	name := strings.TrimSpace(args[0])

	matches, err := filepath.Glob(filepath.Join(dir, "*."+ext))
	if err != nil {
		return fmt.Errorf("scan migrations dir: %w", err)
	}

	next, err := nextSeqVersion(matches, 6)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create migrations dir: %w", err)
	}

	ext = "." + strings.TrimPrefix(ext, ".")
	for _, direction := range []string{"up", "down"} {
		filename := filepath.Join(dir, fmt.Sprintf("%s_%s.%s%s", next, name, direction, ext))
		f, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0o666)
		if err != nil {
			return fmt.Errorf("create %s: %w", filename, err)
		}
		if err := f.Close(); err != nil {
			return fmt.Errorf("close %s: %w", filename, err)
		}
		abs, _ := filepath.Abs(filename)
		fmt.Println(abs)
	}
	return nil
}

// nextSeqVersion computes the next zero-padded sequence number from existing
// migration filenames, mirroring golang-migrate's sequential versioning.
func nextSeqVersion(matches []string, digits int) (string, error) {
	if digits <= 0 {
		digits = 6
	}
	maxSeq := -1
	for _, m := range matches {
		base := filepath.Base(m)
		parts := strings.SplitN(base, "_", 2)
		if len(parts) < 2 {
			continue
		}
		n, err := strconv.Atoi(parts[0])
		if err != nil {
			continue
		}
		if n > maxSeq {
			maxSeq = n
		}
	}
	next := maxSeq + 1
	version := fmt.Sprintf("%0[2]*[1]d", next, digits)
	if len(version) > digits {
		return "", fmt.Errorf("next sequence number %s too large: at most %d digits allowed", version, digits)
	}
	return version, nil
}
