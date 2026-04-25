package domain

import "fmt"

type TruthErrorCode string

const (
	TruthErrInvalidQuery     TruthErrorCode = "INVALID_TRUTH_QUERY"
	TruthErrMissingScope     TruthErrorCode = "MISSING_TRUTH_SCOPE"
	TruthErrNotFound         TruthErrorCode = "TRUTH_NOT_FOUND"
	TruthErrUnsupported      TruthErrorCode = "TRUTH_UNSUPPORTED"
	TruthErrInvalidOperation TruthErrorCode = "TRUTH_INVALID_OPERATION"
	TruthErrPersistence      TruthErrorCode = "TRUTH_PERSISTENCE_ERROR"
)

type TruthError struct {
	Code    TruthErrorCode `json:"code"`
	Field   string         `json:"field,omitempty"`
	Message string         `json:"message"`
}

func (e TruthError) Error() string {
	if e.Field == "" {
		return fmt.Sprintf("%s: %s", e.Code, e.Message)
	}
	return fmt.Sprintf("%s (%s): %s", e.Code, e.Field, e.Message)
}

type TruthQuery struct {
	Scope                 ForgeScope `json:"scope"`
	Key                   string     `json:"key,omitempty"`
	ObjectID              string     `json:"objectId,omitempty"`
	ObjectType            string     `json:"objectType,omitempty"`
	IncludeHistory        bool       `json:"includeHistory,omitempty"`
	IncludeEvidence       bool       `json:"includeEvidence,omitempty"`
	IncludeContradictions bool       `json:"includeContradictions,omitempty"`
	IncludeSupersessions  bool       `json:"includeSupersessions,omitempty"`
	Limit                 int        `json:"limit,omitempty"`
}

type StateTimelineEntry struct {
	VersionID     int64          `json:"versionId"`
	StateItemID   string         `json:"stateItemId"`
	Key           string         `json:"key"`
	PreviousValue map[string]any `json:"previousValue"`
	NewValue      map[string]any `json:"newValue"`
	ChangedBy     string         `json:"changedBy"`
	DerivedFrom   []string       `json:"derivedFrom"`
	SyscallID     string         `json:"syscallId"`
	AuditID       string         `json:"auditId"`
	CorrelationID string         `json:"correlationId"`
	TraceID       string         `json:"traceId"`
	UpdatedAt     int64          `json:"updatedAt"`
	Metadata      map[string]any `json:"metadata"`
}

type StateExplanation struct {
	Key               string               `json:"key"`
	Scope             ForgeScope           `json:"scope"`
	CurrentValue      map[string]any       `json:"currentValue,omitempty"`
	CurrentStateID    string               `json:"currentStateId,omitempty"`
	UpdatedAt         int64                `json:"updatedAt,omitempty"`
	DerivedFrom       []string             `json:"derivedFrom,omitempty"`
	PreviousValues    []StateTimelineEntry `json:"previousValues,omitempty"`
	Contradictions    []string             `json:"contradictions,omitempty"`
	SupersessionChain []string             `json:"supersessionChain,omitempty"`
	Confidence        float64              `json:"confidence,omitempty"`
	AuditID           string               `json:"auditId,omitempty"`
	CorrelationID     string               `json:"correlationId,omitempty"`
	TraceID           string               `json:"traceId,omitempty"`
}

type OpenLoopExplanation struct {
	LoopID        string        `json:"loopId"`
	Scope         ForgeScope    `json:"scope"`
	State         OpenLoopState `json:"state"`
	Priority      string        `json:"priority"`
	Owner         string        `json:"owner"`
	Blocker       string        `json:"blocker,omitempty"`
	NextAction    string        `json:"nextAction,omitempty"`
	CreatedFrom   string        `json:"createdFrom,omitempty"`
	RelatedNotes  []string      `json:"relatedNotes,omitempty"`
	CreatedAt     int64         `json:"createdAt"`
	UpdatedAt     int64         `json:"updatedAt"`
	IsStale       bool          `json:"isStale"`
	StaleCutoffMs int64         `json:"staleCutoffMs,omitempty"`
	Warnings      []string      `json:"warnings,omitempty"`
	CorrelationID string        `json:"correlationId,omitempty"`
	TraceID       string        `json:"traceId,omitempty"`
	SyscallID     string        `json:"syscallId,omitempty"`
	AuditID       string        `json:"auditId,omitempty"`
}

