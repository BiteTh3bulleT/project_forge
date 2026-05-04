package lymphatic

import (
	"sort"
	"strings"
	"time"

	"forge/projectforge/services/core/internal/forgek/contextcompiler"
	"forge/projectforge/services/core/internal/forgek/court"
	"forge/projectforge/services/core/internal/forgek/kv"
	forgekRuntime "forge/projectforge/services/core/internal/forgek/runtime"
	"forge/projectforge/services/core/internal/forgek/snapshots"
)

type LymphaticPolicy struct {
	PolicyID                      string         `json:"policy_id"`
	WorkspaceID                   string         `json:"workspace_id"`
	SweepKindsEnabled             []SweepKind    `json:"sweep_kinds_enabled"`
	MaxReportItems                int            `json:"max_report_items"`
	StaleAfterDurationMS          int64          `json:"stale_after_duration_ms"`
	ExpireAfterDurationMS         int64          `json:"expire_after_duration_ms"`
	KvColdAfterReuseCount         int            `json:"kv_cold_after_reuse_count"`
	KvExpireAfterDurationMS       int64          `json:"kv_expire_after_duration_ms"`
	SnapshotExpireAfterDurationMS int64          `json:"snapshot_expire_after_duration_ms"`
	RuntimeResultStaleAfterMS     int64          `json:"runtime_result_stale_after_ms"`
	IncludeRejectedEvidence       bool           `json:"include_rejected_evidence"`
	IncludeSupersededObjects      bool           `json:"include_superseded_objects"`
	IncludeInvalidatedKV          bool           `json:"include_invalidated_kv"`
	DryRun                        bool           `json:"dry_run"`
	CreatedAt                     time.Time      `json:"created_at"`
	Metadata                      map[string]any `json:"metadata,omitempty"`
}

type LymphaticSweepRequest struct {
	RequestID      string         `json:"request_id"`
	WorkspaceID    string         `json:"workspace_id"`
	CaseID         string         `json:"case_id,omitempty"`
	SweepKinds     []SweepKind    `json:"sweep_kinds"`
	PolicyID       string         `json:"policy_id,omitempty"`
	DryRun         bool           `json:"dry_run"`
	IncludeDetails bool           `json:"include_details"`
	RequestedBy    string         `json:"requested_by"`
	CreatedAt      time.Time      `json:"created_at"`
	Metadata       map[string]any `json:"metadata,omitempty"`
}

type SweepSources struct {
	Snapshots       []snapshots.Snapshot
	KVManifests     []kv.KVCacheManifest
	RuntimeResults  []forgekRuntime.RuntimeGenerateResult
	Contradictions  []court.Contradiction
	ContextBundles  []contextcompiler.ContextBundle
	ContextBlocks   []contextcompiler.ContextBlock
	KnownObjectRefs []string
	KnownRefKinds   map[string]string
	Now             time.Time
}

type LymphaticFinding struct {
	FindingID         string          `json:"finding_id"`
	FindingType       FindingType     `json:"finding_type"`
	Severity          FindingSeverity `json:"severity"`
	WorkspaceID       string          `json:"workspace_id"`
	CaseID            string          `json:"case_id,omitempty"`
	ObjectRefs        []string        `json:"object_refs,omitempty"`
	SourceRefs        []string        `json:"source_refs,omitempty"`
	Reason            string          `json:"reason"`
	RecommendedAction ProposalType    `json:"recommended_action"`
	CreatedAt         time.Time       `json:"created_at"`
	Metadata          map[string]any  `json:"metadata,omitempty"`
}

type CleanupProposal struct {
	ProposalID          string         `json:"proposal_id"`
	ProposalType        ProposalType   `json:"proposal_type"`
	WorkspaceID         string         `json:"workspace_id"`
	CaseID              string         `json:"case_id,omitempty"`
	TargetObjectRefs    []string       `json:"target_object_refs,omitempty"`
	ProposedSyscallName string         `json:"proposed_syscall_name,omitempty"`
	ProposedPayload     map[string]any `json:"proposed_payload,omitempty"`
	Reason              string         `json:"reason"`
	SafetyNotes         []string       `json:"safety_notes,omitempty"`
	RequiresReview      bool           `json:"requires_review"`
	CreatedAt           time.Time      `json:"created_at"`
	Metadata            map[string]any `json:"metadata,omitempty"`
}

