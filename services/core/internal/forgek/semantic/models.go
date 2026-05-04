package semantic

import "time"

type SemanticObjectInput struct {
	SemanticObjectID    string
	WorkspaceID         string
	ObjectType          string
	SourceObjectRefs    []string
	SourceRefs          []string
	ContentSummary      string
	NormalizedContent   string
	Confidence          float64
	AuthorityLevel      string
	AdmissibilityStatus string
	ProvenanceRefs      []string
	SupersededBy        []string
	ContradictedBy      []string
	CreatedAt           time.Time
	Metadata            map[string]any
}

type SemanticObject struct {
	SemanticObjectID    string
	WorkspaceID         string
	ObjectType          string
	SourceObjectRefs    []string
	SourceRefs          []string
	ContentSummary      string
	NormalizedContent   string
	Confidence          float64
	AuthorityLevel      string
	AdmissibilityStatus string
	ProvenanceRefs      []string
	SupersededBy        []string
	ContradictedBy      []string
	CreatedAt           time.Time
	UpdatedAt           time.Time
	JournalRefs         []string
	Metadata            map[string]any
}

func NewSemanticObject(input SemanticObjectInput) (SemanticObject, error) {
	if input.SemanticObjectID == "" || input.WorkspaceID == "" || input.ObjectType == "" {
		return SemanticObject{}, ErrInvalidSemanticObject
	}
	if input.ContentSummary == "" && input.NormalizedContent == "" && len(input.SourceObjectRefs)+len(input.SourceRefs) == 0 {
		return SemanticObject{}, ErrInvalidSemanticObject
	}
	authority := input.AuthorityLevel
	if authority == "" {
		authority = AuthorityProposal
	}
	return SemanticObject{
		SemanticObjectID:    input.SemanticObjectID,
		WorkspaceID:         input.WorkspaceID,
		ObjectType:          input.ObjectType,
		SourceObjectRefs:    append([]string(nil), input.SourceObjectRefs...),
		SourceRefs:          append([]string(nil), input.SourceRefs...),
		ContentSummary:      input.ContentSummary,
		NormalizedContent:   input.NormalizedContent,
		Confidence:          input.Confidence,
		AuthorityLevel:      authority,
		AdmissibilityStatus: input.AdmissibilityStatus,
		ProvenanceRefs:      append([]string(nil), input.ProvenanceRefs...),
		SupersededBy:        append([]string(nil), input.SupersededBy...),
		ContradictedBy:      append([]string(nil), input.ContradictedBy...),
		CreatedAt:           input.CreatedAt,
		UpdatedAt:           input.CreatedAt,
		Metadata:            cloneMap(input.Metadata),
	}, nil
}

func (o SemanticObject) Clone() SemanticObject {
	o.SourceObjectRefs = append([]string(nil), o.SourceObjectRefs...)
	o.SourceRefs = append([]string(nil), o.SourceRefs...)
	o.ProvenanceRefs = append([]string(nil), o.ProvenanceRefs...)
	o.SupersededBy = append([]string(nil), o.SupersededBy...)
	o.ContradictedBy = append([]string(nil), o.ContradictedBy...)
	o.JournalRefs = append([]string(nil), o.JournalRefs...)
	o.Metadata = cloneMap(o.Metadata)
	return o
}

func (o SemanticObject) IsCanonicalTruth() bool {
	return false
}

type SemanticOperationInput struct {
	OperationID      string
	OperationType    string
	WorkspaceID      string
	CaseID           string
	InputObjectRefs  []string
	OutputObjectRefs []string
	OperatorVersion  string
	Parameters       map[string]any
	ReasoningSummary string
	ProvenanceRefs   []string
	CreatedBy        string
	CreatedAt        time.Time
	Metadata         map[string]any
}

type SemanticOperation struct {
	OperationID      string
	OperationType    string
	WorkspaceID      string
	CaseID           string
	InputObjectRefs  []string
	OutputObjectRefs []string
	OperatorVersion  string
	Parameters       map[string]any
	ReasoningSummary string
	ProvenanceRefs   []string
	CreatedBy        string
	CreatedAt        time.Time
	JournalRef       string
	Metadata         map[string]any
}

