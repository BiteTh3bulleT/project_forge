package store

import (
	"errors"
	"strings"
)

var (
	ErrPostgresDSNRequired = errors.New("postgres DSN required")
	ErrInvalidPostgresDSN  = errors.New("invalid postgres DSN")
)

type PostgresConnector struct {
	dsn string
}

func NewPostgresConnector(dsn string) (PostgresConnector, error) {
	normalized := strings.TrimSpace(dsn)
	if normalized == "" {
		return PostgresConnector{}, ErrPostgresDSNRequired
	}
	if !strings.HasPrefix(normalized, "postgres://") && !strings.HasPrefix(normalized, "postgresql://") {
		return PostgresConnector{}, ErrInvalidPostgresDSN
	}
	return PostgresConnector{dsn: normalized}, nil
}

func (c PostgresConnector) DSN() string {
	return c.dsn
}
