package librarian

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"time"

	"forge/projectforge/services/core/internal/aios/controllane"
	"forge/projectforge/services/core/internal/aios/domain"
	"forge/projectforge/services/core/internal/aios/truth"
)

type IngestPipelineOptions struct {
	Kernel           controllane.ForgeKernelProcessor
	Repositories     CellReadRepositories
	Cells            []RuntimeCell
	Semantic         SemanticInferenceService
	AutonomyPass     AutonomyPassFunc
	MaxAutonomyDepth int
	NowMillis        func() int64
	FeatureFlags     map[string]bool
}

type AutonomyPassFunc func(
	ctx context.Context,
	req domain.IngestRequest,
	result domain.IngestResult,
	truthEngine truth.TruthEngine,
	depth int,
) ([]domain.AutonomyRunSummary, error)

type IngestPipeline struct {
	kernel           controllane.ForgeKernelProcessor
	repositories     CellReadRepositories
	cells            []RuntimeCell
	semantic         SemanticInferenceService
	truthEngine      *truth.Engine
	autonomyPass     AutonomyPassFunc
	maxAutonomyDepth int
	nowMillis        func() int64
	featureFlags     map[string]bool
}

func NewIngestPipeline(opts IngestPipelineOptions) *IngestPipeline {
	nowFn := opts.NowMillis
	if nowFn == nil {
		nowFn = domain.NowMillis
	}
	cells := opts.Cells
	if len(cells) == 0 {
		cells = DefaultCells()
	}
	semantic := opts.Semantic
	if semantic == nil {
		semantic = NoopSemanticInference{}
	}
	flags := map[string]bool{}
	for k, v := range opts.FeatureFlags {
		flags[k] = v
	}
	truthEngine := truth.NewEngine(truth.EngineOptions{
		Kernel: opts.Kernel,
		Repositories: truth.Repositories{
			State:         opts.Repositories.State,
			Loops:         opts.Repositories.Loops,
			Notes:         opts.Repositories.Notes,
			Models:        opts.Repositories.Models,
			Contradiction: opts.Repositories.Contradictions,
			Supersession:  opts.Repositories.Supersessions,
		},
		NowMillis: nowFn,
	})
	return &IngestPipeline{
		kernel:           opts.Kernel,
		repositories:     opts.Repositories,
		cells:            cells,
		semantic:         semantic,
		truthEngine:      truthEngine,
		autonomyPass:     opts.AutonomyPass,
		maxAutonomyDepth: nonZeroInt(opts.MaxAutonomyDepth, 1),
		nowMillis:        nowFn,
		featureFlags:     flags,
	}
}

