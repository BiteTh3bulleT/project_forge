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
	db     *sql.DB
	log    *events.Logger
	extCSV string
}

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
	hash, mtimeNs, err := fileHash(abs)
	if err != nil {
		_ = s.log.Emit(ctx, "error.raised", map[string]any{"where": "file.hash", "path": abs, "message": err.Error()})
		return false, false, nil
	}
	fi, err := os.Stat(abs)
	if err != nil {
		return false, false, nil
	}
	size := fi.Size()

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
		body, err := os.ReadFile(abs)
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
		body, err := os.ReadFile(abs)
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

func (s *Service) IndexSource(ctx context.Context, sourceID int64, rootPath string) error {
	allowed := s.ExtensionsMap()
	now := time.Now().UnixMilli()
	if _, err := s.db.ExecContext(ctx, `UPDATE sources SET last_scan_started_at = ?, last_error = NULL WHERE id = ?`, now, sourceID); err != nil {
		return err
	}
	if err := s.log.Emit(ctx, "source.scan.started", map[string]any{"sourceId": sourceID, "path": rootPath}); err != nil {
		return err
	}

	info, err := os.Stat(rootPath)
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
	_ = filepath.WalkDir(rootPath, func(abs string, d os.DirEntry, we error) error {
		if we != nil {
			_ = s.log.Emit(ctx, "error.raised", map[string]any{"where": "walk", "path": abs, "message": we.Error()})
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
		rel, err := filepath.Rel(rootPath, abs)
		if err != nil {
			return nil
		}
		rel = filepath.ToSlash(rel)
		if _, _, err := s.upsertFile(ctx, sourceID, abs, rel); err != nil {
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
	_ = s.log.Emit(ctx, "source.scan.completed", map[string]any{"sourceId": sourceID, "path": rootPath})
	return nil
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
