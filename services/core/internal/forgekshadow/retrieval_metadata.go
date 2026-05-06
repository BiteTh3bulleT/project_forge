package forgekshadow

import (
	"fmt"
	"strings"
	"time"
)

const (
	retrievalMetadataObservationType = "retrieval_metadata"
	retrievalMetadataRouteClass      = "retrieval"
)

const (
	maxRetrievalMetadataWarnings      = 8
	maxRetrievalMetadataWarningLength = 160
)

func normalizeRetrievalMetadataInput(input RetrievalMetadataInput, now time.Time, observationID string) (RetrievalMetadataObservation, map[string]any, error) {
	runID, err := safeRetrievalMetadataRef(input.RetrievalRunID, "retrieval_run_id")
	if err != nil {
		return RetrievalMetadataObservation{}, nil, err
	}
	resultID, err := safeRetrievalMetadataRef(input.RetrievalResultID, "retrieval_result_id")
	if err != nil {
		return RetrievalMetadataObservation{}, nil, err
	}
	sourceRefID, err := safeRetrievalMetadataRef(input.SourceRefID, "source_ref_id")
	if err != nil {
		return RetrievalMetadataObservation{}, nil, err
	}
	sourceHash, err := safeRetrievalMetadataRef(input.SourceHash, "source_hash")
	if err != nil {
		return RetrievalMetadataObservation{}, nil, err
	}
	scoreSummary, err := safeRetrievalMetadataRef(input.ScoreSummary, "score_summary")
	if err != nil {
		return RetrievalMetadataObservation{}, nil, err
	}
	embeddingModelID, err := safeRetrievalMetadataRef(input.EmbeddingModelID, "embedding_model_id")
	if err != nil {
		return RetrievalMetadataObservation{}, nil, err
	}
	warnings, err := safeRetrievalMetadataWarnings(input.Warnings)
	if err != nil {
		return RetrievalMetadataObservation{}, nil, err
	}
	sourceType := normalizeRetrievalSourceType(input.SourceType)
	strategy := normalizeRetrievalStrategy(input.RetrievalStrategy)
	indexType := normalizeRetrievalIndexType(input.IndexType)
	freshnessStatus := normalizeRetrievalFreshnessStatus(input.FreshnessStatus)
	resultCount := input.ResultCount
	if resultCount < 0 {
		resultCount = 0
	}
	selectedCount := input.SelectedCount
	if selectedCount < 0 {
		selectedCount = 0
	}
	rankingPosition := input.RankingPosition
	if rankingPosition < 0 {
		rankingPosition = 0
	}
	durationMS := input.Duration.Milliseconds()
	if durationMS < 0 {
		durationMS = 0
	}

	metadata := map[string]any{
		"observation_type": retrievalMetadataObservationType,
		"route_class":      retrievalMetadataRouteClass,
		"duration_ms":      durationMS,
	}
	if runID != "" {
		metadata["retrieval_run_id"] = runID
	}
	if resultID != "" {
		metadata["retrieval_result_id"] = resultID
	}
	if sourceType != "" {
		metadata["source_type"] = sourceType
	}
	if sourceRefID != "" {
		metadata["source_ref_id"] = sourceRefID
	}
	if sourceHash != "" {
		metadata["source_hash"] = sourceHash
	}
	if resultCount > 0 {
		metadata["result_count"] = resultCount
	}
	if selectedCount > 0 {
		metadata["selected_count"] = selectedCount
	}
	if scoreSummary != "" {
		metadata["score_summary"] = scoreSummary
	}
	if rankingPosition > 0 {
		metadata["ranking_position"] = rankingPosition
	}
	if strategy != "" {
		metadata["retrieval_strategy"] = strategy
	}
	if indexType != "" {
		metadata["index_type"] = indexType
	}
	if embeddingModelID != "" {
		metadata["embedding_model_id"] = embeddingModelID
	}
	if freshnessStatus != "" {
		metadata["freshness_status"] = freshnessStatus
	}
	if requestID := strings.TrimSpace(input.RequestID); requestID != "" {
		metadata["request_id"] = requestID
	}
	if workspaceID := strings.TrimSpace(input.WorkspaceID); workspaceID != "" {
		metadata["workspace_id"] = workspaceID
	}
	if correlationID := strings.TrimSpace(input.CorrelationID); correlationID != "" {
		metadata["correlation_id"] = correlationID
	}
	if len(warnings) > 0 {
		metadata["warning_count"] = len(warnings)
	}
	for key, value := range input.Metadata {
		trimmedKey := strings.TrimSpace(key)
		if isReservedRetrievalMetadataKey(trimmedKey) {
			continue
		}
		if _, exists := metadata[trimmedKey]; exists {
			continue
		}
		if err := validateRetrievalCallerMetadataValue(trimmedKey, value); err != nil {
			return RetrievalMetadataObservation{}, nil, err
		}
		metadata[trimmedKey] = value
	}
	safe, err := safeMetadata(metadata)
	if err != nil {
		return RetrievalMetadataObservation{}, nil, err
	}
	return RetrievalMetadataObservation{
		ObservationID:     observationID,
		ObservedAt:        now,
		WorkspaceID:       strings.TrimSpace(input.WorkspaceID),
		RequestID:         strings.TrimSpace(input.RequestID),
		CorrelationID:     strings.TrimSpace(input.CorrelationID),
		RetrievalRunID:    runID,
		RetrievalResultID: resultID,
		SourceType:        sourceType,
		SourceRefID:       sourceRefID,
		SourceHash:        sourceHash,
		ResultCount:       resultCount,
		SelectedCount:     selectedCount,
		ScoreSummary:      scoreSummary,
		RankingPosition:   rankingPosition,
		RetrievalStrategy: strategy,
		IndexType:         indexType,
		EmbeddingModelID:  embeddingModelID,
		FreshnessStatus:   freshnessStatus,
		DurationMS:        durationMS,
		Warnings:          warnings,
		Metadata:          safe,
	}, safe, nil
}