func (p *IngestPipeline) Run(ctx context.Context, req domain.IngestRequest) (domain.IngestResult, error) {
	req = p.normalizeRequest(req)
	result := domain.IngestResult{
		Success:            true,
		Scope:              req.Scope,
		CorrelationID:      req.CorrelationID,
		TraceID:            req.TraceID,
		CellRunID:          "cellrun-" + shortHash(req.ID, req.CorrelationID),
		ProposedActions:    []domain.SyscallRequest{},
		AcceptedActions:    []domain.IngestActionOutcome{},
		RejectedActions:    []domain.IngestActionOutcome{},
		CommittedObjectIDs: []string{},
		Warnings:           []string{},
		Errors:             []domain.IngestError{},
		AuditIDs:           []string{},
		DryRun:             req.DryRun || req.CommitMode == domain.IngestValidateOnly,
		Diagnostics:        []domain.CellDiagnostic{},
		Batches:            []domain.CandidateActionBatch{},
		AutonomyRuns:       []domain.AutonomyRunSummary{},
		TruthDiagnostics:   map[string]any{},
	}
	if p.kernel == nil {
		result.Success = false
		result.Errors = append(result.Errors, domain.IngestError{
			Code:    domain.IngestErrInvalidRequest,
			Field:   "kernel",
			Message: "kernel processor is required",
		})
		return result, nil
	}
	if errs := req.Validate(); len(errs) > 0 {
		result.Success = false
		result.Errors = append(result.Errors, errs...)
		return result, nil
	}

	if depErrs := p.validateCellDependencies(); len(depErrs) > 0 {
		result.Success = false
		result.Errors = append(result.Errors, depErrs...)
		return result, nil
	}

	event, eventPersisted, eventReused, appendErr := p.appendOrVirtualizeEvent(ctx, req)
	result.EventID = event.ID
	if appendErr != nil {
		result.Success = false
		result.Errors = append(result.Errors, domain.IngestError{
			Code:    domain.IngestErrEventAppend,
			Field:   "journal.append",
			Message: appendErr.Error(),
		})
		return result, nil
	}
	if !eventPersisted {
		result.Warnings = append(result.Warnings, "journal append skipped due dry-run/validate-only mode")
	}
	if eventReused {
		result.Warnings = append(result.Warnings, "reused existing journal event for idempotent ingest replay")
	}

	seedCtx := p.buildCellContext(ctx, req, event, nil)
	workingActions := []domain.SyscallRequest{}
	for _, cell := range p.cells {
		cellCtx := seedCtx
		cellCtx.ExistingActions = append([]domain.SyscallRequest{}, workingActions...)
		canRun, reason := cell.CanRun(ctx, cellCtx)
		if !canRun {
			result.Diagnostics = append(result.Diagnostics, domain.CellDiagnostic{
				CellName:      cell.Name(),
				CellVersion:   cell.Version(),
				ProposedCount: 0,
				Warnings:      []string{},
				Errors:        []domain.IngestError{},
				Skipped:       true,
				SkippedReason: reason,
			})
			continue
		}
		runStart := time.Now()
		runRes, err := cell.Run(ctx, cellCtx)
		if runRes.CellName == "" {
			runRes.CellName = cell.Name()
		}
		if runRes.CellVersion == "" {
			runRes.CellVersion = cell.Version()
		}
		runRes.Duration = time.Since(runStart)
		if err != nil {
			runRes.Errors = append(runRes.Errors, domain.IngestError{
				Code:    domain.IngestErrCellRun,
				Field:   cell.Name(),
				Message: err.Error(),
			})
		}
		batchID := "batch-" + shortHash(req.ID, event.ID, runRes.CellName, fmt.Sprintf("%d", len(result.Batches)))
		annotated := make([]domain.SyscallRequest, 0, len(runRes.ProposedActions))
		for _, action := range runRes.ProposedActions {
			annotated = append(annotated, p.annotateAction(req, event, action, runRes.CellName, runRes.CellVersion, batchID))
		}
		workingActions = append(workingActions, annotated...)
		result.Batches = append(result.Batches, domain.CandidateActionBatch{
			ID:            batchID,
			SourceEventID: event.ID,
			ProducedBy:    runRes.CellName,
			WorkspaceID:   req.Scope.WorkspaceID,
			LaneID:        req.Scope.LaneID,
			CorrelationID: req.CorrelationID,
			TraceID:       req.TraceID,
			Actions:       annotated,
			Warnings:      append([]string{}, runRes.Warnings...),
			Confidence:    runRes.Confidence,
			Metadata:      map[string]any{"version": runRes.CellVersion},
		})
		result.Diagnostics = append(result.Diagnostics, runRes.Diagnostic())
	}

	proposed := p.orderAndDedupe(workingActions)
	result.ProposedActions = append(result.ProposedActions, proposed...)
	result.Summary.ProposedCount = len(proposed)
	result.Summary.CellCount = len(p.cells)

	switch req.CommitMode {
	case domain.IngestValidateOnly:
		p.processActions(ctx, req, event, proposed, true, false, &result)
	case domain.IngestCommitAllOrFail:
		result.Success = false
		result.Errors = append(result.Errors, domain.IngestError{
			Code:    domain.IngestErrUnsupported,
			Field:   "commitMode",
			Message: "commit_all_or_fail is not supported because cross-action kernel batch transactions are not available",
		})
		result.Warnings = append(result.Warnings, "use commit_valid for deterministic partial commits or validate_only for no-write validation")
	default:
		p.processActions(ctx, req, event, proposed, req.DryRun, false, &result)
	}

	result.CommittedObjectIDs = uniqueStrings(result.CommittedObjectIDs)
	result.AuditIDs = uniqueStrings(result.AuditIDs)
	result.Summary.AcceptedCount = len(result.AcceptedActions)
	result.Summary.RejectedCount = len(result.RejectedActions)
	result.Summary.CommittedCount = len(result.CommittedObjectIDs)
	if len(result.RejectedActions) > 0 {
		result.Success = false
	}
	p.runAutonomyPass(ctx, req, &result)
	return result, nil
}

func (p *IngestPipeline) runAutonomyPass(ctx context.Context, req domain.IngestRequest, result *domain.IngestResult) {
	if p.autonomyPass == nil || result == nil {
		return
	}
	if req.DryRun || req.CommitMode == domain.IngestValidateOnly {
		result.Warnings = append(result.Warnings, "autonomy pass skipped in dry-run/validate-only mode")
		return
	}
	depth := ingestAutonomyDepth(req)
	if depth >= p.maxAutonomyDepth {
		result.Warnings = append(result.Warnings, fmt.Sprintf("autonomy pass depth capped at %d", p.maxAutonomyDepth))
		return
	}
	runs, err := p.autonomyPass(ctx, req, *result, p.truthEngine, depth+1)
	if err != nil {
		result.Warnings = append(result.Warnings, "autonomy pass failed: "+err.Error())
		return
	}
	if len(runs) == 0 {
		return
	}
	result.AutonomyRuns = append(result.AutonomyRuns, runs...)
	if result.TruthDiagnostics == nil {
		result.TruthDiagnostics = map[string]any{}
	}
	result.TruthDiagnostics["autonomyRuns"] = runs
}

