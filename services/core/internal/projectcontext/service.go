package projectcontext

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"forge/projectforge/services/core/internal/events"
)

const ContextVersion = 1
const maxContextSourceBytes = 8 << 20

type Record struct {
	ID                    int64           `json:"id"`
	ContextVersion        int             `json:"contextVersion"`
	CreatedAtMs           int64           `json:"createdAtMs"`
	GeneratedAtMs         int64           `json:"generatedAtMs"`
	SourcePath            string          `json:"sourcePath"`
	SourceHash            string          `json:"sourceHash"`
	SourceSizeBytes       int64           `json:"sourceSizeBytes"`
	NormalizedSummary     json.RawMessage `json:"normalizedSummary"`
	BriefingMarkdown      string          `json:"briefingMarkdown"`
	AgentsMarkdown        string          `json:"agentsMarkdown"`
	ClaudeMarkdown        string          `json:"claudeMarkdown"`
	CursorMarkdown        string          `json:"cursorMarkdown"`
	GeneratedAgentsPath   string          `json:"generatedAgentsPath"`
	GeneratedClaudePath   string          `json:"generatedClaudePath"`
	GeneratedBriefingPath string          `json:"generatedBriefingPath"`
	GeneratedCursorPath   string          `json:"generatedCursorPath"`
	Notes                 string          `json:"notes"`
}

type ImportRequest struct {
	SourcePath string `json:"sourcePath"`
	Notes      string `json:"notes"`
}

type summary struct {
	Title          string   `json:"title"`
	Headings       []string `json:"headings"`
	KeyPoints      []string `json:"keyPoints"`
	Phase          string   `json:"phase"`
	CoreObjectives []string `json:"coreObjectives"`
	Deferrals      []string `json:"deferrals"`
}

type Service struct {
	db           *sql.DB
	log          *events.Logger
	workspaceDir string
	dataDir      string
}

func New(db *sql.DB, log *events.Logger, workspaceDir, dataDir string) *Service {
	return &Service{db: db, log: log, workspaceDir: workspaceDir, dataDir: dataDir}
}

