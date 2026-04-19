package truth

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"forge/projectforge/services/core/internal/aios/controllane"
	"forge/projectforge/services/core/internal/aios/domain"
)

type TruthEngine interface {
	ApplySyscallResult(ctx context.Context, req domain.SyscallRequest, result domain.SyscallResult) (domain.TruthApplySummary, error)
	RebuildProjection(ctx context.Context, query domain.TruthQuery, dryRun bool) (domain.ProjectionRebuildReport, error)
	GetCurrentTruth(ctx context.Context, query domain.TruthQuery) (domain.TruthExplanation, error)
	GetTruthTimeline(ctx context.Context, query domain.TruthQuery) ([]domain.StateTimelineEntry, error)
	GetTruthEvidence(ctx context.Context, query domain.TruthQuery) (domain.TruthExplanation, error)
	ExplainCurrentTruth(ctx context.Context, query domain.TruthQuery) (domain.TruthExplanation, error)
}

type StateProjectionService interface {
	ProjectStateUpdate(ctx context.Context, req domain.SyscallRequest, result domain.SyscallResult) (*domain.StateItem, error)
	GetCurrentState(ctx context.Context, key string, scope domain.ForgeScope) (domain.StateItem, bool, error)
	ListCurrentState(ctx context.Context, scope domain.ForgeScope, limit int) ([]domain.StateItem, error)
	GetStateTimeline(ctx context.Context, key string, scope domain.ForgeScope, limit int) ([]domain.StateTimelineEntry, error)
	GetRecentlyChangedState(ctx context.Context, scope domain.ForgeScope, limit int) ([]domain.StateItem, error)
	ExplainState(ctx context.Context, key string, scope domain.ForgeScope) (domain.StateExplanation, error)
}

type OpenLoopLifecycleService interface {
	OpenLoop(ctx context.Context, req LoopMutationRequest) (domain.SyscallResult, error)
	TransitionLoop(ctx context.Context, req LoopMutationRequest) (domain.SyscallResult, error)
	ResolveLoop(ctx context.Context, req LoopMutationRequest) (domain.SyscallResult, error)
	BlockLoop(ctx context.Context, req LoopMutationRequest) (domain.SyscallResult, error)
	ReopenLoop(ctx context.Context, req LoopMutationRequest) (domain.SyscallResult, error)
	ArchiveLoop(ctx context.Context, req LoopMutationRequest) (domain.SyscallResult, error)
	ListActiveLoops(ctx context.Context, scope domain.ForgeScope, limit int) ([]domain.OpenLoop, error)
	ListBlockedLoops(ctx context.Context, scope domain.ForgeScope, limit int) ([]domain.OpenLoop, error)
	ListLoopsByPriority(ctx context.Context, scope domain.ForgeScope, priority string, limit int) ([]domain.OpenLoop, error)
	ListLoopsByOwner(ctx context.Context, scope domain.ForgeScope, owner string, limit int) ([]domain.OpenLoop, error)
	ListStaleLoops(ctx context.Context, scope domain.ForgeScope, cutoffMillis int64, limit int) ([]domain.OpenLoop, error)
	ExplainLoop(ctx context.Context, loopID string, scope domain.ForgeScope, cutoffMillis int64) (domain.OpenLoopExplanation, error)
}

type ContradictionService interface {
	RegisterContradiction(ctx context.Context, req ContradictionRequest) (domain.SyscallResult, error)
	ListContradictionsForObject(ctx context.Context, objectID string, scope domain.ForgeScope, limit int) ([]controllane.ContradictionRecord, error)
	ListContradictionsByScope(ctx context.Context, scope domain.ForgeScope, limit int) ([]controllane.ContradictionRecord, error)
	CalculateSeverity(reason string, leftKind, rightKind string) string
	ExplainContradiction(ctx context.Context, recordID string) (domain.ContradictionExplanation, error)
}

type SupersessionService interface {
	MarkSuperseded(ctx context.Context, req SupersessionRequest) (domain.SyscallResult, error)
	GetSuccessor(ctx context.Context, objectID string, scope domain.ForgeScope) (controllane.SupersessionRecord, bool, error)
	GetSupersessionChain(ctx context.Context, objectID string, scope domain.ForgeScope, limit int) ([]controllane.SupersessionRecord, error)
	IsCurrentObject(ctx context.Context, objectID string, scope domain.ForgeScope) (bool, error)
	ExplainSupersession(ctx context.Context, objectID string, scope domain.ForgeScope) (domain.SupersessionExplanation, error)
}

type CurrentObjectResolver interface {
	Resolve(ctx context.Context, objectID string, scope domain.ForgeScope) (domain.CurrentObjectResolution, error)
	ResolveMany(ctx context.Context, objectIDs []string, scope domain.ForgeScope) ([]domain.CurrentObjectResolution, error)
	FilterCurrent(ctx context.Context, objectIDs []string, scope domain.ForgeScope) ([]string, error)
	ExplainResolution(ctx context.Context, objectID string, scope domain.ForgeScope) (domain.CurrentObjectResolution, error)
}

type Repositories struct {
	State         controllane.StateRepository
	Loops         controllane.OpenLoopRepository
	Notes         controllane.MemoryNoteRepository
	Models        controllane.DerivedModelRepository
	Contradiction controllane.ContradictionRepository
	Supersession  controllane.SupersessionRepository
}

type EngineOptions struct {
	Kernel           controllane.ForgeKernelProcessor
	Repositories     Repositories
	NowMillis        func() int64
	StaleAfterMillis int64
}

type Engine struct {
	kernel           controllane.ForgeKernelProcessor
	repos            Repositories
	nowMillis        func() int64
	staleAfterMillis int64
}

