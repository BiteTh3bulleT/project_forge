package search

import (
	"context"
	"database/sql"
	"reflect"
	"strings"
	"testing"

	_ "modernc.org/sqlite"
)

func TestFTSMatchQuery(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{
			name: "empty query",
			raw:  " \t\n ",
			want: "",
		},
		{
			name: "trims and joins terms",
			raw:  "  alpha   beta\tgamma  ",
			want: `"alpha" AND "beta" AND "gamma"`,
		},
		{
			name: "removes fts operator characters",
			raw:  `"alpha" (beta)^ gamma:delta`,
			want: `"alpha" AND "beta" AND "gamma delta"`,
		},
		{
			name: "drops tokens made only of operators",
			raw:  `* "()" useful`,
			want: `"useful"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ftsMatchQuery(tt.raw); got != tt.want {
				t.Fatalf("ftsMatchQuery(%q) = %q, want %q", tt.raw, got, tt.want)
			}
		})
	}
}

func TestPlaceholders(t *testing.T) {
	tests := []struct {
		name string
		n    int
		want string
	}{
		{name: "zero", n: 0, want: ""},
		{name: "negative", n: -2, want: ""},
		{name: "one", n: 1, want: "?"},
		{name: "many", n: 3, want: "?,?,?"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := placeholders(tt.n); got != tt.want {
				t.Fatalf("placeholders(%d) = %q, want %q", tt.n, got, tt.want)
			}
		})
	}
}

func TestSearchScoped(t *testing.T) {
	ctx := context.Background()
	db := newSearchTestDB(t)
	svc := New(db)

	tests := []struct {
		name      string
		query     string
		limit     int
		sourceIDs []int64
		wantPaths []string
	}{
		{
			name:      "empty query returns no hits",
			query:     "  ",
			limit:     10,
			sourceIDs: nil,
			wantPaths: nil,
		},
		{
			name:      "matches all sources by default",
			query:     "alpha",
			limit:     10,
			sourceIDs: nil,
			wantPaths: []string{"src1/a.txt", "src2/c.txt"},
		},
		{
			name:      "filters by source ids",
			query:     "alpha",
			limit:     10,
			sourceIDs: []int64{2},
			wantPaths: []string{"src2/c.txt"},
		},
		{
			name:      "applies explicit limit",
			query:     "alpha",
			limit:     1,
			sourceIDs: nil,
			wantPaths: []string{"src1/a.txt"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hits, err := svc.SearchScoped(ctx, tt.query, tt.limit, tt.sourceIDs)
			if err != nil {
				t.Fatalf("SearchScoped returned error: %v", err)
			}
			gotPaths := make([]string, 0, len(hits))
			for _, hit := range hits {
				gotPaths = append(gotPaths, hit.RelPath)
				if hit.ContentLen != len(hit.Content) {
					t.Fatalf("hit %d ContentLen = %d, want %d", hit.ChunkID, hit.ContentLen, len(hit.Content))
				}
				if hit.Snippet == "" || !strings.Contains(hit.Snippet, "alpha") {
					t.Fatalf("hit %d snippet = %q, want highlighted alpha snippet", hit.ChunkID, hit.Snippet)
				}
			}
			if len(gotPaths) == 0 {
				gotPaths = nil
			}
			if !reflect.DeepEqual(gotPaths, tt.wantPaths) {
				t.Fatalf("SearchScoped paths = %#v, want %#v", gotPaths, tt.wantPaths)
			}
		})
	}
}

func TestChunkByID(t *testing.T) {
	ctx := context.Background()
	db := newSearchTestDB(t)
	svc := New(db)

	hit, err := svc.ChunkByID(ctx, 2)
	if err != nil {
		t.Fatalf("ChunkByID returned error: %v", err)
	}

	if hit.ChunkID != 2 || hit.FileID != 1 || hit.SourceID != 1 {
		t.Fatalf("unexpected identifiers: chunk=%d file=%d source=%d", hit.ChunkID, hit.FileID, hit.SourceID)
	}
	if hit.ChunkIdx != 1 {
		t.Fatalf("ChunkIdx = %d, want 1", hit.ChunkIdx)
	}
	if hit.RelPath != "src1/a.txt" || hit.AbsPath != "/workspace/src1/a.txt" {
		t.Fatalf("paths = (%q, %q), want fixture paths", hit.RelPath, hit.AbsPath)
	}
	if hit.Content != "beta only" || hit.ContentLen != len("beta only") {
		t.Fatalf("content = %q len=%d, want beta fixture", hit.Content, hit.ContentLen)
	}
}

func newSearchTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	schema := []string{
		`CREATE TABLE sources (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			path TEXT NOT NULL UNIQUE,
			created_at INTEGER NOT NULL
		)`,
		`CREATE TABLE files (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			source_id INTEGER NOT NULL REFERENCES sources(id) ON DELETE CASCADE,
			rel_path TEXT NOT NULL,
			abs_path TEXT NOT NULL,
			size_bytes INTEGER NOT NULL,
			mtime_ns INTEGER NOT NULL,
			content_sha256 TEXT NOT NULL,
			indexed_at INTEGER NOT NULL,
			UNIQUE(source_id, rel_path)
		)`,
		`CREATE TABLE chunks (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			file_id INTEGER NOT NULL REFERENCES files(id) ON DELETE CASCADE,
			chunk_index INTEGER NOT NULL,
			content TEXT NOT NULL,
			UNIQUE(file_id, chunk_index)
		)`,
		`CREATE VIRTUAL TABLE chunks_fts USING fts5(
			content,
			content='chunks',
			content_rowid='id',
			tokenize = 'porter unicode61'
		)`,
		`CREATE TRIGGER chunks_ai AFTER INSERT ON chunks BEGIN
			INSERT INTO chunks_fts(rowid, content) VALUES (new.id, new.content);
		END`,
	}
	for _, stmt := range schema {
		if _, err := db.Exec(stmt); err != nil {
			t.Fatalf("create fixture schema: %v", err)
		}
	}

	fixtures := []string{
		`INSERT INTO sources(id, path, created_at) VALUES
			(1, '/workspace/src1', 1000),
			(2, '/workspace/src2', 1000)`,
		`INSERT INTO files(id, source_id, rel_path, abs_path, size_bytes, mtime_ns, content_sha256, indexed_at) VALUES
			(1, 1, 'src1/a.txt', '/workspace/src1/a.txt', 10, 100, 'sha-a', 1000),
			(2, 1, 'src1/b.txt', '/workspace/src1/b.txt', 10, 200, 'sha-b', 1000),
			(3, 2, 'src2/c.txt', '/workspace/src2/c.txt', 10, 300, 'sha-c', 1000)`,
		`INSERT INTO chunks(id, file_id, chunk_index, content) VALUES
			(1, 1, 0, 'alpha source one'),
			(2, 1, 1, 'beta only'),
			(3, 2, 0, 'gamma text'),
			(4, 3, 0, 'alpha source two')`,
	}
	for _, stmt := range fixtures {
		if _, err := db.Exec(stmt); err != nil {
			t.Fatalf("insert fixture data: %v", err)
		}
	}

	return db
}