func (p *IngestPipeline) processActions(
	ctx context.Context,
	req domain.IngestRequest,
	event domain.JournalEvent,
	actions []domain.SyscallRequest,
	dryRun bool,
	stopOnFailure bool,
	result *domain.IngestResult,
) {
	for _, action := range actions {
		call := action
		call.DryRun = dryRun
		call = p.withIdempotency(req, call)
		if !p.processActionValidationSeams(ctx, req, event, call, result) {
			if stopOnFailure {
				result.Warnings = append(result.Warnings, "commit phase stopped after validation seam rejection")
				return
			}
			continue
		}
		res, err := p.kernel.Process(ctx, call)
		if err != nil {
			result.Success = false
			result.Errors = append(result.Errors, domain.IngestError{
				Code:    domain.IngestErrKernel,
				Field:   string(call.Action),
				Message: err.Error(),
			})
			if stopOnFailure {
				return
			}
			continue
		}
		outcome := domain.IngestActionOutcome{
			Action:         call,
			Result:         res,
			CellName:       readStringAny(call.Metadata, "cellName"),
			CellVersion:    readStringAny(call.Metadata, "cellVersion"),
			CandidateBatch: readStringAny(call.Metadata, "candidateBatch"),
		}
		if res.Success {
			result.AcceptedActions = append(result.AcceptedActions, outcome)
			result.CommittedObjectIDs = append(result.CommittedObjectIDs, res.CommittedObjectIDs...)
		} else {
			result.RejectedActions = append(result.RejectedActions, outcome)
			result.Success = false
			if stopOnFailure {
				result.Warnings = append(result.Warnings, "commit phase stopped after first failure in commit_all_or_fail mode")
				return
			}
		}
		if strings.TrimSpace(res.AuditID) != "" {
			result.AuditIDs = append(result.AuditIDs, res.AuditID)
		}
		if p.truthEngine != nil {
			applySummary, applyErr := p.truthEngine.ApplySyscallResult(ctx, call, res)
			if applyErr != nil {
				result.Warnings = append(result.Warnings, "truth engine apply diagnostics failed: "+applyErr.Error())
			} else {
				appendTruthApply(result, applySummary)
			}
		}
		_ = event
	}
}

func (p *IngestPipeline) processActionValidationSeams(
	ctx context.Context,
	req domain.IngestRequest,
	event domain.JournalEvent,
	action domain.SyscallRequest,
	result *domain.IngestResult,
) bool {
	if result == nil || p.kernel == nil {
		return true
	}
	validations := p.validationSeamRequests(req, event, action)
	if len(validations) == 0 {
		return true
	}
	accepted := true
	for _, validation := range validations {
		call := p.withIdempotency(req, validation)
		res, err := p.kernel.Process(ctx, call)
		if err != nil {
			result.Success = false
			result.Errors = append(result.Errors, domain.IngestError{
				Code:    domain.IngestErrKernel,
				Field:   string(call.Action),
				Message: err.Error(),
			})
			accepted = false
			continue
		}
		appendValidationSeamDiagnostic(result, call, res)
		if strings.TrimSpace(res.AuditID) != "" {
			result.AuditIDs = append(result.AuditIDs, res.AuditID)
		}
		if !res.Success {
			result.Success = false
			result.RejectedActions = append(result.RejectedActions, domain.IngestActionOutcome{
				Action:         call,
				Result:         res,
				CellName:       readStringAny(action.Metadata, "cellName"),
				CellVersion:    readStringAny(action.Metadata, "cellVersion"),
				CandidateBatch: readStringAny(action.Metadata, "candidateBatch"),
			})
			accepted = false
		}
	}
	return accepted
}