func normalizeRetrievalSourceType(value string) string {
	switch strings.TrimSpace(value) {
	case "source", "file", "chunk", "memory_ref", "packet", "dossier", "retrieval_result":
		return strings.TrimSpace(value)
	default:
		return ""
	}
}

func normalizeRetrievalStrategy(value string) string {
	switch strings.TrimSpace(value) {
	case "keyword", "semantic", "hybrid", "vsa", "rerank", "unknown":
		return strings.TrimSpace(value)
	default:
		return ""
	}
}

func normalizeRetrievalIndexType(value string) string {
	switch strings.TrimSpace(value) {
	case "fts", "vector", "hybrid", "vsa", "memory", "unknown":
		return strings.TrimSpace(value)
	default:
		return ""
	}
}

func normalizeRetrievalFreshnessStatus(value string) string {
	switch strings.TrimSpace(value) {
	case "fresh", "stale", "unknown":
		return strings.TrimSpace(value)
	default:
		return ""
	}
}

func safeRetrievalMetadataRef(value, field string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", nil
	}
	if containsUnsafeTerm(value) || containsRawContentTerm(value) || len(value) > maxMetadataStringLength {
		return "", fmt.Errorf("%w: %s", ErrUnsafeMetadata, field)
	}
	return value, nil
}

func safeRetrievalMetadataWarnings(in []string) ([]string, error) {
	if len(in) == 0 {
		return nil, nil
	}
	out := make([]string, 0, len(in))
	for _, warning := range in {
		text := strings.TrimSpace(warning)
		if text == "" {
			continue
		}
		if containsUnsafeTerm(text) || containsRawContentTerm(text) || len(text) > maxRetrievalMetadataWarningLength {
			return nil, fmt.Errorf("%w: retrieval metadata warning", ErrUnsafeMetadata)
		}
		out = append(out, text)
		if len(out) == maxRetrievalMetadataWarnings {
			break
		}
	}
	if len(out) == 0 {
		return nil, nil
	}
	return out, nil
}

func validateRetrievalCallerMetadataValue(key string, value any) error {
	switch v := value.(type) {
	case string:
		if containsRawContentTerm(v) {
			return fmt.Errorf("%w: value for %q", ErrUnsafeMetadata, strings.TrimSpace(key))
		}
	case fmt.Stringer:
		if containsRawContentTerm(v.String()) {
			return fmt.Errorf("%w: value for %q", ErrUnsafeMetadata, strings.TrimSpace(key))
		}
	}
	return nil
}

func isReservedRetrievalMetadataKey(key string) bool {
	switch normalizeMetadataToken(key) {
	case "observation_type", "route_class", "duration_ms", "retrieval_run_id",
		"retrieval_result_id", "source_type", "source_ref_id", "source_hash",
		"result_count", "selected_count", "score_summary", "ranking_position",
		"retrieval_strategy", "index_type", "embedding_model_id", "freshness_status",
		"request_id", "workspace_id", "correlation_id", "warning_count":
		return true
	default:
		return false
	}
}
