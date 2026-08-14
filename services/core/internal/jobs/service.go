package jobs

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"forge/projectforge/services/core/internal/adapters"
	"forge/projectforge/services/core/internal/aios/domain"
	"forge/projectforge/services/core/internal/approvals"
	"forge/projectforge/services/core/internal/artifacts"
	"forge/projectforge/services/core/internal/dossiers"
	"forge/projectforge/services/core/internal/events"
	"forge/projectforge/services/core/internal/gateway"
	"forge/projectforge/services/core/internal/ingest"
	"forge/projectforge/services/core/internal/packets"
	"forge/projectforge/services/core/internal/projectcontext"
	"forge/projectforge/services/core/internal/retrieval"
)

type jobMetadata struct {
	TemplateID             string         `json:"templateId"`
	UserRequest            string         `json:"userRequest"`
	Objective              string         `json:"objective"`
	Query                  string         `json:"query"`
	Scope                  ScopeInput     `json:"scope"`
	SourceContextRecordIDs []int64        `json:"sourceContextRecordIds"`
	Constraints            []string       `json:"constraints"`
	Instructions           string         `json:"instructions"`
	ExecutionMode          string         `json:"executionMode"`
	ExpectedOutput         map[string]any `json:"expectedOutput"`
	RequestPayload         map[string]any `json:"requestPayload"`
	ExpectedArtifactTypes  []string       `json:"expectedArtifactTypes"`
	CreatedBy              string         `json:"createdBy"`
}

type Detail struct {
	Job             Job                  `json:"job"`
	Events          []JobEvent           `json:"events"`
	StatusHistory   []StatusTransition   `json:"statusHistory"`
	ApprovalRequest *approvals.Request   `json:"approvalRequest,omitempty"`
	Packet          *packets.Packet      `json:"packet,omitempty"`
	Artifacts       []artifacts.Artifact `json:"artifacts"`
}

type Service struct {
	db           *sql.DB
	log          *events.Logger
	ingest       *ingest.Service
	adapters     *adapters.Registry
	packets      *packets.Service
	approvals    *approvals.Service
	artifacts    *artifacts.Service
	projectCtx   *projectcontext.Service
	dossiers     *dossiers.Service
	retrieval    *retrieval.Service
	gateway      *gateway.Gateway
	workspaceDir string

	queue      chan string
	stop       chan struct{}
	wg         sync.WaitGroup
	closeOnce  sync.Once
	rootCtx    context.Context
	cancelRoot context.CancelFunc

	mu          sync.Mutex
	cancelFuncs map[string]context.CancelFunc
}

type executionError struct {
	Code    FailureCode
	Message string
}

var errExecutionDeferred = errors.New("execution_deferred")

func (e executionError) Error() string {
	return e.Message
}

type Dependencies struct {
	DB           *sql.DB
	Logger       *events.Logger
	Ingest       *ingest.Service
	Adapters     *adapters.Registry
	Packets      *packets.Service
	Approvals    *approvals.Service
	Artifacts    *artifacts.Service
	ProjectCtx   *projectcontext.Service
	Dossiers     *dossiers.Service
	Retrieval    *retrieval.Service
	Gateway      *gateway.Gateway
	WorkspaceDir string
}

func NewService(dep Dependencies) *Service {
	rootCtx, cancelRoot := context.WithCancel(context.Background())
	s := &Service{
		db:           dep.DB,
		log:          dep.Logger,
		ingest:       dep.Ingest,
		adapters:     dep.Adapters,
		packets:      dep.Packets,
		approvals:    dep.Approvals,
		artifacts:    dep.Artifacts,
		projectCtx:   dep.ProjectCtx,
		dossiers:     dep.Dossiers,
		retrieval:    dep.Retrieval,
		gateway:      dep.Gateway,
		workspaceDir: dep.WorkspaceDir,
		queue:        make(chan string, 256),
		stop:         make(chan struct{}),
		rootCtx:      rootCtx,
		cancelRoot:   cancelRoot,
		cancelFuncs:  map[string]context.CancelFunc{},
	}
	_ = s.recoverInterruptedJobs(context.Background())
	s.wg.Add(1)
	go s.worker()
	return s
}

func (s *Service) Close() {
	s.closeOnce.Do(func() {
		if s.cancelRoot != nil {
			s.cancelRoot()
		}
		s.cancelRunning()
		close(s.stop)
		s.wg.Wait()
	})
}

func (s *Service) ListTemplates() []Template {
	return ListTemplates()
}

