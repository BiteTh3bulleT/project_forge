package contextcompiler

import (
	"sort"
	"strings"
	"time"
)

type ContextBlockInput struct {
	BlockID               string
	BlockType             ContextBlockType
	WorkspaceID           string
	CaseID                string
	SnapshotID            string
	RestoreSeedID         string
	SourceObjectRefs      []string
	SourceRefs            []string
	AdmittedExhibitRefs   []string
	RejectedExhibitRefs   []string
	RulingRefs            []string
	ContradictionRefs     []string
	SupersessionRefs      []string
	PalaceRouteRefs       []string
	SemanticOperationRefs []string
	DerivedObjectRefs     []string
	ContentSummary        string
	CanonicalText         string
	LayoutPosition        int
	CacheEligibility      CacheEligibility
	InvalidationScope     string
	PolicyVersion         string
	SyscallSchemaVersion  string
	CreatedBy             string
	CreatedAt             time.Time
	JournalRefs           []string
	Metadata              map[string]any
}

type ContextBlock struct {
	BlockID               string           `json:"block_id"`
	BlockType             ContextBlockType `json:"block_type"`
	WorkspaceID           string           `json:"workspace_id"`
	CaseID                string           `json:"case_id,omitempty"`
	SnapshotID            string           `json:"snapshot_id,omitempty"`
	RestoreSeedID         string           `json:"restore_seed_id,omitempty"`
	SourceObjectRefs      []string         `json:"source_object_refs,omitempty"`
	SourceRefs            []string         `json:"source_refs,omitempty"`
	AdmittedExhibitRefs   []string         `json:"admitted_exhibit_refs,omitempty"`
	RejectedExhibitRefs   []string         `json:"rejected_exhibit_refs,omitempty"`
	RulingRefs            []string         `json:"ruling_refs,omitempty"`
	ContradictionRefs     []string         `json:"contradiction_refs,omitempty"`
	SupersessionRefs      []string         `json:"supersession_refs,omitempty"`
	PalaceRouteRefs       []string         `json:"palace_route_refs,omitempty"`
	SemanticOperationRefs []string         `json:"semantic_operation_refs,omitempty"`
	DerivedObjectRefs     []string         `json:"derived_object_refs,omitempty"`
	ContentSummary        string           `json:"content_summary,omitempty"`
	CanonicalText         string           `json:"canonical_text"`
	ContentHash           string           `json:"content_hash"`
	TokenInputHash        string           `json:"token_input_hash"`
	TokenCountEstimate    int              `json:"token_count_estimate"`
	LayoutPosition        int              `json:"layout_position"`
	CacheEligibility      CacheEligibility `json:"cache_eligibility"`
	InvalidationScope     string           `json:"invalidation_scope"`
	PolicyVersion         string           `json:"policy_version"`
	SyscallSchemaVersion  string           `json:"syscall_schema_version"`
	CreatedBy             string           `json:"created_by"`
	CreatedAt             time.Time        `json:"created_at"`
	JournalRefs           []string         `json:"journal_refs,omitempty"`
	Metadata              map[string]any   `json:"metadata,omitempty"`
}