func (s *Service) Latest(ctx context.Context) (*Record, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT id, context_version, created_at, generated_at, source_path, source_hash, source_size_bytes,
       normalized_summary_json, briefing_markdown, agents_markdown, claude_markdown, cursor_markdown,
       generated_agents_path, generated_claude_path, generated_briefing_path, generated_cursor_path, notes
FROM project_context_records ORDER BY id DESC LIMIT 1`)
	var r Record
	var summaryJSON string
	if err := row.Scan(
		&r.ID,
		&r.ContextVersion,
		&r.CreatedAtMs,
		&r.GeneratedAtMs,
		&r.SourcePath,
		&r.SourceHash,
		&r.SourceSizeBytes,
		&summaryJSON,
		&r.BriefingMarkdown,
		&r.AgentsMarkdown,
		&r.ClaudeMarkdown,
		&r.CursorMarkdown,
		&r.GeneratedAgentsPath,
		&r.GeneratedClaudePath,
		&r.GeneratedBriefingPath,
		&r.GeneratedCursorPath,
		&r.Notes,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	r.NormalizedSummary = json.RawMessage(summaryJSON)
	return &r, nil
}

func (s *Service) ImportAndNormalize(ctx context.Context, req ImportRequest) (*Record, error) {
	sourcePath, err := s.resolveSourcePath(ctx, req.SourcePath)
	if err != nil {
		return nil, err
	}

	body, err := readContextSource(sourcePath)
	if err != nil {
		return nil, fmt.Errorf("read context source: %w", err)
	}
	h := sha256.Sum256(body)
	sourceHash := hex.EncodeToString(h[:])

	sm := parseSummary(string(body))
	summaryJSON, _ := json.Marshal(sm)
	genAt := time.Now()
	generatedAtMs := genAt.UnixMilli()

	docsDir := filepath.Join(s.workspaceDir, "docs")
	cursorDir := filepath.Join(s.workspaceDir, ".cursor", "rules")
	if err := os.MkdirAll(docsDir, 0o755); err != nil {
		return nil, fmt.Errorf("mkdir docs: %w", err)
	}
	if err := os.MkdirAll(cursorDir, 0o755); err != nil {
		return nil, fmt.Errorf("mkdir cursor rules: %w", err)
	}

	briefing := buildBriefingMarkdown(sourcePath, genAt, sm)
	agents := buildAgentsMarkdown(sourcePath, genAt, sm)
	claude := buildClaudeMarkdown(sourcePath, genAt, sm)
	cursor := buildCursorMarkdown(sourcePath, genAt, sm)

	agentsPath := filepath.Join(s.workspaceDir, "AGENTS.md")
	claudePath := filepath.Join(s.workspaceDir, "CLAUDE.md")
	briefingPath := filepath.Join(docsDir, "FORGE_PROJECT_BRIEFING.md")
	cursorPath := filepath.Join(cursorDir, "forge-context.mdc")

	if err := os.WriteFile(agentsPath, []byte(agents), 0o644); err != nil {
		return nil, fmt.Errorf("write AGENTS.md: %w", err)
	}
	if err := os.WriteFile(claudePath, []byte(claude), 0o644); err != nil {
		return nil, fmt.Errorf("write CLAUDE.md: %w", err)
	}
	if err := os.WriteFile(briefingPath, []byte(briefing), 0o644); err != nil {
		return nil, fmt.Errorf("write FORGE briefing: %w", err)
	}
	if err := os.WriteFile(cursorPath, []byte(cursor), 0o644); err != nil {
		return nil, fmt.Errorf("write cursor guidance: %w", err)
	}

	archiveDir := filepath.Join(s.dataDir, "project_context", "imports")
	if err := os.MkdirAll(archiveDir, 0o755); err != nil {
		return nil, fmt.Errorf("mkdir context archive: %w", err)
	}
	archivePath := filepath.Join(archiveDir, fmt.Sprintf("%d-%s.md", generatedAtMs, safeStem(sourcePath)))
	if err := os.WriteFile(archivePath, body, 0o644); err != nil {
		return nil, fmt.Errorf("archive context source: %w", err)
	}

	now := time.Now().UnixMilli()
	res, err := s.db.ExecContext(ctx, `
INSERT INTO project_context_records(
  context_version, created_at, generated_at, source_path, source_hash, source_size_bytes,
  normalized_summary_json, briefing_markdown, agents_markdown, claude_markdown, cursor_markdown,
  generated_agents_path, generated_claude_path, generated_briefing_path, generated_cursor_path, notes
) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		ContextVersion,
		now,
		generatedAtMs,
		sourcePath,
		sourceHash,
		int64(len(body)),
		string(summaryJSON),
		briefing,
		agents,
		claude,
		cursor,
		agentsPath,
		claudePath,
		briefingPath,
		cursorPath,
		strings.TrimSpace(req.Notes),
	)
	if err != nil {
		return nil, fmt.Errorf("insert project context record: %w", err)
	}
	id, _ := res.LastInsertId()

	_ = s.setSetting(ctx, "project_context_source_path", sourcePath)
	_ = s.setSetting(ctx, "project_context_last_archive_path", archivePath)
	_ = s.setSetting(ctx, "project_context_last_record_id", fmt.Sprintf("%d", id))

	_ = s.log.Emit(ctx, "project_context.normalized", map[string]any{
		"recordId": id,
		"source":   sourcePath,
		"hash":     sourceHash,
		"version":  ContextVersion,
		"generated": []string{
			agentsPath,
			claudePath,
			briefingPath,
			cursorPath,
		},
	})

	return s.Latest(ctx)
}

func (s *Service) resolveSourcePath(ctx context.Context, requested string) (string, error) {
	if strings.TrimSpace(requested) != "" {
		abs, err := filepath.Abs(strings.TrimSpace(requested))
		if err != nil {
			return "", err
		}
		if _, err := os.Stat(abs); err != nil {
			return "", fmt.Errorf("context file not found: %s", abs)
		}
		return abs, nil
	}
	if v := s.getSetting(ctx, "project_context_source_path"); strings.TrimSpace(v) != "" {
		if _, err := os.Stat(v); err == nil {
			return v, nil
		}
	}
	candidate := filepath.Join(s.workspaceDir, "FORGE_CONTEXT.md")
	if _, err := os.Stat(candidate); err == nil {
		return candidate, nil
	}
	return "", fmt.Errorf("no context source file found; pass sourcePath or place FORGE_CONTEXT.md in workspace root")
}

