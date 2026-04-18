package embeddings

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	ProviderLocalHash = "local_hash"
	ProviderOllama    = "ollama"
)

type Provider interface {
	ID() string
	Model() string
	Embed(ctx context.Context, text string) ([]float64, error)
}

type Config struct {
	Provider  string `json:"provider"`
	Model     string `json:"model"`
	Dims      int    `json:"dims"`
	OllamaURL string `json:"ollamaUrl"`
}

type Service struct {
	db     *sql.DB
	client *http.Client
}

func New(db *sql.DB) *Service {
	return &Service{db: db, client: &http.Client{Timeout: 60 * time.Second}}
}

type ReembedResult struct {
	Provider string `json:"provider"`
	Model    string `json:"model"`
	Total    int    `json:"total"`
	Ready    int    `json:"ready"`
	Failed   int    `json:"failed"`
}

type SourceEmbeddingStatus struct {
	SourceID int64  `json:"sourceId"`
	Path     string `json:"path"`
	Total    int    `json:"totalChunks"`
	Ready    int    `json:"readyChunks"`
	Failed   int    `json:"failedChunks"`
}

type SemanticHit struct {
	ChunkID       int64   `json:"chunkId"`
	FileID        int64   `json:"fileId"`
	SourceID      int64   `json:"sourceId"`
	AbsPath       string  `json:"absPath"`
	RelPath       string  `json:"relPath"`
	ChunkIndex    int     `json:"chunkIndex"`
	Content       string  `json:"content"`
	Snippet       string  `json:"snippet"`
	SemanticScore float64 `json:"semanticScore"`
}

type SemanticSearchRequest struct {
	Query      string
	Limit      int
	SourceIDs  []int64
	Provider   string
	Model      string
	DossierID  *int64
}

func (s *Service) CurrentConfig(ctx context.Context) Config {
	provider := strings.TrimSpace(s.setting(ctx, "embedding_provider", ProviderLocalHash))
	if provider != ProviderLocalHash && provider != ProviderOllama {
		provider = ProviderLocalHash
	}
	dims, _ := strconv.Atoi(strings.TrimSpace(s.setting(ctx, "embedding_dims", "128")))
	if dims <= 0 {
		dims = 128
	}
	model := strings.TrimSpace(s.setting(ctx, "embedding_model", ""))
	if model == "" {
		if provider == ProviderLocalHash {
			model = fmt.Sprintf("local-hash-%d", dims)
		} else {
			model = strings.TrimSpace(s.setting(ctx, "ollama_model", ""))
		}
	}
	if model == "" {
		model = "default"
	}
	return Config{
		Provider:  provider,
		Model:     model,
		Dims:      dims,
		OllamaURL: strings.TrimSpace(s.setting(ctx, "ollama_base_url", "http://127.0.0.1:11434")),
	}
}

func (s *Service) Provider(ctx context.Context, overrideProvider, overrideModel string) Provider {
	cfg := s.CurrentConfig(ctx)
	if strings.TrimSpace(overrideProvider) != "" {
		cfg.Provider = strings.TrimSpace(overrideProvider)
	}
	if strings.TrimSpace(overrideModel) != "" {
		cfg.Model = strings.TrimSpace(overrideModel)
	}
	if cfg.Provider == ProviderOllama {
		return &ollamaProvider{client: s.client, baseURL: cfg.OllamaURL, model: cfg.Model}
	}
	return &localHashProvider{dims: cfg.Dims, model: cfg.Model}
}

func (s *Service) ReembedAll(ctx context.Context, provider, model string) (ReembedResult, error) {
	return s.reembed(ctx, 0, provider, model)
}

func (s *Service) ReembedSource(ctx context.Context, sourceID int64, provider, model string) (ReembedResult, error) {
	if sourceID <= 0 {
		return ReembedResult{}, fmt.Errorf("source id must be > 0")
	}
	return s.reembed(ctx, sourceID, provider, model)
}