func (p *IngestPipeline) validationSeamRequests(req domain.IngestRequest, event domain.JournalEvent, action domain.SyscallRequest) []domain.SyscallRequest {
	if isValidationSeamAction(action.Action) {
		return nil
	}
	workspaceID := firstNonEmptyString(action.Scope.WorkspaceID, req.Scope.WorkspaceID)
	if strings.TrimSpace(workspaceID) == "" {
		return nil
	}
	refs := validationRefs(workspaceID, event.ID, action.ID)
	if len(refs) == 0 {
		return nil
	}
	sourceRefs := []any{refs[0]}
	if len(refs) > 1 {
		sourceRefs = append(sourceRefs, refs[1:]...)
	}
	reasons := selectionReasonsForRefs(refs)
	out := []domain.SyscallRequest{
		p.validationSeamRequest(req, action, domain.ActionValidateRefShape, "ref-shape", map[string]any{
			"workspace_id": workspaceID,
			"refs":         refs,
		}),
		p.validationSeamRequest(req, action, domain.ActionCompareRefShape, "compare-ref-shape", map[string]any{
			"workspace_id":    workspaceID,
			"candidate_refs":  refs,
			"observed_refs":   refs,
			"comparison_mode": "candidate_preflight",
		}),
		p.validationSeamRequest(req, action, domain.ActionValidateSemanticOperation, "semantic-operation", map[string]any{
			"workspace_id":    workspaceID,
			"operation_type":  validationOperationType(action.Action),
			"source_refs":     sourceRefs,
			"derived_refs":    []any{objectRefPayload("semantic_operation", action.ID, workspaceID)},
			"provenance_refs": []any{objectRefPayload("diagnostic_report", event.ID, workspaceID)},
			"claims":          noAuthorityClaims(),
		}),
		p.validationSeamRequest(req, action, domain.ActionValidateAdmissionCandidate, "admission-candidate", map[string]any{
			"workspace_id":    workspaceID,
			"case_id":         "case-" + safeValidationID(action.ID),
			"admission_mode":  "admission_candidate",
			"evidence_refs":   refs,
			"source_refs":     sourceRefs,
			"provenance_refs": []any{objectRefPayload("diagnostic_report", event.ID, workspaceID)},
			"claims":          noAuthorityClaims(),
		}),
		p.validationSeamRequest(req, action, domain.ActionValidateContextAttribution, "context-attribution", map[string]any{
			"workspace_id":        workspaceID,
			"query":               validationAttributionQuery(req, action),
			"context_purpose":     "diagnostic_attribution",
			"source_refs":         refs,
			"selection_reasons":   reasons,
			"claims":              noAuthorityClaims(),
			"downstream_action":   string(action.Action),
			"candidate_action_id": action.ID,
		}),
	}
	if kvPayload, ok := kvIdentityValidationPayload(action.Metadata); ok {
		out = append(out, p.validationSeamRequest(req, action, domain.ActionValidateKVIdentity, "kv-identity", kvPayload))
	}
	return out
}

func (p *IngestPipeline) validationSeamRequest(req domain.IngestRequest, action domain.SyscallRequest, validationAction domain.SemanticActionType, suffix string, payload map[string]any) domain.SyscallRequest {
	scope := action.Scope
	if strings.TrimSpace(scope.WorkspaceID) == "" {
		scope = req.Scope
	}
	idBase := strings.TrimSpace(action.ID)
	if idBase == "" {
		idBase = "candidate-" + shortHash(string(action.Action), payloadFingerprint(action.Payload))
	}
	return domain.SyscallRequest{
		ID:     idBase + ":validation:" + suffix,
		Action: validationAction,
		Actor: domain.ActorIdentity{
			ID:   "forge.validation.seam",
			Kind: "system",
		},
		Source:  domain.SourceSystem,
		Scope:   scope,
		Payload: payload,
		Provenance: domain.Provenance{
			Actor:     "forge.validation.seam",
			ActorType: "system",
			Source:    "control_lane_validation_preflight",
			TraceID:   firstNonEmptyString(action.TraceID, req.TraceID),
		},
		CorrelationID: firstNonEmptyString(action.CorrelationID, req.CorrelationID),
		TraceID:       firstNonEmptyString(action.TraceID, req.TraceID),
		DryRun:        true,
		RequestedAt:   nonZeroInt64(action.RequestedAt, req.RequestedAt),
		Metadata: map[string]any{
			"validationPreflight": true,
			"parentActionId":      idBase,
			"parentAction":        string(action.Action),
		},
	}
}

func appendValidationSeamDiagnostic(result *domain.IngestResult, call domain.SyscallRequest, res domain.SyscallResult) {
	if result.TruthDiagnostics == nil {
		result.TruthDiagnostics = map[string]any{}
	}
	raw, _ := result.TruthDiagnostics["validationSeams"].([]map[string]any)
	raw = append(raw, map[string]any{
		"action":         string(call.Action),
		"requestId":      call.ID,
		"parentActionId": readStringAny(call.Metadata, "parentActionId"),
		"parentAction":   readStringAny(call.Metadata, "parentAction"),
		"success":        res.Success,
		"dryRun":         res.DryRun,
		"auditId":        res.AuditID,
		"stateSummary":   cloneMap(res.StateSummary),
	})
	result.TruthDiagnostics["validationSeams"] = raw
}

func isValidationSeamAction(action domain.SemanticActionType) bool {
	switch action {
	case domain.ActionValidateKVIdentity,
		domain.ActionValidateRefShape,
		domain.ActionCompareRefShape,
		domain.ActionValidateSemanticOperation,
		domain.ActionValidateAdmissionCandidate,
		domain.ActionValidateContextAttribution:
		return true
	default:
		return false
	}
}

