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

func TestShadowDiagnosticPersistenceDefaultsDisabled(t *testing.T) {
	t.Setenv("FORGE_SHADOW_DIAGNOSTIC_PERSISTENCE_ENABLED", "")
	t.Setenv("FORGE_SHADOW_DIAGNOSTIC_RETENTION_DAYS", "")
	t.Setenv("FORGE_SHADOW_DIAGNOSTIC_MAX_PAYLOAD_BYTES", "")

	cfg := Load()
	if cfg.ShadowDiagnosticPersistenceEnabled {
		t.Fatalf("expected shadow diagnostic persistence disabled by default")
	}
	if cfg.ShadowDiagnosticRetentionDays <= 0 {
		t.Fatalf("expected positive default retention days")
	}
	if cfg.ShadowDiagnosticMaxPayloadBytes <= 0 {
		t.Fatalf("expected positive default max payload bytes")
	}
	if err := cfg.ValidateShadowDiagnosticPersistence(); err != nil {
		t.Fatalf("disabled persistence should validate without postgres config: %v", err)
	}
}

func TestShadowDiagnosticPersistenceRequiresPostgresConfigWhenEnabled(t *testing.T) {
	cfg := Config{
		StoreBackend:                       "sqlite",
		ShadowDiagnosticPersistenceEnabled: true,
		ShadowDiagnosticRetentionDays:      30,
		ShadowDiagnosticMaxPayloadBytes:    1024,
	}
	if err := cfg.ValidateShadowDiagnosticPersistence(); err == nil {
		t.Fatalf("expected enabled persistence without postgres DSN to fail closed")
	}

	cfg.PostgresDSN = "postgres://forge:forge@localhost:5432/forge?sslmode=disable"
	if err := cfg.ValidateShadowDiagnosticPersistence(); err != nil {
		t.Fatalf("expected explicit postgres DSN to allow diagnostic persistence: %v", err)
	}
	if cfg.StoreBackend != "sqlite" {
		t.Fatalf("enabling diagnostic persistence must not switch main backend, got %q", cfg.StoreBackend)
	}
}

func TestShadowDiagnosticPersistenceInvalidConfigFailsSafe(t *testing.T) {
	cfg := Config{
		PostgresDSN:                        "postgres://forge:forge@localhost:5432/forge?sslmode=disable",
		ShadowDiagnosticPersistenceEnabled: true,
		ShadowDiagnosticRetentionDays:      0,
		ShadowDiagnosticMaxPayloadBytes:    1024,
	}
	if err := cfg.ValidateShadowDiagnosticPersistence(); err == nil {
		t.Fatalf("expected invalid retention to fail safe")
	}
	cfg.ShadowDiagnosticRetentionDays = 30
	cfg.ShadowDiagnosticMaxPayloadBytes = 0
	if err := cfg.ValidateShadowDiagnosticPersistence(); err == nil {
		t.Fatalf("expected invalid payload limit to fail safe")
	}
}

func TestQdrantShadowIndexDefaultsDisabled(t *testing.T) {
	t.Setenv("FORGE_QDRANT_SHADOW_INDEX_ENABLED", "")
	t.Setenv("FORGE_QDRANT_COLLECTION", "")
	t.Setenv("FORGE_QDRANT_VECTOR_SIZE", "")
	t.Setenv("FORGE_QDRANT_TIMEOUT_MS", "")
	t.Setenv("FORGE_QDRANT_URL", "")
	t.Setenv("FORGE_STORE_BACKEND", "sqlite")

	cfg := Load()
	if cfg.QdrantShadowIndexEnabled {
		t.Fatalf("expected qdrant shadow index disabled by default")
	}
	if cfg.QdrantCollection != "forge_shadow_embeddings" {
		t.Fatalf("unexpected default collection %q", cfg.QdrantCollection)
	}
	if cfg.QdrantTimeoutMs <= 0 {
		t.Fatalf("expected positive qdrant timeout default")
	}
	if err := cfg.ValidateQdrantShadowIndex(); err != nil {
		t.Fatalf("disabled qdrant shadow index should validate without url: %v", err)
	}
}

func TestQdrantShadowIndexRequiresURLWhenEnabled(t *testing.T) {
	cfg := Config{
		StoreBackend:                    "sqlite",
		QdrantShadowIndexEnabled:        true,
		QdrantCollection:                "forge_shadow_embeddings",
		QdrantVectorSize:                128,
		QdrantTimeoutMs:                 3000,
		ShadowDiagnosticMaxPayloadBytes: 1024,
	}
	if err := cfg.ValidateQdrantShadowIndex(); err == nil {
		t.Fatalf("expected enabled qdrant shadow index without URL to fail closed")
	}

	cfg.QdrantURL = "http://qdrant:6333"
	if err := cfg.ValidateQdrantShadowIndex(); err != nil {
		t.Fatalf("expected explicit qdrant URL to allow shadow index: %v", err)
	}
	if cfg.StoreBackend != "sqlite" {
		t.Fatalf("enabling qdrant shadow index must not switch live backend")
	}
}

func TestQdrantShadowIndexInvalidConfigFailsSafe(t *testing.T) {
	cfg := Config{
		QdrantURL:                "http://qdrant:6333",
		QdrantShadowIndexEnabled: true,
		QdrantCollection:         "",
		QdrantTimeoutMs:          3000,
	}
	if err := cfg.ValidateQdrantShadowIndex(); err == nil {
		t.Fatalf("expected empty collection to fail safe")
	}
	cfg.QdrantCollection = "forge_shadow_embeddings"
	cfg.QdrantTimeoutMs = 0
	if err := cfg.ValidateQdrantShadowIndex(); err == nil {
		t.Fatalf("expected invalid timeout to fail safe")
	}
}

func TestQdrantShadowIndexDoesNotSwitchRetrievalBackend(t *testing.T) {
	t.Setenv("FORGE_STORE_BACKEND", "sqlite")
	t.Setenv("FORGE_QDRANT_URL", "http://qdrant:6333")
	t.Setenv("FORGE_QDRANT_SHADOW_INDEX_ENABLED", "true")

	cfg := Load()
	if cfg.StoreBackend != "sqlite" {
		t.Fatalf("qdrant shadow index must not change store backend, got %q", cfg.StoreBackend)
	}
	backendCfg, err := cfg.StorageBackendConfig()
	if err != nil {
		t.Fatalf("StorageBackendConfig failed: %v", err)
	}
	if backendCfg.Kind != storagebackend.BackendSQLite {
		t.Fatalf("qdrant shadow index must not switch retrieval/storage backend, got %q", backendCfg.Kind)
	}
}