type ContradictionExplanation struct {
	RecordID        string     `json:"recordId"`
	Scope           ForgeScope `json:"scope"`
	LeftObjectID    string     `json:"leftObjectId"`
	LeftObjectKind  string     `json:"leftObjectKind"`
	RightObjectID   string     `json:"rightObjectId"`
	RightObjectKind string     `json:"rightObjectKind"`
	Reason          string     `json:"reason"`
	Severity        string     `json:"severity"`
	Confidence      float64    `json:"confidence"`
	CreatedAt       int64      `json:"createdAt"`
	CorrelationID   string     `json:"correlationId,omitempty"`
	TraceID         string     `json:"traceId,omitempty"`
	SyscallID       string     `json:"syscallId,omitempty"`
	AuditID         string     `json:"auditId,omitempty"`
}

type SupersessionExplanation struct {
	Scope           ForgeScope `json:"scope"`
	RootObjectID    string     `json:"rootObjectId"`
	CurrentObjectID string     `json:"currentObjectId"`
	Chain           []string   `json:"chain"`
	Reasons         []string   `json:"reasons,omitempty"`
	RecordIDs       []string   `json:"recordIds,omitempty"`
}

type CurrentObjectResolution struct {
	ObjectID          string     `json:"objectId"`
	Scope             ForgeScope `json:"scope"`
	Current           bool       `json:"current"`
	CurrentObjectID   string     `json:"currentObjectId,omitempty"`
	Archived          bool       `json:"archived"`
	Superseded        bool       `json:"superseded"`
	Deprecated        bool       `json:"deprecated"`
	Contradicted      bool       `json:"contradicted"`
	IncludeInActive   bool       `json:"includeInActive"`
	Warnings          []string   `json:"warnings,omitempty"`
	SupersessionChain []string   `json:"supersessionChain,omitempty"`
}

type TruthExplanation struct {
	Query          TruthQuery                 `json:"query"`
	Status         string                     `json:"status"`
	CurrentState   *StateExplanation          `json:"currentState,omitempty"`
	CurrentObject  *CurrentObjectResolution   `json:"currentObject,omitempty"`
	Loops          []OpenLoopExplanation      `json:"loops,omitempty"`
	Contradictions []ContradictionExplanation `json:"contradictions,omitempty"`
	Supersession   *SupersessionExplanation   `json:"supersession,omitempty"`
	Timeline       []StateTimelineEntry       `json:"timeline,omitempty"`
	Warnings       []string                   `json:"warnings,omitempty"`
}

type TruthApplySummary struct {
	Action             SemanticActionType `json:"action"`
	RequestID          string             `json:"requestId"`
	Success            bool               `json:"success"`
	Scope              ForgeScope         `json:"scope"`
	CommittedObjectIDs []string           `json:"committedObjectIds"`
	Warnings           []string           `json:"warnings,omitempty"`
}

type ProjectionRebuildDiff struct {
	Category string         `json:"category"`
	Key      string         `json:"key,omitempty"`
	ObjectID string         `json:"objectId,omitempty"`
	Message  string         `json:"message"`
	Severity string         `json:"severity"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

type ProjectionRebuildReport struct {
	Scope       ForgeScope              `json:"scope"`
	DryRun      bool                    `json:"dryRun"`
	GeneratedAt int64                   `json:"generatedAt"`
	Differences []ProjectionRebuildDiff `json:"differences"`
	Warnings    []string                `json:"warnings,omitempty"`
	Applied     bool                    `json:"applied"`
}