func validationRefs(workspaceID, eventID, actionID string) []any {
	refs := []any{}
	if strings.TrimSpace(eventID) != "" {
		refs = append(refs, objectRefPayload("diagnostic_report", eventID, workspaceID))
	}
	if strings.TrimSpace(actionID) != "" {
		refs = append(refs, objectRefPayload("semantic_operation", actionID, workspaceID))
	}
	return refs
}

func objectRefPayload(refType, refID, workspaceID string) map[string]any {
	return map[string]any{
		"ref_type":     strings.TrimSpace(refType),
		"ref_id":       safeValidationID(refID),
		"workspace_id": strings.TrimSpace(workspaceID),
	}
}

func selectionReasonsForRefs(refs []any) map[string]any {
	out := map[string]any{}
	for _, item := range refs {
		ref, ok := item.(map[string]any)
		if !ok {
			continue
		}
		refType := strings.ToLower(strings.TrimSpace(fmt.Sprintf("%v", ref["ref_type"])))
		refID := strings.TrimSpace(fmt.Sprintf("%v", ref["ref_id"]))
		if refType == "" || refID == "" {
			continue
		}
		out[refType+":"+refID] = "candidate action preflight provenance"
	}
	return out
}

func validationOperationType(action domain.SemanticActionType) string {
	switch action {
	case domain.ActionCompileContext:
		return "context_prepare"
	case domain.ActionCreateLink:
		return "link"
	case domain.ActionRegisterContradict:
		return "contradiction_check"
	case domain.ActionMarkSuperseded, domain.ActionArchiveNote, domain.ActionCloseLoop:
		return "supersede"
	case domain.ActionDeriveModel:
		return "derive"
	case domain.ActionCreateNote, domain.ActionUpdateState, domain.ActionOpenLoop:
		return "classify"
	default:
		return "classify"
	}
}

func validationAttributionQuery(req domain.IngestRequest, action domain.SyscallRequest) string {
	if query := strings.TrimSpace(readStringAny(action.Payload, "query")); query != "" {
		return query
	}
	if content := strings.TrimSpace(req.Content); content != "" {
		return content
	}
	return "candidate action validation preflight"
}

func noAuthorityClaims() map[string]any {
	return map[string]any{
		"commit":                   false,
		"write_memory":             false,
		"memory_mutation":          false,
		"admit_evidence":           false,
		"call_modelruntime":        false,
		"gateway_execution":        false,
		"compile_context":          false,
		"live_kv_reuse":            false,
		"live_authority_migration": false,
	}
}

func kvIdentityValidationPayload(metadata map[string]any) (map[string]any, bool) {
	raw, ok := metadata["kvIdentityValidation"]
	if !ok {
		raw, ok = metadata["kv_identity_validation"]
	}
	payload, ok := raw.(map[string]any)
	if !ok {
		return nil, false
	}
	manifest, manifestOK := payload["manifest"].(map[string]any)
	request, requestOK := payload["request"].(map[string]any)
	if !manifestOK || !requestOK {
		return nil, false
	}
	out := cloneMap(payload)
	out["manifest"] = cloneMap(manifest)
	out["request"] = cloneMap(request)
	return out, true
}

func safeValidationID(value string) string {
	clean := strings.TrimSpace(value)
	if clean == "" {
		return "unknown"
	}
	var b strings.Builder
	for _, r := range clean {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' {
			b.WriteRune(r)
			continue
		}
		switch r {
		case '-', '_', ':', '.', '/':
			b.WriteRune(r)
		default:
			b.WriteRune('-')
		}
	}
	out := b.String()
	if len(out) > 240 {
		return out[:240]
	}
	return out
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func nonZeroInt64(values ...int64) int64 {
	for _, value := range values {
		if value > 0 {
			return value
		}
	}
	return domain.NowMillis()
}

func (p *IngestPipeline) appendOrVirtualizeEvent(ctx context.Context, req domain.IngestRequest) (domain.JournalEvent, bool, bool, error) {
	event := domain.JournalEvent{
		ID:        "ingest-event-" + shortHash(req.ID, req.CorrelationID),
		Type:      "ingest." + string(req.InputKind),
		Timestamp: req.RequestedAt,
		Source:    string(req.Source),
		Scope:     req.Scope,
		Payload: map[string]any{
			"content":    req.Content,
			"payload":    cloneMap(req.Payload),
			"metadata":   cloneMap(req.Metadata),
			"dryRun":     req.DryRun,
			"commitMode": req.CommitMode,
			"ingestId":   req.ID,
		},
		CorrelationID: req.CorrelationID,
		Provenance:    req.Provenance,
	}
	shouldPersist := !req.DryRun && req.CommitMode != domain.IngestValidateOnly
	if !shouldPersist {
		event.ID = "virtual-" + event.ID
		return event, false, false, nil
	}
	if p.repositories.Journal == nil {
		return event, false, false, fmt.Errorf("journal repository is required for commit modes")
	}
	if err := p.repositories.Journal.Append(ctx, event); err != nil {
		existing, ok, getErr := p.repositories.Journal.GetByID(ctx, event.ID)
		if getErr != nil {
			return event, false, false, err
		}
		if ok {
			return existing, true, true, nil
		}
		return event, false, false, err
	}
	return event, true, false, nil
}