func NewEngine(opts EngineOptions) *Engine {
	nowFn := opts.NowMillis
	if nowFn == nil {
		nowFn = domain.NowMillis
	}
	staleAfter := opts.StaleAfterMillis
	if staleAfter <= 0 {
		staleAfter = int64((72 * time.Hour).Milliseconds())
	}
	return &Engine{
		kernel:           opts.Kernel,
		repos:            opts.Repositories,
		nowMillis:        nowFn,
		staleAfterMillis: staleAfter,
	}
}

type MutationBase struct {
	Actor          domain.ActorIdentity
	Source         domain.ActionSource
	Scope          domain.ForgeScope
	Provenance     domain.Provenance
	CorrelationID  string
	TraceID        string
	RequestedAt    int64
	IdempotencyKey string
	Metadata       map[string]any
}

type LoopMutationRequest struct {
	MutationBase
	LoopID       string
	Title        string
	NextState    domain.OpenLoopState
	Priority     string
	Owner        string
	Blocker      string
	NextAction   string
	RelatedNotes []string
	CreatedFrom  string
	Reason       string
	Outcome      string
}

type ContradictionRequest struct {
	MutationBase
	LeftObjectID    string
	LeftObjectKind  string
	RightObjectID   string
	RightObjectKind string
	Reason          string
	Severity        string
	Confidence      float64
}

type SupersessionRequest struct {
	MutationBase
	OldObjectID   string
	OldObjectKind string
	NewObjectID   string
	NewObjectKind string
	Reason        string
}

var (
	_ TruthEngine              = (*Engine)(nil)
	_ StateProjectionService   = (*Engine)(nil)
	_ OpenLoopLifecycleService = (*Engine)(nil)
	_ ContradictionService     = (*Engine)(nil)
	_ SupersessionService      = (*Engine)(nil)
	_ CurrentObjectResolver    = (*Engine)(nil)
)

func (e *Engine) ApplySyscallResult(_ context.Context, req domain.SyscallRequest, result domain.SyscallResult) (domain.TruthApplySummary, error) {
	return domain.TruthApplySummary{
		Action:             req.Action,
		RequestID:          req.ID,
		Success:            result.Success,
		Scope:              req.Scope,
		CommittedObjectIDs: append([]string{}, result.CommittedObjectIDs...),
		Warnings:           append([]string{}, result.Warnings...),
	}, nil
}

func (e *Engine) GetCurrentState(ctx context.Context, key string, scope domain.ForgeScope) (domain.StateItem, bool, error) {
	if e.repos.State == nil {
		return domain.StateItem{}, false, domain.TruthError{Code: domain.TruthErrUnsupported, Field: "state", Message: "state repository is not configured"}
	}
	return e.repos.State.GetCurrent(ctx, strings.TrimSpace(key), toScope(scope))
}

func (e *Engine) ListCurrentState(ctx context.Context, scope domain.ForgeScope, limit int) ([]domain.StateItem, error) {
	if e.repos.State == nil {
		return nil, domain.TruthError{Code: domain.TruthErrUnsupported, Field: "state", Message: "state repository is not configured"}
	}
	return e.repos.State.ListCurrent(ctx, toScope(scope), limit)
}

func (e *Engine) GetRecentlyChangedState(ctx context.Context, scope domain.ForgeScope, limit int) ([]domain.StateItem, error) {
	if e.repos.State == nil {
		return nil, domain.TruthError{Code: domain.TruthErrUnsupported, Field: "state", Message: "state repository is not configured"}
	}
	return e.repos.State.ListRecentlyChanged(ctx, toScope(scope), limit)
}

func (e *Engine) ProjectStateUpdate(ctx context.Context, req domain.SyscallRequest, result domain.SyscallResult) (*domain.StateItem, error) {
	if req.Action != domain.ActionUpdateState || !result.Success {
		return nil, nil
	}
	key := strings.TrimSpace(fmt.Sprintf("%v", req.Payload["key"]))
	if key == "" {
		return nil, nil
	}
	item, ok, err := e.GetCurrentState(ctx, key, req.Scope)
	if err != nil || !ok {
		return nil, err
	}
	return &item, nil
}

func (e *Engine) GetStateTimeline(ctx context.Context, key string, scope domain.ForgeScope, limit int) ([]domain.StateTimelineEntry, error) {
	if e.repos.State == nil {
		return nil, domain.TruthError{Code: domain.TruthErrUnsupported, Field: "state", Message: "state repository is not configured"}
	}
	rows, err := e.repos.State.GetTimeline(ctx, strings.TrimSpace(key), toScope(scope), limit)
	if err != nil {
		return nil, err
	}
	out := make([]domain.StateTimelineEntry, 0, len(rows))
	for _, row := range rows {
		out = append(out, timelineEntry(row))
	}
	return out, nil
}