func (s *Service) reembed(ctx context.Context, sourceID int64, provider, model string) (ReembedResult, error) {
	p := s.Provider(ctx, provider, model)
	out := ReembedResult{Provider: p.ID(), Model: p.Model()}

	query := `
SELECT c.id, f.id, f.source_id, f.content_sha256, c.content
FROM chunks c
JOIN files f ON f.id = c.file_id`
	args := []any{}
	if sourceID > 0 {
		query += ` WHERE f.source_id = ?`
		args = append(args, sourceID)
	}
	query += ` ORDER BY c.id ASC`

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return out, err
	}
	defer rows.Close()

	now := time.Now().UnixMilli()
	for rows.Next() {
		out.Total++
		var chunkID, fileID, srcID int64
		var sha, content string
		if err := rows.Scan(&chunkID, &fileID, &srcID, &sha, &content); err != nil {
			return out, err
		}

		vec, err := p.Embed(ctx, content)
		if err != nil {
			out.Failed++
			_, _ = s.db.ExecContext(ctx, `
INSERT INTO embedding_records(chunk_id, file_id, source_id, provider, model, vector_json, dims, norm, content_sha256, status, error_message, updated_at)
VALUES(?,?,?,?,?,?,?,?,?,?,?,?)
ON CONFLICT(chunk_id, provider, model)
DO UPDATE SET
  file_id=excluded.file_id,
  source_id=excluded.source_id,
  vector_json=NULL,
  dims=0,
  norm=0,
  content_sha256=excluded.content_sha256,
  status='failed',
  error_message=excluded.error_message,
  updated_at=excluded.updated_at`,
				chunkID, fileID, srcID, p.ID(), p.Model(), nil, 0, 0, sha, "failed", err.Error(), now,
			)
			continue
		}

		n := vectorNorm(vec)
		vb, _ := json.Marshal(vec)
		_, err = s.db.ExecContext(ctx, `
INSERT INTO embedding_records(chunk_id, file_id, source_id, provider, model, vector_json, dims, norm, content_sha256, status, error_message, updated_at)
VALUES(?,?,?,?,?,?,?,?,?,?,?,?)
ON CONFLICT(chunk_id, provider, model)
DO UPDATE SET
  file_id=excluded.file_id,
  source_id=excluded.source_id,
  vector_json=excluded.vector_json,
  dims=excluded.dims,
  norm=excluded.norm,
  content_sha256=excluded.content_sha256,
  status='ready',
  error_message='',
  updated_at=excluded.updated_at`,
			chunkID, fileID, srcID, p.ID(), p.Model(), string(vb), len(vec), n, sha, "ready", "", now,
		)
		if err != nil {
			return out, err
		}
		out.Ready++
	}
	if err := rows.Err(); err != nil {
		return out, err
	}
	return out, nil
}

