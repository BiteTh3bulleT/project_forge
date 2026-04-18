package packets

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"forge/projectforge/services/core/internal/adapters"
	"forge/projectforge/services/core/internal/memory"
	"forge/projectforge/services/core/internal/search"
)

const PacketVersion = 1

type SourceReference struct {
	ChunkID    int64   `json:"chunkId"`
	FileID     int64   `json:"fileId"`
	AbsPath    string  `json:"absPath"`
	RelPath    string  `json:"relPath"`
	ChunkIdx   int     `json:"chunkIndex"`
	Snippet    string  `json:"snippet"`
	Score      float64 `json:"score"`
	ContentLen int     `json:"contentLength"`
}

type Packet struct {
	ID                     int64           `json:"id"`
	PacketVersion          int             `json:"packetVersion"`
	CreatedAtMs            int64           `json:"createdAtMs"`
	GeneratedAtMs          int64           `json:"generatedAtMs"`
	Title                  string          `json:"title"`
	UserRequest            string          `json:"userRequest"`
	Objective              string          `json:"objective"`
	AdapterTarget          string          `json:"adapterTarget"`
	ExecutionMode          string          `json:"executionMode"`
	RiskClass              string          `json:"riskClass"`
	ExpectedOutput         json.RawMessage `json:"expectedOutput"`
	Constraints            json.RawMessage `json:"constraints"`
	Instructions           string          `json:"instructions"`
	SelectedPaths          json.RawMessage `json:"selectedPaths"`
	ScopeSnapshot          json.RawMessage `json:"scopeSnapshot"`
	SourceReferences       json.RawMessage `json:"sourceReferences"`
	RetrievedContext       json.RawMessage `json:"retrievedContext"`
	ProjectNotes           string          `json:"projectNotes"`
	SourceContextRecordIDs json.RawMessage `json:"sourceContextRecordIds"`
	RequestPayload         json.RawMessage `json:"requestPayload"`
}

type BuildRequest struct {
	Title                  string
	UserRequest            string
	Objective              string
	AdapterTarget          string
	ExecutionMode          string
	RiskClass              string
	ExpectedOutput         map[string]any
	Constraints            []string
	Instructions           string
	Scope                  adapters.Scope
	SourceContextRecordIDs []int64
	Query                  string
	Limit                  int
	RequestPayload         map[string]any
	ProjectNotes           string
	RetrievedItems         []RetrievedItem
	RetrievalRunID         *int64
}

type RetrievedItem struct {
	ChunkID       int64
	FileID        int64
	AbsPath       string
	RelPath       string
	ChunkIndex    int
	Snippet       string
	Score         float64
	KeywordScore  float64
	SemanticScore float64
	HybridScore   float64
}

type Service struct {
	db     *sql.DB
	search *search.Service
	memory *memory.Service
}

func New(db *sql.DB, searchSvc *search.Service, memorySvc *memory.Service) *Service {
	return &Service{db: db, search: searchSvc, memory: memorySvc}
}

