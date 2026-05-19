package api

import (
	"context"
	"fmt"
	"strings"

	"forge/projectforge/services/core/internal/aios/domain"
)

// RunSourceIndexIngest runs the librarian pipeline as a dry-run preflight after
// a source folder has been (re)indexed. This is the live production caller for
// the candidate-action validation seams: every cell-proposed action emitted by
// the pipeline routes through VALIDATE_REF_SHAPE, COMPARE_REF_SHAPE,
// VALIDATE_SEMANTIC_OPERATION, VALIDATE_ADMISSION_CANDIDATE, and
// VALIDATE_CONTEXT_ATTRIBUTION before reaching commit. Forced to dry-run
// commit mode so no canonical state mutates from this path.
func (l *AutonomyMaintenanceLoop) RunSourceIndexIngest(ctx context.Context, sourceID int64, sourcePath string) (domain.IngestResult, error) {
	if l == nil || l.librarianPipeline == nil {
		return domain.IngestResult{}, nil
	}
	now := l.nowMillis()
	sourcePath = strings.TrimSpace(sourcePath)
	correlationID := fmt.Sprintf("source-index:%d:%d", sourceID, now)
	req := domain.IngestRequest{
		ID:        fmt.Sprintf("ingest-source-index-%d-%d", sourceID, now),
		InputKind: domain.IngestArtifactEvent,
		Content:   sourcePath,
		Payload: map[string]any{
			"sourceId":   sourceID,
			"sourcePath": sourcePath,
			"trigger":    "ingest.source.index",
		},
		Actor: domain.ActorIdentity{
			ID:   "forge.librarian.source_index",
			Kind: "system",
		},
		Source: domain.SourceSystem,
		Scope:  l.scope,
		Provenance: domain.Provenance{
			Actor:     "forge.librarian.source_index",
			ActorType: "system",
			Source:    "librarian.source_index",
			TraceID:   correlationID,
		},
		CorrelationID: correlationID,
		TraceID:       correlationID,
		DryRun:        true,
		CommitMode:    domain.IngestValidateOnly,
		RequestedAt:   now,
		Metadata: map[string]any{
			"sourceId":   sourceID,
			"sourcePath": sourcePath,
			"trigger":    "ingest.source.index",
		},
	}
	result, runErr := l.librarianPipeline.Run(ctx, req)
	if l.events != nil {
		summary := map[string]any{
			"sourceId":        sourceID,
			"sourcePath":      sourcePath,
			"success":         result.Success,
			"dryRun":          result.DryRun,
			"proposedCount":   result.Summary.ProposedCount,
			"acceptedCount":   result.Summary.AcceptedCount,
			"rejectedCount":   result.Summary.RejectedCount,
			"cellCount":       result.Summary.CellCount,
			"validationSeams": countValidationSeamDiagnostics(result),
			"correlationId":   correlationID,
		}
		if runErr != nil {
			summary["error"] = runErr.Error()
		}
		_ = l.events.Emit(context.Background(), "librarian.ingest.source_indexed", summary)
	}
	return result, runErr
}

// LibrarianPipelineConfigured reports whether the autonomy loop has an active
// librarian pipeline. Used by server hooks to short-circuit early when the
// loop is disabled (e.g. autonomy_dream_enabled=false).
func (l *AutonomyMaintenanceLoop) LibrarianPipelineConfigured() bool {
	return l != nil && l.librarianPipeline != nil
}

func countValidationSeamDiagnostics(result domain.IngestResult) int {
	if result.TruthDiagnostics == nil {
		return 0
	}
	switch v := result.TruthDiagnostics["validationSeams"].(type) {
	case []map[string]any:
		return len(v)
	case []any:
		return len(v)
	}
	return 0
}