type ContextCompileRequest struct {
	RequestID                      string         `json:"request_id"`
	BundleID                       string         `json:"bundle_id"`
	WorkspaceID                    string         `json:"workspace_id"`
	CaseID                         string         `json:"case_id,omitempty"`
	SnapshotID                     string         `json:"snapshot_id,omitempty"`
	RestoreSeedID                  string         `json:"restore_seed_id,omitempty"`
	UserMessage                    string         `json:"user_message,omitempty"`
	CurrentTaskSummary             string         `json:"current_task_summary,omitempty"`
	ActiveConstraints              []string       `json:"active_constraints,omitempty"`
	SourceObjectRefs               []string       `json:"source_object_refs,omitempty"`
	SourceRefs                     []string       `json:"source_refs,omitempty"`
	AdmittedExhibitRefs            []string       `json:"admitted_exhibit_refs,omitempty"`
	RejectedExhibitRefs            []string       `json:"rejected_exhibit_refs,omitempty"`
	RulingRefs                     []string       `json:"ruling_refs,omitempty"`
	ContradictionRefs              []string       `json:"contradiction_refs,omitempty"`
	SupersessionRefs               []string       `json:"supersession_refs,omitempty"`
	PalaceRouteRefs                []string       `json:"palace_route_refs,omitempty"`
	SemanticOperationRefs          []string       `json:"semantic_operation_refs,omitempty"`
	DerivedObjectRefs              []string       `json:"derived_object_refs,omitempty"`
	IncludeRejectedEvidenceSummary bool           `json:"include_rejected_evidence_summary"`
	IncludeContradictions          bool           `json:"include_contradictions"`
	IncludeRestoreSeed             bool           `json:"include_restore_seed"`
	LayoutVersion                  string         `json:"layout_version"`
	PolicyVersion                  string         `json:"policy_version"`
	SyscallSchemaVersion           string         `json:"syscall_schema_version"`
	TokenBudget                    int            `json:"token_budget,omitempty"`
	CreatedBy                      string         `json:"created_by"`
	CreatedAt                      time.Time      `json:"created_at"`
	Metadata                       map[string]any `json:"metadata,omitempty"`
}

type ContextCompileResult struct {
	ResultID       string         `json:"result_id"`
	RequestID      string         `json:"request_id"`
	WorkspaceID    string         `json:"workspace_id"`
	CaseID         string         `json:"case_id,omitempty"`
	SnapshotID     string         `json:"snapshot_id,omitempty"`
	RestoreSeedID  string         `json:"restore_seed_id,omitempty"`
	Bundle         ContextBundle  `json:"bundle"`
	Blocks         []ContextBlock `json:"blocks"`
	Warnings       []string       `json:"warnings,omitempty"`
	Errors         []string       `json:"errors,omitempty"`
	ProvenanceRefs []string       `json:"provenance_refs,omitempty"`
	CreatedAt      time.Time      `json:"created_at"`
	Metadata       map[string]any `json:"metadata,omitempty"`
}

func NewContextBlock(input ContextBlockInput) (ContextBlock, error) {
	if input.BlockID == "" || input.WorkspaceID == "" || input.CreatedBy == "" {
		return ContextBlock{}, ErrInvalidContextBlock
	}
	if !ValidBlockType(input.BlockType) {
		return ContextBlock{}, ErrInvalidBlockType
	}
	block := ContextBlock{
		BlockID:               input.BlockID,
		BlockType:             input.BlockType,
		WorkspaceID:           input.WorkspaceID,
		CaseID:                strings.TrimSpace(input.CaseID),
		SnapshotID:            strings.TrimSpace(input.SnapshotID),
		RestoreSeedID:         strings.TrimSpace(input.RestoreSeedID),
		SourceObjectRefs:      NormalizeRefs(input.SourceObjectRefs),
		SourceRefs:            NormalizeRefs(input.SourceRefs),
		AdmittedExhibitRefs:   NormalizeRefs(input.AdmittedExhibitRefs),
		RejectedExhibitRefs:   NormalizeRefs(input.RejectedExhibitRefs),
		RulingRefs:            NormalizeRefs(input.RulingRefs),
		ContradictionRefs:     NormalizeRefs(input.ContradictionRefs),
		SupersessionRefs:      NormalizeRefs(input.SupersessionRefs),
		PalaceRouteRefs:       NormalizeRefs(input.PalaceRouteRefs),
		SemanticOperationRefs: NormalizeRefs(input.SemanticOperationRefs),
		DerivedObjectRefs:     NormalizeRefs(input.DerivedObjectRefs),
		ContentSummary:        NormalizeWhitespace(input.ContentSummary),
		LayoutPosition:        input.LayoutPosition,
		CacheEligibility:      input.CacheEligibility,
		InvalidationScope:     strings.TrimSpace(input.InvalidationScope),
		PolicyVersion:         strings.TrimSpace(input.PolicyVersion),
		SyscallSchemaVersion:  strings.TrimSpace(input.SyscallSchemaVersion),
		CreatedBy:             input.CreatedBy,
		CreatedAt:             input.CreatedAt,
		JournalRefs:           NormalizeRefs(input.JournalRefs),
		Metadata:              CloneMap(input.Metadata),
	}
	if block.CacheEligibility == "" {
		block.CacheEligibility = DefaultCacheEligibility(block.BlockType)
	}
	if block.InvalidationScope == "" {
		block.InvalidationScope = DefaultContextInvalidationScope
	}
	if block.PolicyVersion == "" {
		block.PolicyVersion = DefaultPolicyVersion
	}
	if block.SyscallSchemaVersion == "" {
		block.SyscallSchemaVersion = DefaultSyscallSchemaVersion
	}
	if input.CanonicalText != "" {
		block.CanonicalText = NormalizeWhitespace(input.CanonicalText)
	} else {
		block.CanonicalText = SerializeBlock(block)
	}
	block.ContentHash = ContentHash(block)
	block.TokenInputHash = TokenInputHash(block.CanonicalText)
	block.TokenCountEstimate = EstimateTokens(block.CanonicalText)
	return block, nil
}

