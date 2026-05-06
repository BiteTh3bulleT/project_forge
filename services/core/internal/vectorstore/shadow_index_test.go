package vectorstore

import (
	"context"
	"errors"
	"testing"
)

type fakeStore struct {
	ensureCalls int
	upsertCalls int
	searchCalls int
	deleteCalls int
	lastPoint   VectorPoint
	err         error
}

func (f *fakeStore) EnsureCollection(_ context.Context, _ CollectionSpec) error {
	f.ensureCalls++
	return f.err
}

func (f *fakeStore) UpsertVector(_ context.Context, point VectorPoint) error {
	f.upsertCalls++
	f.lastPoint = point
	return f.err
}

func (f *fakeStore) SearchVector(_ context.Context, _ SearchRequest) (SearchResult, error) {
	f.searchCalls++
	return SearchResult{}, f.err
}

func (f *fakeStore) DeleteVector(_ context.Context, _, _ string) error {
	f.deleteCalls++
	return f.err
}

func (f *fakeStore) Health(_ context.Context) error {
	return f.err
}

func TestShadowIndexSkipsWhenDisabled(t *testing.T) {
	store := &fakeStore{}
	service := NewShadowIndexService(store, ShadowIndexConfig{
		Enabled:    false,
		Collection: "forge_shadow_embeddings",
	})
	result, err := service.Index(context.Background(), ShadowIndexRequest{
		Vector:  []float64{0.1, 0.2, 0.3},
		Payload: validPayload(),
	})
	if err != nil {
		t.Fatalf("disabled shadow index should not fail: %v", err)
	}
	if !result.Skipped || result.Reason != "disabled" {
		t.Fatalf("expected disabled skip result, got %#v", result)
	}
	if store.ensureCalls != 0 || store.upsertCalls != 0 || store.searchCalls != 0 {
		t.Fatalf("disabled shadow index called vector store: %#v", store)
	}
}

func TestShadowIndexUpsertsPrecomputedVectorWithSafeRefs(t *testing.T) {
	store := &fakeStore{}
	service := NewShadowIndexService(store, ShadowIndexConfig{
		Enabled:          true,
		Collection:       "forge_shadow_embeddings",
		VectorSize:       3,
		EnsureCollection: true,
	})
	result, err := service.Index(context.Background(), ShadowIndexRequest{
		Vector:  []float64{0.1, 0.2, 0.3},
		Payload: validPayload(),
	})
	if err != nil {
		t.Fatalf("index safe vector: %v", err)
	}
	if !result.Indexed || result.PointID == "" {
		t.Fatalf("expected indexed result, got %#v", result)
	}
	if store.ensureCalls != 1 || store.upsertCalls != 1 {
		t.Fatalf("expected ensure+upsert calls, got ensure=%d upsert=%d", store.ensureCalls, store.upsertCalls)
	}
	if store.searchCalls != 0 {
		t.Fatalf("shadow index must not execute search")
	}
	if store.lastPoint.Payload.SourceRefID != "source:1" || store.lastPoint.Payload.ProvenanceRefID == "" {
		t.Fatalf("payload refs/provenance not preserved: %#v", store.lastPoint.Payload)
	}
}

func TestShadowIndexRejectsUnsafePayload(t *testing.T) {
	store := &fakeStore{}
	service := NewShadowIndexService(store, ShadowIndexConfig{Enabled: true, Collection: "forge_shadow_embeddings"})
	payload := validPayload()
	payload.Metadata["chunk_text"] = "raw chunk"
	_, err := service.Index(context.Background(), ShadowIndexRequest{
		Vector:  []float64{0.1, 0.2, 0.3},
		Payload: payload,
	})
	if !errors.Is(err, ErrUnsafePayload) {
		t.Fatalf("expected unsafe payload rejection, got %v", err)
	}
	if store.upsertCalls != 0 {
		t.Fatalf("unsafe payload must not upsert")
	}
}

func TestShadowIndexRejectsDimensionMismatch(t *testing.T) {
	store := &fakeStore{}
	service := NewShadowIndexService(store, ShadowIndexConfig{Enabled: true, Collection: "forge_shadow_embeddings", VectorSize: 4})
	_, err := service.Index(context.Background(), ShadowIndexRequest{
		Vector:  []float64{0.1, 0.2, 0.3},
		Payload: validPayload(),
	})
	if !errors.Is(err, ErrInvalidVectorDimensions) {
		t.Fatalf("expected dimension rejection, got %v", err)
	}
}

func TestShadowIndexDoesNotProvideRetrievalOrEmbeddingExecution(t *testing.T) {
	service := NewShadowIndexService(&fakeStore{}, ShadowIndexConfig{Enabled: true})
	if service.CanExecuteRetrieval() {
		t.Fatalf("shadow index must not execute retrieval")
	}
	if service.CanCreateEmbeddings() {
		t.Fatalf("shadow index must not create embeddings")
	}
}

func TestShadowIndexErrorsAreIsolatedFromStore(t *testing.T) {
	store := &fakeStore{err: errors.New("qdrant unavailable")}
	service := NewShadowIndexService(store, ShadowIndexConfig{Enabled: true, Collection: "forge_shadow_embeddings"})
	_, err := service.Index(context.Background(), ShadowIndexRequest{
		Vector:  []float64{0.1, 0.2, 0.3},
		Payload: validPayload(),
	})
	if err == nil || err.Error() != "qdrant unavailable" {
		t.Fatalf("expected store error to return clearly, got %v", err)
	}
}