func (s *Service) BuildAndStore(ctx context.Context, req BuildRequest) (*Packet, error) {
	if strings.TrimSpace(req.UserRequest) == "" {
		return nil, fmt.Errorf("packet user request is required")
	}
	if strings.TrimSpace(req.Objective) == "" {
		return nil, fmt.Errorf("packet objective is required")
	}
	if strings.TrimSpace(req.AdapterTarget) == "" {
		return nil, fmt.Errorf("packet adapter target is required")
	}
	if req.Limit <= 0 || req.Limit > 20 {
		req.Limit = 8
	}
	query := strings.TrimSpace(req.Query)
	if query == "" {
		query = req.UserRequest
	}

	refs := []SourceReference{}
	retrieved := []map[string]any{}
	if len(req.RetrievedItems) > 0 {
		for _, item := range req.RetrievedItems {
			chunk := item.ChunkID
			if chunk <= 0 {
				continue
			}
			h, err := s.search.ChunkByID(ctx, chunk)
			if err != nil {
				continue
			}
			score := item.Score
			if score == 0 {
				score = item.HybridScore
			}
			refs = append(refs, SourceReference{
				ChunkID:    h.ChunkID,
				FileID:     h.FileID,
				AbsPath:    nonEmpty(item.AbsPath, h.AbsPath),
				RelPath:    nonEmpty(item.RelPath, h.RelPath),
				ChunkIdx:   h.ChunkIdx,
				Snippet:    nonEmpty(item.Snippet, h.Snippet),
				Score:      score,
				ContentLen: h.ContentLen,
			})
			retrieved = append(retrieved, map[string]any{
				"chunkId":       h.ChunkID,
				"fileId":        h.FileID,
				"absPath":       nonEmpty(item.AbsPath, h.AbsPath),
				"relPath":       nonEmpty(item.RelPath, h.RelPath),
				"chunkIndex":    h.ChunkIdx,
				"snippet":       nonEmpty(item.Snippet, h.Snippet),
				"score":         score,
				"keywordScore":  item.KeywordScore,
				"semanticScore": item.SemanticScore,
				"hybridScore":   item.HybridScore,
				"content":       h.Content,
				"contentLength": h.ContentLen,
			})
		}
	} else {
		hits, err := s.search.Search(ctx, query, req.Limit)
		if err != nil {
			return nil, fmt.Errorf("packet retrieval failed: %w", err)
		}
		refs = make([]SourceReference, 0, len(hits))
		retrieved = make([]map[string]any, 0, len(hits))
		for _, h := range hits {
			refs = append(refs, SourceReference{
				ChunkID:    h.ChunkID,
				FileID:     h.FileID,
				AbsPath:    h.AbsPath,
				RelPath:    h.RelPath,
				ChunkIdx:   h.ChunkIdx,
				Snippet:    h.Snippet,
				Score:      h.Score,
				ContentLen: h.ContentLen,
			})
			retrieved = append(retrieved, map[string]any{
				"chunkId":       h.ChunkID,
				"fileId":        h.FileID,
				"absPath":       h.AbsPath,
				"relPath":       h.RelPath,
				"chunkIndex":    h.ChunkIdx,
				"snippet":       h.Snippet,
				"score":         h.Score,
				"content":       h.Content,
				"contentLength": h.ContentLen,
			})
		}
	}

	now := time.Now().UnixMilli()

	expectedJSON, _ := json.Marshal(emptyMap(req.ExpectedOutput))
	constraintsJSON, _ := json.Marshal(req.Constraints)
	selectedJSON, _ := json.Marshal(req.Scope.SelectedPaths)
	scopeJSON, _ := json.Marshal(req.Scope)
	refsJSON, _ := json.Marshal(refs)
	retrievedJSON, _ := json.Marshal(retrieved)
	sourceCtxJSON, _ := json.Marshal(req.SourceContextRecordIDs)
	payload := emptyMap(req.RequestPayload)
	if req.RetrievalRunID != nil {
		payload["retrievalRunId"] = *req.RetrievalRunID
	}
	requestJSON, _ := json.Marshal(payload)

	res, err := s.db.ExecContext(ctx, `
INSERT INTO task_packets(
  packet_version, created_at, generated_at, title, user_request, objective,
  adapter_target, execution_mode, risk_class,
  expected_output_json, constraints_json, instructions,
  selected_paths_json, scope_snapshot_json, source_references_json, retrieved_context_json,
  project_notes, source_context_record_ids_json, request_payload_json
) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		PacketVersion,
		now,
		now,
		nonEmpty(req.Title, "Task packet"),
		req.UserRequest,
		req.Objective,
		req.AdapterTarget,
		nonEmpty(req.ExecutionMode, "scoped_execution"),
		nonEmpty(req.RiskClass, "read_only"),
		string(expectedJSON),
		string(constraintsJSON),
		req.Instructions,
		string(selectedJSON),
		string(scopeJSON),
		string(refsJSON),
		string(retrievedJSON),
		req.ProjectNotes,
		string(sourceCtxJSON),
		string(requestJSON),
	)
	if err != nil {
		return nil, fmt.Errorf("insert packet: %w", err)
	}
	id, _ := res.LastInsertId()
	if s.memory != nil {
		for _, ref := range refs {
			retrievalResultID := s.retrievalResultForChunk(ctx, req.RetrievalRunID, ref.ChunkID)
			reason := fmt.Sprintf("Included %s (chunk %d) for packet objective alignment; score %.3f and snippet coverage.",
				nonEmpty(ref.RelPath, ref.AbsPath),
				ref.ChunkID,
				ref.Score,
			)
			_, _ = s.memory.AddAlignmentNote(ctx, memory.AddAlignmentNoteRequest{
				PacketID:          id,
				RetrievalResultID: retrievalResultID,
				Note:              reason,
			})
		}
	}

	return s.GetByID(ctx, id)
}

func (s *Service) GetByID(ctx context.Context, id int64) (*Packet, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT id, packet_version, created_at, generated_at, title, user_request, objective,
       adapter_target, execution_mode, risk_class,
       expected_output_json, constraints_json, instructions,
       selected_paths_json, scope_snapshot_json, source_references_json, retrieved_context_json,
       project_notes, source_context_record_ids_json, request_payload_json
FROM task_packets WHERE id = ?`, id)
	var p Packet
	var expected, constraints, selected, scope, refs, retrieved, sourceCtx, request string
	if err := row.Scan(
		&p.ID,
		&p.PacketVersion,
		&p.CreatedAtMs,
		&p.GeneratedAtMs,
		&p.Title,
		&p.UserRequest,
		&p.Objective,
		&p.AdapterTarget,
		&p.ExecutionMode,
		&p.RiskClass,
		&expected,
		&constraints,
		&p.Instructions,
		&selected,
		&scope,
		&refs,
		&retrieved,
		&p.ProjectNotes,
		&sourceCtx,
		&request,
	); err != nil {
		return nil, err
	}
	p.ExpectedOutput = json.RawMessage(expected)
	p.Constraints = json.RawMessage(constraints)
	p.SelectedPaths = json.RawMessage(selected)
	p.ScopeSnapshot = json.RawMessage(scope)
	p.SourceReferences = json.RawMessage(refs)
	p.RetrievedContext = json.RawMessage(retrieved)
	p.SourceContextRecordIDs = json.RawMessage(sourceCtx)
	p.RequestPayload = json.RawMessage(request)
	return &p, nil
}

func emptyMap(v map[string]any) map[string]any {
	if v == nil {
		return map[string]any{}
	}
	return v
}

func nonEmpty(v, def string) string {
	if strings.TrimSpace(v) == "" {
		return def
	}
	return strings.TrimSpace(v)
}

func (s *Service) retrievalResultForChunk(ctx context.Context, retrievalRunID *int64, chunkID int64) *int64 {
	if retrievalRunID == nil || *retrievalRunID <= 0 || chunkID <= 0 {
		return nil
	}
	var resultID sql.NullInt64
	err := s.db.QueryRowContext(ctx, `
SELECT id
FROM retrieval_results
WHERE retrieval_run_id = ? AND chunk_id = ?
ORDER BY rank_index ASC
LIMIT 1`, *retrievalRunID, chunkID).Scan(&resultID)
	if err != nil || !resultID.Valid {
		return nil
	}
	v := resultID.Int64
	return &v
}
