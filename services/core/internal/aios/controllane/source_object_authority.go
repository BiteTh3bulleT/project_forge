package controllane

import (
	"strings"

	"forge/projectforge/services/core/internal/aios/domain"
	"forge/projectforge/services/core/internal/refvalidation"
)

const (
	SourceObjectDecisionAccepted  = "accepted"
	SourceObjectDecisionRejected  = "rejected"
	SourceObjectDecisionMalformed = "malformed"

	SourceObjectPolicyVersion    = "phase-14m-v1"
	SourceObjectValidatorVersion = "source-object-authority-v1"
)

type SourceObjectAuthorityDecision struct {
	Accepted               bool                              `json:"accepted"`
	Decision               string                            `json:"decision"`
	Reason                 string                            `json:"reason"`
	Source                 string                            `json:"source"`
	Failures               []refvalidation.ValidationFailure `json:"failures,omitempty"`
	Warnings               []string                          `json:"warnings,omitempty"`
	NormalizedRefs         []refvalidation.ObjectRef         `json:"normalizedRefs,omitempty"`
	ResolvedRefs           []SourceObjectResolvedRef         `json:"resolvedRefs,omitempty"`
	MemoryMutation         bool                              `json:"memoryMutation"`
	RuntimeMutation        bool                              `json:"runtimeMutation"`
	ModelRuntimeCall       bool                              `json:"modelRuntimeCall"`
	EvidenceAdmission      bool                              `json:"evidenceAdmission"`
	ContextCompilation     bool                              `json:"contextCompilation"`
	LiveAuthorityMigration bool                              `json:"liveAuthorityMigration"`
	ValidatorVersion       string                            `json:"validatorVersion"`
	PolicyVersion          string                            `json:"policyVersion"`
	ValidationResult       refvalidation.ValidationResult    `json:"validationResult"`
}

type SourceObjectResolvedRef struct {
	RefType     string `json:"ref_type"`
	RefID       string `json:"ref_id"`
	ObjectKind  string `json:"object_kind"`
	WorkspaceID string `json:"workspace_id"`
	LaneID      string `json:"lane_id,omitempty"`
}

func EnforceSourceObjectAuthority(req domain.SyscallRequest, store SemanticReadStore) SourceObjectAuthorityDecision {
	base := SourceObjectAuthorityDecision{
		Source:                 string(req.Source),
		MemoryMutation:         false,
		RuntimeMutation:        false,
		ModelRuntimeCall:       false,
		EvidenceAdmission:      false,
		ContextCompilation:     false,
		LiveAuthorityMigration: false,
		ValidatorVersion:       SourceObjectValidatorVersion,
		PolicyVersion:          SourceObjectPolicyVersion,
	}
	if issues := validateSourceObjectAuthority(req); len(issues) > 0 {
		base.Accepted = false
		base.Decision = SourceObjectDecisionMalformed
		base.Reason = issues[0].Message
		return base
	}
	result := refvalidation.ValidateRefs(refValidationRequestFromPayload(req))
	base.ValidationResult = result
	base.Failures = append([]refvalidation.ValidationFailure{}, result.Failures...)
	base.Warnings = append([]string{}, result.Warnings...)
	base.NormalizedRefs = append([]refvalidation.ObjectRef{}, result.NormalizedRefs...)
	if !result.Passed {
		base.Accepted = false
		base.Decision = SourceObjectDecisionRejected
		base.Reason = "source object authority rejected by ref shape validation"
		if len(result.Failures) > 0 {
			base.Reason = result.Failures[0].Message
		}
		return base
	}
	if store == nil {
		base.Accepted = false
		base.Decision = SourceObjectDecisionRejected
		base.Reason = "source object authority lookup store is unavailable"
		base.Failures = append(base.Failures, refvalidation.ValidationFailure{
			Gate:    "source_object_lookup",
			Field:   "store",
			Message: base.Reason,
		})
		return base
	}
	for _, ref := range result.NormalizedRefs {
		resolved, failure, ok := resolveSourceObjectRef(store, req.Scope, ref)
		if !ok {
			base.Failures = append(base.Failures, failure)
			continue
		}
		base.ResolvedRefs = append(base.ResolvedRefs, resolved)
	}
	if len(base.Failures) > 0 {
		base.Accepted = false
		base.Decision = SourceObjectDecisionRejected
		base.Reason = base.Failures[0].Message
		return base
	}
	base.Accepted = true
	base.Decision = SourceObjectDecisionAccepted
	base.Reason = "source object authority validation accepted"
	return base
}