func (p *IngestPipeline) buildCellContext(ctx context.Context, req domain.IngestRequest, event domain.JournalEvent, actions []domain.SyscallRequest) CellRunContext {
	scope := controllane.ScopeFilter{WorkspaceID: req.Scope.WorkspaceID, LaneID: req.Scope.LaneID}
	notes := []domain.MemoryNote{}
	loops := []domain.OpenLoop{}
	state := []domain.StateItem{}
	artifacts := []domain.ArtifactRef{}
	if p.repositories.Notes != nil {
		if rows, err := p.repositories.Notes.ListActive(ctx, scope); err == nil {
			notes = rows
		}
	}
	if p.repositories.Loops != nil {
		if rows, err := p.repositories.Loops.ListActive(ctx, scope, 120); err == nil {
			loops = rows
		}
	}
	if p.repositories.State != nil {
		if rows, err := p.repositories.State.ListCurrent(ctx, scope, 120); err == nil {
			state = rows
		}
	}
	if p.repositories.Artifacts != nil {
		if rows, err := p.repositories.Artifacts.ListByScope(ctx, scope, 120); err == nil {
			artifacts = rows
		}
	}
	flags := map[string]bool{}
	for k, v := range p.featureFlags {
		flags[k] = v
	}
	return CellRunContext{
		Request:         req,
		Event:           event,
		Scope:           req.Scope,
		Actor:           req.Actor,
		Source:          req.Source,
		Provenance:      req.Provenance,
		CorrelationID:   req.CorrelationID,
		TraceID:         req.TraceID,
		DryRun:          req.DryRun || req.CommitMode == domain.IngestValidateOnly,
		CurrentState:    state,
		ActiveNotes:     notes,
		ActiveLoops:     loops,
		RecentArtifacts: artifacts,
		ExistingActions: append([]domain.SyscallRequest{}, actions...),
		Repositories:    p.repositories,
		Semantic:        p.semantic,
		Truth:           p.truthEngine,
		FeatureFlags:    flags,
	}
}

func (p *IngestPipeline) normalizeRequest(req domain.IngestRequest) domain.IngestRequest {
	now := p.nowMillis()
	if strings.TrimSpace(req.ID) == "" {
		req.ID = fmt.Sprintf("ingest-%d", now)
	}
	if strings.TrimSpace(string(req.InputKind)) == "" {
		req.InputKind = domain.IngestSystemEvent
	}
	if strings.TrimSpace(string(req.Source)) == "" {
		req.Source = domain.SourceSystem
	}
	if strings.TrimSpace(req.Actor.ID) == "" {
		req.Actor.ID = "forge.ingest.pipeline"
	}
	if strings.TrimSpace(req.Actor.Kind) == "" {
		req.Actor.Kind = string(req.Source)
	}
	if req.Payload == nil {
		req.Payload = map[string]any{}
	}
	if req.Metadata == nil {
		req.Metadata = map[string]any{}
	}
	if strings.TrimSpace(req.Provenance.Actor) == "" {
		req.Provenance.Actor = req.Actor.ID
	}
	if strings.TrimSpace(req.Provenance.ActorType) == "" {
		req.Provenance.ActorType = req.Actor.Kind
	}
	if strings.TrimSpace(req.Provenance.Source) == "" {
		req.Provenance.Source = string(req.Source)
	}
	if req.RequestedAt <= 0 {
		req.RequestedAt = now
	}
	if strings.TrimSpace(req.CorrelationID) == "" {
		req.CorrelationID = "corr-" + req.ID
	}
	if strings.TrimSpace(req.TraceID) == "" {
		req.TraceID = req.CorrelationID
	}
	if strings.TrimSpace(req.Provenance.TraceID) == "" {
		req.Provenance.TraceID = req.TraceID
	}
	if req.CommitMode == "" {
		req.CommitMode = domain.IngestCommitValid
	}
	return req
}

