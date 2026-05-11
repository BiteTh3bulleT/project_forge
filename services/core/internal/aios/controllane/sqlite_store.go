package controllane

import (
	"context"
	"database/sql"
)

type sqlExecutor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

type SQLiteSemanticStore struct {
	exec       sqlExecutor
	meta       CommitMetadata
	background context.Context
}

func NewSQLiteSemanticStore(db *sql.DB) *SQLiteSemanticStore {
	return &SQLiteSemanticStore{exec: db, background: context.Background()}
}

func newSQLiteSemanticStore(exec sqlExecutor) *SQLiteSemanticStore {
	return &SQLiteSemanticStore{exec: exec, background: context.Background()}
}

func (s *SQLiteSemanticStore) SetCommitMetadata(meta CommitMetadata) {
	s.meta = meta
}
