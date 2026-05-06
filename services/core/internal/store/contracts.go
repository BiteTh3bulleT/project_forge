package store

import (
	"context"
	"database/sql"
)

type StoreBackend interface {
	Kind() BackendKind
	Migrator() Migrator
	HealthChecker() HealthChecker
	TransactionRunner() TransactionRunner
}

type Migrator interface {
	Run(context.Context, *sql.DB) error
}

type HealthChecker interface {
	CheckHealth(context.Context, *sql.DB) error
}

type TransactionRunner interface {
	RunInTransaction(context.Context, *sql.DB, func(context.Context, *sql.Tx) error) error
}
