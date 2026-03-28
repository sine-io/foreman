package sqlite

import (
	"database/sql"
	_ "embed"
	"fmt"

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

	if err := applyMigrations(db); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("apply migrations: %w", err)
	}

	return db, nil
}

func applyMigrations(db *sql.DB) error {
	if _, err := db.Exec(`
create table if not exists schema_migrations (
  version text primary key
)`); err != nil {
		return fmt.Errorf("ensure schema migrations table: %w", err)
	}

	for _, migration := range migrations {
		applied, err := migrationApplied(db, migration.version)
		if err != nil {
			return fmt.Errorf("check migration %s: %w", migration.version, err)
		}
		if applied {
			continue
		}

		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("begin migration %s: %w", migration.version, err)
		}
		if _, err := tx.Exec(migration.sql); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("apply migration %s: %w", migration.version, err)
		}
		if _, err := tx.Exec(
			`insert into schema_migrations(version) values (?)`,
			migration.version,
		); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("record migration %s: %w", migration.version, err)
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit migration %s: %w", migration.version, err)
		}
	}

	return nil
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