func (e *Engine) ExplainState(ctx context.Context, key string, scope domain.ForgeScope) (domain.StateExplanation, error) {
	if strings.TrimSpace(scope.WorkspaceID) == "" {
		return domain.StateExplanation{}, domain.TruthError{Code: domain.TruthErrMissingScope, Field: "scope.workspaceId", Message: "workspace scope is required"}
	}
	current, ok, err := e.GetCurrentState(ctx, key, scope)
	if err != nil {
		return domain.StateExplanation{}, err
	}
	if !ok {
		return domain.StateExplanation{
			Key:   key,
			Scope: scope,
		}, nil
	}
	timeline, err := e.GetStateTimeline(ctx, key, scope, 200)
	if err != nil {
		return domain.StateExplanation{}, err
	}
	contradictions := []string{}
	if e.repos.Contradiction != nil {
		rows, err := e.repos.Contradiction.ListByObject(ctx, current.ID, toScope(scope), 50)
		if err == nil {
			for _, row := range rows {
				contradictions = append(contradictions, row.ID)
			}
		}
	}
	supersessionIDs := []string{}
	if e.repos.Supersession != nil {
		chain, err := e.GetSupersessionChain(ctx, current.ID, scope, 30)
		if err == nil {
			for _, row := range chain {
				supersessionIDs = append(supersessionIDs, row.ID)
			}
		}
	}
	previous := []domain.StateTimelineEntry{}
	if len(timeline) > 1 {
		previous = append(previous, timeline[:len(timeline)-1]...)
	}
	return domain.StateExplanation{
		Key:               key,
		Scope:             scope,
		CurrentValue:      current.Value,
		CurrentStateID:    current.ID,
		UpdatedAt:         current.UpdatedAt,
		DerivedFrom:       append([]string{}, current.DerivedFrom...),
		PreviousValues:    previous,
		Contradictions:    contradictions,
		SupersessionChain: supersessionIDs,
	}, nil
}

func (e *Engine) OpenLoop(ctx context.Context, req LoopMutationRequest) (domain.SyscallResult, error) {
	call := e.buildLoopSyscall(domain.ActionOpenLoop, req)
	call.Payload["state"] = nonEmptyString(string(req.NextState), string(domain.LoopOpen))
	call.Payload["title"] = nonEmptyString(req.Title, "Open loop")
	call.Payload["createdFrom"] = nonEmptyString(req.CreatedFrom, req.CorrelationID)
	return e.process(ctx, call)
}

func (e *Engine) TransitionLoop(ctx context.Context, req LoopMutationRequest) (domain.SyscallResult, error) {
	next := req.NextState
	if next == domain.LoopResolved {
		return e.ResolveLoop(ctx, req)
	}
	call := e.buildLoopSyscall(domain.ActionOpenLoop, req)
	if strings.TrimSpace(req.LoopID) != "" {
		call.Payload["id"] = strings.TrimSpace(req.LoopID)
	}
	call.Payload["state"] = string(next)
	if strings.TrimSpace(req.Title) != "" {
		call.Payload["title"] = req.Title
	}
	if strings.TrimSpace(req.Reason) != "" {
		call.Payload["reason"] = req.Reason
	}
	return e.process(ctx, call)
}

func (e *Engine) ResolveLoop(ctx context.Context, req LoopMutationRequest) (domain.SyscallResult, error) {
	call := e.baseSyscall(req.MutationBase, domain.ActionCloseLoop, "truth-loop-resolve-"+strings.TrimSpace(req.LoopID))
	call.Payload = map[string]any{
		"loopId":  strings.TrimSpace(req.LoopID),
		"reason":  nonEmptyString(req.Reason, "resolved"),
		"outcome": strings.TrimSpace(req.Outcome),
	}
	return e.process(ctx, call)
}

func (e *Engine) BlockLoop(ctx context.Context, req LoopMutationRequest) (domain.SyscallResult, error) {
	req.NextState = domain.LoopBlocked
	return e.TransitionLoop(ctx, req)
}

func (e *Engine) ReopenLoop(ctx context.Context, req LoopMutationRequest) (domain.SyscallResult, error) {
	req.NextState = domain.LoopOpen
	return e.TransitionLoop(ctx, req)
}

func (e *Engine) ArchiveLoop(ctx context.Context, req LoopMutationRequest) (domain.SyscallResult, error) {
	req.NextState = domain.LoopArchived
	return e.TransitionLoop(ctx, req)
}

func (e *Engine) ListActiveLoops(ctx context.Context, scope domain.ForgeScope, limit int) ([]domain.OpenLoop, error) {
	if e.repos.Loops == nil {
		return nil, domain.TruthError{Code: domain.TruthErrUnsupported, Field: "loops", Message: "loop repository is not configured"}
	}
	return e.repos.Loops.ListActive(ctx, toScope(scope), limit)
}

func (e *Engine) ListBlockedLoops(ctx context.Context, scope domain.ForgeScope, limit int) ([]domain.OpenLoop, error) {
	if e.repos.Loops == nil {
		return nil, domain.TruthError{Code: domain.TruthErrUnsupported, Field: "loops", Message: "loop repository is not configured"}
	}
	return e.repos.Loops.ListByState(ctx, domain.LoopBlocked, toScope(scope), limit)
}

func (e *Engine) ListLoopsByPriority(ctx context.Context, scope domain.ForgeScope, priority string, limit int) ([]domain.OpenLoop, error) {
	if e.repos.Loops == nil {
		return nil, domain.TruthError{Code: domain.TruthErrUnsupported, Field: "loops", Message: "loop repository is not configured"}
	}
	return e.repos.Loops.ListByPriority(ctx, strings.TrimSpace(priority), toScope(scope), limit)
}

func (e *Engine) ListLoopsByOwner(ctx context.Context, scope domain.ForgeScope, owner string, limit int) ([]domain.OpenLoop, error) {
	if e.repos.Loops == nil {
		return nil, domain.TruthError{Code: domain.TruthErrUnsupported, Field: "loops", Message: "loop repository is not configured"}
	}
	owner = strings.TrimSpace(owner)
	loops, err := e.repos.Loops.ListActive(ctx, toScope(scope), nonZero(limit, 200))
	if err != nil || owner == "" {
		return loops, err
	}
	out := make([]domain.OpenLoop, 0, len(loops))
	for _, loop := range loops {
		if strings.EqualFold(strings.TrimSpace(loop.Owner), owner) {
			out = append(out, loop)
		}
	}
	return out, nil
}