type MaintenanceReport struct {
	ReportID         string             `json:"report_id"`
	WorkspaceID      string             `json:"workspace_id"`
	CaseID           string             `json:"case_id,omitempty"`
	SweepKinds       []SweepKind        `json:"sweep_kinds"`
	PolicyID         string             `json:"policy_id"`
	DryRun           bool               `json:"dry_run"`
	Status           ReportStatus       `json:"status"`
	Findings         []LymphaticFinding `json:"findings,omitempty"`
	CleanupProposals []CleanupProposal  `json:"cleanup_proposals,omitempty"`
	Warnings         []string           `json:"warnings,omitempty"`
	Errors           []string           `json:"errors,omitempty"`
	Summary          string             `json:"summary"`
	CreatedBy        string             `json:"created_by"`
	CreatedAt        time.Time          `json:"created_at"`
	JournalRefs      []string           `json:"journal_refs,omitempty"`
	Metadata         map[string]any     `json:"metadata,omitempty"`
}

type ReportListFilter struct {
	WorkspaceID string
	CaseID      string
	Status      ReportStatus
	SweepKind   SweepKind
}

type ProposalListFilter struct {
	WorkspaceID  string
	CaseID       string
	ProposalType ProposalType
}

func DefaultPolicy(workspaceID string, createdAt time.Time) LymphaticPolicy {
	return LymphaticPolicy{
		PolicyID:                      "lymphatic-policy-default",
		WorkspaceID:                   strings.TrimSpace(workspaceID),
		SweepKindsEnabled:             AllSweepKinds(),
		MaxReportItems:                100,
		StaleAfterDurationMS:          int64((7 * 24 * time.Hour) / time.Millisecond),
		ExpireAfterDurationMS:         int64((30 * 24 * time.Hour) / time.Millisecond),
		KvColdAfterReuseCount:         0,
		KvExpireAfterDurationMS:       int64((30 * 24 * time.Hour) / time.Millisecond),
		SnapshotExpireAfterDurationMS: int64((30 * 24 * time.Hour) / time.Millisecond),
		RuntimeResultStaleAfterMS:     int64((7 * 24 * time.Hour) / time.Millisecond),
		IncludeSupersededObjects:      true,
		IncludeInvalidatedKV:          true,
		DryRun:                        true,
		CreatedAt:                     createdAt,
	}
}

func NormalizePolicy(policy LymphaticPolicy) LymphaticPolicy {
	policy.PolicyID = strings.TrimSpace(policy.PolicyID)
	policy.WorkspaceID = strings.TrimSpace(policy.WorkspaceID)
	policy.SweepKindsEnabled = NormalizeSweepKinds(policy.SweepKindsEnabled)
	if len(policy.SweepKindsEnabled) == 0 {
		policy.SweepKindsEnabled = AllSweepKinds()
	}
	if policy.MaxReportItems == 0 {
		policy.MaxReportItems = 100
	}
	policy.DryRun = true
	policy.Metadata = CloneMap(policy.Metadata)
	return policy
}

func ValidatePolicy(policy LymphaticPolicy) error {
	policy = NormalizePolicy(policy)
	if policy.PolicyID == "" || policy.WorkspaceID == "" || !policy.DryRun || policy.MaxReportItems < 0 ||
		policy.StaleAfterDurationMS < 0 || policy.ExpireAfterDurationMS < 0 ||
		policy.KvExpireAfterDurationMS < 0 || policy.SnapshotExpireAfterDurationMS < 0 ||
		policy.RuntimeResultStaleAfterMS < 0 {
		return ErrInvalidPolicy
	}
	for _, kind := range policy.SweepKindsEnabled {
		if !ValidSweepKind(kind) {
			return ErrInvalidSweepKind
		}
	}
	if containsSecretMetadata(policy.Metadata) {
		return ErrInvalidPolicy
	}
	return nil
}

func NormalizeSweepRequest(request LymphaticSweepRequest) LymphaticSweepRequest {
	request.RequestID = strings.TrimSpace(request.RequestID)
	request.WorkspaceID = strings.TrimSpace(request.WorkspaceID)
	request.CaseID = strings.TrimSpace(request.CaseID)
	request.SweepKinds = NormalizeSweepKinds(request.SweepKinds)
	if len(request.SweepKinds) == 0 {
		request.SweepKinds = []SweepKind{SweepSnapshotHygiene, SweepKVHygiene, SweepRuntimeResult, SweepContradiction, SweepOrphanRef}
	}
	request.PolicyID = strings.TrimSpace(request.PolicyID)
	request.DryRun = true
	request.RequestedBy = strings.TrimSpace(request.RequestedBy)
	request.Metadata = CloneMap(request.Metadata)
	return request
}