func (s *Service) Create(ctx context.Context, req CreateRequest) (*Job, error) {
	tpl, ok := TemplateByID(strings.TrimSpace(req.TemplateID))
	if !ok {
		return nil, fmt.Errorf("unknown job template %q", req.TemplateID)
	}
	if strings.TrimSpace(req.UserRequest) == "" {
		return nil, fmt.Errorf("userRequest is required")
	}

	now := time.Now().UnixMilli()
	id := newJobID()
	status := StatusQueued
	approval := ApprovalNotRequired
	if tpl.ApprovalRequired {
		approval = ApprovalPending
	}
	meta := jobMetadata{
		TemplateID:             tpl.ID,
		UserRequest:            req.UserRequest,
		Objective:              nonEmpty(req.Objective, req.UserRequest),
		Query:                  req.Query,
		Scope:                  req.Scope,
		SourceContextRecordIDs: req.SourceContextRecordIDs,
		Constraints:            req.Constraints,
		Instructions:           req.Instructions,
		ExecutionMode:          nonEmpty(req.ExecutionMode, tpl.DefaultExecutionMode),
		ExpectedOutput:         nonNilMap(req.ExpectedOutput),
		RequestPayload:         nonNilMap(req.RequestPayload),
		ExpectedArtifactTypes:  tpl.ExpectedArtifactTypes,
		CreatedBy:              nonEmpty(req.InitiatingSource, "command_bar"),
	}
	metaJSON, _ := json.Marshal(meta)

	_, err := s.db.ExecContext(ctx, `
INSERT INTO jobs(
  id, created_at, updated_at, queued_at,
  title, requested_action, target_adapter, initiating_source,
  execution_boundary, risk_class, status, approval_status, write_intent,
  metadata_json
) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		id,
		now,
		now,
		now,
		nonEmpty(req.Title, tpl.Name),
		tpl.RequestedAction,
		tpl.TargetAdapter,
		nonEmpty(req.InitiatingSource, "command_bar"),
		tpl.ExecutionBoundary,
		string(tpl.RiskClass),
		string(status),
		string(approval),
		boolToInt(tpl.WriteIntent),
		string(metaJSON),
	)
	if err != nil {
		return nil, fmt.Errorf("insert job: %w", err)
	}

	_ = s.appendStatusTransition(ctx, id, nil, status, "job created")
	_, _ = s.appendEvent(ctx, id, "job.created", "Job created from template", map[string]any{
		"templateId":        tpl.ID,
		"riskClass":         tpl.RiskClass,
		"executionBoundary": tpl.ExecutionBoundary,
		"approvalRequired":  tpl.ApprovalRequired,
	})

	job, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	s.Enqueue(id)
	return job, nil
}

func (s *Service) Enqueue(jobID string) {
	select {
	case s.queue <- jobID:
	case <-s.stop:
		return
	default:
		_, _ = s.appendEvent(context.Background(), jobID, "job.queue.full", "Job queue is full; job remains queued for recovery", map[string]any{})
	}
}

func (s *Service) worker() {
	defer s.wg.Done()
	for {
		select {
		case <-s.stop:
			return
		case id := <-s.queue:
			if err := s.process(id); err != nil {
				_ = s.log.Emit(context.Background(), "job.processor.error", map[string]any{"jobId": id, "message": err.Error()})
			}
		}
	}
}

func (s *Service) cancelRunning() {
	s.mu.Lock()
	cancels := make([]context.CancelFunc, 0, len(s.cancelFuncs))
	for _, cancel := range s.cancelFuncs {
		if cancel != nil {
			cancels = append(cancels, cancel)
		}
	}
	s.mu.Unlock()
	for _, cancel := range cancels {
		cancel()
	}
}

func (s *Service) recoverInterruptedJobs(ctx context.Context) error {
	rows, err := s.db.QueryContext(ctx, `
SELECT id, status
FROM jobs
WHERE status IN (?, ?, ?)`,
		string(StatusQueued),
		string(StatusPreparing),
		string(StatusRunning),
	)
	if err != nil {
		return err
	}
	defer rows.Close()
	type candidate struct {
		id     string
		status Status
	}
	var candidates []candidate
	for rows.Next() {
		var c candidate
		if err := rows.Scan(&c.id, &c.status); err != nil {
			return err
		}
		candidates = append(candidates, c)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	for _, c := range candidates {
		switch c.status {
		case StatusQueued:
			_, _ = s.appendEventIfMissing(ctx, c.id, "job.recovered", "Queued job recovered on startup", map[string]any{"fromStatus": c.status})
			s.Enqueue(c.id)
		case StatusPreparing, StatusRunning:
			_ = s.fail(ctx, c.id, FailInterrupted, "Job interrupted by prior shutdown; operator retry required")
			_, _ = s.appendEventIfMissing(ctx, c.id, "job.recovered", "Interrupted job reconciled on startup", map[string]any{"fromStatus": c.status, "recoverable": true})
		}
	}
	return nil
}

func (s *Service) process(jobID string) error {
	ctx := s.context()
	job, err := s.Get(ctx, jobID)
	if err != nil {
		return err
	}
	if isTerminal(job.Status) {
		return nil
	}

	if job.CancelRequested {
		return s.markCancelled(ctx, jobID, FailUserCancellation, "Cancellation requested before execution")
	}

	if err := s.transitionStatus(ctx, jobID, StatusPreparing, "packet and policy checks"); err != nil {
		return err
	}

	meta, err := s.jobMetadata(ctx, jobID)
	if err != nil {
		_ = s.fail(ctx, jobID, FailValidation, fmt.Sprintf("metadata decode failed: %v", err))
		return err
	}
	tpl, ok := TemplateByID(meta.TemplateID)
	if !ok {
		return s.fail(ctx, jobID, FailValidation, fmt.Sprintf("template %q missing", meta.TemplateID))
	}

	packet, err := s.ensurePacket(ctx, jobID, job, meta, tpl)
	if err != nil {
		_ = s.fail(ctx, jobID, FailPacketBuild, err.Error())
		return err
	}

	if tpl.ApprovalRequired {
		if job.ApprovalStatus != ApprovalGranted {
			r, err := s.approvals.OpenRequestForJob(ctx, jobID, approvals.CreateRequestInput{
				JobID:            jobID,
				RequestedAction:  job.RequestedAction,
				RiskClass:        string(job.RiskClass),
				RequestedAdapter: job.TargetAdapter,
				WriteIntent:      job.WriteIntent,
				ScopeSnapshot:    meta.Scope.ToMap(),
				TaskPacketID:     packetID(packet),
				RequestSummary:   fmt.Sprintf("%s via %s", job.RequestedAction, job.TargetAdapter),
			})
			if err != nil {
				_ = s.fail(ctx, jobID, FailPersistence, fmt.Sprintf("approval request failed: %v", err))
				return err
			}
			_ = s.setApprovalStatus(ctx, jobID, ApprovalPending)
			if err := s.transitionStatus(ctx, jobID, StatusAwaitingApproval, "awaiting operator approval"); err != nil {
				return err
			}
			_, _ = s.appendEvent(ctx, jobID, "job.approval.requested", "Approval required before execution", map[string]any{
				"requestId": r.ID,
				"riskClass": r.RiskClass,
			})
			return nil
		}
	}

	if err := s.transitionStatus(ctx, jobID, StatusRunning, "execution started"); err != nil {
		return err
	}
	_, _ = s.appendEvent(ctx, jobID, "job.started", "Execution started", map[string]any{"templateId": tpl.ID})

	runCtx, cancel := context.WithCancel(ctx)
	s.setCancel(jobID, cancel)
	defer s.clearCancel(jobID)

	resultSummary, err := s.execute(runCtx, jobID, job, meta, tpl, packet)
	if err != nil {
		commitCtx := context.WithoutCancel(ctx)
		if errors.Is(err, errExecutionDeferred) {
			return nil
		}
		if errors.Is(err, context.Canceled) {
			return s.markCancelled(commitCtx, jobID, FailUserCancellation, "Execution cancelled")
		}
		var ex executionError
		if errors.As(err, &ex) {
			return s.fail(commitCtx, jobID, ex.Code, ex.Message)
		}
		return s.fail(commitCtx, jobID, FailExecution, err.Error())
	}

	if err := s.complete(context.WithoutCancel(ctx), jobID, resultSummary); err != nil {
		return err
	}
	return nil
}

func (s *Service) context() context.Context {
	if s.rootCtx != nil {
		return s.rootCtx
	}
	return context.Background()
}

func (s *Service) execute(ctx context.Context, jobID string, job *Job, meta jobMetadata, tpl Template, packet *packets.Packet) (string, error) {
	if job.CancelRequested {
		return "", context.Canceled
	}

	switch tpl.ID {
	case "search_packet":
		summary := fmt.Sprintf("Packet %d prepared with memory references.", packet.ID)
		_, _ = s.appendEvent(ctx, jobID, "job.completed.search_packet", "Packet-only job completed", map[string]any{"packetId": packet.ID})
		return summary, nil
	case "reindex_sources":
		sourceID := readInt(meta.RequestPayload, "sourceId", 0)
		if sourceID > 0 {
			var path string
			if err := s.db.QueryRowContext(ctx, `SELECT path FROM sources WHERE id = ?`, sourceID).Scan(&path); err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					return "", executionError{Code: FailValidation, Message: fmt.Sprintf("source %d not found", sourceID)}
				}
				return "", err
			}
			if err := s.ingest.IndexSource(ctx, sourceID, path); err != nil {
				return "", err
			}
			_, _ = s.appendEvent(ctx, jobID, "job.reindex.completed", "Source reindex completed", map[string]any{
				"scope":    "one",
				"sourceId": sourceID,
				"path":     path,
			})
			return fmt.Sprintf("Re-indexed source %d.", sourceID), nil
		}
		if err := s.ingest.IndexAllSources(ctx); err != nil {
			return "", err
		}
		_, _ = s.appendEvent(ctx, jobID, "job.reindex.completed", "Source reindex completed", map[string]any{
			"scope": "all",
		})
		return "Re-indexed all configured sources.", nil
	case "normalize_project_context":
		rec, err := s.projectCtx.ImportAndNormalize(ctx, projectcontext.ImportRequest{
			SourcePath: readString(meta.RequestPayload, "sourcePath"),
			Notes:      readString(meta.RequestPayload, "notes"),
		})
		if err != nil {
			return "", err
		}
		_, _ = s.appendEvent(ctx, jobID, "job.context.normalized", "Project context normalized", map[string]any{
			"recordId": rec.ID,
			"version":  rec.ContextVersion,
		})
		_, _ = s.artifacts.CreateTextArtifact(ctx, artifacts.CreateTextArtifactRequest{
			JobID:    &jobID,
			PacketID: packetID(packet),
			Type:     "context_normalization",
			Title:    "Project context normalization record",
			FileName: fmt.Sprintf("context-normalization-%s.md", jobID),
			Subdir:   "context",
			Content: fmt.Sprintf("Record: %d\nVersion: %d\nSource: %s\nGenerated files:\n- %s\n- %s\n- %s\n- %s\n",
				rec.ID,
				rec.ContextVersion,
				rec.SourcePath,
				rec.GeneratedAgentsPath,
				rec.GeneratedClaudePath,
				rec.GeneratedBriefingPath,
				rec.GeneratedCursorPath,
			),
			MimeType: "text/markdown",
			Metadata: map[string]any{"recordId": rec.ID},
		})
		return fmt.Sprintf("Project context normalized (record %d).", rec.ID), nil
	case "gateway_action":
		if s.gateway == nil {
			return "", executionError{Code: FailExecution, Message: "gateway dependency unavailable"}
		}
		toolID := strings.TrimSpace(readString(meta.RequestPayload, "toolId"))
		laneID := strings.TrimSpace(readString(meta.RequestPayload, "laneId"))
		action := strings.TrimSpace(readString(meta.RequestPayload, "action"))
		domain := strings.TrimSpace(readString(meta.RequestPayload, "domain"))
		riskClass := strings.TrimSpace(readString(meta.RequestPayload, "riskClass"))
		level := strings.TrimSpace(readString(meta.RequestPayload, "executionLevel"))
		correlationID := strings.TrimSpace(readString(meta.RequestPayload, "correlationId"))
		if correlationID == "" {
			correlationID = fmt.Sprintf("job-%s-gateway", jobID)
		}
		initiator := strings.TrimSpace(readString(meta.RequestPayload, "initiator"))
		if initiator == "" {
			initiator = "job"
		}
		reqPaths := readStringSlice(meta.RequestPayload, "paths")
		invokeInput := readMap(meta.RequestPayload, "input")
		invokeInput = enrichGatewayActionInput(toolID, invokeInput, meta.UserRequest)
		dryRun := readBool(meta.RequestPayload, "dryRun", false)
		packetID := packetID(packet)
		result, err := s.gateway.Execute(ctx, gateway.Request{
			ToolID:              toolID,
			LaneID:              laneID,
			Domain:              domain,
			Action:              action,
			RiskClass:           riskClass,
			ExecutionLevel:      level,
			CorrelationID:       correlationID,
			Paths:               reqPaths,
			Input:               invokeInput,
			JobID:               &jobID,
			PacketID:            packetID,
			Initiator:           initiator,
			Source:              strings.TrimSpace(readString(meta.RequestPayload, "source")),
			WorkspaceID:         strings.TrimSpace(readString(meta.RequestPayload, "workspaceId")),
			ProvenanceActor:     strings.TrimSpace(readString(meta.RequestPayload, "provenanceActor")),
			ProvenanceActorType: strings.TrimSpace(readString(meta.RequestPayload, "provenanceActorType")),
			DryRun:              dryRun,
		})
		if err != nil {
			return "", executionError{Code: FailExecution, Message: "gateway invoke failed: " + err.Error()}
		}
		_, _ = s.appendEvent(ctx, jobID, "job.gateway.result", "Gateway action result received", map[string]any{
			"status":           result.Status,
			"policyOutcome":    result.PolicyOutcome,
			"invocationId":     result.InvocationID,
			"correlationId":    result.CorrelationID,
			"tool":             result.Tool,
			"lane":             result.Lane,
			"riskClass":        result.RiskClass,
			"executionLevel":   result.ExecutionLevel,
			"approvalRequired": result.Status == gateway.StatusNeedsApprov,
		})
		if result.Status == gateway.StatusNeedsApprov {
			_ = s.setApprovalStatus(ctx, jobID, ApprovalPending)
			if err := s.transitionStatus(ctx, jobID, StatusAwaitingApproval, "awaiting gateway approval"); err != nil {
				return "", err
			}
			return "", errExecutionDeferred
		}
		if !result.Allowed || result.Status == gateway.StatusDenied {
			return "", executionError{Code: FailApprovalDenied, Message: nonEmpty(result.DeniedReason, "gateway denied request")}
		}
		payload, _ := json.MarshalIndent(result, "", "  ")
		_, _ = s.artifacts.CreateTextArtifact(ctx, artifacts.CreateTextArtifactRequest{
			JobID:    &jobID,
			PacketID: packetID,
			Type:     "job_result",
			Title:    "Gateway action result",
			FileName: fmt.Sprintf("gateway-result-%s.json", jobID),
			Subdir:   "gateway",
			Content:  string(payload),
			MimeType: "application/json",
			Metadata: map[string]any{"invocationId": result.InvocationID, "correlationId": result.CorrelationID},
		})
		return nonEmpty(result.Message, "Gateway action completed"), nil
	}

	invoke := adapters.InvokeRequest{
		AdapterID:     job.TargetAdapter,
		Capability:    tpl.Capability,
		Scope:         toAdapterScope(meta.Scope),
		WriteIntent:   tpl.WriteIntent,
		TaskPacketRef: packetID(packet),
		TimeoutMs:     int(readInt(meta.RequestPayload, "timeoutMs", 45_000)),
		DryRun:        readBool(meta.RequestPayload, "dryRun", false),
		CorrelationID: fmt.Sprintf("%s:%s", job.ID, tpl.ID),
		Input:         s.buildAdapterInput(meta, tpl, packet),
	}

	_, _ = s.appendEvent(ctx, jobID, "gateway.adapter.request.sent", "Adapter request sent through gateway", map[string]any{
		"adapter":       invoke.AdapterID,
		"capability":    invoke.Capability,
		"correlationId": invoke.CorrelationID,
		"writeIntent":   invoke.WriteIntent,
		"dryRun":        invoke.DryRun,
	})

	res, err := s.invokeLegacyAdapterThroughGateway(ctx, job, packet, invoke)
	if err != nil {
		if errors.Is(err, errExecutionDeferred) {
			return "", err
		}
		return "", executionError{Code: FailAdapterUnavailable, Message: err.Error()}
	}

	_, _ = s.appendEvent(ctx, jobID, "gateway.adapter.response.received", "Gateway adapter response received", map[string]any{
		"adapter":     invoke.AdapterID,
		"ok":          res.OK,
		"failureCode": res.FailureCode,
	})

	if !res.OK {
		if res.FailureCode == "adapter_timeout" {
			return "", executionError{Code: FailAdapterTimeout, Message: "adapter timeout: " + res.Message}
		}
		if res.FailureCode == "adapter_unavailable" {
			return "", executionError{Code: FailAdapterUnavailable, Message: "adapter unavailable: " + res.Message}
		}
		return "", executionError{Code: FailExecution, Message: "adapter failure: " + res.Message}
	}

	content, title := adapterContent(res, tpl)
	atype := "adapter_output"
	if tpl.TargetAdapter == "codex" || tpl.TargetAdapter == "claude_code" {
		atype = "agent_guidance"
	}
	_, _ = s.artifacts.CreateTextArtifact(ctx, artifacts.CreateTextArtifactRequest{
		JobID:    &jobID,
		PacketID: packetID(packet),
		Type:     atype,
		Title:    title,
		FileName: fmt.Sprintf("%s-%s.md", tpl.TargetAdapter, jobID),
		Subdir:   "adapter",
		Content:  content,
		MimeType: "text/markdown",
		Metadata: map[string]any{"adapter": tpl.TargetAdapter, "capability": tpl.Capability},
	})

	return nonEmpty(res.Message, "Adapter execution completed"), nil
}

func (s *Service) invokeLegacyAdapterThroughGateway(ctx context.Context, job *Job, packet *packets.Packet, invoke adapters.InvokeRequest) (adapters.InvokeResult, error) {
	if s.gateway == nil {
		return adapters.InvokeResult{}, fmt.Errorf("gateway dependency unavailable")
	}
	if job == nil {
		return adapters.InvokeResult{}, fmt.Errorf("job is required")
	}
	input := map[string]any{}
	raw, err := json.Marshal(invoke)
	if err != nil {
		return adapters.InvokeResult{}, err
	}
	if err := json.Unmarshal(raw, &input); err != nil {
		return adapters.InvokeResult{}, err
	}
	packetID := packetID(packet)
	result, err := s.gateway.Execute(ctx, gateway.Request{
		ToolID:         "legacy.adapter.invoke",
		LaneID:         "legacy.adapter.invoke",
		Domain:         "gateway",
		Action:         "invoke",
		RiskClass:      "low",
		ExecutionLevel: "L0",
		CorrelationID:  invoke.CorrelationID,
		Input:          input,
		JobID:          &job.ID,
		PacketID:       packetID,
		Initiator:      "job",
		DryRun:         invoke.DryRun,
	})
	if err != nil {
		return adapters.InvokeResult{}, err
	}
	if result.Status == gateway.StatusNeedsApprov {
		_ = s.setApprovalStatus(ctx, job.ID, ApprovalPending)
		if err := s.transitionStatus(ctx, job.ID, StatusAwaitingApproval, "awaiting legacy adapter gateway approval"); err != nil {
			return adapters.InvokeResult{}, err
		}
		return adapters.InvokeResult{}, errExecutionDeferred
	}
	if !result.Allowed || result.Status == gateway.StatusDenied {
		return adapters.InvokeResult{}, errors.New(nonEmpty(result.DeniedReason, "gateway denied legacy adapter invocation"))
	}
	if result.Status == gateway.StatusError {
		return adapters.InvokeResult{}, errors.New(nonEmpty(result.Message, "gateway legacy adapter invocation failed"))
	}
	return decodeLegacyAdapterGatewayResult(result.Data)
}

func decodeLegacyAdapterGatewayResult(data map[string]any) (adapters.InvokeResult, error) {
	rawResult, ok := data["result"]
	if !ok {
		return adapters.InvokeResult{}, fmt.Errorf("gateway legacy adapter result missing")
	}
	raw, err := json.Marshal(rawResult)
	if err != nil {
		return adapters.InvokeResult{}, err
	}
	var out adapters.InvokeResult
	if err := json.Unmarshal(raw, &out); err != nil {
		return adapters.InvokeResult{}, err
	}
	return out, nil
}

func enrichGatewayActionInput(toolID string, input map[string]any, userRequest string) map[string]any {
	if strings.TrimSpace(toolID) != "desktop.open" {
		return input
	}
	if input == nil {
		input = map[string]any{}
	}
	if raw, ok := input["query"]; ok && strings.TrimSpace(fmt.Sprintf("%v", raw)) != "" {
		return input
	}
	text := strings.TrimSpace(userRequest)
	if text == "" {
		return input
	}
	if strings.HasPrefix(strings.ToLower(text), "chat gateway:") {
		return input
	}
	input["query"] = text
	return input
}

func (s *Service) buildAdapterInput(meta jobMetadata, tpl Template, packet *packets.Packet) map[string]any {
	input := map[string]any{}
	for k, v := range meta.RequestPayload {
		input[k] = v
	}
	input["objective"] = meta.Objective
	input["userRequest"] = meta.UserRequest
	input["expectedDeliverable"] = readString(meta.ExpectedOutput, "deliverableType")
	input["packetId"] = packet.ID
	if tpl.TargetAdapter == "ollama" {
		input["prompt"] = buildPrompt(meta, tpl, packet)
		if model := readString(meta.RequestPayload, "model"); strings.TrimSpace(model) != "" {
			input["model"] = model
		}
	}
	return input
}

func buildPrompt(meta jobMetadata, tpl Template, packet *packets.Packet) string {
	ctxPreview := string(packet.RetrievedContext)
	if len(ctxPreview) > 4000 {
		ctxPreview = ctxPreview[:4000]
	}
	return fmt.Sprintf("Template: %s\nObjective: %s\nUser request: %s\nConstraints: %s\nContext: %s",
		tpl.ID,
		meta.Objective,
		meta.UserRequest,
		strings.Join(meta.Constraints, "; "),
		ctxPreview,
	)
}

func (s *Service) ensurePacket(ctx context.Context, jobID string, job *Job, meta jobMetadata, tpl Template) (*packets.Packet, error) {
	if job.TaskPacketID != nil {
		return s.packets.GetByID(ctx, *job.TaskPacketID)
	}

	dossierID := readOptionalInt(meta.RequestPayload, "dossierId")

	if len(meta.SourceContextRecordIDs) == 0 {
		if latest, _ := s.projectCtx.Latest(ctx); latest != nil {
			meta.SourceContextRecordIDs = []int64{latest.ID}
		}
	}

	var run *retrieval.Run
	if s.retrieval != nil {
		mode := retrieval.Mode(strings.TrimSpace(readString(meta.RequestPayload, "retrievalMode")))
		if mode == "" {
			mode = retrieval.ModeHybrid
		}
		selectForPacket := int(readInt(meta.RequestPayload, "retrievalSelectCount", 8))
		runReq := retrieval.RunRequest{
			Query:           nonEmpty(meta.Query, meta.UserRequest),
			Mode:            mode,
			Limit:           int(readInt(meta.RequestPayload, "retrievalLimit", 24)),
			SelectForPacket: selectForPacket,
			WeightKeyword:   readFloat(meta.RequestPayload, "retrievalWeightKeyword", 0),
			WeightSemantic:  readFloat(meta.RequestPayload, "retrievalWeightSemantic", 0),
			Provider:        readString(meta.RequestPayload, "embeddingProvider"),
			Model:           readString(meta.RequestPayload, "embeddingModel"),
			JobID:           &jobID,
			DossierID:       dossierID,
			Notes:           "job.packet.prep",
			Actor:           domain.ActorIdentity{ID: "forge.jobs", Kind: "service"},
			Source:          domain.SourceInternal,
			Scope: domain.ForgeScope{
				WorkspaceID: strings.TrimSpace(s.workspaceDir), LaneID: "control.semantic",
			},
			Provenance:     domain.Provenance{Actor: "forge.jobs", ActorType: "system", Source: "job.packet.prep"},
			CorrelationID:  "job-" + jobID + "-retrieval",
			TraceID:        "job-" + jobID + "-retrieval",
			RequestID:      "retrieval-evidence-job-" + jobID,
			IdempotencyKey: "retrieval-evidence-job-" + jobID,
			RequestedAt:    job.CreatedAtMs,
		}
		r, err := s.retrieval.Run(ctx, runReq)
		if err != nil {
			return nil, fmt.Errorf("retrieval run failed: %w", err)
		}
		run = r
	}

	retrievedItems := []packets.RetrievedItem{}
	var runID *int64
	if run != nil {
		id := run.ID
		runID = &id
		for _, rr := range run.Results {
			if !rr.SelectedForPacket || rr.ChunkID == nil {
				continue
			}
			retrievedItems = append(retrievedItems, packets.RetrievedItem{
				ChunkID:       *rr.ChunkID,
				FileID:        derefInt64(rr.FileID),
				AbsPath:       rr.AbsPath,
				RelPath:       rr.RelPath,
				Snippet:       rr.Snippet,
				Score:         rr.HybridScore,
				KeywordScore:  rr.KeywordScore,
				SemanticScore: rr.SemanticScore,
				HybridScore:   rr.HybridScore,
			})
		}
	}

	packet, err := s.packets.BuildAndStore(ctx, packets.BuildRequest{
		Title:                  nonEmpty(job.Title, tpl.Name),
		UserRequest:            meta.UserRequest,
		Objective:              meta.Objective,
		AdapterTarget:          tpl.TargetAdapter,
		ExecutionMode:          meta.ExecutionMode,
		RiskClass:              string(tpl.RiskClass),
		ExpectedOutput:         meta.ExpectedOutput,
		Constraints:            meta.Constraints,
		Instructions:           meta.Instructions,
		Scope:                  toAdapterScope(meta.Scope),
		SourceContextRecordIDs: meta.SourceContextRecordIDs,
		Query:                  meta.Query,
		RequestPayload:         meta.RequestPayload,
		ProjectNotes:           "Derived from FORGE context and indexed source memory.",
		RetrievedItems:         retrievedItems,
		RetrievalRunID:         runID,
	})
	if err != nil {
		return nil, err
	}

	if _, err := s.db.ExecContext(ctx, `UPDATE jobs SET task_packet_id = ?, updated_at = ? WHERE id = ?`, packet.ID, time.Now().UnixMilli(), jobID); err != nil {
		return nil, err
	}

	_, _ = s.appendEvent(ctx, jobID, "job.packet.prepared", "Task packet prepared", map[string]any{
		"packetId":         packet.ID,
		"packetVersion":    packet.PacketVersion,
		"riskClass":        packet.RiskClass,
		"adapterTarget":    packet.AdapterTarget,
		"scopeSnapshot":    json.RawMessage(packet.ScopeSnapshot),
		"sourceContextIds": json.RawMessage(packet.SourceContextRecordIDs),
	})

	if s.dossiers != nil && dossierID != nil {
		_ = s.dossiers.AttachJob(ctx, *dossierID, jobID)
		_ = s.dossiers.AttachPacket(ctx, *dossierID, packet.ID)
	}

	payload, _ := json.MarshalIndent(packet, "", "  ")
	_, _ = s.artifacts.CreateTextArtifact(ctx, artifacts.CreateTextArtifactRequest{
		JobID:    &jobID,
		PacketID: packetID(packet),
		Type:     "task_packet",
		Title:    fmt.Sprintf("Task packet %d", packet.ID),
		FileName: fmt.Sprintf("task-packet-%d.json", packet.ID),
		Subdir:   "packets",
		Content:  string(payload),
		MimeType: "application/json",
		Metadata: map[string]any{"packetId": packet.ID, "packetVersion": packet.PacketVersion},
	})

	return packet, nil
}

func (s *Service) ApplyApprovalDecision(ctx context.Context, requestID int64, decision, actor, note string) (*approvals.Decision, error) {
	d, err := s.approvals.Decide(ctx, requestID, actor, decision, note)
	if err != nil {
		return nil, err
	}
	req, err := s.approvals.GetRequest(ctx, requestID)
	if err != nil {
		return nil, err
	}
	jobID := req.JobID

	switch decision {
	case "approved":
		_ = s.setApprovalStatus(ctx, jobID, ApprovalGranted)
		_, _ = s.appendEvent(ctx, jobID, "job.approval.granted", "Approval granted", map[string]any{"requestId": requestID, "actor": actor, "decisionId": d.ID})
		job, _ := s.Get(ctx, jobID)
		if job != nil && job.Status == StatusAwaitingApproval {
			_ = s.transitionStatus(ctx, jobID, StatusQueued, "approval granted")
			s.Enqueue(jobID)
		}
	case "denied":
		_ = s.setApprovalStatus(ctx, jobID, ApprovalDenied)
		_, _ = s.appendEvent(ctx, jobID, "job.approval.denied", "Approval denied", map[string]any{"requestId": requestID, "actor": actor, "decisionId": d.ID})
		_ = s.markCancelled(ctx, jobID, FailApprovalDenied, "Approval denied")
	case "cancelled":
		_ = s.setApprovalStatus(ctx, jobID, ApprovalDenied)
		_, _ = s.appendEvent(ctx, jobID, "job.approval.cancelled", "Approval cancelled", map[string]any{"requestId": requestID, "actor": actor, "decisionId": d.ID})
		_ = s.markCancelled(ctx, jobID, FailUserCancellation, "Approval flow cancelled")
	}
	return d, nil
}

func (s *Service) RequestCancel(ctx context.Context, jobID, actor string) error {
	job, err := s.Get(ctx, jobID)
	if err != nil {
		return err
	}
	if isTerminal(job.Status) {
		return nil
	}
	if _, err := s.db.ExecContext(ctx, `UPDATE jobs SET cancel_requested = 1, updated_at = ? WHERE id = ?`, time.Now().UnixMilli(), jobID); err != nil {
		return err
	}
	_, _ = s.appendEvent(ctx, jobID, "job.cancel.requested", "Cancellation requested", map[string]any{"actor": nonEmpty(actor, "operator")})

	if job.Status == StatusQueued || job.Status == StatusPreparing || job.Status == StatusAwaitingApproval {
		return s.markCancelled(ctx, jobID, FailUserCancellation, "Cancelled before execution")
	}
	if job.Status == StatusRunning {
		s.mu.Lock()
		cancel := s.cancelFuncs[jobID]
		s.mu.Unlock()
		if cancel != nil {
			cancel()
		}
	}
	return nil
}

func (s *Service) List(ctx context.Context, status string, limit int) ([]Job, error) {
	if limit <= 0 || limit > 300 {
		limit = 120
	}
	query := `SELECT id, title, requested_action, target_adapter, status,
 created_at, updated_at, queued_at, started_at, completed_at,
 initiating_source, execution_boundary, risk_class, approval_status, write_intent,
 cancel_requested, task_packet_id, result_summary, failure_info, last_failure_code, last_error, metadata_json
FROM jobs`
	args := []any{}
	if strings.TrimSpace(status) != "" {
		query += ` WHERE status = ?`
		args = append(args, status)
	}
	query += ` ORDER BY id DESC LIMIT ?`
	args = append(args, limit)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]Job, 0, limit)
	for rows.Next() {
		j, err := scanJob(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *j)
	}
	return out, rows.Err()
}

func (s *Service) Get(ctx context.Context, jobID string) (*Job, error) {
	row := s.db.QueryRowContext(ctx, `SELECT id, title, requested_action, target_adapter, status,
 created_at, updated_at, queued_at, started_at, completed_at,
 initiating_source, execution_boundary, risk_class, approval_status, write_intent,
 cancel_requested, task_packet_id, result_summary, failure_info, last_failure_code, last_error, metadata_json
FROM jobs WHERE id = ?`, jobID)
	return scanJob(row)
}

func (s *Service) Detail(ctx context.Context, jobID string, afterEventID int64) (*Detail, error) {
	job, err := s.Get(ctx, jobID)
	if err != nil {
		return nil, err
	}
	evts, err := s.Events(ctx, jobID, afterEventID, 500)
	if err != nil {
		return nil, err
	}
	history, err := s.StatusHistory(ctx, jobID)
	if err != nil {
		return nil, err
	}
	apr, _ := s.approvals.LatestRequestByJob(ctx, jobID)
	var pkt *packets.Packet
	if job.TaskPacketID != nil {
		pkt, _ = s.packets.GetByID(ctx, *job.TaskPacketID)
	}
	arts, _ := s.artifacts.ListByJob(ctx, jobID)
	return &Detail{Job: *job, Events: evts, StatusHistory: history, ApprovalRequest: apr, Packet: pkt, Artifacts: arts}, nil
}

func (s *Service) Events(ctx context.Context, jobID string, afterID int64, limit int) ([]JobEvent, error) {
	if limit <= 0 || limit > 500 {
		limit = 300
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT id, job_id, created_at, type, message, payload_json
FROM job_events
WHERE job_id = ? AND id > ?
ORDER BY id ASC
LIMIT ?`, jobID, afterID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]JobEvent, 0, limit)
	for rows.Next() {
		var e JobEvent
		var payload string
		if err := rows.Scan(&e.ID, &e.JobID, &e.CreatedAtMs, &e.Type, &e.Message, &payload); err != nil {
			return nil, err
		}
		e.Payload = json.RawMessage(payload)
		out = append(out, e)
	}
	return out, rows.Err()
}