func NewSemanticOperation(input SemanticOperationInput) (SemanticOperation, error) {
	if input.OperationID == "" || input.OperationType == "" || input.WorkspaceID == "" || input.CreatedBy == "" {
		return SemanticOperation{}, ErrInvalidSemanticOperation
	}
	return SemanticOperation{
		OperationID:      input.OperationID,
		OperationType:    input.OperationType,
		WorkspaceID:      input.WorkspaceID,
		CaseID:           input.CaseID,
		InputObjectRefs:  append([]string(nil), input.InputObjectRefs...),
		OutputObjectRefs: append([]string(nil), input.OutputObjectRefs...),
		OperatorVersion:  input.OperatorVersion,
		Parameters:       cloneMap(input.Parameters),
		ReasoningSummary: input.ReasoningSummary,
		ProvenanceRefs:   append([]string(nil), input.ProvenanceRefs...),
		CreatedBy:        input.CreatedBy,
		CreatedAt:        input.CreatedAt,
		Metadata:         cloneMap(input.Metadata),
	}, nil
}

func (o SemanticOperation) Clone() SemanticOperation {
	o.InputObjectRefs = append([]string(nil), o.InputObjectRefs...)
	o.OutputObjectRefs = append([]string(nil), o.OutputObjectRefs...)
	o.Parameters = cloneMap(o.Parameters)
	o.ProvenanceRefs = append([]string(nil), o.ProvenanceRefs...)
	o.Metadata = cloneMap(o.Metadata)
	return o
}

type SemanticSyscallRequest struct {
	RequestID          string
	SyscallName        string
	WorkspaceID        string
	CaseID             string
	Payload            map[string]any
	Reason             string
	RequiredCapability string
	CreatedAt          time.Time
}

func (r SemanticSyscallRequest) Clone() SemanticSyscallRequest {
	r.Payload = cloneMap(r.Payload)
	return r
}

type SemanticTransformResult struct {
	ResultID          string
	OperationID       string
	OperationType     string
	WorkspaceID       string
	CaseID            string
	OutputObjects     []SemanticObject
	OutputRefs        []string
	RequestedSyscalls []SemanticSyscallRequest
	Warnings          []string
	Errors            []string
	ProvenanceRefs    []string
	CreatedAt         time.Time
	Metadata          map[string]any
}

func (r SemanticTransformResult) Clone() SemanticTransformResult {
	r.OutputObjects = cloneObjects(r.OutputObjects)
	r.OutputRefs = append([]string(nil), r.OutputRefs...)
	r.RequestedSyscalls = cloneSyscallRequests(r.RequestedSyscalls)
	r.Warnings = append([]string(nil), r.Warnings...)
	r.Errors = append([]string(nil), r.Errors...)
	r.ProvenanceRefs = append([]string(nil), r.ProvenanceRefs...)
	r.Metadata = cloneMap(r.Metadata)
	return r
}

type OperationRequest struct {
	OperationID   string
	ResultID      string
	OperationType string
	WorkspaceID   string
	CaseID        string
	InputObjects  []SemanticObject
	Parameters    map[string]any
	CreatedBy     string
	CreatedAt     time.Time
	NextObjectID  func() string
}

func cloneObjects(in []SemanticObject) []SemanticObject {
	out := make([]SemanticObject, len(in))
	for i, obj := range in {
		out[i] = obj.Clone()
	}
	return out
}

func cloneSyscallRequests(in []SemanticSyscallRequest) []SemanticSyscallRequest {
	out := make([]SemanticSyscallRequest, len(in))
	for i, request := range in {
		out[i] = request.Clone()
	}
	return out
}

func cloneMap(in map[string]any) map[string]any {
	if in == nil {
		return nil
	}
	out := make(map[string]any, len(in))
	for key, value := range in {
		switch typed := value.(type) {
		case []string:
			out[key] = append([]string(nil), typed...)
		default:
			out[key] = value
		}
	}
	return out
}