func FinalizeBlock(block ContextBlock) ContextBlock {
	block.SourceObjectRefs = NormalizeRefs(block.SourceObjectRefs)
	block.SourceRefs = NormalizeRefs(block.SourceRefs)
	block.AdmittedExhibitRefs = NormalizeRefs(block.AdmittedExhibitRefs)
	block.RejectedExhibitRefs = NormalizeRefs(block.RejectedExhibitRefs)
	block.RulingRefs = NormalizeRefs(block.RulingRefs)
	block.ContradictionRefs = NormalizeRefs(block.ContradictionRefs)
	block.SupersessionRefs = NormalizeRefs(block.SupersessionRefs)
	block.PalaceRouteRefs = NormalizeRefs(block.PalaceRouteRefs)
	block.SemanticOperationRefs = NormalizeRefs(block.SemanticOperationRefs)
	block.DerivedObjectRefs = NormalizeRefs(block.DerivedObjectRefs)
	block.JournalRefs = NormalizeRefs(block.JournalRefs)
	block.ContentSummary = NormalizeWhitespace(block.ContentSummary)
	block.CanonicalText = SerializeBlock(block)
	block.ContentHash = ContentHash(block)
	block.TokenInputHash = TokenInputHash(block.CanonicalText)
	block.TokenCountEstimate = EstimateTokens(block.CanonicalText)
	return block
}

func (b ContextBlock) Clone() ContextBlock {
	b.SourceObjectRefs = CloneStrings(b.SourceObjectRefs)
	b.SourceRefs = CloneStrings(b.SourceRefs)
	b.AdmittedExhibitRefs = CloneStrings(b.AdmittedExhibitRefs)
	b.RejectedExhibitRefs = CloneStrings(b.RejectedExhibitRefs)
	b.RulingRefs = CloneStrings(b.RulingRefs)
	b.ContradictionRefs = CloneStrings(b.ContradictionRefs)
	b.SupersessionRefs = CloneStrings(b.SupersessionRefs)
	b.PalaceRouteRefs = CloneStrings(b.PalaceRouteRefs)
	b.SemanticOperationRefs = CloneStrings(b.SemanticOperationRefs)
	b.DerivedObjectRefs = CloneStrings(b.DerivedObjectRefs)
	b.JournalRefs = CloneStrings(b.JournalRefs)
	b.Metadata = CloneMap(b.Metadata)
	return b
}

