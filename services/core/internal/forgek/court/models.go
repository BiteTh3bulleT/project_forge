package court

import "time"

type ExhibitInput struct {
	ExhibitID      string
	CaseID         string
	WorkspaceID    string
	SourceObjectID string
	SubmittedBy    string
	SourceType     string
	SourceRefs     []string
	ClaimRefs      []string
	ContentSummary string
	RawRef         string
	CreatedAt      time.Time
	Metadata       map[string]any
}

type Exhibit struct {
	ExhibitID           string
	CaseID              string
	WorkspaceID         string
	SourceObjectID      string
	SubmittedBy         string
	SourceType          string
	SourceRefs          []string
	ClaimRefs           []string
	ContentSummary      string
	RawRef              string
	AdmissibilityStatus string
	AdmissionReason     string
	RejectionReason     string
	ContradictionRefs   []string
	SupersessionRefs    []string
	CreatedAt           time.Time
	UpdatedAt           time.Time
	JournalRefs         []string
	Metadata            map[string]any
}

func NewExhibit(input ExhibitInput) (Exhibit, error) {
	if input.ExhibitID == "" || input.CaseID == "" || input.WorkspaceID == "" || input.SubmittedBy == "" || input.SourceType == "" {
		return Exhibit{}, ErrInvalidExhibit
	}
	if input.ContentSummary == "" && input.SourceObjectID == "" && len(input.SourceRefs) == 0 {
		return Exhibit{}, ErrInvalidExhibit
	}
	return Exhibit{
		ExhibitID:           input.ExhibitID,
		CaseID:              input.CaseID,
		WorkspaceID:         input.WorkspaceID,
		SourceObjectID:      input.SourceObjectID,
		SubmittedBy:         input.SubmittedBy,
		SourceType:          input.SourceType,
		SourceRefs:          append([]string(nil), input.SourceRefs...),
		ClaimRefs:           append([]string(nil), input.ClaimRefs...),
		ContentSummary:      input.ContentSummary,
		RawRef:              input.RawRef,
		AdmissibilityStatus: StatusSubmitted,
		CreatedAt:           input.CreatedAt,
		UpdatedAt:           input.CreatedAt,
		Metadata:            cloneMap(input.Metadata),
	}, nil
}

func (e Exhibit) Clone() Exhibit {
	e.SourceRefs = append([]string(nil), e.SourceRefs...)
	e.ClaimRefs = append([]string(nil), e.ClaimRefs...)
	e.ContradictionRefs = append([]string(nil), e.ContradictionRefs...)
	e.SupersessionRefs = append([]string(nil), e.SupersessionRefs...)
	e.JournalRefs = append([]string(nil), e.JournalRefs...)
	e.Metadata = cloneMap(e.Metadata)
	return e
}

type Claim struct {
	ClaimID        string
	CaseID         string
	WorkspaceID    string
	ExhibitID      string
	ClaimType      string
	Statement      string
	Confidence     float64
	SourceRefs     []string
	Status         string
	ContradictedBy []string
	SupersededBy   []string
	CreatedAt      time.Time
	JournalRefs    []string
}

type RulingInput struct {
	RulingID            string
	CaseID              string
	WorkspaceID         string
	RulingType          string
	AdmittedExhibitRefs []string
	RejectedExhibitRefs []string
	ContradictionRefs   []string
	SupersessionRefs    []string
	ReasoningSummary    string
	PolicyRefs          []string
	CreatedBy           string
	CreatedAt           time.Time
	Metadata            map[string]any
}

type Ruling struct {
	RulingID            string
	CaseID              string
	WorkspaceID         string
	RulingType          string
	AdmittedExhibitRefs []string
	RejectedExhibitRefs []string
	ContradictionRefs   []string
	SupersessionRefs    []string
	ReasoningSummary    string
	PolicyRefs          []string
	CreatedBy           string
	CreatedAt           time.Time
	JournalRef          string
	Metadata            map[string]any
}

