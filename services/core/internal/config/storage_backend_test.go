package config

import (
	"errors"
	"testing"

	"forge/projectforge/services/core/internal/storagebackend"
)

func TestLoadStorageBackendDefaultsToSQLite(t *testing.T) {
	t.Setenv("FORGE_STORE_BACKEND", "")
	t.Setenv("FORGE_POSTGRES_DSN", "")
	t.Setenv("FORGE_REDIS_ADDR", "")
	t.Setenv("FORGE_QDRANT_URL", "")

	cfg := Load()
	if cfg.StoreBackend != "sqlite" {
		t.Fatalf("expected sqlite default backend, got %q", cfg.StoreBackend)
	}
	if cfg.PostgresDSN != "" || cfg.RedisAddr != "" || cfg.QdrantURL != "" {
		t.Fatalf("expected storage service endpoints unset by default, got postgres=%q redis=%q qdrant=%q", cfg.PostgresDSN, cfg.RedisAddr, cfg.QdrantURL)
	}
	backendCfg, err := cfg.StorageBackendConfig()
	if err != nil {
		t.Fatalf("StorageBackendConfig failed: %v", err)
	}
	if backendCfg.Kind != storagebackend.BackendSQLite {
		t.Fatalf("expected sqlite backend config, got %q", backendCfg.Kind)
	}
}

func TestLoadStorageBackendOverrides(t *testing.T) {
	t.Setenv("FORGE_STORE_BACKEND", "postgres")
	t.Setenv("FORGE_POSTGRES_DSN", "postgres://forge:forge@postgres:5432/forge?sslmode=disable")
	t.Setenv("FORGE_REDIS_ADDR", "redis:6379")
	t.Setenv("FORGE_QDRANT_URL", "http://qdrant:6333")

	cfg := Load()
	if cfg.StoreBackend != "postgres" {
		t.Fatalf("expected postgres backend override, got %q", cfg.StoreBackend)
	}
	if cfg.PostgresDSN != "postgres://forge:forge@postgres:5432/forge?sslmode=disable" {
		t.Fatalf("expected postgres DSN override, got %q", cfg.PostgresDSN)
	}
	if cfg.RedisAddr != "redis:6379" {
		t.Fatalf("expected redis addr override, got %q", cfg.RedisAddr)
	}
	if cfg.QdrantURL != "http://qdrant:6333" {
		t.Fatalf("expected qdrant url override, got %q", cfg.QdrantURL)
	}
	backendCfg, err := cfg.StorageBackendConfig()
	if err != nil {
		t.Fatalf("StorageBackendConfig failed: %v", err)
	}
	if backendCfg.Kind != storagebackend.BackendPostgres {
		t.Fatalf("expected postgres backend config, got %q", backendCfg.Kind)
	}
}

func TestStorageBackendConfigRejectsInvalidBackend(t *testing.T) {
	cfg := Config{StoreBackend: "mysql"}
	_, err := cfg.StorageBackendConfig()
	if !errors.Is(err, storagebackend.ErrInvalidBackend) {
		t.Fatalf("expected ErrInvalidBackend, got %v", err)
	}
}
