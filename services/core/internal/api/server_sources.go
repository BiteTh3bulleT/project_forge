package api

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"forge/projectforge/services/core/internal/permissions"
)

func (s *Server) handleListSources(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	rows, err := s.st.DB.QueryContext(ctx, `
SELECT id, path, created_at, last_scan_started_at, last_scan_completed_at, last_error
FROM sources ORDER BY id`)
	if err != nil {
		writeAPIRequestError(w, http.StatusInternalServerError, err)
		return
	}
	defer rows.Close()
	type src struct {
		ID                  int64   `json:"id"`
		Path                string  `json:"path"`
		CreatedAtMs         int64   `json:"createdAtMs"`
		LastScanStartedMs   *int64  `json:"lastScanStartedMs"`
		LastScanCompletedMs *int64  `json:"lastScanCompletedMs"`
		LastError           *string `json:"lastError"`
	}
	var out []src
	for rows.Next() {
		var srow src
		var ls, lc sql.NullInt64
		var le sql.NullString
		if err := rows.Scan(&srow.ID, &srow.Path, &srow.CreatedAtMs, &ls, &lc, &le); err != nil {
			writeAPIRequestError(w, http.StatusInternalServerError, err)
			return
		}
		if ls.Valid {
			v := ls.Int64
			srow.LastScanStartedMs = &v
		}
		if lc.Valid {
			v := lc.Int64
			srow.LastScanCompletedMs = &v
		}
		if le.Valid {
			v := le.String
			srow.LastError = &v
		}
		out = append(out, srow)
	}
	writeJSON(w, http.StatusOK, map[string]any{"sources": out})
}

type addSourceBody struct {
	Path string `json:"path"`
}

func canonicalExistingDir(path string) (string, error) {
	p := strings.TrimSpace(path)
	if p == "" {
		return "", fmt.Errorf("path required")
	}
	abs, err := filepath.Abs(p)
	if err != nil {
		return "", err
	}
	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return "", err
	}
	resolved = filepath.Clean(resolved)
	fi, err := os.Stat(resolved)
	if err != nil || !fi.IsDir() {
		return "", fmt.Errorf("not a directory")
	}
	if isFilesystemRoot(resolved) {
		return "", fmt.Errorf("filesystem root cannot be indexed as a source")
	}
	return resolved, nil
}

func isFilesystemRoot(path string) bool {
	clean := filepath.Clean(path)
	parent := filepath.Dir(clean)
	return clean == parent
}

func (s *Server) admittedSourcePath(ctx context.Context, raw string) (string, int, string) {
	resolved, err := canonicalExistingDir(raw)
	if err != nil {
		if err.Error() == "path required" || err.Error() == "not a directory" {
			return "", http.StatusBadRequest, err.Error()
		}
		return "", http.StatusBadRequest, err.Error()
	}
	decision, _, err := s.permissions.Check(ctx, permissions.CheckRequest{
		ToolID:    "fs.read",
		Action:    "source.index",
		Paths:     []string{resolved},
		Reads:     true,
		RiskClass: "low",
	})
	if err != nil {
		return "", http.StatusInternalServerError, err.Error()
	}
	if decision == nil || !decision.Allowed {
		reason := "source path denied by active permission profile"
		if decision != nil && strings.TrimSpace(decision.Reason) != "" {
			reason = decision.Reason
		}
		return "", http.StatusForbidden, reason
	}
	if decision.RequiresApproval {
		reason := strings.TrimSpace(decision.Reason)
		if reason == "" {
			reason = "source path outside active read scope"
		}
		return "", http.StatusForbidden, "source path outside active read scope: " + reason
	}
	return resolved, http.StatusOK, ""
}

