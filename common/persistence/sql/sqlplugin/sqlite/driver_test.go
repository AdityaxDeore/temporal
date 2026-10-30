package sqlite

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

func TestIsDupEntryError(t *testing.T) {
	dsn := "file:" + filepath.Join(t.TempDir(), "t.db")
	conn, err := sqlx.Connect(goSQLDriverName, dsn)
	require.NoError(t, err)
	defer func() { _ = conn.Close() }()
	_, err = conn.Exec(`CREATE TABLE t (id INTEGER PRIMARY KEY, uq TEXT UNIQUE, nn TEXT NOT NULL)`)
	require.NoError(t, err)
	_, err = conn.Exec(`INSERT INTO t VALUES (1, 'a', 'x')`)
	require.NoError(t, err)

	_, dupPK := conn.Exec(`INSERT INTO t VALUES (1, 'b', 'y')`)
	require.Error(t, dupPK)
	_, dupUnique := conn.Exec(`INSERT INTO t VALUES (2, 'a', 'y')`)
	require.Error(t, dupUnique)
	_, notNull := conn.Exec(`INSERT INTO t (id, uq) VALUES (3, 'c')`)
	require.Error(t, notNull)
	_, noTable := conn.Exec(`SELECT * FROM missing`)
	require.Error(t, noTable)

	other, err := sqlx.Connect(goSQLDriverName, dsn)
	require.NoError(t, err)
	defer func() { _ = other.Close() }()
	lock, err := other.Conn(context.Background())
	require.NoError(t, err)
	defer func() { _ = lock.Close() }()
	_, err = lock.ExecContext(context.Background(), "BEGIN IMMEDIATE")
	require.NoError(t, err)
	_, busy := conn.Exec(`INSERT INTO t VALUES (4, 'd', 'z')`)
	require.ErrorContains(t, busy, "database is locked")
	_, err = lock.ExecContext(context.Background(), "ROLLBACK")
	require.NoError(t, err)

	d := &db{}
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "primary key violation",
			err:  dupPK,
			want: true,
		},
		{
			name: "unique violation",
			err:  dupUnique,
			want: true,
		},
		{
			name: "wrapped primary key violation",
			err:  fmt.Errorf("insert failed: %w", dupPK),
			want: true,
		},
		{
			name: "not null violation",
			err:  notNull,
			want: false,
		},
		{
			name: "no such table",
			err:  noTable,
			want: false,
		},
		{
			name: "database is locked",
			err:  busy,
			want: false,
		},
		{
			name: "unrelated error",
			err:  errors.New("boom"),
			want: false,
		},
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, d.IsDupEntryError(tc.err))
		})
	}
}