func readContextSource(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	if info, err := f.Stat(); err == nil && info.Size() > maxContextSourceBytes {
		return nil, fmt.Errorf("context source too large: %d bytes exceeds %d byte limit", info.Size(), maxContextSourceBytes)
	}
	body, err := io.ReadAll(io.LimitReader(f, maxContextSourceBytes+1))
	if err != nil {
		return nil, err
	}
	if len(body) > maxContextSourceBytes {
		return nil, fmt.Errorf("context source too large: exceeds %d byte limit", maxContextSourceBytes)
	}
	return body, nil
}

func (s *Service) getSetting(ctx context.Context, key string) string {
	var v string
	err := s.db.QueryRowContext(ctx, `SELECT value FROM settings WHERE key = ?`, key).Scan(&v)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(v)
}

func (s *Service) setSetting(ctx context.Context, key, value string) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO settings(key,value) VALUES(?,?) ON CONFLICT(key) DO UPDATE SET value=excluded.value`, key, value)
	return err
}

func parseSummary(raw string) summary {
	lines := strings.Split(strings.ReplaceAll(raw, "\r\n", "\n"), "\n")
	sm := summary{}
	for _, line := range lines {
		t := strings.TrimSpace(line)
		if strings.HasPrefix(t, "# ") && sm.Title == "" {
			sm.Title = strings.TrimSpace(strings.TrimPrefix(t, "# "))
		}
		if strings.HasPrefix(t, "## ") {
			sm.Headings = append(sm.Headings, strings.TrimSpace(strings.TrimPrefix(t, "## ")))
		}
		if strings.HasPrefix(t, "- ") {
			sm.KeyPoints = append(sm.KeyPoints, strings.TrimSpace(strings.TrimPrefix(t, "- ")))
		}
	}
	if sm.Title == "" {
		sm.Title = "FORGE Project Context"
	}
	sm.Phase = pickPhase(lines)
	sm.CoreObjectives = pickByKeywords(sm.KeyPoints, []string{"objective", "local-first", "job", "approval", "adapter", "context"}, 10)
	sm.Deferrals = pickByKeywords(sm.KeyPoints, []string{"defer", "phase 3", "later", "not", "limit"}, 10)
	if len(sm.Headings) > 24 {
		sm.Headings = sm.Headings[:24]
	}
	if len(sm.KeyPoints) > 64 {
		sm.KeyPoints = sm.KeyPoints[:64]
	}
	return sm
}

func pickPhase(lines []string) string {
	for _, line := range lines {
		lt := strings.ToLower(strings.TrimSpace(line))
		if strings.Contains(lt, "phase 2") {
			return "phase_2"
		}
		if strings.Contains(lt, "phase 1") {
			return "phase_1"
		}
	}
	return "unknown"
}

func pickByKeywords(items []string, keywords []string, max int) []string {
	type row struct {
		Text  string
		Score int
	}
	if len(items) == 0 {
		return nil
	}
	rows := make([]row, 0, len(items))
	for _, it := range items {
		trimmed := strings.TrimSpace(it)
		if len(trimmed) < 18 {
			continue
		}
		if len(strings.Fields(trimmed)) < 3 {
			continue
		}
		score := 0
		lit := strings.ToLower(trimmed)
		for _, kw := range keywords {
			if strings.Contains(lit, kw) {
				score++
			}
		}
		if score > 0 {
			rows = append(rows, row{Text: trimmed, Score: score})
		}
	}
	if len(rows) == 0 {
		if len(items) > max {
			return items[:max]
		}
		return items
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Score == rows[j].Score {
			return rows[i].Text < rows[j].Text
		}
		return rows[i].Score > rows[j].Score
	})
	out := make([]string, 0, min(max, len(rows)))
	for _, r := range rows {
		out = append(out, r.Text)
		if len(out) >= max {
			break
		}
	}
	return out
}

func buildBriefingMarkdown(sourcePath string, generatedAt time.Time, sm summary) string {
	return fmt.Sprintf(`# FORGE Project Briefing

