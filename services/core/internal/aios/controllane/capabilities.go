package controllane

import (
	"context"
	"fmt"
	"strings"

	"forge/projectforge/services/core/internal/aios/domain"
)

type CapabilityService interface {
	HasCapability(ctx context.Context, req domain.SyscallRequest, capability string) (bool, string, error)
}

type StaticCapabilityService struct {
	sourceCapabilities map[domain.ActionSource]map[string]struct{}
}

func NewStaticCapabilityService() *StaticCapabilityService {
	all := setOf(
		CapMemoryNoteCreate,
		CapMemoryNoteArchive,
		CapMemoryLinkCreate,
		CapStateUpdate,
		CapLoopOpen,
		CapLoopClose,
		CapMemoryContradictionReg,
		CapMemorySupersessionMark,
		CapModelDerive,
		CapContextCompile,
		CapKVIdentityValidate,
		CapRefShapeValidate,
		CapRefShapeCompare,
		CapSourceObjectValidate,
		CapSemanticOperationValidate,
		CapAdmissionCandidateValidate,
	)
	return &StaticCapabilityService{
		sourceCapabilities: map[domain.ActionSource]map[string]struct{}{
			domain.SourceUser:   all,
			domain.SourceSystem: all,
			domain.SourceTest:   all,
			domain.SourceInternal: setOf(
				CapMemoryNoteCreate,
				CapMemoryNoteArchive,
				CapMemoryLinkCreate,
				CapStateUpdate,
				CapLoopOpen,
				CapLoopClose,
				CapMemoryContradictionReg,
				CapMemorySupersessionMark,
				CapModelDerive,
				CapContextCompile,
				CapKVIdentityValidate,
				CapRefShapeValidate,
				CapRefShapeCompare,
				CapSourceObjectValidate,
				CapSemanticOperationValidate,
				CapAdmissionCandidateValidate,
			),
			domain.SourceAdapter: setOf(
				CapMemoryNoteCreate,
				CapMemoryLinkCreate,
				CapContextCompile,
				CapKVIdentityValidate,
				CapRefShapeValidate,
				CapRefShapeCompare,
				CapSourceObjectValidate,
				CapSemanticOperationValidate,
				CapAdmissionCandidateValidate,
			),
			domain.SourceFutureIRIS: setOf(
				CapMemoryNoteCreate,
				CapMemoryLinkCreate,
				CapMemoryContradictionReg,
				CapModelDerive,
				CapContextCompile,
			),
		},
	}
}

func (s *StaticCapabilityService) HasCapability(_ context.Context, req domain.SyscallRequest, capability string) (bool, string, error) {
	capability = strings.TrimSpace(capability)
	if capability == "" {
		return false, "required capability missing", nil
	}
	caps, ok := s.sourceCapabilities[req.Source]
	if !ok {
		return false, fmt.Sprintf("unknown action source %q", req.Source), nil
	}
	if _, ok := caps[capability]; !ok {
		return false, fmt.Sprintf("source %q lacks capability %q", req.Source, capability), nil
	}
	if req.RequiredCapability != "" && !strings.EqualFold(strings.TrimSpace(req.RequiredCapability), capability) {
		return false, fmt.Sprintf("requiredCapability %q does not match action capability %q", req.RequiredCapability, capability), nil
	}
	return true, "capability granted", nil
}

func setOf(items ...string) map[string]struct{} {
	out := make(map[string]struct{}, len(items))
	for _, item := range items {
		v := strings.TrimSpace(item)
		if v == "" {
			continue
		}
		out[v] = struct{}{}
	}
	return out
}