func (s *Server) handleAddSource(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var body addSourceBody
	if err := decodeServerJSONBody(r, &body); err != nil {
		writeServerDecodeError(w, err)
		return
	}
	abs, status, reason := s.admittedSourcePath(ctx, body.Path)
	if status != http.StatusOK {
		http.Error(w, reason, status)
		return
	}
	res, err := s.st.DB.ExecContext(ctx,
		`INSERT INTO sources (path, created_at) VALUES (?, ?)`,
		abs, time.Now().UnixMilli(),
	)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			http.Error(w, "source already exists", http.StatusConflict)
			return
		}
		writeAPIRequestError(w, http.StatusInternalServerError, err)
		return
	}
	id, _ := res.LastInsertId()
	_ = s.log.Emit(ctx, "source.added", map[string]any{"sourceId": id, "path": abs})
	if err := s.ingest.IndexSource(ctx, id, abs); err != nil {
		_ = s.log.Emit(ctx, "error.raised", map[string]any{"where": "ingest after add", "message": err.Error(), "sourceId": id})
	} else {
		s.runLibrarianAfterSourceIndex(ctx, id, abs)
	}
	if s.watch != nil {
		_ = s.watch.SyncSources(ctx, listSourcePaths(s.st.DB))
	}
	writeJSON(w, http.StatusOK, map[string]any{"id": id, "path": abs})
}

func (s *Server) handleDeleteSource(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "bad id", http.StatusBadRequest)
		return
	}
	if _, err := s.st.DB.ExecContext(ctx, `DELETE FROM sources WHERE id = ?`, id); err != nil {
		writeAPIRequestError(w, http.StatusInternalServerError, err)
		return
	}
	_ = s.log.Emit(ctx, "command.executed", map[string]any{"command": "source.delete", "sourceId": id})
	if s.watch != nil {
		_ = s.watch.SyncSources(ctx, listSourcePaths(s.st.DB))
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleReindex(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	q := r.URL.Query().Get("sourceId")
	_ = s.log.Emit(ctx, "command.executed", map[string]any{"command": "reindex", "sourceId": q})
	if q == "" {
		if err := s.ingest.IndexAllSources(ctx); err != nil {
			writeAPIRequestError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "scope": "all"})
		return
	}
	id, err := strconv.ParseInt(q, 10, 64)
	if err != nil {
		http.Error(w, "bad sourceId", http.StatusBadRequest)
		return
	}
	var path string
	if err := s.st.DB.QueryRowContext(ctx, `SELECT path FROM sources WHERE id = ?`, id).Scan(&path); err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if err := s.ingest.IndexSource(ctx, id, path); err != nil {
		writeAPIRequestError(w, http.StatusInternalServerError, err)
		return
	}
	s.runLibrarianAfterSourceIndex(ctx, id, path)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "scope": "one", "sourceId": id})
}

// runLibrarianAfterSourceIndex fires the librarian ingest pipeline (dry-run)
// for a source folder that has just been indexed. Errors and panics are logged
// and swallowed: the source index itself has already succeeded, and the
// librarian path is non-mutating diagnostic infrastructure that must never
// fail the user-facing source operation.
func (s *Server) runLibrarianAfterSourceIndex(ctx context.Context, sourceID int64, sourcePath string) {
	if s == nil || s.autonomy == nil || !s.autonomy.LibrarianPipelineConfigured() {
		return
	}
	defer func() {
		if r := recover(); r != nil {
			_ = s.log.Emit(ctx, "error.raised", map[string]any{
				"where":      "librarian after source index",
				"message":    fmt.Sprintf("panic: %v", r),
				"sourceId":   sourceID,
				"sourcePath": sourcePath,
			})
		}
	}()
	if _, err := s.autonomy.RunSourceIndexIngest(ctx, sourceID, sourcePath); err != nil {
		_ = s.log.Emit(ctx, "error.raised", map[string]any{
			"where":      "librarian after source index",
			"message":    err.Error(),
			"sourceId":   sourceID,
			"sourcePath": sourcePath,
		})
	}
}

func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	hits, err := s.search.Search(ctx, q, limit)
	if err != nil {
		_ = s.log.Emit(ctx, "error.raised", map[string]any{"where": "search", "message": err.Error()})
		writeAPIRequestError(w, http.StatusBadRequest, err)
		return
	}
	_ = s.log.Emit(ctx, "search.executed", map[string]any{"q": q, "hits": len(hits)})
	writeJSON(w, http.StatusOK, map[string]any{"hits": hits})
}

func (s *Server) handleChunk(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "bad id", http.StatusBadRequest)
		return
	}
	h, err := s.search.ChunkByID(ctx, id)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, h)
}