func NewRuling(input RulingInput) (Ruling, error) {
	if input.RulingID == "" || input.CaseID == "" || input.WorkspaceID == "" || input.RulingType == "" || input.CreatedBy == "" {
		return Ruling{}, ErrInvalidRuling
	}
	if len(input.AdmittedExhibitRefs)+len(input.RejectedExhibitRefs)+len(input.ContradictionRefs)+len(input.SupersessionRefs) == 0 && input.RulingType != RulingCaseSummary {
		return Ruling{}, ErrInvalidRuling
	}
	return Ruling{
		RulingID:            input.RulingID,
		CaseID:              input.CaseID,
		WorkspaceID:         input.WorkspaceID,
		RulingType:          input.RulingType,
		AdmittedExhibitRefs: append([]string(nil), input.AdmittedExhibitRefs...),
		RejectedExhibitRefs: append([]string(nil), input.RejectedExhibitRefs...),
		ContradictionRefs:   append([]string(nil), input.ContradictionRefs...),
		SupersessionRefs:    append([]string(nil), input.SupersessionRefs...),
		ReasoningSummary:    input.ReasoningSummary,
		PolicyRefs:          append([]string(nil), input.PolicyRefs...),
		CreatedBy:           input.CreatedBy,
		CreatedAt:           input.CreatedAt,
		Metadata:            cloneMap(input.Metadata),
	}, nil
}

func (r Ruling) Clone() Ruling {
	r.AdmittedExhibitRefs = append([]string(nil), r.AdmittedExhibitRefs...)
	r.RejectedExhibitRefs = append([]string(nil), r.RejectedExhibitRefs...)
	r.ContradictionRefs = append([]string(nil), r.ContradictionRefs...)
	r.SupersessionRefs = append([]string(nil), r.SupersessionRefs...)
	r.PolicyRefs = append([]string(nil), r.PolicyRefs...)
	r.Metadata = cloneMap(r.Metadata)
	return r
}

type ContradictionInput struct {
	ContradictionID   string
	CaseID            string
	WorkspaceID       string
	ExhibitAID        string
	ExhibitBID        string
	ClaimAID          string
	ClaimBID          string
	ContradictionType string
	Description       string
	Severity          string
	CreatedAt         time.Time
}

type Contradiction struct {
	ContradictionID   string
	CaseID            string
	WorkspaceID       string
	ExhibitAID        string
	ExhibitBID        string
	ClaimAID          string
	ClaimBID          string
	ContradictionType string
	Description       string
	Severity          string
	Status            string
	CreatedAt         time.Time
	ResolvedAt        *time.Time
	JournalRefs       []string
}

func NewContradiction(input ContradictionInput) (Contradiction, error) {
	if input.ContradictionID == "" || input.CaseID == "" || input.WorkspaceID == "" || input.ExhibitAID == "" || input.ExhibitBID == "" {
		return Contradiction{}, ErrInvalidContradiction
	}
	return Contradiction{
		ContradictionID:   input.ContradictionID,
		CaseID:            input.CaseID,
		WorkspaceID:       input.WorkspaceID,
		ExhibitAID:        input.ExhibitAID,
		ExhibitBID:        input.ExhibitBID,
		ClaimAID:          input.ClaimAID,
		ClaimBID:          input.ClaimBID,
		ContradictionType: input.ContradictionType,
		Description:       input.Description,
		Severity:          input.Severity,
		Status:            ContradictionOpen,
		CreatedAt:         input.CreatedAt,
	}, nil
}

func (c Contradiction) Clone() Contradiction {
	c.JournalRefs = append([]string(nil), c.JournalRefs...)
	if c.ResolvedAt != nil {
		resolvedAt := *c.ResolvedAt
		c.ResolvedAt = &resolvedAt
	}
	return c
}

type SupersessionInput struct {
	SupersessionID string
	CaseID         string
	WorkspaceID    string
	OldObjectID    string
	NewObjectID    string
	Reason         string
	CreatedAt      time.Time
}

type Supersession struct {
	SupersessionID string
	CaseID         string
	WorkspaceID    string
	OldObjectID    string
	NewObjectID    string
	Reason         string
	CreatedAt      time.Time
	JournalRef     string
}

func NewSupersession(input SupersessionInput) (Supersession, error) {
	if input.SupersessionID == "" || input.CaseID == "" || input.WorkspaceID == "" || input.OldObjectID == "" || input.NewObjectID == "" || input.Reason == "" {
		return Supersession{}, ErrInvalidSupersession
	}
	return Supersession{
		SupersessionID: input.SupersessionID,
		CaseID:         input.CaseID,
		WorkspaceID:    input.WorkspaceID,
		OldObjectID:    input.OldObjectID,
		NewObjectID:    input.NewObjectID,
		Reason:         input.Reason,
		CreatedAt:      input.CreatedAt,
	}, nil
}

func cloneMap(in map[string]any) map[string]any {
	if in == nil {
		return nil
	}
	out := make(map[string]any, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}