func (e *Engine) ListStaleLoops(ctx context.Context, scope domain.ForgeScope, cutoffMillis int64, limit int) ([]domain.OpenLoop, error) {
	if e.repos.Loops == nil {
		return nil, domain.TruthError{Code: domain.TruthErrUnsupported, Field: "loops", Message: "loop repository is not configured"}
	}
	if cutoffMillis <= 0 {
		cutoffMillis = e.nowMillis() - e.staleAfterMillis
	}
	return e.repos.Loops.ListStale(ctx, cutoffMillis, toScope(scope), limit)
}

func (e *Engine) ExplainLoop(ctx context.Context, loopID string, scope domain.ForgeScope, cutoffMillis int64) (domain.OpenLoopExplanation, error) {
	if e.repos.Loops == nil {
		return domain.OpenLoopExplanation{}, domain.TruthError{Code: domain.TruthErrUnsupported, Field: "loops", Message: "loop repository is not configured"}
	}
	loop, ok, err := e.repos.Loops.GetByID(ctx, strings.TrimSpace(loopID))
	if err != nil {
		return domain.OpenLoopExplanation{}, err
	}
	if !ok {
		return domain.OpenLoopExplanation{}, domain.TruthError{Code: domain.TruthErrNotFound, Field: "loopId", Message: "loop not found"}
	}
	if !scopeMatches(scope, loop.Scope) {
		return domain.OpenLoopExplanation{}, domain.TruthError{Code: domain.TruthErrNotFound, Field: "loopId", Message: "loop not found in requested scope"}
	}
	if cutoffMillis <= 0 {
		cutoffMillis = e.nowMillis() - e.staleAfterMillis
	}
	return domain.OpenLoopExplanation{
		LoopID:        loop.ID,
		Scope:         loop.Scope,
		State:         loop.State,
		Priority:      loop.Priority,
		Owner:         loop.Owner,
		Blocker:       loop.Blocker,
		NextAction:    loop.NextAction,
		CreatedFrom:   loop.CreatedFrom,
		RelatedNotes:  append([]string{}, loop.RelatedNotes...),
		CreatedAt:     loop.CreatedAt,
		UpdatedAt:     loop.UpdatedAt,
		IsStale:       isLoopStale(loop, cutoffMillis),
		StaleCutoffMs: cutoffMillis,
	}, nil
}

func (e *Engine) RegisterContradiction(ctx context.Context, req ContradictionRequest) (domain.SyscallResult, error) {
	severity := strings.TrimSpace(req.Severity)
	if severity == "" {
		severity = e.CalculateSeverity(req.Reason, req.LeftObjectKind, req.RightObjectKind)
	}
	call := e.baseSyscall(req.MutationBase, domain.ActionRegisterContradict, "truth-contradiction-"+hashParts(req.LeftObjectID, req.RightObjectID, req.Reason))
	call.Payload = map[string]any{
		"leftObjectId":    strings.TrimSpace(req.LeftObjectID),
		"leftObjectKind":  nonEmptyString(req.LeftObjectKind, "object"),
		"rightObjectId":   strings.TrimSpace(req.RightObjectID),
		"rightObjectKind": nonEmptyString(req.RightObjectKind, "object"),
		"reason":          strings.TrimSpace(req.Reason),
		"severity":        severity,
		"confidence":      clampConfidence(req.Confidence, 0.65),
	}
	return e.process(ctx, call)
}

func (e *Engine) ListContradictionsForObject(ctx context.Context, objectID string, scope domain.ForgeScope, limit int) ([]controllane.ContradictionRecord, error) {
	if e.repos.Contradiction == nil {
		return nil, nil
	}
	return e.repos.Contradiction.ListByObject(ctx, strings.TrimSpace(objectID), toScope(scope), limit)
}

func (e *Engine) ListContradictionsByScope(ctx context.Context, scope domain.ForgeScope, limit int) ([]controllane.ContradictionRecord, error) {
	if e.repos.Contradiction == nil {
		return nil, nil
	}
	return e.repos.Contradiction.ListByScope(ctx, toScope(scope), limit)
}

func (e *Engine) CalculateSeverity(reason string, leftKind, rightKind string) string {
	text := strings.ToLower(strings.TrimSpace(reason))
	if leftKind == "state_item" && rightKind == "state_item" {
		return "high"
	}
	if strings.Contains(text, "instead of") || strings.Contains(text, "replace") || strings.Contains(text, "not anymore") || strings.Contains(text, "wrong") {
		return "high"
	}
	if strings.Contains(text, "tension") || strings.Contains(text, "mismatch") {
		return "medium"
	}
	return "low"
}

func (e *Engine) ExplainContradiction(ctx context.Context, recordID string) (domain.ContradictionExplanation, error) {
	if e.repos.Contradiction == nil {
		return domain.ContradictionExplanation{}, domain.TruthError{Code: domain.TruthErrUnsupported, Field: "contradiction", Message: "contradiction repository is not configured"}
	}
	row, ok, err := e.repos.Contradiction.GetByID(ctx, strings.TrimSpace(recordID))
	if err != nil {
		return domain.ContradictionExplanation{}, err
	}
	if !ok {
		return domain.ContradictionExplanation{}, domain.TruthError{Code: domain.TruthErrNotFound, Field: "recordId", Message: "contradiction record not found"}
	}
	return contradictionExplanation(row), nil
}

