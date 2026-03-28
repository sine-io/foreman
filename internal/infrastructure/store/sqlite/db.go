package sqlite

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"sort"

	_ "modernc.org/sqlite"
)

//go:embed migrations/001_init.sql
var initSchema string

//go:embed migrations/002_control_plane_hardening.sql
var controlPlaneHardeningSchema string

type migration struct {
	version string
	sql     string
}

var migrations = []migration{
	{version: "001_init.sql", sql: initSchema},
	{version: "002_control_plane_hardening.sql", sql: controlPlaneHardeningSchema},
}

func Open(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	if _, err := db.Exec(`PRAGMA foreign_keys = ON`); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("enable foreign keys: %w", err)
	}
	if _, err := db.Exec(`PRAGMA busy_timeout = 5000`); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("set busy timeout: %w", err)
	}

	if err := applyMigrations(db); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("apply migrations: %w", err)
	}

	return db, nil
}

func applyMigrations(db *sql.DB) error {
	sorted := append([]migration(nil), migrations...)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].version < sorted[j].version
	})

	for i, migration := range migrations {
		if migration.version != sorted[i].version {
			return fmt.Errorf("migrations must be declared in filename order")
		}
	}

	for _, migration := range sorted {
		ctx := context.Background()
		var applied bool

		hasMigrationsTable, err := schemaMigrationsTableExists(db)
		if err != nil {
			return fmt.Errorf("check schema_migrations table: %w", err)
		}
		if hasMigrationsTable {
			applied, err := migrationApplied(db, migration.version)
			if err != nil {
				return fmt.Errorf("check migration %s: %w", migration.version, err)
			}
			if applied {
				continue
			}
		}

		conn, err := db.Conn(ctx)
		if err != nil {
			return fmt.Errorf("acquire connection for migration %s: %w", migration.version, err)
		}
		if _, err := conn.ExecContext(ctx, `PRAGMA busy_timeout = 5000`); err != nil {
			_ = conn.Close()
			return fmt.Errorf("set connection busy timeout for migration %s: %w", migration.version, err)
		}

		if _, err := conn.ExecContext(ctx, `BEGIN IMMEDIATE`); err != nil {
			_ = conn.Close()
			return fmt.Errorf("lock migration %s: %w", migration.version, err)
		}
		if _, err := conn.ExecContext(ctx, `
create table if not exists schema_migrations (
  version text primary key
)`); err != nil {
			_, _ = conn.ExecContext(ctx, `ROLLBACK`)
			_ = conn.Close()
			return fmt.Errorf("ensure schema migrations table: %w", err)
		}

		applied, err = migrationAppliedConn(ctx, conn, migration.version)
		if err != nil {
			_, _ = conn.ExecContext(ctx, `ROLLBACK`)
			_ = conn.Close()
			return fmt.Errorf("check migration %s: %w", migration.version, err)
		}
		if applied {
			if _, err := conn.ExecContext(ctx, `COMMIT`); err != nil {
				_ = conn.Close()
				return fmt.Errorf("commit skipped migration %s: %w", migration.version, err)
			}
			_ = conn.Close()
			continue
		}
		if _, err := conn.ExecContext(ctx, migration.sql); err != nil {
			_, _ = conn.ExecContext(ctx, `ROLLBACK`)
			_ = conn.Close()
			return fmt.Errorf("apply migration %s: %w", migration.version, err)
		}
		if _, err := conn.ExecContext(
			ctx,
			`insert into schema_migrations(version) values (?)`,
			migration.version,
		); err != nil {
			_, _ = conn.ExecContext(ctx, `ROLLBACK`)
			_ = conn.Close()
			return fmt.Errorf("record migration %s: %w", migration.version, err)
		}
		if _, err := conn.ExecContext(ctx, `COMMIT`); err != nil {
			_ = conn.Close()
			return fmt.Errorf("commit migration %s: %w", migration.version, err)
		}
		_ = conn.Close()
	}

	return nil
}

func schemaMigrationsTableExists(db *sql.DB) (bool, error) {
	var count int
	if err := db.QueryRow(
		`select count(1) from sqlite_master where type = 'table' and name = 'schema_migrations'`,
	).Scan(&count); err != nil {
		return false, err
	}
	return count > 0, nil
}

func migrationApplied(db *sql.DB, version string) (bool, error) {
	var count int
	if err := db.QueryRow(
		`select count(1) from schema_migrations where version = ?`,
		version,
	).Scan(&count); err != nil {
		return false, err
	}
	return count > 0, nil
}

func migrationAppliedConn(ctx context.Context, conn *sql.Conn, version string) (bool, error) {
	var count int
	if err := conn.QueryRowContext(
		ctx,
		`select count(1) from schema_migrations where version = ?`,
		version,
	).Scan(&count); err != nil {
		return false, err
	}
	return count > 0, nil
}
