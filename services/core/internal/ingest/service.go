package ingest

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"forge/projectforge/services/core/internal/events"
)

type Service struct {
	db        *sql.DB
	log       *events.Logger
	extCSV    string
	rootScope string
}

const maxIngestFileBytes = 8 << 20

func New(db *sql.DB, log *events.Logger, extCSV string) *Service {
	if strings.TrimSpace(extCSV) == "" {
		extCSV = DefaultExtensionsCSV()
	}
	return &Service{db: db, log: log, extCSV: extCSV}
}

func (s *Service) ExtensionsMap() map[string]struct{} {
	return ParseExtensionList(s.extCSV)
}

func (s *Service) SetExtensionsCSV(csv string) {
	if strings.TrimSpace(csv) == "" {
		csv = DefaultExtensionsCSV()
	}
	s.extCSV = csv
}

func (s *Service) ExtensionsCSV() string { return s.extCSV }

func (s *Service) SetRootScope(root string) {
	s.rootScope = strings.TrimSpace(root)
}

func fileHash(path string) (string, int64, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return "", 0, err
	}
	f, err := os.Open(path)
	if err != nil {
		return "", 0, err
	}
	defer f.Close()
	h := sha256.New()
	n, err := io.Copy(h, f)
	if err != nil {
		return "", 0, err
	}
	if n != fi.Size() {
		return "", 0, fmt.Errorf("size mismatch while hashing")
	}
	return hex.EncodeToString(h.Sum(nil)), fi.ModTime().UnixNano(), nil
}