func (s *Service) StatusHistory(ctx context.Context, jobID string) ([]StatusTransition, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT id, job_id, created_at, from_status, to_status, reason
FROM job_status_history
WHERE job_id = ?
ORDER BY id ASC`, jobID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []StatusTransition
	for rows.Next() {
		var tr StatusTransition
		var from sql.NullString
		if err := rows.Scan(&tr.ID, &tr.JobID, &tr.CreatedAtMs, &from, &tr.ToStatus, &tr.Reason); err != nil {
			return nil, err
		}
		if from.Valid {
			v := Status(from.String)
			tr.FromStatus = &v
		}
		out = append(out, tr)
	}
	return out, rows.Err()
}

func (s *Service) transitionStatus(ctx context.Context, jobID string, to Status, reason string) error {
	job, err := s.Get(ctx, jobID)
	if err != nil {
		return err
	}
	if job.Status == to {
		return nil
	}
	now := time.Now().UnixMilli()
	fields := "status = ?, updated_at = ?"
	args := []any{to, now}
	if to == StatusRunning {
		fields += ", started_at = COALESCE(started_at, ?)"
		args = append(args, now)
	}
	if to == StatusSucceeded || to == StatusFailed || to == StatusCancelled {
		fields += ", completed_at = COALESCE(completed_at, ?)"
		args = append(args, now)
	}
	args = append(args, jobID)
	if _, err := s.db.ExecContext(ctx, fmt.Sprintf("UPDATE jobs SET %s WHERE id = ?", fields), args...); err != nil {
		return err
	}
	_ = s.appendStatusTransition(ctx, jobID, &job.Status, to, reason)
	_, _ = s.appendEvent(ctx, jobID, "job.status.changed", "Status transition", map[string]any{
		"from":   job.Status,
		"to":     to,
		"reason": reason,
	})
	return nil
}

func (s *Service) appendStatusTransition(ctx context.Context, jobID string, from *Status, to Status, reason string) error {
	now := time.Now().UnixMilli()
	var fromAny any
	if from != nil {
		fromAny = string(*from)
	}
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO job_status_history(job_id, created_at, from_status, to_status, reason) VALUES(?,?,?,?,?)`,
		jobID, now, fromAny, string(to), reason,
	)
	return err
}

