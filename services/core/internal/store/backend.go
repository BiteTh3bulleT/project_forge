package store

type BackendKind string

const (
	BackendSQLite   BackendKind = "sqlite"
	BackendPostgres BackendKind = "postgres"
)