func (s *Service) upsertFile(ctx context.Context, sourceID int64, abs, rel string) (indexed bool, reindexed bool, err error) {
	fi, err := os.Stat(abs)
	if err != nil {
		return false, false, nil
	}
	size := fi.Size()
	if size > maxIngestFileBytes {
		_ = s.clearIndexedFile(ctx, sourceID, rel)
		_ = s.log.Emit(ctx, "source.scan.skipped_oversize", map[string]any{"sourceId": sourceID, "path": abs, "relPath": rel, "sizeBytes": size, "limitBytes": maxIngestFileBytes})
		return false, false, nil
	}

	hash, mtimeNs, err := fileHash(abs)
	if err != nil {
		_ = s.log.Emit(ctx, "error.raised", map[string]any{"where": "file.hash", "path": abs, "message": err.Error()})
		return false, false, nil
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return false, false, err
	}
	defer func() { _ = tx.Rollback() }()

	var fileID int64
	var oldHash string
	row := tx.QueryRowContext(ctx,
		`SELECT id, content_sha256 FROM files WHERE source_id = ? AND rel_path = ?`, sourceID, rel)
	switch err := row.Scan(&fileID, &oldHash); err {
	case sql.ErrNoRows:
		res, err := tx.ExecContext(ctx,
			`INSERT INTO files (source_id, rel_path, abs_path, size_bytes, mtime_ns, content_sha256, indexed_at)
			 VALUES (?,?,?,?,?,?,?)`,
			sourceID, rel, abs, size, mtimeNs, hash, time.Now().UnixMilli(),
		)
		if err != nil {
			return false, false, err
		}
		id, _ := res.LastInsertId()
		fileID = id
		body, err := readIngestFile(abs)
		if err != nil {
			return false, false, err
		}
		parts := ChunkText(string(body))
		for i, c := range parts {
			if _, err := tx.ExecContext(ctx,
				`INSERT INTO chunks (file_id, chunk_index, content) VALUES (?,?,?)`,
				fileID, i, c,
			); err != nil {
				return false, false, err
			}
		}
		if err := tx.Commit(); err != nil {
			return false, false, err
		}
		_ = s.log.Emit(ctx, "file.ingested", map[string]any{"sourceId": sourceID, "path": abs, "relPath": rel, "chunks": len(parts)})
		return true, false, nil
	case nil:
		if oldHash == hash {
			return false, false, nil
		}
		if _, err := tx.ExecContext(ctx,
			`UPDATE files SET abs_path=?, size_bytes=?, mtime_ns=?, content_sha256=?, indexed_at=? WHERE id=?`,
			abs, size, mtimeNs, hash, time.Now().UnixMilli(), fileID,
		); err != nil {
			return false, false, err
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM chunks WHERE file_id = ?`, fileID); err != nil {
			return false, false, err
		}
		body, err := readIngestFile(abs)
		if err != nil {
			return false, false, err
		}
		parts := ChunkText(string(body))
		for i, c := range parts {
			if _, err := tx.ExecContext(ctx,
				`INSERT INTO chunks (file_id, chunk_index, content) VALUES (?,?,?)`,
				fileID, i, c,
			); err != nil {
				return false, false, err
			}
		}
		if err := tx.Commit(); err != nil {
			return false, false, err
		}
		_ = s.log.Emit(ctx, "file.reindexed", map[string]any{"sourceId": sourceID, "path": abs, "relPath": rel, "chunks": len(parts)})
		return false, true, nil
	default:
		return false, false, err
	}
}

func (s *Service) clearIndexedFile(ctx context.Context, sourceID int64, rel string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, `DELETE FROM chunks WHERE file_id IN (SELECT id FROM files WHERE source_id = ? AND rel_path = ?)`, sourceID, rel); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM files WHERE source_id = ? AND rel_path = ?`, sourceID, rel); err != nil {
		return err
	}
	return tx.Commit()
}

func readIngestFile(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	if info, err := f.Stat(); err == nil && info.Size() > maxIngestFileBytes {
		return nil, fmt.Errorf("ingest file too large: %d bytes exceeds %d byte limit", info.Size(), maxIngestFileBytes)
	}
	body, err := io.ReadAll(io.LimitReader(f, maxIngestFileBytes+1))
	if err != nil {
		return nil, err
	}
	if len(body) > maxIngestFileBytes {
		return nil, fmt.Errorf("ingest file too large: exceeds %d byte limit", maxIngestFileBytes)
	}
	return body, nil
}

func (s *Service) IndexSource(ctx context.Context, sourceID int64, rootPath string) error {
	allowed := s.ExtensionsMap()
	now := time.Now().UnixMilli()
	if _, err := s.db.ExecContext(ctx, `UPDATE sources SET last_scan_started_at = ?, last_error = NULL WHERE id = ?`, now, sourceID); err != nil {
		return err
	}
	if err := s.log.Emit(ctx, "source.scan.started", map[string]any{"sourceId": sourceID, "path": rootPath}); err != nil {
		return err
	}

	resolvedRoot, err := canonicalExistingDir(rootPath)
	if err == nil && strings.TrimSpace(s.rootScope) != "" {
		scope, scopeErr := canonicalExistingScope(s.rootScope)
		if scopeErr != nil {
			err = fmt.Errorf("ingest root scope unavailable: %w", scopeErr)
		} else if !pathWithinScope(scope, resolvedRoot) {
			err = fmt.Errorf("source path outside configured ingest root scope")
		}
	}
	if err != nil {
		msg := err.Error()
		_, _ = s.db.ExecContext(ctx, `UPDATE sources SET last_error = ?, last_scan_completed_at = ? WHERE id = ?`, msg, time.Now().UnixMilli(), sourceID)
		_ = s.log.Emit(ctx, "error.raised", map[string]any{"where": "ingest.IndexSource", "message": msg, "sourceId": sourceID})
		return fmt.Errorf("%s", msg)
	}

	info, err := os.Stat(resolvedRoot)
	if err != nil || !info.IsDir() {
		msg := "path not found or not a directory"
		if err != nil {
			msg = err.Error()
		}
		_, _ = s.db.ExecContext(ctx, `UPDATE sources SET last_error = ?, last_scan_completed_at = ? WHERE id = ?`, msg, time.Now().UnixMilli(), sourceID)
		_ = s.log.Emit(ctx, "error.raised", map[string]any{"where": "ingest.IndexSource", "message": msg, "sourceId": sourceID})
		return fmt.Errorf("%s", msg)
	}

	var walkErr error
	_ = filepath.WalkDir(resolvedRoot, func(abs string, d os.DirEntry, we error) error {
		if we != nil {
			_ = s.log.Emit(ctx, "error.raised", map[string]any{"where": "walk", "path": abs, "message": we.Error()})
			return nil
		}
		if d.Type()&os.ModeSymlink != 0 {
			_ = s.log.Emit(ctx, "source.scan.skipped_symlink", map[string]any{"sourceId": sourceID, "path": abs})
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			base := filepath.Base(abs)
			if base == ".git" || base == "node_modules" || base == "target" || base == "dist" {
				return filepath.SkipDir
			}
			return nil
		}
		if !IsSupportedPath(abs, allowed) {
			return nil
		}
		filePath, err := filepath.EvalSymlinks(abs)
		if err != nil {
			return nil
		}
		filePath = filepath.Clean(filePath)
		if !pathWithinScope(resolvedRoot, filePath) {
			_ = s.log.Emit(ctx, "source.scan.skipped_escape", map[string]any{"sourceId": sourceID, "path": abs})
			return nil
		}
		rel, err := filepath.Rel(resolvedRoot, filePath)
		if err != nil {
			return nil
		}
		rel = filepath.ToSlash(rel)
		if _, _, err := s.upsertFile(ctx, sourceID, filePath, rel); err != nil {
			walkErr = err
			return err
		}
		return nil
	})

	done := time.Now().UnixMilli()
	if walkErr != nil {
		_, _ = s.db.ExecContext(ctx, `UPDATE sources SET last_error = ?, last_scan_completed_at = ? WHERE id = ?`, walkErr.Error(), done, sourceID)
		_ = s.log.Emit(ctx, "error.raised", map[string]any{"where": "ingest.walk", "message": walkErr.Error(), "sourceId": sourceID})
		return walkErr
	}
	if _, err := s.db.ExecContext(ctx, `UPDATE sources SET last_scan_completed_at = ?, last_error = NULL WHERE id = ?`, done, sourceID); err != nil {
		return err
	}
	_ = s.log.Emit(ctx, "source.scan.completed", map[string]any{"sourceId": sourceID, "path": resolvedRoot})
	return nil
}

func canonicalExistingDir(path string) (string, error) {
	resolved, err := canonicalExistingScope(path)
	if err != nil {
		return "", err
	}
	if filepath.Dir(resolved) == resolved {
		return "", fmt.Errorf("filesystem root cannot be indexed as a source")
	}
	return resolved, nil
}

func canonicalExistingScope(path string) (string, error) {
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
	info, err := os.Stat(resolved)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", fmt.Errorf("not a directory")
	}
	return resolved, nil
}

func pathWithinScope(scope, target string) bool {
	cleanScope := filepath.Clean(scope)
	cleanTarget := filepath.Clean(target)
	if cleanTarget == cleanScope {
		return true
	}
	rel, err := filepath.Rel(cleanScope, cleanTarget)
	if err != nil {
		return false
	}
	rel = filepath.Clean(rel)
	return rel != "." && !strings.HasPrefix(rel, "..") && !strings.HasPrefix(rel, string(filepath.Separator))
}

func (s *Service) IndexAllSources(ctx context.Context) error {
	rows, err := s.db.QueryContext(ctx, `SELECT id, path FROM sources ORDER BY id`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		var p string
		if err := rows.Scan(&id, &p); err != nil {
			return err
		}
		_ = s.IndexSource(ctx, id, p)
	}
	return rows.Err()
}