func (p *IngestPipeline) annotateAction(req domain.IngestRequest, event domain.JournalEvent, action domain.SyscallRequest, cellName, cellVersion, batchID string) domain.SyscallRequest {
	next := action
	if strings.TrimSpace(next.ID) == "" {
		next.ID = "action-" + shortHash(req.ID, event.ID, cellName, string(next.Action), payloadFingerprint(next.Payload))
	}
	if strings.TrimSpace(string(next.Source)) == "" {
		next.Source = domain.SourceInternal
	}
	if strings.TrimSpace(next.Actor.ID) == "" {
		next.Actor.ID = "cell." + strings.ToLower(strings.TrimSuffix(cellName, "Cell"))
	}
	if strings.TrimSpace(next.Actor.Kind) == "" {
		next.Actor.Kind = string(next.Source)
	}
	if strings.TrimSpace(next.Scope.WorkspaceID) == "" {
		next.Scope = req.Scope
	}
	if strings.TrimSpace(next.CorrelationID) == "" {
		next.CorrelationID = req.CorrelationID
	}
	if strings.TrimSpace(next.TraceID) == "" {
		next.TraceID = req.TraceID
	}
	if next.RequestedAt <= 0 {
		next.RequestedAt = req.RequestedAt
	}
	next.Provenance = withCellProvenance(cellName, next.Provenance, req.TraceID)
	next.Metadata = mergeMetadata(next.Metadata, map[string]any{
		"cellName":        cellName,
		"cellVersion":     cellVersion,
		"eventId":         event.ID,
		"candidateBatch":  batchID,
		"ingestId":        req.ID,
		"correlationId":   req.CorrelationID,
		"proposedByCells": uniqueStrings(append(readStringSliceAny(next.Metadata, "proposedByCells"), cellName)),
	})
	return next
}

func (p *IngestPipeline) withIdempotency(req domain.IngestRequest, action domain.SyscallRequest) domain.SyscallRequest {
	if strings.TrimSpace(action.IdempotencyKey) != "" {
		return action
	}
	base := strings.TrimSpace(req.IdempotencyKey)
	if base == "" {
		base = strings.TrimSpace(req.ID)
	}
	if base == "" {
		base = "ingest"
	}
	action.IdempotencyKey = base + ":" + shortHash(string(action.Action), payloadFingerprint(action.Payload), action.ID)
	return action
}

func (p *IngestPipeline) orderAndDedupe(actions []domain.SyscallRequest) []domain.SyscallRequest {
	if len(actions) == 0 {
		return nil
	}
	byID := map[string]domain.SyscallRequest{}
	orderedIDs := []string{}
	for _, action := range actions {
		id := strings.TrimSpace(action.ID)
		if id == "" {
			id = "anon-" + shortHash(string(action.Action), payloadFingerprint(action.Payload))
			action.ID = id
		}
		if existing, ok := byID[id]; ok {
			action.Metadata = mergeActionMetadata(existing.Metadata, action.Metadata)
		}
		if _, seen := byID[id]; !seen {
			orderedIDs = append(orderedIDs, id)
		}
		byID[id] = action
	}
	collapsed := make([]domain.SyscallRequest, 0, len(byID))
	for _, id := range orderedIDs {
		collapsed = append(collapsed, byID[id])
	}
	semanticIdx := map[string]int{}
	out := []domain.SyscallRequest{}
	for _, action := range collapsed {
		key := semanticKey(action)
		if idx, ok := semanticIdx[key]; ok {
			out[idx].Metadata = mergeActionMetadata(out[idx].Metadata, action.Metadata)
			continue
		}
		semanticIdx[key] = len(out)
		out = append(out, action)
	}
	sort.SliceStable(out, func(i, j int) bool {
		ri := actionOrder(out[i].Action)
		rj := actionOrder(out[j].Action)
		if ri != rj {
			return ri < rj
		}
		return out[i].ID < out[j].ID
	})
	return out
}

func semanticKey(action domain.SyscallRequest) string {
	switch action.Action {
	case domain.ActionCreateNote:
		return fmt.Sprintf("%s|%s|%s|%s|%s",
			action.Action,
			action.Scope.WorkspaceID,
			strings.ToLower(strings.TrimSpace(readPayloadString(action.Payload, "type"))),
			strings.ToLower(strings.TrimSpace(readPayloadString(action.Payload, "title"))),
			strings.ToLower(strings.TrimSpace(readPayloadString(action.Payload, "content"))),
		)
	case domain.ActionCreateLink:
		return fmt.Sprintf("%s|%s|%s|%s|%s",
			action.Action,
			action.Scope.WorkspaceID,
			strings.ToLower(strings.TrimSpace(readPayloadString(action.Payload, "type"))),
			strings.TrimSpace(readPayloadString(action.Payload, "sourceId")),
			strings.TrimSpace(readPayloadString(action.Payload, "targetId")),
		)
	case domain.ActionUpdateState:
		return fmt.Sprintf("%s|%s|%s|%s",
			action.Action,
			action.Scope.WorkspaceID,
			strings.ToLower(strings.TrimSpace(readPayloadString(action.Payload, "key"))),
			strings.ToLower(strings.TrimSpace(fmt.Sprintf("%v", action.Payload["value"]))),
		)
	case domain.ActionOpenLoop:
		return fmt.Sprintf("%s|%s|%s|%s",
			action.Action,
			action.Scope.WorkspaceID,
			strings.ToLower(strings.TrimSpace(readPayloadString(action.Payload, "id"))),
			strings.ToLower(strings.TrimSpace(readPayloadString(action.Payload, "state"))),
		)
	default:
		return fmt.Sprintf("%s|%s|%s", action.Action, action.Scope.WorkspaceID, payloadFingerprint(action.Payload))
	}
}

