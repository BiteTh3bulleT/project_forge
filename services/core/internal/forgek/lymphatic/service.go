package lymphatic

import (
	"sort"
	"sync"
	"time"
)

type Service struct {
	mu        sync.RWMutex
	reports   map[string]MaintenanceReport
	proposals map[string]CleanupProposal
}

func NewService() *Service {
	return &Service{
		reports:   make(map[string]MaintenanceReport),
		proposals: make(map[string]CleanupProposal),
	}
}

func (s *Service) RunSweep(request LymphaticSweepRequest, policy LymphaticPolicy, sources SweepSources) (MaintenanceReport, error) {
	request = NormalizeSweepRequest(request)
	if policy.PolicyID == "" {
		policy = DefaultPolicy(request.WorkspaceID, request.CreatedAt)
	}
	policy = NormalizePolicy(policy)
	if request.PolicyID == "" {
		request.PolicyID = policy.PolicyID
	}
	if sources.Now.IsZero() {
		sources.Now = firstNonZeroTime(request.CreatedAt, policy.CreatedAt, time.Now().UTC())
	}
	if request.CreatedAt.IsZero() {
		request.CreatedAt = sources.Now
	}
	if policy.CreatedAt.IsZero() {
		policy.CreatedAt = request.CreatedAt
	}
	if err := ValidateSweepRequest(request, policy); err != nil {
		return MaintenanceReport{}, err
	}
	report := BuildReport(request, policy, sources)
	s.StoreReport(report)
	return report.Clone(), nil
}

func (s *Service) StoreReport(report MaintenanceReport) {
	s.mu.Lock()
	defer s.mu.Unlock()
	report = report.Clone()
	s.reports[report.ReportID] = report
	for _, proposal := range report.CleanupProposals {
		s.proposals[proposal.ProposalID] = proposal.Clone()
	}
}

func (s *Service) StoreProposal(proposal CleanupProposal) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.proposals[proposal.ProposalID] = proposal.Clone()
}

func (s *Service) CreateProposal(proposal CleanupProposal) (CleanupProposal, error) {
	proposal = NormalizeProposal(proposal)
	if err := ValidateProposal(proposal); err != nil {
		return CleanupProposal{}, err
	}
	s.StoreProposal(proposal)
	return proposal.Clone(), nil
}

func (s *Service) GetReport(reportID string) (MaintenanceReport, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	report, ok := s.reports[reportID]
	if !ok {
		return MaintenanceReport{}, false
	}
	return report.Clone(), true
}

func (s *Service) ListReports(filter ReportListFilter) []MaintenanceReport {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]MaintenanceReport, 0)
	for _, report := range s.reports {
		if filter.WorkspaceID != "" && report.WorkspaceID != filter.WorkspaceID {
			continue
		}
		if filter.CaseID != "" && report.CaseID != filter.CaseID {
			continue
		}
		if filter.Status != "" && report.Status != filter.Status {
			continue
		}
		if filter.SweepKind != "" && !reportHasSweepKind(report, filter.SweepKind) {
			continue
		}
		out = append(out, report.Clone())
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ReportID < out[j].ReportID })
	return out
}

func (s *Service) GetProposal(proposalID string) (CleanupProposal, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	proposal, ok := s.proposals[proposalID]
	if !ok {
		return CleanupProposal{}, false
	}
	return proposal.Clone(), true
}

func (s *Service) ListProposals(filter ProposalListFilter) []CleanupProposal {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]CleanupProposal, 0)
	for _, proposal := range s.proposals {
		if filter.WorkspaceID != "" && proposal.WorkspaceID != filter.WorkspaceID {
			continue
		}
		if filter.CaseID != "" && proposal.CaseID != filter.CaseID {
			continue
		}
		if filter.ProposalType != "" && proposal.ProposalType != filter.ProposalType {
			continue
		}
		out = append(out, proposal.Clone())
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ProposalID < out[j].ProposalID })
	return out
}

func NormalizeProposal(proposal CleanupProposal) CleanupProposal {
	proposal.ProposalID = trim(proposal.ProposalID)
	proposal.WorkspaceID = trim(proposal.WorkspaceID)
	proposal.CaseID = trim(proposal.CaseID)
	proposal.TargetObjectRefs = NormalizeRefs(proposal.TargetObjectRefs)
	proposal.ProposedSyscallName = trim(proposal.ProposedSyscallName)
	proposal.ProposedPayload = CloneMap(proposal.ProposedPayload)
	proposal.Reason = trim(proposal.Reason)
	proposal.SafetyNotes = NormalizeRefs(proposal.SafetyNotes)
	proposal.RequiresReview = true
	proposal.Metadata = CloneMap(proposal.Metadata)
	return proposal
}

func ValidateProposal(proposal CleanupProposal) error {
	proposal = NormalizeProposal(proposal)
	if proposal.ProposalID == "" || proposal.ProposalType == "" || proposal.WorkspaceID == "" ||
		len(proposal.TargetObjectRefs) == 0 || proposal.Reason == "" || !proposal.RequiresReview {
		return ErrInvalidProposal
	}
	if containsSecretMetadata(proposal.Metadata) {
		return ErrInvalidProposal
	}
	return nil
}

func reportHasSweepKind(report MaintenanceReport, kind SweepKind) bool {
	for _, candidate := range report.SweepKinds {
		if candidate == kind {
			return true
		}
	}
	return false
}

func firstNonZeroTime(values ...time.Time) time.Time {
	for _, value := range values {
		if !value.IsZero() {
			return value
		}
	}
	return time.Time{}
}

func trim(value string) string {
	return stringsTrimSpace(value)
}