func (e *Engine) MarkSuperseded(ctx context.Context, req SupersessionRequest) (domain.SyscallResult, error) {
	if strings.TrimSpace(req.OldObjectID) == strings.TrimSpace(req.NewObjectID) {
		return domain.SyscallResult{
			Success:              false,
			Action:               domain.ActionMarkSuperseded,
			RequestID:            "truth-supersession-invalid",
			ApprovalStatus:       domain.ApprovalAllowed,
			RejectedReasons:      []domain.SyscallError{{Code: domain.ErrInvalidPayload, Field: "oldObjectId", Message: "old and new objects must differ"}},
			DeterministicErrCode: domain.ErrInvalidPayload,
		}, nil
	}
	chain, err := e.GetSupersessionChain(ctx, req.NewObjectID, req.Scope, 30)
	if err != nil {
		return domain.SyscallResult{}, err
	}
	for _, row := range chain {
		if strings.TrimSpace(row.NewID) == strings.TrimSpace(req.OldObjectID) {
			return domain.SyscallResult{
				Success:              false,
				Action:               domain.ActionMarkSuperseded,
				RequestID:            "truth-supersession-cycle",
				ApprovalStatus:       domain.ApprovalAllowed,
				RejectedReasons:      []domain.SyscallError{{Code: domain.ErrInvalidStateTransition, Field: "oldObjectId", Message: "supersession cycle detected"}},
				DeterministicErrCode: domain.ErrInvalidStateTransition,
			}, nil
		}
	}
	call := e.baseSyscall(req.MutationBase, domain.ActionMarkSuperseded, "truth-supersede-"+hashParts(req.OldObjectID, req.NewObjectID, req.Reason))
	call.Payload = map[string]any{
		"oldObjectId":   strings.TrimSpace(req.OldObjectID),
		"oldObjectKind": nonEmptyString(req.OldObjectKind, "object"),
		"newObjectId":   strings.TrimSpace(req.NewObjectID),
		"newObjectKind": nonEmptyString(req.NewObjectKind, "object"),
		"reason":        strings.TrimSpace(req.Reason),
	}
	return e.process(ctx, call)
}

func (e *Engine) GetSuccessor(ctx context.Context, objectID string, scope domain.ForgeScope) (controllane.SupersessionRecord, bool, error) {
	if e.repos.Supersession == nil {
		return controllane.SupersessionRecord{}, false, nil
	}
	return e.repos.Supersession.GetCurrentSuccessor(ctx, strings.TrimSpace(objectID), toScope(scope))
}

func (e *Engine) GetSupersessionChain(ctx context.Context, objectID string, scope domain.ForgeScope, limit int) ([]controllane.SupersessionRecord, error) {
	if e.repos.Supersession == nil {
		return nil, nil
	}
	if limit <= 0 {
		limit = 100
	}
	current := strings.TrimSpace(objectID)
	out := []controllane.SupersessionRecord{}
	seen := map[string]struct{}{}
	for len(out) < limit {
		row, ok, err := e.repos.Supersession.GetCurrentSuccessor(ctx, current, toScope(scope))
		if err != nil || !ok {
			return out, err
		}
		if _, exists := seen[row.ID]; exists {
			break
		}
		seen[row.ID] = struct{}{}
		out = append(out, row)
		current = row.NewID
	}
	return out, nil
}

func (e *Engine) IsCurrentObject(ctx context.Context, objectID string, scope domain.ForgeScope) (bool, error) {
	resolution, err := e.Resolve(ctx, objectID, scope)
	if err != nil {
		return false, err
	}
	return resolution.Current, nil
}

func (e *Engine) ExplainSupersession(ctx context.Context, objectID string, scope domain.ForgeScope) (domain.SupersessionExplanation, error) {
	chain, err := e.GetSupersessionChain(ctx, objectID, scope, 100)
	if err != nil {
		return domain.SupersessionExplanation{}, err
	}
	explain := domain.SupersessionExplanation{
		Scope:           scope,
		RootObjectID:    objectID,
		CurrentObjectID: objectID,
		Chain:           []string{objectID},
		Reasons:         []string{},
		RecordIDs:       []string{},
	}
	current := objectID
	for _, row := range chain {
		current = row.NewID
		explain.Chain = append(explain.Chain, row.NewID)
		explain.Reasons = append(explain.Reasons, row.Reason)
		explain.RecordIDs = append(explain.RecordIDs, row.ID)
	}
	explain.CurrentObjectID = current
	return explain, nil
}