func mergeActionMetadata(a, b map[string]any) map[string]any {
	out := mergeMetadata(a, b)
	out["proposedByCells"] = uniqueStrings(append(readStringSliceAny(a, "proposedByCells"), readStringSliceAny(b, "proposedByCells")...))
	return out
}

func actionOrder(action domain.SemanticActionType) int {
	switch action {
	case domain.ActionCreateNote:
		return 10
	case domain.ActionOpenLoop:
		return 20
	case domain.ActionUpdateState:
		return 30
	case domain.ActionCreateLink:
		return 40
	case domain.ActionMarkSuperseded:
		return 50
	case domain.ActionRegisterContradict:
		return 60
	case domain.ActionDeriveModel:
		return 70
	case domain.ActionArchiveNote:
		return 80
	case domain.ActionCloseLoop:
		return 90
	case domain.ActionCompileContext:
		return 100
	default:
		return 999
	}
}

func shortHash(parts ...string) string {
	sum := sha1.Sum([]byte(strings.Join(parts, "|")))
	return hex.EncodeToString(sum[:8])
}

func appendTruthApply(result *domain.IngestResult, summary domain.TruthApplySummary) {
	if result.TruthDiagnostics == nil {
		result.TruthDiagnostics = map[string]any{}
	}
	entries, _ := result.TruthDiagnostics["apply"].([]domain.TruthApplySummary)
	entries = append(entries, summary)
	result.TruthDiagnostics["apply"] = entries
}

func ingestAutonomyDepth(req domain.IngestRequest) int {
	if req.Metadata == nil {
		return 0
	}
	raw, ok := req.Metadata["autonomyDepth"]
	if !ok || raw == nil {
		return 0
	}
	switch v := raw.(type) {
	case int:
		return v
	case int32:
		return int(v)
	case int64:
		return int(v)
	case float64:
		return int(v)
	case float32:
		return int(v)
	default:
		return 0
	}
}

func nonZeroInt(v, fallback int) int {
	if v > 0 {
		return v
	}
	return fallback
}

func (p *IngestPipeline) validateCellDependencies() []domain.IngestError {
	if len(p.cells) == 0 {
		return nil
	}
	type mark int
	const (
		markNone mark = iota
		markTemp
		markDone
	)
	byName := map[string]RuntimeCell{}
	state := map[string]mark{}
	var errs []domain.IngestError
	for _, cell := range p.cells {
		name := strings.TrimSpace(cell.Name())
		if name == "" {
			errs = append(errs, domain.IngestError{
				Code:    domain.IngestErrCellDependency,
				Field:   "cells.name",
				Message: "cell name cannot be empty",
			})
			continue
		}
		if _, exists := byName[name]; exists {
			errs = append(errs, domain.IngestError{
				Code:    domain.IngestErrCellDependency,
				Field:   "cells.name",
				Message: "duplicate cell name: " + name,
			})
			continue
		}
		byName[name] = cell
	}
	var visit func(name string, stack []string)
	visit = func(name string, stack []string) {
		switch state[name] {
		case markDone:
			return
		case markTemp:
			cycle := append(stack, name)
			errs = append(errs, domain.IngestError{
				Code:    domain.IngestErrCellDependency,
				Field:   "cells.dependencies",
				Message: "dependency cycle detected: " + strings.Join(cycle, " -> "),
			})
			return
		}
		cell, ok := byName[name]
		if !ok {
			errs = append(errs, domain.IngestError{
				Code:    domain.IngestErrCellDependency,
				Field:   "cells.dependencies",
				Message: "missing dependency cell: " + name,
			})
			return
		}
		state[name] = markTemp
		nextStack := append(stack, name)
		for _, dep := range cell.Dependencies() {
			dep = strings.TrimSpace(dep)
			if dep == "" {
				continue
			}
			if _, ok := byName[dep]; !ok {
				errs = append(errs, domain.IngestError{
					Code:    domain.IngestErrCellDependency,
					Field:   "cells.dependencies",
					Message: "unknown dependency " + dep + " for cell " + name,
				})
				continue
			}
			visit(dep, nextStack)
		}
		state[name] = markDone
	}
	for name := range byName {
		visit(name, nil)
	}
	return errs
}
