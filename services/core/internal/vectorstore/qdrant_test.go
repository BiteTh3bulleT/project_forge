package vectorstore

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func TestQdrantClientUpsertUsesSafePayloadOnly(t *testing.T) {
	var captured map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/collections/forge_shadow_embeddings/points" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		_, _ = w.Write([]byte(`{"result":true}`))
	}))
	defer server.Close()

	client, err := NewQdrantClient(server.URL, time.Second)
	if err != nil {
		t.Fatalf("new qdrant client: %v", err)
	}
	if err := client.UpsertVector(context.Background(), VectorPoint{
		Collection: "forge_shadow_embeddings",
		PointID:    DeterministicPointID(validPayload()),
		Vector:     []float64{0.1, 0.2, 0.3},
		Payload:    validPayload(),
	}); err != nil {
		t.Fatalf("upsert vector: %v", err)
	}
	raw, _ := json.Marshal(captured)
	lower := strings.ToLower(string(raw))
	for _, forbidden := range []string{"source_text", "chunk_text", "prompt", "completion", "message_body", "memory_content", "raw_query", "secret"} {
		if strings.Contains(lower, forbidden) {
			t.Fatalf("qdrant upsert body leaked forbidden field %q: %s", forbidden, lower)
		}
	}
	if !strings.Contains(lower, "provenance_ref_id") || !strings.Contains(lower, "embedding_record_id") {
		t.Fatalf("qdrant upsert body missing refs/provenance: %s", lower)
	}
}

func TestQdrantClientEnsureCollection(t *testing.T) {
	var captured map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/collections/forge_shadow_embeddings" {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		_, _ = w.Write([]byte(`{"result":true}`))
	}))
	defer server.Close()

	client, err := NewQdrantClient(server.URL, time.Second)
	if err != nil {
		t.Fatalf("new qdrant client: %v", err)
	}
	if err := client.EnsureCollection(context.Background(), CollectionSpec{Name: "forge_shadow_embeddings", VectorSize: 3}); err != nil {
		t.Fatalf("ensure collection: %v", err)
	}
	vectors, ok := captured["vectors"].(map[string]any)
	if !ok || int(vectors["size"].(float64)) != 3 {
		t.Fatalf("unexpected collection body: %#v", captured)
	}
}

func TestQdrantClientRejectsOversizeSuccessResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/collections/forge_shadow_embeddings/points/search" {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"result":[]}`))
		_, _ = w.Write([]byte(strings.Repeat(" ", qdrantResponseBodyLimit+1)))
	}))
	defer server.Close()

	client, err := NewQdrantClient(server.URL, time.Second)
	if err != nil {
		t.Fatalf("new qdrant client: %v", err)
	}
	_, err = client.SearchVector(context.Background(), SearchRequest{
		Collection: "forge_shadow_embeddings",
		Vector:     []float64{0.1, 0.2, 0.3},
		Limit:      1,
	})
	if err == nil || !strings.Contains(err.Error(), "too large") {
		t.Fatalf("expected oversized response error, got %v", err)
	}
}

func TestQdrantClientErrorResponseIsBoundedAndMarked(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(strings.Repeat("x", qdrantErrorBodyLimit+1)))
	}))
	defer server.Close()

	client, err := NewQdrantClient(server.URL, time.Second)
	if err != nil {
		t.Fatalf("new qdrant client: %v", err)
	}
	err = client.EnsureCollection(context.Background(), CollectionSpec{Name: "forge_shadow_embeddings", VectorSize: 3})
	if err == nil || !strings.Contains(err.Error(), "truncated") {
		t.Fatalf("expected truncated error response, got %v", err)
	}
}

func TestQdrantIntegrationEnvGated(t *testing.T) {
	url := requireIntegrationEnvOrSkip(t, "FORGE_QDRANT_TEST_URL")
	client, err := NewQdrantClient(url, 3*time.Second)
	if err != nil {
		t.Fatalf("new qdrant client: %v", err)
	}
	if err := client.Health(context.Background()); err != nil {
		t.Fatalf("qdrant health: %v", err)
	}
	collection := "forge_shadow_embeddings_test"
	if err := client.EnsureCollection(context.Background(), CollectionSpec{Name: collection, VectorSize: 3}); err != nil {
		t.Fatalf("ensure collection: %v", err)
	}
	payload := validPayload()
	payload.EmbeddingRecordID = "embedding:test"
	pointID := DeterministicPointID(payload)
	if err := client.UpsertVector(context.Background(), VectorPoint{
		Collection: collection,
		PointID:    pointID,
		Vector:     []float64{0.1, 0.2, 0.3},
		Payload:    payload,
	}); err != nil {
		t.Fatalf("upsert vector: %v", err)
	}
	result, err := client.SearchVector(context.Background(), SearchRequest{
		Collection: collection,
		Vector:     []float64{0.1, 0.2, 0.3},
		Limit:      1,
	})
	if err != nil {
		t.Fatalf("search vector: %v", err)
	}
	if len(result.Matches) == 0 {
		t.Fatalf("expected qdrant match")
	}
	if err := client.DeleteVector(context.Background(), collection, pointID); err != nil {
		t.Fatalf("delete vector: %v", err)
	}
}

func requireIntegrationEnvOrSkip(t *testing.T, name string) string {
	t.Helper()
	value := strings.TrimSpace(os.Getenv(name))
	if value != "" {
		return value
	}
	if os.Getenv("CI") != "" || os.Getenv("GITHUB_ACTIONS") != "" {
		t.Fatalf("%s must be set in CI for integration coverage", name)
	}
	t.Skipf("%s not set", name)
	return ""
}
