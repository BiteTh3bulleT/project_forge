package storagebackend

import (
	"errors"
	"testing"
)

func TestConfigDefaultsToSQLite(t *testing.T) {
	cfg, err := NewConfig(ConfigInput{})
	if err != nil {
		t.Fatalf("NewConfig failed: %v", err)
	}
	if cfg.Kind != BackendSQLite {
		t.Fatalf("expected default backend %q, got %q", BackendSQLite, cfg.Kind)
	}
	if !cfg.Capabilities.CanonicalTruthAllowed {
		t.Fatalf("expected sqlite to remain canonical-capable")
	}
}

func TestPostgresBackendRequiresDSN(t *testing.T) {
	_, err := NewConfig(ConfigInput{Backend: "postgres"})
	if !errors.Is(err, ErrPostgresDSNRequired) {
		t.Fatalf("expected ErrPostgresDSNRequired, got %v", err)
	}

	cfg, err := NewConfig(ConfigInput{
		Backend:     "postgres",
		PostgresDSN: "postgres://forge:forge@localhost:5432/forge?sslmode=disable",
	})
	if err != nil {
		t.Fatalf("NewConfig with postgres DSN failed: %v", err)
	}
	if cfg.Kind != BackendPostgres {
		t.Fatalf("expected postgres backend, got %q", cfg.Kind)
	}
}

func TestInvalidBackendRejected(t *testing.T) {
	_, err := NewConfig(ConfigInput{Backend: "mysql"})
	if !errors.Is(err, ErrInvalidBackend) {
		t.Fatalf("expected ErrInvalidBackend, got %v", err)
	}
}

func TestRedisAndQdrantDoNotImplyBackendSwitch(t *testing.T) {
	cfg, err := NewConfig(ConfigInput{
		RedisAddr: "redis:6379",
		QdrantURL: "http://qdrant:6333",
	})
	if err != nil {
		t.Fatalf("NewConfig failed: %v", err)
	}
	if cfg.Kind != BackendSQLite {
		t.Fatalf("expected redis/qdrant env to preserve sqlite default, got %q", cfg.Kind)
	}
}
