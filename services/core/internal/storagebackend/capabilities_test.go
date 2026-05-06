package storagebackend

import "testing"

func TestBackendCapabilities(t *testing.T) {
	sqlite := CapabilitiesFor(BackendSQLite)
	if !sqlite.DurableRelational || !sqlite.Transactional || !sqlite.CanonicalTruthAllowed {
		t.Fatalf("expected sqlite durable transactional canonical-capable, got %+v", sqlite)
	}
	if sqlite.VectorIndex || sqlite.EphemeralCache {
		t.Fatalf("sqlite should not be modeled as vector/cache infrastructure, got %+v", sqlite)
	}

	postgres := CapabilitiesFor(BackendPostgres)
	if !postgres.DurableRelational || !postgres.Transactional || !postgres.CanonicalTruthAllowed {
		t.Fatalf("expected postgres durable transactional canonical-capable for future migration, got %+v", postgres)
	}
	if !postgres.AdvisoryLocksAvailable {
		t.Fatalf("expected postgres advisory locks to be available")
	}
}

func TestRedisAndQdrantCannotBeCanonicalTruth(t *testing.T) {
	redis := InfrastructureCapabilities(InfrastructureRedis)
	if !redis.EphemeralCache {
		t.Fatalf("expected redis ephemeral cache capability, got %+v", redis)
	}
	if redis.CanonicalTruthAllowed || redis.DurableRelational {
		t.Fatalf("redis must not be canonical truth or durable relational, got %+v", redis)
	}

	qdrant := InfrastructureCapabilities(InfrastructureQdrant)
	if !qdrant.VectorIndex {
		t.Fatalf("expected qdrant vector index capability, got %+v", qdrant)
	}
	if qdrant.CanonicalTruthAllowed || qdrant.DurableRelational {
		t.Fatalf("qdrant must not be canonical truth or durable relational, got %+v", qdrant)
	}
}
