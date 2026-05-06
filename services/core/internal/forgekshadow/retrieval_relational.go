package forgekshadow

import (
	"encoding/json"
	"strings"
)

type RetrievalMetadataRelationalAdapter struct{}

type RetrievalMetadataRelationalDTO struct {
	ObservationID     string         `json:"observation_id,omitempty"`
	ObservedAtUnix    int64          `json:"observed_at_unix,omitempty"`
	WorkspaceID       string         `json:"workspace_id,omitempty"`
	RequestID         string         `json:"request_id,omitempty"`
	CorrelationID     string         `json:"correlation_id,omitempty"`
	RetrievalRunID    string         `json:"retrieval_run_id,omitempty"`
	RetrievalResultID string         `json:"retrieval_result_id,omitempty"`
	SourceType        string         `json:"source_type,omitempty"`
	SourceRefID       string         `json:"source_ref_id,omitempty"`
	SourceHash        string         `json:"source_hash,omitempty"`
	ResultCount       int            `json:"result_count,omitempty"`
	SelectedCount     int            `json:"selected_count,omitempty"`
	ScoreSummary      string         `json:"score_summary,omitempty"`
	RankingPosition   int            `json:"ranking_position,omitempty"`
	RetrievalStrategy string         `json:"retrieval_strategy,omitempty"`
	IndexType         string         `json:"index_type,omitempty"`
	FreshnessStatus   string         `json:"freshness_status,omitempty"`
	DurationMS        int64          `json:"duration_ms,omitempty"`
	MetadataJSON      map[string]any `json:"metadata_json,omitempty"`
	CanonicalJSON     string         `json:"canonical_json,omitempty"`

	RawQuery      string `json:"-"`
	SourceText    string `json:"-"`
	ChunkText     string `json:"-"`
	VectorJSON    string `json:"-"`
	EmbeddingJSON string `json:"-"`
	MemoryContent string `json:"-"`
}

func NewRetrievalMetadataRelationalAdapter() RetrievalMetadataRelationalAdapter {
	return RetrievalMetadataRelationalAdapter{}
}

func (RetrievalMetadataRelationalAdapter) CanExecuteRetrieval() bool {
	return false
}

func (RetrievalMetadataRelationalAdapter) MapObservation(observation RetrievalMetadataObservation) (RetrievalMetadataRelationalDTO, error) {
	metadata, err := safeRetrievalRelationalMetadata(observation.Metadata)
	if err != nil {
		return RetrievalMetadataRelationalDTO{}, err
	}
	dto := RetrievalMetadataRelationalDTO{
		ObservationID:     strings.TrimSpace(observation.ObservationID),
		ObservedAtUnix:    observation.ObservedAt.UTC().Unix(),
		WorkspaceID:       strings.TrimSpace(observation.WorkspaceID),
		RequestID:         strings.TrimSpace(observation.RequestID),
		CorrelationID:     strings.TrimSpace(observation.CorrelationID),
		RetrievalRunID:    strings.TrimSpace(observation.RetrievalRunID),
		RetrievalResultID: strings.TrimSpace(observation.RetrievalResultID),
		SourceType:        strings.TrimSpace(observation.SourceType),
		SourceRefID:       strings.TrimSpace(observation.SourceRefID),
		SourceHash:        strings.TrimSpace(observation.SourceHash),
		ResultCount:       nonNegativeInt(observation.ResultCount),
		SelectedCount:     nonNegativeInt(observation.SelectedCount),
		ScoreSummary:      strings.TrimSpace(observation.ScoreSummary),
		RankingPosition:   nonNegativeInt(observation.RankingPosition),
		RetrievalStrategy: strings.TrimSpace(observation.RetrievalStrategy),
		IndexType:         strings.TrimSpace(observation.IndexType),
		FreshnessStatus:   strings.TrimSpace(observation.FreshnessStatus),
		DurationMS:        nonNegativeInt64(observation.DurationMS),
		MetadataJSON:      metadata,
	}
	canonical, err := canonicalRetrievalDTOJSON(dto)
	if err != nil {
		return RetrievalMetadataRelationalDTO{}, err
	}
	dto.CanonicalJSON = canonical
	return dto, nil
}

func safeRetrievalRelationalMetadata(in map[string]any) (map[string]any, error) {
	if len(in) == 0 {
		return nil, nil
	}
	for key, value := range in {
		normalized := normalizeMetadataToken(key)
		if containsRawContentTerm(normalized) || containsUnsafeTerm(normalized) {
			return nil, ErrUnsafeMetadata
		}
		if _, ok := value.([]float64); ok {
			return nil, ErrUnsafeMetadata
		}
	}
	return safeMetadata(in)
}

func canonicalRetrievalDTOJSON(dto RetrievalMetadataRelationalDTO) (string, error) {
	payload := map[string]any{
		"observation_id":       dto.ObservationID,
		"observed_at_unix":     dto.ObservedAtUnix,
		"workspace_id":         dto.WorkspaceID,
		"request_id":           dto.RequestID,
		"correlation_id":       dto.CorrelationID,
		"retrieval_run_id":     dto.RetrievalRunID,
		"retrieval_result_id":  dto.RetrievalResultID,
		"source_type":          dto.SourceType,
		"source_ref_id":        dto.SourceRefID,
		"source_hash":          dto.SourceHash,
		"result_count":         dto.ResultCount,
		"selected_count":       dto.SelectedCount,
		"score_summary":        dto.ScoreSummary,
		"ranking_position":     dto.RankingPosition,
		"retrieval_strategy":   dto.RetrievalStrategy,
		"index_type":           dto.IndexType,
		"freshness_status":     dto.FreshnessStatus,
		"duration_ms":          dto.DurationMS,
		"metadata_json":        dto.MetadataJSON,
		"diagnostic_only":      true,
		"retrieval_executes":   false,
		"relational_metadata":  true,
		"raw_content_present":  false,
		"source_content_saved": false,
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

func nonNegativeInt(value int) int {
	if value < 0 {
		return 0
	}
	return value
}

func nonNegativeInt64(value int64) int64 {
	if value < 0 {
		return 0
	}
	return value
}
