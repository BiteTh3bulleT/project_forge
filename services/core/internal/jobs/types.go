package jobs

import "encoding/json"

type Status string

const (
	StatusQueued           Status = "queued"
	StatusPreparing        Status = "preparing"
	StatusAwaitingApproval Status = "awaiting_approval"
	StatusRunning          Status = "running"
	StatusSucceeded        Status = "succeeded"
	StatusFailed           Status = "failed"
	StatusCancelled        Status = "cancelled"
)

type RiskClass string

const (
	RiskReadOnly          RiskClass = "read_only"
	RiskExternalReasoning RiskClass = "external_reasoning"
	RiskWriteFiles        RiskClass = "write_files"
	RiskRunCommands       RiskClass = "run_commands"
)

type ApprovalStatus string

const (
	ApprovalNotRequired ApprovalStatus = "not_required"
	ApprovalPending     ApprovalStatus = "pending"
	ApprovalGranted     ApprovalStatus = "granted"
	ApprovalDenied      ApprovalStatus = "denied"
)

type FailureCode string

const (
	FailValidation         FailureCode = "validation"
	FailApprovalDenied     FailureCode = "approval_denied"
	FailAdapterUnavailable FailureCode = "adapter_unavailable"
	FailAdapterTimeout     FailureCode = "adapter_timeout"
	FailPacketBuild        FailureCode = "packet_build_failure"
	FailPersistence        FailureCode = "persistence_failure"
	FailUserCancellation   FailureCode = "user_cancellation"
	FailExecution          FailureCode = "execution_failure"
	FailInterrupted        FailureCode = "interrupted"
)

type Job struct {
	ID                string          `json:"id"`
	Title             string          `json:"title"`
	RequestedAction   string          `json:"requestedAction"`
	TargetAdapter     string          `json:"targetAdapter"`
	Status            Status          `json:"status"`
	CreatedAtMs       int64           `json:"createdAtMs"`
	UpdatedAtMs       int64           `json:"updatedAtMs"`
	QueuedAtMs        *int64          `json:"queuedAtMs"`
	StartedAtMs       *int64          `json:"startedAtMs"`
	CompletedAtMs     *int64          `json:"completedAtMs"`
	InitiatingSource  string          `json:"initiatingSource"`
	ExecutionBoundary string          `json:"executionBoundary"`
	RiskClass         RiskClass       `json:"riskClass"`
	ApprovalStatus    ApprovalStatus  `json:"approvalStatus"`
	WriteIntent       bool            `json:"writeIntent"`
	CancelRequested   bool            `json:"cancelRequested"`
	TaskPacketID      *int64          `json:"taskPacketId"`
	ResultSummary     *string         `json:"resultSummary"`
	FailureInfo       *string         `json:"failureInfo"`
	LastFailureCode   *FailureCode    `json:"lastFailureCode"`
	LastError         *string         `json:"lastError"`
	Metadata          json.RawMessage `json:"metadata"`
}

type JobEvent struct {
	ID          int64           `json:"id"`
	JobID       string          `json:"jobId"`
	CreatedAtMs int64           `json:"createdAtMs"`
	Type        string          `json:"type"`
	Message     string          `json:"message"`
	Payload     json.RawMessage `json:"payload"`
}

type StatusTransition struct {
	ID          int64   `json:"id"`
	JobID       string  `json:"jobId"`
	CreatedAtMs int64   `json:"createdAtMs"`
	FromStatus  *Status `json:"fromStatus"`
	ToStatus    Status  `json:"toStatus"`
	Reason      string  `json:"reason"`
}

type CreateRequest struct {
	TemplateID             string         `json:"templateId"`
	Title                  string         `json:"title"`
	UserRequest            string         `json:"userRequest"`
	Objective              string         `json:"objective"`
	InitiatingSource       string         `json:"initiatingSource"`
	Query                  string         `json:"query"`
	Scope                  ScopeInput     `json:"scope"`
	SourceContextRecordIDs []int64        `json:"sourceContextRecordIds"`
	Constraints            []string       `json:"constraints"`
	Instructions           string         `json:"instructions"`
	ExecutionMode          string         `json:"executionMode"`
	ExpectedOutput         map[string]any `json:"expectedOutput"`
	RequestPayload         map[string]any `json:"requestPayload"`
}

type ScopeInput struct {
	AllowedPaths   []string `json:"allowedPaths"`
	ForbiddenPaths []string `json:"forbiddenPaths"`
	SelectedPaths  []string `json:"selectedPaths"`
}

func (s ScopeInput) ToMap() map[string]any {
	return map[string]any{
		"allowedPaths":   s.AllowedPaths,
		"forbiddenPaths": s.ForbiddenPaths,
		"selectedPaths":  s.SelectedPaths,
	}
}