func (e *Engine) Resolve(ctx context.Context, objectID string, scope domain.ForgeScope) (domain.CurrentObjectResolution, error) {
	objectID = strings.TrimSpace(objectID)
	out := domain.CurrentObjectResolution{
		ObjectID:        objectID,
		Scope:           scope,
		Current:         true,
		CurrentObjectID: objectID,
		IncludeInActive: true,
		Warnings:        []string{},
	}
	if objectID == "" {
		return out, domain.TruthError{Code: domain.TruthErrInvalidQuery, Field: "objectId", Message: "objectId is required"}
	}
	chain, err := e.GetSupersessionChain(ctx, objectID, scope, 100)
	if err != nil {
		return out, err
	}
	if len(chain) > 0 {
		out.Superseded = true
		out.Current = false
		out.IncludeInActive = false
		for _, row := range chain {
			out.SupersessionChain = append(out.SupersessionChain, row.NewID)
		}
		out.CurrentObjectID = chain[len(chain)-1].NewID
	}
	if e.repos.Contradiction != nil {
		rows, err := e.repos.Contradiction.ListByObject(ctx, objectID, toScope(scope), 20)
		if err == nil && len(rows) > 0 {
			out.Contradicted = true
			out.Warnings = append(out.Warnings, "object has contradiction records")
		}
	}
	if e.repos.Notes != nil {
		if note, ok, err := e.repos.Notes.GetByID(ctx, objectID); err == nil && ok {
			if !scopeMatches(scope, note.Scope) {
				return out, domain.TruthError{Code: domain.TruthErrNotFound, Field: "objectId", Message: "object not found in requested scope"}
			}
			if note.Status == domain.NoteArchived {
				out.Archived = true
				out.Current = false
				out.IncludeInActive = false
			}
			if note.Status == domain.NoteSuperseded {
				out.Superseded = true
				out.Current = false
				out.IncludeInActive = false
			}
			return out, nil
		}
	}
	if e.repos.Models != nil {
		if model, ok, err := e.repos.Models.GetByID(ctx, objectID); err == nil && ok {
			if !scopeMatches(scope, model.Scope) {
				return out, domain.TruthError{Code: domain.TruthErrNotFound, Field: "objectId", Message: "object not found in requested scope"}
			}
			if model.Status == domain.ModelDeprecated {
				out.Deprecated = true
				out.Current = false
				out.IncludeInActive = false
			}
			return out, nil
		}
	}
	if e.repos.Loops != nil {
		if loop, ok, err := e.repos.Loops.GetByID(ctx, objectID); err == nil && ok {
			if !scopeMatches(scope, loop.Scope) {
				return out, domain.TruthError{Code: domain.TruthErrNotFound, Field: "objectId", Message: "object not found in requested scope"}
			}
			if loop.State == domain.LoopResolved || loop.State == domain.LoopArchived {
				out.Current = false
				out.IncludeInActive = false
			}
			return out, nil
		}
	}
	if len(chain) > 0 {
		out.Warnings = append(out.Warnings, "supersession chain exists but root object was not found in repositories")
		return out, nil
	}
	return out, domain.TruthError{Code: domain.TruthErrNotFound, Field: "objectId", Message: "object not found in requested scope"}
}

func (e *Engine) ResolveMany(ctx context.Context, objectIDs []string, scope domain.ForgeScope) ([]domain.CurrentObjectResolution, error) {
	out := make([]domain.CurrentObjectResolution, 0, len(objectIDs))
	for _, id := range objectIDs {
		res, err := e.Resolve(ctx, id, scope)
		if err != nil {
			return nil, err
		}
		out = append(out, res)
	}
	return out, nil
}

func (e *Engine) FilterCurrent(ctx context.Context, objectIDs []string, scope domain.ForgeScope) ([]string, error) {
	out := []string{}
	for _, id := range objectIDs {
		res, err := e.Resolve(ctx, id, scope)
		if err != nil {
			return nil, err
		}
		if res.IncludeInActive {
			out = append(out, id)
		}
	}
	return out, nil
}

func (e *Engine) ExplainResolution(ctx context.Context, objectID string, scope domain.ForgeScope) (domain.CurrentObjectResolution, error) {
	return e.Resolve(ctx, objectID, scope)
}

func (e *Engine) GetCurrentTruth(ctx context.Context, query domain.TruthQuery) (domain.TruthExplanation, error) {
	return e.ExplainCurrentTruth(ctx, query)
}

func (e *Engine) GetTruthTimeline(ctx context.Context, query domain.TruthQuery) ([]domain.StateTimelineEntry, error) {
	if strings.TrimSpace(query.Key) == "" {
		return nil, domain.TruthError{Code: domain.TruthErrInvalidQuery, Field: "key", Message: "truth timeline requires key"}
	}
	return e.GetStateTimeline(ctx, query.Key, query.Scope, query.Limit)
}

func (e *Engine) GetTruthEvidence(ctx context.Context, query domain.TruthQuery) (domain.TruthExplanation, error) {
	return e.ExplainCurrentTruth(ctx, query)
}

func (e *Engine) ExplainCurrentTruth(ctx context.Context, query domain.TruthQuery) (domain.TruthExplanation, error) {
	if strings.TrimSpace(query.Scope.WorkspaceID) == "" {
		return domain.TruthExplanation{}, domain.TruthError{Code: domain.TruthErrMissingScope, Field: "scope.workspaceId", Message: "workspace scope is required"}
	}
	out := domain.TruthExplanation{
		Query:    query,
		Status:   "ok",
		Warnings: []string{},
	}
	if strings.TrimSpace(query.Key) != "" {
		explain, err := e.ExplainState(ctx, query.Key, query.Scope)
		if err != nil {
			return domain.TruthExplanation{}, err
		}
		out.CurrentState = &explain
		if query.IncludeHistory {
			out.Timeline = append([]domain.StateTimelineEntry{}, explain.PreviousValues...)
		}
	}
	if strings.TrimSpace(query.ObjectID) != "" {
		resolution, err := e.Resolve(ctx, query.ObjectID, query.Scope)
		if err != nil {
			return domain.TruthExplanation{}, err
		}
		out.CurrentObject = &resolution
		if query.IncludeSupersessions {
			sup, err := e.ExplainSupersession(ctx, query.ObjectID, query.Scope)
			if err == nil {
				out.Supersession = &sup
			}
		}
		if query.IncludeContradictions && e.repos.Contradiction != nil {
			rows, err := e.repos.Contradiction.ListByObject(ctx, query.ObjectID, toScope(query.Scope), query.Limit)
			if err == nil {
				for _, row := range rows {
					out.Contradictions = append(out.Contradictions, contradictionExplanation(row))
				}
			}
		}
	}
	if strings.TrimSpace(query.Key) == "" && strings.TrimSpace(query.ObjectID) == "" {
		loops, err := e.ListActiveLoops(ctx, query.Scope, nonZero(query.Limit, 50))
		if err == nil {
			for _, loop := range loops {
				out.Loops = append(out.Loops, domain.OpenLoopExplanation{
					LoopID:       loop.ID,
					Scope:        loop.Scope,
					State:        loop.State,
					Priority:     loop.Priority,
					Owner:        loop.Owner,
					Blocker:      loop.Blocker,
					NextAction:   loop.NextAction,
					CreatedFrom:  loop.CreatedFrom,
					RelatedNotes: append([]string{}, loop.RelatedNotes...),
					CreatedAt:    loop.CreatedAt,
					UpdatedAt:    loop.UpdatedAt,
					IsStale:      isLoopStale(loop, e.nowMillis()-e.staleAfterMillis),
				})
			}
		}
		sort.SliceStable(out.Loops, func(i, j int) bool { return out.Loops[i].UpdatedAt > out.Loops[j].UpdatedAt })
	}
	return out, nil
}