func (b ContextBlock) AllRefs() []string {
	refs := []string{b.CaseID, b.SnapshotID, b.RestoreSeedID}
	for _, values := range [][]string{
		b.SourceObjectRefs,
		b.SourceRefs,
		b.AdmittedExhibitRefs,
		b.RejectedExhibitRefs,
		b.RulingRefs,
		b.ContradictionRefs,
		b.SupersessionRefs,
		b.PalaceRouteRefs,
		b.SemanticOperationRefs,
		b.DerivedObjectRefs,
	} {
		refs = append(refs, values...)
	}
	return NormalizeRefs(refs)
}

func (b ContextBlock) IsCanonicalTruth() bool { return false }
func (b ContextBlock) IsKVCache() bool        { return false }

func NormalizeCompileRequest(request ContextCompileRequest) ContextCompileRequest {
	request.ActiveConstraints = NormalizeRefs(request.ActiveConstraints)
	request.SourceObjectRefs = NormalizeRefs(request.SourceObjectRefs)
	request.SourceRefs = NormalizeRefs(request.SourceRefs)
	request.AdmittedExhibitRefs = NormalizeRefs(request.AdmittedExhibitRefs)
	request.RejectedExhibitRefs = NormalizeRefs(request.RejectedExhibitRefs)
	request.RulingRefs = NormalizeRefs(request.RulingRefs)
	request.ContradictionRefs = NormalizeRefs(request.ContradictionRefs)
	request.SupersessionRefs = NormalizeRefs(request.SupersessionRefs)
	request.PalaceRouteRefs = NormalizeRefs(request.PalaceRouteRefs)
	request.SemanticOperationRefs = NormalizeRefs(request.SemanticOperationRefs)
	request.DerivedObjectRefs = NormalizeRefs(request.DerivedObjectRefs)
	request.UserMessage = NormalizeWhitespace(request.UserMessage)
	request.CurrentTaskSummary = NormalizeWhitespace(request.CurrentTaskSummary)
	request.Metadata = CloneMap(request.Metadata)
	if request.LayoutVersion == "" {
		request.LayoutVersion = DefaultLayoutVersion
	}
	if request.PolicyVersion == "" {
		request.PolicyVersion = DefaultPolicyVersion
	}
	if request.SyscallSchemaVersion == "" {
		request.SyscallSchemaVersion = DefaultSyscallSchemaVersion
	}
	return request
}

func ValidateCompileRequest(request ContextCompileRequest) error {
	if request.WorkspaceID == "" || request.CreatedBy == "" {
		return ErrInvalidCompileRequest
	}
	if request.CaseID == "" && request.SnapshotID == "" && request.RestoreSeedID == "" &&
		len(request.SourceObjectRefs)+len(request.SourceRefs)+len(request.AdmittedExhibitRefs)+len(request.RejectedExhibitRefs)+
			len(request.RulingRefs)+len(request.ContradictionRefs)+len(request.SupersessionRefs)+len(request.PalaceRouteRefs)+
			len(request.SemanticOperationRefs)+len(request.DerivedObjectRefs) == 0 {
		return ErrInvalidCompileRequest
	}
	return nil
}

func NormalizeRefs(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func CloneStrings(values []string) []string {
	return append([]string(nil), values...)
}

func CloneMap(in map[string]any) map[string]any {
	if in == nil {
		return nil
	}
	out := make(map[string]any, len(in))
	for key, value := range in {
		switch typed := value.(type) {
		case []string:
			out[key] = CloneStrings(typed)
		case []any:
			out[key] = append([]any(nil), typed...)
		case map[string]any:
			out[key] = CloneMap(typed)
		case map[string]string:
			nested := make(map[string]string, len(typed))
			for nestedKey, nestedValue := range typed {
				nested[nestedKey] = nestedValue
			}
			out[key] = nested
		default:
			out[key] = value
		}
	}
	return out
}
