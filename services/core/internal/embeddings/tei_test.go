package embeddings

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	_ "modernc.org/sqlite"
)

func TestTEIUnavailableIsDegraded(t *testing.T) {
	svc := New(testSettingsDB(t))
	_, err := svc.db.Exec(`INSERT INTO settings(key,value) VALUES('embedding_provider','tei')`)
	if err != nil {
		t.Fatalf("seed provider: %v", err)
	}
	health := svc.ProviderHealth(context.Background(), "", "")
	if health.Healthy || health.State != "degraded" {
		t.Fatalf("expected degraded TEI health when endpoint missing, got %+v", health)
	}
}

func TestTEIEmbedTexts(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/health":
			w.WriteHeader(http.StatusOK)
		case "/embed":
			var req struct {
				Inputs []string `json:"inputs"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("decode request: %v", err)
			}
			if len(req.Inputs) != 2 {
				t.Fatalf("expected batched texts, got %+v", req)
			}
			_ = json.NewEncoder(w).Encode([][]float64{{1, 0}, {0, 1}})
		default:
			http.NotFound(w, r)
		}
	}))
	defer ts.Close()

	db := testSettingsDB(t)
	mustSetting(t, db, "embedding_provider", "tei")
	mustSetting(t, db, "embedding_tei_endpoint", ts.URL)
	svc := New(db)

	health := svc.ProviderHealth(context.Background(), "", "")
	if !health.Healthy {
		t.Fatalf("expected healthy TEI preflight, got %+v", health)
	}
	vectors, err := svc.EmbedTexts(context.Background(), []string{"a", "b"}, "", "", true)
	if err != nil {
		t.Fatalf("embed texts: %v", err)
	}
	if len(vectors) != 2 || len(vectors[0]) != 2 || vectors[0][0] != 1 || vectors[1][1] != 1 {
		t.Fatalf("unexpected vectors: %+v", vectors)
	}
}

func TestTEIEmbedTextsRejectsOversizeResponse(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/embed" {
			w.WriteHeader(http.StatusOK)
			return
		}
		_, _ = w.Write([]byte(strings.Repeat(" ", teiEmbeddingResponseLimit+1)))
	}))
	defer ts.Close()

	provider := &teiProvider{
		client:   ts.Client(),
		endpoint: ts.URL,
		model:    "tei-test",
	}
	_, err := provider.EmbedTexts(context.Background(), []string{"a"})
	if err == nil {
		t.Fatalf("expected oversize TEI embedding response to be rejected")
	}
	if !strings.Contains(err.Error(), "too large") {
		t.Fatalf("expected too-large error, got %v", err)
	}
}

func TestOllamaEmbedRejectsOversizeResponse(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/embeddings" {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte(`{"embedding":[1,0]}`))
		_, _ = w.Write([]byte(strings.Repeat(" ", teiEmbeddingResponseLimit+1)))
	}))
	defer ts.Close()

	provider := &ollamaProvider{
		client:  ts.Client(),
		baseURL: ts.URL,
		model:   "ollama-embed-test",
	}
	_, err := provider.Embed(context.Background(), "a")
	if err == nil {
		t.Fatalf("expected oversize Ollama embedding response to be rejected")
	}
	if !strings.Contains(err.Error(), "too large") {
		t.Fatalf("expected too-large error, got %v", err)
	}
}

func TestEmbeddingProviderDoesNotMutateCanonicalTruth(t *testing.T) {
	db := testSettingsDB(t)
	svc := New(db)
	before := canonicalTruthCount(t, db)
	if _, err := svc.EmbedTexts(context.Background(), []string{"truth remains syscall-owned"}, ProviderLocalHash, "", false); err != nil {
		t.Fatalf("embed texts: %v", err)
	}
	if after := canonicalTruthCount(t, db); after != before {
		t.Fatalf("embedding call mutated canonical truth tables: before=%d after=%d", before, after)
	}
}

func testSettingsDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if _, err := db.Exec(`CREATE TABLE settings(key TEXT PRIMARY KEY, value TEXT NOT NULL);
CREATE TABLE memory_notes(id TEXT PRIMARY KEY);
CREATE TABLE state_items(id TEXT PRIMARY KEY);
CREATE TABLE open_loops(id TEXT PRIMARY KEY);
CREATE TABLE journal_events(id TEXT PRIMARY KEY);`); err != nil {
		t.Fatalf("create tables: %v", err)
	}
	return db
}

func mustSetting(t *testing.T, db *sql.DB, key, value string) {
	t.Helper()
	if _, err := db.Exec(`INSERT INTO settings(key,value) VALUES(?,?)`, key, value); err != nil {
		t.Fatalf("insert setting %s: %v", key, err)
	}
}

func canonicalTruthCount(t *testing.T, db *sql.DB) int {
	t.Helper()
	total := 0
	for _, table := range []string{"memory_notes", "state_items", "open_loops", "journal_events"} {
		var count int
		if err := db.QueryRow(`SELECT COUNT(*) FROM ` + table).Scan(&count); err != nil {
			t.Fatalf("count %s: %v", table, err)
		}
		total += count
	}
	return total
}
