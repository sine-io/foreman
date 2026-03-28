package sqlite

import (
	"database/sql"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMigrationsCreateCoreTables(t *testing.T) {
	db := OpenTestDB(t)

	requireTable(t, db, "projects")
	requireTable(t, db, "modules")
	requireTable(t, db, "tasks")
	requireTable(t, db, "leases")
	requireTable(t, db, "artifacts")
}

func TestOpenAppliesAllMigrationsInOrder(t *testing.T) {
	db := OpenTestDB(t)

	requireColumn(t, db, "runs", "created_at")
	requireColumn(t, db, "approvals", "created_at")
	requireColumn(t, db, "artifacts", "created_at")
}

func TestOpenIsIdempotentAcrossRepeatedBoots(t *testing.T) {
	path := filepath.Join(t.TempDir(), "foreman.db")

	db, err := Open(path)
	require.NoError(t, err)
	require.NoError(t, db.Close())

	db, err = Open(path)
	require.NoError(t, err)
	requireMigrationVersions(t, db, "001_init.sql", "002_control_plane_hardening.sql")
	require.NoError(t, db.Close())
}

func TestOpenUpgradesLegacyDatabaseAndRecordsMigrations(t *testing.T) {
	path := filepath.Join(t.TempDir(), "legacy.db")

	db, err := sql.Open("sqlite", path)
	require.NoError(t, err)

	_, err = db.Exec(`PRAGMA foreign_keys = ON`)
	require.NoError(t, err)
	_, err = db.Exec(initSchema)
	require.NoError(t, err)
	require.NoError(t, db.Close())

	db, err = Open(path)
	require.NoError(t, err)
	defer func() { require.NoError(t, db.Close()) }()

	requireColumn(t, db, "runs", "created_at")
	requireColumn(t, db, "approvals", "created_at")
	requireColumn(t, db, "artifacts", "created_at")
	requireMigrationVersions(t, db, "001_init.sql", "002_control_plane_hardening.sql")
}

func TestOpenIsSafeUnderConcurrentBoots(t *testing.T) {
	path := filepath.Join(t.TempDir(), "concurrent.db")

	var wg sync.WaitGroup
	errs := make(chan error, 2)
	start := make(chan struct{})

	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			db, err := Open(path)
			if err == nil {
				err = db.Close()
			}
			errs <- err
		}()
	}

	close(start)
	wg.Wait()
	close(errs)

	for err := range errs {
		require.NoError(t, err)
	}

	db, err := Open(path)
	require.NoError(t, err)
	defer func() { require.NoError(t, db.Close()) }()
	requireMigrationVersions(t, db, "001_init.sql", "002_control_plane_hardening.sql")
}

func TestOpenDoesNotNeedWriteLockWhenDatabaseIsAlreadyMigrated(t *testing.T) {
	path := filepath.Join(t.TempDir(), "writer-lock.db")

	db, err := Open(path)
	require.NoError(t, err)
	require.NoError(t, db.Close())

	lockDB, err := sql.Open("sqlite", path)
	require.NoError(t, err)
	defer func() { require.NoError(t, lockDB.Close()) }()
	_, err = lockDB.Exec(`PRAGMA busy_timeout = 5000`)
	require.NoError(t, err)
	_, err = lockDB.Exec(`BEGIN IMMEDIATE`)
	require.NoError(t, err)
	defer func() {
		_, rollbackErr := lockDB.Exec(`ROLLBACK`)
		require.NoError(t, rollbackErr)
	}()

	reopened, err := Open(path)
	require.NoError(t, err)
	require.NoError(t, reopened.Close())
}

func OpenTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := Open("file:foreman-test?mode=memory&cache=shared")
	require.NoError(t, err)

	t.Cleanup(func() {
		require.NoError(t, db.Close())
	})

	return db
}

func requireTable(t *testing.T, db *sql.DB, name string) {
	t.Helper()

	var found string
	err := db.QueryRow(
		`select name from sqlite_master where type = 'table' and name = ?`,
		name,
	).Scan(&found)
	require.NoError(t, err)
	require.Equal(t, name, found)
}

func requireColumn(t *testing.T, db *sql.DB, table, column string) {
	t.Helper()

	rows, err := db.Query(`pragma table_info(` + table + `)`)
	require.NoError(t, err)
	defer rows.Close()

	var (
		cid        int
		name       string
		dataType   string
		notNull    int
		defaultVal sql.NullString
		pk         int
	)

	for rows.Next() {
		err := rows.Scan(&cid, &name, &dataType, &notNull, &defaultVal, &pk)
		require.NoError(t, err)
		if name == column {
			require.NoError(t, rows.Err())
			return
		}
	}

	require.NoError(t, rows.Err())
	t.Fatalf("column %q not found in table %q", column, table)
}

func requireMigrationVersions(t *testing.T, db *sql.DB, versions ...string) {
	t.Helper()

	rows, err := db.Query(`select version from schema_migrations order by version`)
	require.NoError(t, err)
	defer rows.Close()

	var found []string
	for rows.Next() {
		var version string
		require.NoError(t, rows.Scan(&version))
		found = append(found, version)
	}
	require.NoError(t, rows.Err())
	require.Equal(t, versions, found)
}
