package sqlite

import (
	"database/sql"
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
