package artifacts

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

type Artifact struct {
	ID          int64           `json:"id"`
	CreatedAtMs int64           `json:"createdAtMs"`
	JobID       *string         `json:"jobId"`
	PacketID    *int64          `json:"packetId"`
	Type        string          `json:"type"`
	Title       string          `json:"title"`
	FilePath    string          `json:"filePath"`
	MimeType    string          `json:"mimeType"`
	Metadata    json.RawMessage `json:"metadata"`
}

type CreateTextArtifactRequest struct {
	JobID    *string
	PacketID *int64
	Type     string
	Title    string
	FileName string
	Subdir   string
	Content  string
	MimeType string
	Metadata map[string]any
}

type CreateFileArtifactRequest struct {
	JobID      *string
	PacketID   *int64
	Type       string
	Title      string
	FileName   string
	Subdir     string
	Reader     io.Reader
	MimeType   string
	Metadata   map[string]any
	MaxBytes   int64
	DefaultExt string
}

type Service struct {
	db      *sql.DB
	baseDir string
}

const maxArtifactTextReadBytes = 2 << 20

func New(db *sql.DB, dataDir string) *Service {
	return &Service{db: db, baseDir: filepath.Join(dataDir, "artifacts")}
}

func (s *Service) ensureDir(dir string) error {
	return os.MkdirAll(dir, 0o755)
}

var cleanNameRE = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)

func safeName(in string) string {
	v := strings.TrimSpace(in)
	if v == "" {
		return "artifact"
	}
	v = strings.ReplaceAll(v, " ", "-")
	v = cleanNameRE.ReplaceAllString(v, "-")
	v = strings.Trim(v, "-.")
	if v == "" {
		return "artifact"
	}
	return strings.ToLower(v)
}

func (s *Service) resolveArtifactWritePath(subdir, fileName, defaultExt string) (string, error) {
	sub := safeName(subdir)
	if sub == "artifact" {
		sub = time.Now().UTC().Format("20060102")
	}
	dir := filepath.Join(s.baseDir, sub)
	if err := s.ensureDir(dir); err != nil {
		return "", fmt.Errorf("mkdir artifact dir: %w", err)
	}
	name := safeName(fileName)
	if filepath.Ext(name) == "" {
		ext := strings.TrimSpace(defaultExt)
		if ext == "" {
			ext = ".txt"
		}
		if !strings.HasPrefix(ext, ".") {
			ext = "." + ext
		}
		name += ext
	}
	return filepath.Join(dir, name), nil
}

func (s *Service) insertArtifactRow(ctx context.Context, now int64, reqType, title, path, mime string, jobID *string, packetID *int64, metadata map[string]any) (Artifact, error) {
	meta := metadata
	if meta == nil {
		meta = map[string]any{}
	}
	metaBytes, _ := json.Marshal(meta)

	res, err := s.db.ExecContext(ctx,
		`INSERT INTO artifacts(created_at, job_id, packet_id, type, title, file_path, mime_type, metadata_json)
		 VALUES(?,?,?,?,?,?,?,?)`,
		now, jobID, packetID, reqType, title, path, mime, string(metaBytes),
	)
	if err != nil {
		return Artifact{}, fmt.Errorf("insert artifact: %w", err)
	}
	id, _ := res.LastInsertId()
	return Artifact{
		ID:          id,
		CreatedAtMs: now,
		JobID:       jobID,
		PacketID:    packetID,
		Type:        reqType,
		Title:       title,
		FilePath:    path,
		MimeType:    mime,
		Metadata:    metaBytes,
	}, nil
}

func (s *Service) CreateTextArtifact(ctx context.Context, req CreateTextArtifactRequest) (Artifact, error) {
	now := time.Now().UnixMilli()
	path, err := s.resolveArtifactWritePath(req.Subdir, req.FileName, ".txt")
	if err != nil {
		return Artifact{}, err
	}
	if err := os.WriteFile(path, []byte(req.Content), 0o644); err != nil {
		return Artifact{}, fmt.Errorf("write artifact file: %w", err)
	}
	return s.insertArtifactRow(ctx, now, req.Type, req.Title, path, req.MimeType, req.JobID, req.PacketID, req.Metadata)
}

func (s *Service) CreateFileArtifact(ctx context.Context, req CreateFileArtifactRequest) (Artifact, int64, error) {
	if req.Reader == nil {
		return Artifact{}, 0, fmt.Errorf("reader required")
	}
	maxBytes := req.MaxBytes
	if maxBytes <= 0 {
		maxBytes = 20 << 20 // 20MB
	}
	now := time.Now().UnixMilli()
	path, err := s.resolveArtifactWritePath(req.Subdir, req.FileName, req.DefaultExt)
	if err != nil {
		return Artifact{}, 0, err
	}
	f, err := os.Create(path)
	if err != nil {
		return Artifact{}, 0, fmt.Errorf("create artifact file: %w", err)
	}
	defer f.Close()

	written, err := io.Copy(f, io.LimitReader(req.Reader, maxBytes+1))
	if err != nil {
		return Artifact{}, 0, fmt.Errorf("write artifact file: %w", err)
	}
	if written > maxBytes {
		_ = os.Remove(path)
		return Artifact{}, 0, fmt.Errorf("file exceeds %d bytes", maxBytes)
	}
	art, err := s.insertArtifactRow(ctx, now, req.Type, req.Title, path, req.MimeType, req.JobID, req.PacketID, req.Metadata)
	if err != nil {
		_ = os.Remove(path)
		return Artifact{}, 0, err
	}
	return art, written, nil
}