func validateSourceObjectAuthority(req domain.SyscallRequest) []domain.SyscallError {
	return validateRefShape(req)
}

func resolveSourceObjectRef(store SemanticReadStore, reqScope domain.ForgeScope, ref refvalidation.ObjectRef) (SourceObjectResolvedRef, refvalidation.ValidationFailure, bool) {
	kind, scope, ok := resolveSourceObject(store, ref)
	if !ok {
		return SourceObjectResolvedRef{}, refvalidation.ValidationFailure{
			Gate:    "source_object_lookup",
			Field:   "refs.ref_id",
			Message: "source object not found or ref_type has no governed resolver",
		}, false
	}
	if !refTypeMatchesResolvedKind(ref.RefType, kind) {
		return SourceObjectResolvedRef{}, refvalidation.ValidationFailure{
			Gate:    "source_object_kind",
			Field:   "refs.ref_type",
			Message: "source object kind does not match ref_type",
		}, false
	}
	if !scopeCompatible(reqScope, scope) {
		return SourceObjectResolvedRef{}, refvalidation.ValidationFailure{
			Gate:    "source_object_scope",
			Field:   "refs.workspace_id",
			Message: "source object is outside request scope",
		}, false
	}
	if strings.TrimSpace(ref.WorkspaceID) != "" && strings.TrimSpace(ref.WorkspaceID) != strings.TrimSpace(scope.WorkspaceID) {
		return SourceObjectResolvedRef{}, refvalidation.ValidationFailure{
			Gate:    "source_object_scope",
			Field:   "refs.workspace_id",
			Message: "source object workspace does not match ref workspace",
		}, false
	}
	return SourceObjectResolvedRef{
		RefType:     ref.RefType,
		RefID:       ref.RefID,
		ObjectKind:  kind,
		WorkspaceID: scope.WorkspaceID,
		LaneID:      scope.LaneID,
	}, refvalidation.ValidationFailure{}, true
}

func resolveSourceObject(store SemanticReadStore, ref refvalidation.ObjectRef) (kind string, scope domain.ForgeScope, ok bool) {
	switch ref.RefType {
	case "memory_note":
		if note, found := store.FindNote(ref.RefID); found {
			return "memory_note", note.Scope, true
		}
	case "semantic_link":
		if link, found := store.FindLink(ref.RefID); found {
			return "semantic_link", link.Scope, true
		}
	case "state_item":
		if state, found := store.FindState(ref.RefID); found {
			return "state_item", state.Scope, true
		}
	case "open_loop":
		if loop, found := store.FindLoop(ref.RefID); found {
			return "open_loop", loop.Scope, true
		}
	case "semantic_object":
		return resolveKnownSemanticObject(store, ref.RefID)
	}
	return "", domain.ForgeScope{}, false
}

func resolveKnownSemanticObject(store SemanticReadStore, id string) (kind string, scope domain.ForgeScope, ok bool) {
	if note, found := store.FindNote(id); found {
		return "memory_note", note.Scope, true
	}
	if link, found := store.FindLink(id); found {
		return "semantic_link", link.Scope, true
	}
	if state, found := store.FindState(id); found {
		return "state_item", state.Scope, true
	}
	if loop, found := store.FindLoop(id); found {
		return "open_loop", loop.Scope, true
	}
	if model, found := store.FindModel(id); found {
		return "derived_model", model.Scope, true
	}
	return "", domain.ForgeScope{}, false
}