func (s *Service) StatusBySource(ctx context.Context, provider, model string) ([]SourceEmbeddingStatus, error) {
	p := s.Provider(ctx, provider, model)
	rows, err := s.db.QueryContext(ctx, `
SELECT s.id, s.path,
       COUNT(c.id) AS total_chunks,
       SUM(CASE WHEN er.status = 'ready' THEN 1 ELSE 0 END) AS ready_chunks,
       SUM(CASE WHEN er.status = 'failed' THEN 1 ELSE 0 END) AS failed_chunks
FROM sources s
LEFT JOIN files f ON f.source_id = s.id
LEFT JOIN chunks c ON c.file_id = f.id
LEFT JOIN embedding_records er ON er.chunk_id = c.id AND er.provider = ? AND er.model = ?
GROUP BY s.id, s.path
ORDER BY s.id ASC`, p.ID(), p.Model())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []SourceEmbeddingStatus{}
	for rows.Next() {
		var r SourceEmbeddingStatus
		if err := rows.Scan(&r.SourceID, &r.Path, &r.Total, &r.Ready, &r.Failed); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *Service) SemanticSearch(ctx context.Context, req SemanticSearchRequest) ([]SemanticHit, error) {
	if strings.TrimSpace(req.Query) == "" {
		return nil, nil
	}
	if req.Limit <= 0 || req.Limit > 100 {
		req.Limit = 20
	}
	p := s.Provider(ctx, req.Provider, req.Model)
	qvec, err := p.Embed(ctx, req.Query)
	if err != nil {
		return nil, err
	}
	qnorm := vectorNorm(qvec)
	if qnorm == 0 {
		return nil, nil
	}

	sourceIDs := req.SourceIDs
	if req.DossierID != nil && len(sourceIDs) == 0 {
		sids, err := s.sourceIDsForDossier(ctx, *req.DossierID)
		if err != nil {
			return nil, err
		}
		sourceIDs = sids
	}

	query := `
SELECT er.chunk_id, er.file_id, er.source_id, er.vector_json,
       c.chunk_index, c.content,
       f.abs_path, f.rel_path
FROM embedding_records er
JOIN chunks c ON c.id = er.chunk_id
JOIN files f ON f.id = er.file_id
WHERE er.provider = ? AND er.model = ? AND er.status = 'ready'`
	args := []any{p.ID(), p.Model()}
	if len(sourceIDs) > 0 {
		query += ` AND er.source_id IN (` + placeholders(len(sourceIDs)) + `)`
		for _, id := range sourceIDs {
			args = append(args, id)
		}
	}
	query += ` ORDER BY er.chunk_id ASC`

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	hits := make([]SemanticHit, 0, req.Limit)
	type scored struct {
		hit   SemanticHit
		score float64
	}
	all := make([]scored, 0, 256)
	for rows.Next() {
		var chunkID, fileID, sourceID int64
		var vectorJSON string
		var chunkIndex int
		var content, absPath, relPath string
		if err := rows.Scan(&chunkID, &fileID, &sourceID, &vectorJSON, &chunkIndex, &content, &absPath, &relPath); err != nil {
			return nil, err
		}
		var vec []float64
		if err := json.Unmarshal([]byte(vectorJSON), &vec); err != nil {
			continue
		}
		score := cosine(qvec, qnorm, vec)
		if math.IsNaN(score) || math.IsInf(score, 0) {
			continue
		}
		all = append(all, scored{hit: SemanticHit{
			ChunkID:       chunkID,
			FileID:        fileID,
			SourceID:      sourceID,
			AbsPath:       absPath,
			RelPath:       relPath,
			ChunkIndex:    chunkIndex,
			Content:       content,
			Snippet:       snippet(content, 260),
			SemanticScore: score,
		}, score: score})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	sort.Slice(all, func(i, j int) bool { return all[i].score > all[j].score })
	if len(all) > req.Limit {
		all = all[:req.Limit]
	}
	for _, row := range all {
		hits = append(hits, row.hit)
	}
	return hits, nil
}

func (s *Service) sourceIDsForDossier(ctx context.Context, dossierID int64) ([]int64, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT source_id FROM dossier_sources WHERE dossier_id = ? ORDER BY source_id ASC`, dossierID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

func (s *Service) setting(ctx context.Context, key, def string) string {
	var v string
	err := s.db.QueryRowContext(ctx, `SELECT value FROM settings WHERE key = ?`, key).Scan(&v)
	if err != nil || strings.TrimSpace(v) == "" {
		return def
	}
	return strings.TrimSpace(v)
}

type localHashProvider struct {
	dims  int
	model string
}

func (p *localHashProvider) ID() string    { return ProviderLocalHash }
func (p *localHashProvider) Model() string { return p.model }

func (p *localHashProvider) Embed(ctx context.Context, text string) ([]float64, error) {
	_ = ctx
	dims := p.dims
	if dims <= 0 {
		dims = 128
	}
	vec := make([]float64, dims)
	tokens := tokenize(text)
	if len(tokens) == 0 {
		return vec, nil
	}
	for _, tok := range tokens {
		h := fnv.New64a()
		_, _ = h.Write([]byte(tok))
		sum := h.Sum64()
		idx := int(sum % uint64(dims))
		sign := 1.0
		if (sum>>8)&1 == 1 {
			sign = -1.0
		}
		vec[idx] += sign
	}
	n := vectorNorm(vec)
	if n > 0 {
		for i := range vec {
			vec[i] /= n
		}
	}
	return vec, nil
}

type ollamaProvider struct {
	client  *http.Client
	baseURL string
	model   string
}

func (p *ollamaProvider) ID() string    { return ProviderOllama }
func (p *ollamaProvider) Model() string { return p.model }

func (p *ollamaProvider) Embed(ctx context.Context, text string) ([]float64, error) {
	if strings.TrimSpace(p.baseURL) == "" {
		return nil, fmt.Errorf("ollama base URL is empty")
	}
	if strings.TrimSpace(p.model) == "" {
		return nil, fmt.Errorf("ollama embedding model is empty")
	}
	payload := map[string]any{"model": p.model, "prompt": text}
	body, _ := json.Marshal(payload)

	url := strings.TrimRight(p.baseURL, "/") + "/api/embeddings"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	res, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode >= 300 {
		return nil, fmt.Errorf("ollama embeddings returned %s", res.Status)
	}
	var out struct {
		Embedding []float64 `json:"embedding"`
	}
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		return nil, err
	}
	if len(out.Embedding) == 0 {
		return nil, fmt.Errorf("empty embedding from ollama")
	}
	return out.Embedding, nil
}

func tokenize(in string) []string {
	lower := strings.ToLower(in)
	f := strings.FieldsFunc(lower, func(r rune) bool {
		return !(r == '_' || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'))
	})
	out := make([]string, 0, len(f))
	for _, t := range f {
		trim := strings.TrimSpace(t)
		if trim == "" {
			continue
		}
		out = append(out, trim)
	}
	return out
}

func vectorNorm(v []float64) float64 {
	s := 0.0
	for _, x := range v {
		s += x * x
	}
	return math.Sqrt(s)
}

func cosine(a []float64, anorm float64, b []float64) float64 {
	if len(a) == 0 || len(b) == 0 || anorm == 0 {
		return 0
	}
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	dot := 0.0
	bn := 0.0
	for i := 0; i < n; i++ {
		dot += a[i] * b[i]
		bn += b[i] * b[i]
	}
	if bn == 0 {
		return 0
	}
	return dot / (anorm * math.Sqrt(bn))
}

func snippet(in string, max int) string {
	trim := strings.TrimSpace(strings.ReplaceAll(in, "\n", " "))
	if len(trim) <= max {
		return trim
	}
	return trim[:max] + "…"
}

func placeholders(n int) string {
	if n <= 0 {
		return ""
	}
	b := strings.Builder{}
	for i := 0; i < n; i++ {
		if i > 0 {
			b.WriteString(",")
		}
		b.WriteString("?")
	}
	return b.String()
}