func (e *Engine) RebuildProjection(ctx context.Context, query domain.TruthQuery, dryRun bool) (domain.ProjectionRebuildReport, error) {
	report := domain.ProjectionRebuildReport{
		Scope:       query.Scope,
		DryRun:      dryRun,
		GeneratedAt: e.nowMillis(),
		Differences: []domain.ProjectionRebuildDiff{},
		Warnings:    []string{},
		Applied:     false,
	}
	if strings.TrimSpace(query.Scope.WorkspaceID) == "" {
		return report, domain.TruthError{Code: domain.TruthErrMissingScope, Field: "scope.workspaceId", Message: "workspace scope is required"}
	}
	if e.repos.State != nil {
		keys, err := e.repos.State.ListHistoryKeys(ctx, toScope(query.Scope), nonZero(query.Limit, 500))
		if err != nil {
			return report, err
		}
		for _, key := range keys {
			timeline, err := e.repos.State.GetTimeline(ctx, key, toScope(query.Scope), 500)
			if err != nil || len(timeline) == 0 {
				continue
			}
			latest := timeline[len(timeline)-1]
			current, ok, err := e.repos.State.GetCurrent(ctx, key, toScope(query.Scope))
			if err != nil {
				return report, err
			}
			if !ok {
				report.Differences = append(report.Differences, domain.ProjectionRebuildDiff{
					Category: "state_missing_current",
					Key:      key,
					Message:  "state history exists but current projection row is missing",
					Severity: "high",
					Metadata: map[string]any{"latestHistoryValue": latest.NewValue},
				})
				continue
			}
			if !mapsEqual(current.Value, latest.NewValue) {
				report.Differences = append(report.Differences, domain.ProjectionRebuildDiff{
					Category: "state_value_mismatch",
					Key:      key,
					ObjectID: current.ID,
					Message:  "current state value differs from latest timeline value",
					Severity: "medium",
					Metadata: map[string]any{"current": current.Value, "historyLatest": latest.NewValue},
				})
			}
		}
	}
	if e.repos.Contradiction != nil {
		rows, err := e.repos.Contradiction.ListByScope(ctx, toScope(query.Scope), nonZero(query.Limit, 500))
		if err != nil {
			return report, err
		}
		for _, row := range rows {
			if !e.objectExists(ctx, row.LeftID, query.Scope) || !e.objectExists(ctx, row.RightID, query.Scope) {
				report.Differences = append(report.Differences, domain.ProjectionRebuildDiff{
					Category: "orphan_contradiction",
					ObjectID: row.ID,
					Message:  "contradiction references missing object(s)",
					Severity: "low",
					Metadata: map[string]any{"leftObjectId": row.LeftID, "rightObjectId": row.RightID},
				})
			}
		}
	}
	report.Warnings = append(report.Warnings, "loop transition history is not persisted as a dedicated timeline table; rebuild reports current loop-state anomalies only")
	if !dryRun {
		report.Applied = len(report.Differences) == 0
	}
	return report, nil
}

func (e *Engine) process(ctx context.Context, req domain.SyscallRequest) (domain.SyscallResult, error) {
	if e.kernel == nil {
		return domain.SyscallResult{}, domain.TruthError{Code: domain.TruthErrUnsupported, Field: "kernel", Message: "kernel processor is required for truth mutations"}
	}
	return e.kernel.Process(ctx, req)
}

func (e *Engine) baseSyscall(base MutationBase, action domain.SemanticActionType, fallbackID string) domain.SyscallRequest {
	now := e.nowMillis()
	req := domain.SyscallRequest{
		ID:     nonEmptyString(strings.TrimSpace(fallbackID), fmt.Sprintf("truth-%d", now)),
		Action: action,
		Actor: domain.ActorIdentity{
			ID:   nonEmptyString(base.Actor.ID, "forge.truth.engine"),
			Kind: nonEmptyString(base.Actor.Kind, string(domain.SourceSystem)),
		},
		Source:         nonEmptySource(base.Source, domain.SourceSystem),
		Scope:          base.Scope,
		Provenance:     base.Provenance,
		CorrelationID:  nonEmptyString(base.CorrelationID, "corr-"+fmt.Sprintf("%d", now)),
		TraceID:        nonEmptyString(base.TraceID, nonEmptyString(base.CorrelationID, "trace-"+fmt.Sprintf("%d", now))),
		RequestedAt:    nonZeroInt64(base.RequestedAt, now),
		IdempotencyKey: strings.TrimSpace(base.IdempotencyKey),
		Metadata:       cloneMap(base.Metadata),
	}
	if strings.TrimSpace(req.Provenance.Actor) == "" {
		req.Provenance.Actor = req.Actor.ID
	}
	if strings.TrimSpace(req.Provenance.ActorType) == "" {
		req.Provenance.ActorType = "truth_engine"
	}
	if strings.TrimSpace(req.Provenance.Source) == "" {
		req.Provenance.Source = "control.truth"
	}
	if strings.TrimSpace(req.Provenance.TraceID) == "" {
		req.Provenance.TraceID = req.TraceID
	}
	return req
}