func refTypeMatchesResolvedKind(refType, kind string) bool {
	refType = strings.TrimSpace(refType)
	if refType == "semantic_object" {
		return true
	}
	return refType == kind
}

func (d SourceObjectAuthorityDecision) ToStateSummary() map[string]any {
	return map[string]any{
		"sourceObjectAuthorityValidation": d.ToAuditFields(),
		"sourceObjectAuthorityResult":     d.ValidationResult,
		"passed":                          d.Accepted,
		"normalizedRefs":                  refsForSummary(d.NormalizedRefs),
		"resolvedRefs":                    sourceObjectResolvedRefsForSummary(d.ResolvedRefs),
		"failures":                        append([]refvalidation.ValidationFailure{}, d.Failures...),
		"warnings":                        append([]string{}, d.Warnings...),
		"memoryMutation":                  d.MemoryMutation,
		"runtimeMutation":                 d.RuntimeMutation,
		"modelRuntimeCall":                d.ModelRuntimeCall,
		"evidenceAdmission":               d.EvidenceAdmission,
		"contextCompilation":              d.ContextCompilation,
		"liveAuthorityMigration":          d.LiveAuthorityMigration,
		"forgeKActivation":                forgeKActivationSummary(string(domain.ActionValidateSourceObject)),
		"forgeKNoEffect":                  forgeKNoEffectSummary(),
	}
}

func (d SourceObjectAuthorityDecision) ToAuditFields() map[string]any {
	return map[string]any{
		"accepted":               d.Accepted,
		"decision":               d.Decision,
		"reason":                 d.Reason,
		"source":                 d.Source,
		"normalizedRefCount":     len(d.NormalizedRefs),
		"resolvedRefCount":       len(d.ResolvedRefs),
		"failures":               append([]refvalidation.ValidationFailure{}, d.Failures...),
		"warnings":               append([]string{}, d.Warnings...),
		"failureCount":           len(d.Failures),
		"warningCount":           len(d.Warnings),
		"memoryMutation":         d.MemoryMutation,
		"runtimeMutation":        d.RuntimeMutation,
		"modelRuntimeCall":       d.ModelRuntimeCall,
		"evidenceAdmission":      d.EvidenceAdmission,
		"contextCompilation":     d.ContextCompilation,
		"liveAuthorityMigration": d.LiveAuthorityMigration,
		"validatorVersion":       d.ValidatorVersion,
		"policyVersion":          d.PolicyVersion,
		"forgeKActivation":       forgeKActivationSummary(string(domain.ActionValidateSourceObject)),
		"forgeKNoEffect":         forgeKNoEffectSummary(),
	}
}

func (d SourceObjectAuthorityDecision) ToSyscallError() domain.SyscallError {
	code := domain.ErrInvalidPayload
	field := "payload.refs"
	if len(d.Failures) > 0 {
		switch d.Failures[0].Gate {
		case "source_object_lookup":
			code = domain.ErrNotFound
		case "source_object_scope":
			code = domain.ErrInvalidScope
		}
		field = "payload." + strings.TrimPrefix(d.Failures[0].Field, "refs")
		field = strings.ReplaceAll(field, "..", ".")
	}
	return domain.SyscallError{Code: code, Field: field, Message: d.Reason}
}

func sourceObjectResolvedRefsForSummary(refs []SourceObjectResolvedRef) []map[string]string {
	out := make([]map[string]string, 0, len(refs))
	for _, ref := range refs {
		item := map[string]string{
			"ref_type":     ref.RefType,
			"ref_id":       ref.RefID,
			"object_kind":  ref.ObjectKind,
			"workspace_id": ref.WorkspaceID,
		}
		if ref.LaneID != "" {
			item["lane_id"] = ref.LaneID
		}
		out = append(out, item)
	}
	return out
}