func (s *Service) appendEvent(ctx context.Context, jobID, typ, message string, payload map[string]any) (*JobEvent, error) {
	if payload == nil {
		payload = map[string]any{}
	}
	payload["streamVersion"] = 1
	payload["jobId"] = jobID
	b, _ := json.Marshal(payload)
	now := time.Now().UnixMilli()
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO job_events(job_id, created_at, type, message, payload_json) VALUES(?,?,?,?,?)`,
		jobID, now, typ, message, string(b),
	)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	if s.log != nil {
		_ = s.log.Emit(ctx, typ, map[string]any{"jobId": jobID, "message": message, "payload": payload, "eventId": id})
	}
	return &JobEvent{ID: id, JobID: jobID, CreatedAtMs: now, Type: typ, Message: message, Payload: b}, nil
}

func (s *Service) appendEventIfMissing(ctx context.Context, jobID, typ, message string, payload map[string]any) (*JobEvent, error) {
	var id int64
	err := s.db.QueryRowContext(ctx, `SELECT id FROM job_events WHERE job_id = ? AND type = ? ORDER BY id ASC LIMIT 1`, jobID, typ).Scan(&id)
	if err == nil {
		return nil, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	return s.appendEvent(ctx, jobID, typ, message, payload)
}

func (s *Service) setApprovalStatus(ctx context.Context, jobID string, st ApprovalStatus) error {
	_, err := s.db.ExecContext(ctx, `UPDATE jobs SET approval_status = ?, updated_at = ? WHERE id = ?`, string(st), time.Now().UnixMilli(), jobID)
	return err
}

func (s *Service) complete(ctx context.Context, jobID, summary string) error {
	if _, err := s.db.ExecContext(ctx,
		`UPDATE jobs SET result_summary = ?, updated_at = ?, last_error = NULL, last_failure_code = NULL WHERE id = ?`,
		summary,
		time.Now().UnixMilli(),
		jobID,
	); err != nil {
		return err
	}
	if err := s.transitionStatus(ctx, jobID, StatusSucceeded, "execution completed"); err != nil {
		return err
	}
	_, _ = s.appendEvent(ctx, jobID, "job.completed", "Job completed successfully", map[string]any{"resultSummary": summary})
	if s.artifacts != nil {
		_, _ = s.artifacts.CreateTextArtifact(ctx, artifacts.CreateTextArtifactRequest{
			JobID:    &jobID,
			Type:     "job_result",
			Title:    "Job result summary",
			FileName: fmt.Sprintf("job-result-%s.md", jobID),
			Subdir:   "results",
			Content:  "# Job Result\n\n" + summary + "\n",
			MimeType: "text/markdown",
			Metadata: map[string]any{"jobId": jobID},
		})
	}
	return nil
}

func (s *Service) fail(ctx context.Context, jobID string, code FailureCode, message string) error {
	if code == FailAdapterUnavailable {
		message = nonEmpty(message, "adapter unavailable")
	}
	if code == FailAdapterTimeout {
		message = nonEmpty(message, "adapter timeout")
	}

	failureObj := map[string]any{"code": code, "message": message, "at": time.Now().UTC().Format(time.RFC3339)}
	failureJSON, _ := json.Marshal(failureObj)

	_, err := s.db.ExecContext(ctx, `
UPDATE jobs
SET failure_info = ?, last_failure_code = ?, last_error = ?, updated_at = ?
WHERE id = ?`,
		string(failureJSON),
		string(code),
		message,
		time.Now().UnixMilli(),
		jobID,
	)
	if err != nil {
		return err
	}
	if err := s.transitionStatus(ctx, jobID, StatusFailed, "execution failed"); err != nil {
		return err
	}
	_, _ = s.appendEvent(ctx, jobID, "job.failed", "Job failed", map[string]any{"failureCode": code, "message": message})
	if s.artifacts != nil {
		_, _ = s.artifacts.CreateTextArtifact(ctx, artifacts.CreateTextArtifactRequest{
			JobID:    &jobID,
			Type:     "error_report",
			Title:    "Job failure report",
			FileName: fmt.Sprintf("job-error-%s.md", jobID),
			Subdir:   "errors",
			Content:  fmt.Sprintf("# Job Failure\n\nCode: `%s`\n\nMessage: %s\n", code, message),
			MimeType: "text/markdown",
			Metadata: map[string]any{"jobId": jobID, "failureCode": code},
		})
	}
	return nil
}

func (s *Service) markCancelled(ctx context.Context, jobID string, code FailureCode, message string) error {
	failureObj := map[string]any{"code": code, "message": message, "at": time.Now().UTC().Format(time.RFC3339)}
	failureJSON, _ := json.Marshal(failureObj)
	_, err := s.db.ExecContext(ctx, `
UPDATE jobs
SET failure_info = ?, last_failure_code = ?, last_error = ?, updated_at = ?, cancel_requested = 1
WHERE id = ?`,
		string(failureJSON),
		string(code),
		message,
		time.Now().UnixMilli(),
		jobID,
	)
	if err != nil {
		return err
	}
	if err := s.transitionStatus(ctx, jobID, StatusCancelled, "job cancelled"); err != nil {
		return err
	}
	_, _ = s.appendEvent(ctx, jobID, "job.cancelled", "Job cancelled", map[string]any{"failureCode": code, "message": message})
	return nil
}

func (s *Service) setCancel(jobID string, cancel context.CancelFunc) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cancelFuncs[jobID] = cancel
}

func (s *Service) clearCancel(jobID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.cancelFuncs, jobID)
}

func (s *Service) jobMetadata(ctx context.Context, jobID string) (jobMetadata, error) {
	var raw string
	if err := s.db.QueryRowContext(ctx, `SELECT metadata_json FROM jobs WHERE id = ?`, jobID).Scan(&raw); err != nil {
		return jobMetadata{}, err
	}
	var meta jobMetadata
	if err := json.Unmarshal([]byte(raw), &meta); err != nil {
		return jobMetadata{}, err
	}
	return meta, nil
}