func ValidateSweepRequest(request LymphaticSweepRequest, policy LymphaticPolicy) error {
	request = NormalizeSweepRequest(request)
	policy = NormalizePolicy(policy)
	if request.RequestID == "" || request.WorkspaceID == "" || request.RequestedBy == "" || !request.DryRun {
		return ErrInvalidSweepRequest
	}
	if policy.WorkspaceID != request.WorkspaceID {
		return ErrInvalidSweepRequest
	}
	if err := ValidatePolicy(policy); err != nil {
		return err
	}
	enabled := make(map[SweepKind]bool, len(policy.SweepKindsEnabled))
	for _, kind := range policy.SweepKindsEnabled {
		enabled[kind] = true
	}
	for _, kind := range request.SweepKinds {
		if !ValidSweepKind(kind) || !enabled[kind] {
			return ErrInvalidSweepKind
		}
	}
	if containsSecretMetadata(request.Metadata) {
		return ErrInvalidSweepRequest
	}
	return nil
}

func NormalizeSweepKinds(values []SweepKind) []SweepKind {
	seen := make(map[SweepKind]struct{}, len(values))
	out := make([]SweepKind, 0, len(values))
	for _, value := range values {
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	sort.Slice(out, func(i, j int) bool { return sweepOrder(out[i]) < sweepOrder(out[j]) })
	return out
}

func (f LymphaticFinding) Clone() LymphaticFinding {
	f.ObjectRefs = CloneStrings(f.ObjectRefs)
	f.SourceRefs = CloneStrings(f.SourceRefs)
	f.Metadata = CloneMap(f.Metadata)
	return f
}

func (p CleanupProposal) Clone() CleanupProposal {
	p.TargetObjectRefs = CloneStrings(p.TargetObjectRefs)
	p.ProposedPayload = CloneMap(p.ProposedPayload)
	p.SafetyNotes = CloneStrings(p.SafetyNotes)
	p.Metadata = CloneMap(p.Metadata)
	return p
}

func (r MaintenanceReport) Clone() MaintenanceReport {
	r.SweepKinds = NormalizeSweepKinds(r.SweepKinds)
	r.Findings = SortFindings(r.Findings)
	r.CleanupProposals = SortProposals(r.CleanupProposals)
	r.Warnings = NormalizeRefs(r.Warnings)
	r.Errors = NormalizeRefs(r.Errors)
	r.JournalRefs = NormalizeRefs(r.JournalRefs)
	r.Metadata = CloneMap(r.Metadata)
	return r
}

func (r MaintenanceReport) IsCanonicalTruth() bool { return false }
func (p CleanupProposal) ExecutesCleanup() bool    { return false }

func SortFindings(findings []LymphaticFinding) []LymphaticFinding {
	out := make([]LymphaticFinding, 0, len(findings))
	for _, finding := range findings {
		out = append(out, finding.Clone())
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Severity != out[j].Severity {
			return severityOrder(out[i].Severity) > severityOrder(out[j].Severity)
		}
		if out[i].FindingType != out[j].FindingType {
			return out[i].FindingType < out[j].FindingType
		}
		return strings.Join(out[i].ObjectRefs, "\x00") < strings.Join(out[j].ObjectRefs, "\x00")
	})
	return out
}

func SortProposals(proposals []CleanupProposal) []CleanupProposal {
	out := make([]CleanupProposal, 0, len(proposals))
	for _, proposal := range proposals {
		out = append(out, proposal.Clone())
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].ProposalType != out[j].ProposalType {
			return out[i].ProposalType < out[j].ProposalType
		}
		return strings.Join(out[i].TargetObjectRefs, "\x00") < strings.Join(out[j].TargetObjectRefs, "\x00")
	})
	return out
}

func containsSecretMetadata(metadata map[string]any) bool {
	for key := range metadata {
		lower := strings.ToLower(key)
		if strings.Contains(lower, "secret") || strings.Contains(lower, "token") || strings.Contains(lower, "password") {
			return true
		}
	}
	return false
}

func sweepOrder(kind SweepKind) int {
	for i, candidate := range AllSweepKinds() {
		if kind == candidate {
			return i
		}
	}
	return 999
}

func severityOrder(severity FindingSeverity) int {
	switch severity {
	case SeverityCritical:
		return 5
	case SeverityHigh:
		return 4
	case SeverityMedium:
		return 3
	case SeverityLow:
		return 2
	default:
		return 1
	}
}
