package sqlite

import (
	"database/sql"
	"path/filepath"
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
	require.NoError(t, db.Close())
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