func (e *Engine) buildLoopSyscall(action domain.SemanticActionType, req LoopMutationRequest) domain.SyscallRequest {
	call := e.baseSyscall(req.MutationBase, action, "truth-loop-"+hashParts(req.LoopID, string(req.NextState), req.Title))
	call.Payload = map[string]any{
		"id":           strings.TrimSpace(req.LoopID),
		"title":        strings.TrimSpace(req.Title),
		"state":        string(req.NextState),
		"priority":     nonEmptyString(req.Priority, "medium"),
		"owner":        nonEmptyString(req.Owner, call.Actor.ID),
		"blocker":      strings.TrimSpace(req.Blocker),
		"nextAction":   strings.TrimSpace(req.NextAction),
		"relatedNotes": append([]string{}, req.RelatedNotes...),
		"createdFrom":  strings.TrimSpace(req.CreatedFrom),
		"reason":       strings.TrimSpace(req.Reason),
	}
	return call
}

func (e *Engine) objectExists(ctx context.Context, objectID string, scope domain.ForgeScope) bool {
	objectID = strings.TrimSpace(objectID)
	if objectID == "" {
		return false
	}
	if e.repos.Notes != nil {
		if note, ok, err := e.repos.Notes.GetByID(ctx, objectID); err == nil && ok {
			return scopeMatches(scope, note.Scope)
		}
	}
	if e.repos.Loops != nil {
		if loop, ok, err := e.repos.Loops.GetByID(ctx, objectID); err == nil && ok {
			return scopeMatches(scope, loop.Scope)
		}
	}
	if e.repos.Models != nil {
		if model, ok, err := e.repos.Models.GetByID(ctx, objectID); err == nil && ok {
			return scopeMatches(scope, model.Scope)
		}
	}
	return false
}

func toScope(scope domain.ForgeScope) controllane.ScopeFilter {
	return controllane.ScopeFilter{
		WorkspaceID: scope.WorkspaceID,
		LaneID:      scope.LaneID,
	}
}

func contradictionExplanation(row controllane.ContradictionRecord) domain.ContradictionExplanation {
	return domain.ContradictionExplanation{
		RecordID:        row.ID,
		Scope:           domain.ForgeScope{WorkspaceID: row.WorkspaceID, LaneID: row.LaneID},
		LeftObjectID:    row.LeftID,
		LeftObjectKind:  row.LeftKind,
		RightObjectID:   row.RightID,
		RightObjectKind: row.RightKind,
		Reason:          row.Reason,
		Severity:        row.Severity,
		Confidence:      row.Confidence,
		CreatedAt:       row.CreatedAt,
		CorrelationID:   row.CorrelationID,
		TraceID:         row.TraceID,
		SyscallID:       row.SyscallID,
		AuditID:         row.AuditID,
	}
}

func timelineEntry(row controllane.StateVersionRecord) domain.StateTimelineEntry {
	return domain.StateTimelineEntry{
		VersionID:     row.ID,
		StateItemID:   row.StateItemID,
		Key:           row.StateKey,
		PreviousValue: cloneMap(row.PreviousValue),
		NewValue:      cloneMap(row.NewValue),
		ChangedBy:     row.ChangedBy,
		DerivedFrom:   append([]string{}, row.DerivedFrom...),
		SyscallID:     row.SyscallID,
		AuditID:       row.AuditID,
		CorrelationID: row.CorrelationID,
		TraceID:       row.TraceID,
		UpdatedAt:     row.CreatedAt,
		Metadata:      cloneMap(row.Metadata),
	}
}

func isLoopStale(loop domain.OpenLoop, cutoffMillis int64) bool {
	if loop.State == domain.LoopResolved || loop.State == domain.LoopArchived {
		return false
	}
	return loop.UpdatedAt < cutoffMillis
}

func hashParts(parts ...string) string {
	sum := 0
	for _, part := range parts {
		for i := 0; i < len(part); i++ {
			sum = (sum*31 + int(part[i])) % 1000000007
		}
	}
	return fmt.Sprintf("%x", sum)
}

func mapsEqual(a, b map[string]any) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if fmt.Sprintf("%v", v) != fmt.Sprintf("%v", b[k]) {
			return false
		}
	}
	return true
}

func nonZero(v, fallback int) int {
	if v > 0 {
		return v
	}
	return fallback
}

func nonZeroInt64(v, fallback int64) int64 {
	if v > 0 {
		return v
	}
	return fallback
}

func nonEmptyString(v, fallback string) string {
	v = strings.TrimSpace(v)
	if v != "" {
		return v
	}
	return strings.TrimSpace(fallback)
}

func nonEmptySource(v domain.ActionSource, fallback domain.ActionSource) domain.ActionSource {
	if strings.TrimSpace(string(v)) != "" {
		return v
	}
	return fallback
}

func clampConfidence(v, fallback float64) float64 {
	if v <= 0 {
		return fallback
	}
	if v > 1 {
		return 1
	}
	return v
}

func scopeMatches(expected, actual domain.ForgeScope) bool {
	if strings.TrimSpace(expected.WorkspaceID) == "" {
		return false
	}
	if strings.TrimSpace(expected.WorkspaceID) != strings.TrimSpace(actual.WorkspaceID) {
		return false
	}
	expectedLane := strings.TrimSpace(expected.LaneID)
	actualLane := strings.TrimSpace(actual.LaneID)
	if expectedLane == "" || actualLane == "" {
		return true
	}
	return expectedLane == actualLane
}

func cloneMap(in map[string]any) map[string]any {
	if in == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