Generated: %s  
Source: %s  
Context version: %d  
Phase: %s

## Snapshot
%s

## Core Objectives
%s

## Key Headings
%s

## Deferrals / Limits
%s

## Operational Rule
- Regenerate this briefing when source context changes.
- Treat this as durable handoff evidence for packet generation.
`,
		generatedAt.UTC().Format(time.RFC3339),
		sourcePath,
		ContextVersion,
		sm.Phase,
		bulletOrFallback(sm.Title, "No explicit title extracted."),
		bulletsOrFallback(sm.CoreObjectives, "- Preserve local-first architecture and inspectability."),
		bulletsOrFallback(sm.Headings, "- No headings extracted."),
		bulletsOrFallback(sm.Deferrals, "- No explicit deferrals extracted."),
	)
}

func buildAgentsMarkdown(sourcePath string, generatedAt time.Time, sm summary) string {
	return fmt.Sprintf(`# AGENTS

Generated by FORGE at %s from %s.

## Mission
- Execute bounded work with explicit scope and observable outcomes.
- Respect approval gates for write intent or command execution.

## Project Priorities
%s

## Hard Boundaries
- Separate memory retrieval, reasoning, write proposal, write execution, and command execution.
- Never silently escalate between execution boundaries.
- Attach task packet references and artifact evidence to outputs.

## References
- docs/FORGE_PROJECT_BRIEFING.md
- Source context: %s
`,
		generatedAt.UTC().Format(time.RFC3339),
		sourcePath,
		bulletsOrFallback(sm.CoreObjectives, "- Keep the forge local-first and inspectable."),
		sourcePath,
	)
}

func buildClaudeMarkdown(sourcePath string, generatedAt time.Time, sm summary) string {
	return fmt.Sprintf(`# CLAUDE

Generated by FORGE at %s from %s.

## Role
- Operate as a bounded coding worker.
- Keep decision traces explicit.
- Report assumptions and residual risks.

## Current Focus
%s

## Delivery Contract
- Include path scope, write intent, and expected deliverable in each response.
- Distinguish preparation, execution request, and imported execution result states.
- Do not claim command execution occurred unless artifacts prove it.

## References
- AGENTS.md
- docs/FORGE_PROJECT_BRIEFING.md
`,
		generatedAt.UTC().Format(time.RFC3339),
		sourcePath,
		bulletsOrFallback(sm.CoreObjectives, "- Build controlled execution lanes with explicit approvals."),
	)
}

func buildCursorMarkdown(sourcePath string, generatedAt time.Time, sm summary) string {
	return fmt.Sprintf(`---
description: FORGE local project guidance
globs:
  - "**/*"
alwaysApply: false
---

# FORGE Context Rule

Generated: %s
Source: %s
Context version: %d

## Guidance
%s

## Non-Negotiables
- Preserve approval gates for risky operations.
- Keep packets versioned and reproducible.
- Keep artifact outputs typed and inspectable.
`,
		generatedAt.UTC().Format(time.RFC3339),
		sourcePath,
		ContextVersion,
		bulletsOrFallback(sm.CoreObjectives, "- Preserve local-first inspectable workflows."),
	)
}

func bulletsOrFallback(items []string, fallback string) string {
	if len(items) == 0 {
		return fallback
	}
	var b strings.Builder
	for _, it := range items {
		trim := strings.TrimSpace(it)
		if trim == "" {
			continue
		}
		b.WriteString("- ")
		b.WriteString(trim)
		b.WriteString("\n")
	}
	out := strings.TrimSpace(b.String())
	if out == "" {
		return fallback
	}
	return out
}

func bulletOrFallback(item, fallback string) string {
	trim := strings.TrimSpace(item)
	if trim == "" {
		return fallback
	}
	return "- " + trim
}

func safeStem(path string) string {
	base := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	base = strings.TrimSpace(strings.ReplaceAll(base, " ", "-"))
	if base == "" {
		return "context"
	}
	return strings.ToLower(base)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