func (s *Service) ListByJob(ctx context.Context, jobID string) ([]Artifact, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, created_at, job_id, packet_id, type, title, file_path, mime_type, metadata_json
		 FROM artifacts WHERE job_id = ? ORDER BY id ASC`, jobID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Artifact{}
	for rows.Next() {
		var a Artifact
		var jobIDVal sql.NullString
		var packetIDVal sql.NullInt64
		var meta string
		if err := rows.Scan(&a.ID, &a.CreatedAtMs, &jobIDVal, &packetIDVal, &a.Type, &a.Title, &a.FilePath, &a.MimeType, &meta); err != nil {
			return nil, err
		}
		if jobIDVal.Valid {
			v := jobIDVal.String
			a.JobID = &v
		}
		if packetIDVal.Valid {
			v := packetIDVal.Int64
			a.PacketID = &v
		}
		a.Metadata = json.RawMessage(meta)
		out = append(out, a)
	}
	return out, rows.Err()
}

func (s *Service) ReadFile(path string) (string, error) {
	b, err := readBoundedArtifactText(path)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func (s *Service) resolveSafePath(p string) (string, error) {
	abs, err := filepath.Abs(p)
	if err != nil {
		return "", err
	}
	base, err := filepath.Abs(s.baseDir)
	if err != nil {
		return "", err
	}
	abs = filepath.Clean(abs)
	base = filepath.Clean(base)
	if abs == base || strings.HasPrefix(abs, base+string(filepath.Separator)) {
		return abs, nil
	}
	return "", fmt.Errorf("path outside artifact storage")
}

func (s *Service) GetByID(ctx context.Context, id int64) (*Artifact, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, created_at, job_id, packet_id, type, title, file_path, mime_type, metadata_json FROM artifacts WHERE id = ?`, id)
	var a Artifact
	var jobIDVal sql.NullString
	var packetIDVal sql.NullInt64
	var meta string
	if err := row.Scan(&a.ID, &a.CreatedAtMs, &jobIDVal, &packetIDVal, &a.Type, &a.Title, &a.FilePath, &a.MimeType, &meta); err != nil {
		return nil, err
	}
	if jobIDVal.Valid {
		v := jobIDVal.String
		a.JobID = &v
	}
	if packetIDVal.Valid {
		v := packetIDVal.Int64
		a.PacketID = &v
	}
	a.Metadata = json.RawMessage(meta)
	return &a, nil
}

// List returns recent artifacts, optionally filtered by job id.
func (s *Service) List(ctx context.Context, jobID *string, limit int) ([]Artifact, error) {
	if limit <= 0 || limit > 500 {
		limit = 200
	}
	q := `SELECT id, created_at, job_id, packet_id, type, title, file_path, mime_type, metadata_json FROM artifacts`
	args := []any{}
	if jobID != nil && strings.TrimSpace(*jobID) != "" {
		q += ` WHERE job_id = ?`
		args = append(args, strings.TrimSpace(*jobID))
	}
	q += ` ORDER BY id DESC LIMIT ?`
	args = append(args, limit)
	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Artifact{}
	for rows.Next() {
		var a Artifact
		var jobIDVal sql.NullString
		var packetIDVal sql.NullInt64
		var meta string
		if err := rows.Scan(&a.ID, &a.CreatedAtMs, &jobIDVal, &packetIDVal, &a.Type, &a.Title, &a.FilePath, &a.MimeType, &meta); err != nil {
			return nil, err
		}
		if jobIDVal.Valid {
			v := jobIDVal.String
			a.JobID = &v
		}
		if packetIDVal.Valid {
			v := packetIDVal.Int64
			a.PacketID = &v
		}
		a.Metadata = json.RawMessage(meta)
		out = append(out, a)
	}
	return out, rows.Err()
}

// ReadArtifactText returns file contents when path is under artifact base and MIME looks textual.
func (s *Service) ReadArtifactText(ctx context.Context, id int64) (string, *Artifact, bool, error) {
	a, err := s.GetByID(ctx, id)
	if err != nil {
		return "", nil, false, err
	}
	safe, err := s.resolveSafePath(a.FilePath)
	if err != nil {
		return "", a, false, err
	}
	mt := strings.ToLower(strings.TrimSpace(a.MimeType))
	textual := strings.HasPrefix(mt, "text/") || mt == "application/json" || mt == "application/x-json" ||
		strings.HasSuffix(strings.ToLower(safe), ".md") || strings.HasSuffix(strings.ToLower(safe), ".json") ||
		strings.HasSuffix(strings.ToLower(safe), ".txt") || strings.HasSuffix(strings.ToLower(safe), ".go") ||
		strings.HasSuffix(strings.ToLower(safe), ".ts") || strings.HasSuffix(strings.ToLower(safe), ".tsx")
	if !textual {
		return "", a, false, nil
	}
	body, err := readBoundedArtifactText(safe)
	if err != nil {
		return "", a, true, err
	}
	return string(body), a, true, nil
}

func readBoundedArtifactText(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	if info, err := f.Stat(); err == nil && info.Size() > maxArtifactTextReadBytes {
		return nil, fmt.Errorf("artifact text too large: %d bytes exceeds %d byte limit", info.Size(), maxArtifactTextReadBytes)
	}
	body, err := io.ReadAll(io.LimitReader(f, maxArtifactTextReadBytes+1))
	if err != nil {
		return nil, err
	}
	if len(body) > maxArtifactTextReadBytes {
		return nil, fmt.Errorf("artifact text too large: exceeds %d byte limit", maxArtifactTextReadBytes)
	}
	return body, nil
}
