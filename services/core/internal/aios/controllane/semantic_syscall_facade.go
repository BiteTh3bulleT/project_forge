package controllane

import (
	"sort"
	"strings"

	"forge/projectforge/services/core/internal/aios/domain"
)

const SemanticSyscallFacadeSchemaVersion = "semantic-syscall-facade-v1"

type SemanticSyscallFacade struct {
	SchemaVersion      string                    `json:"schemaVersion"`
	SyscallID          string                    `json:"syscallId"`
	Action             domain.SemanticActionType `json:"action"`
	ExpectedEffect     string                    `json:"expectedEffect"`
	TargetObjectType   string                    `json:"targetObjectType"`
	RequiredCapability string                    `json:"requiredCapability"`
	Mutating           bool                      `json:"mutating"`
	DryRun             bool                      `json:"dryRun"`
	WorkspaceID        string                    `json:"workspaceId"`
	LaneID             string                    `json:"laneId,omitempty"`
	Refs               []string                  `json:"refs,omitempty"`
	CapabilityScope    map[string]any            `json:"capabilityScope"`
	RollbackMetadata   map[string]any            `json:"rollbackMetadata"`
	AuthorityEffects   map[string]bool           `json:"authorityEffects"`
}

func BuildSemanticSyscallFacade(req domain.SyscallRequest, def ActionDefinition) SemanticSyscallFacade {
	return SemanticSyscallFacade{
		SchemaVersion:      SemanticSyscallFacadeSchemaVersion,
		SyscallID:          strings.TrimSpace(req.ID),
		Action:             def.Action,
		ExpectedEffect:     expectedEffect(def),
		TargetObjectType:   def.TargetObjectType,
		RequiredCapability: nonEmpty(req.RequiredCapability, def.Capability),
		Mutating:           def.Mutating,
		DryRun:             req.DryRun,
		WorkspaceID:        strings.TrimSpace(req.Scope.WorkspaceID),
		LaneID:             strings.TrimSpace(req.Scope.LaneID),
		Refs:               syscallRefs(req),
		CapabilityScope:    capabilityScope(req),
		RollbackMetadata:   rollbackMetadata(req, def),
		AuthorityEffects:   authorityEffects(def),
	}
}

func (f SemanticSyscallFacade) ToAuditFields() map[string]any {
	return map[string]any{
		"schemaVersion":      f.SchemaVersion,
		"syscallId":          f.SyscallID,
		"action":             string(f.Action),
		"expectedEffect":     f.ExpectedEffect,
		"targetObjectType":   f.TargetObjectType,
		"requiredCapability": f.RequiredCapability,
		"mutating":           f.Mutating,
		"dryRun":             f.DryRun,
		"workspaceId":        f.WorkspaceID,
		"laneId":             f.LaneID,
		"refs":               append([]string{}, f.Refs...),
		"capabilityScope":    cloneMap(f.CapabilityScope),
		"rollbackMetadata":   cloneMap(f.RollbackMetadata),
		"authorityEffects":   boolMap(f.AuthorityEffects),
	}
}

func expectedEffect(def ActionDefinition) string {
	target := strings.TrimSpace(def.TargetObjectType)
	if target == "" {
		target = "semantic_object"
	}
	if def.Mutating {
		return "commit_" + target
	}
	return "validate_" + target
}

func rollbackMetadata(req domain.SyscallRequest, def ActionDefinition) map[string]any {
	if !def.Mutating || req.DryRun {
		return map[string]any{
			"required": false,
			"strategy": "no_state_rollback_required",
		}
	}
	return map[string]any{
		"required":       true,
		"strategy":       "revert_journaled_commit",
		"auditEventName": def.AuditEventName,
		"idempotencyKey": strings.TrimSpace(req.IdempotencyKey),
	}
}

func capabilityScope(req domain.SyscallRequest) map[string]any {
	return map[string]any{
		"workspaceId":   strings.TrimSpace(req.Scope.WorkspaceID),
		"laneId":        strings.TrimSpace(req.Scope.LaneID),
		"selectedPaths": append([]string{}, req.Scope.SelectedPaths...),
	}
}

func authorityEffects(def ActionDefinition) map[string]bool {
	return map[string]bool{
		"controlLaneOwned":     true,
		"mutatesCanonicalData": def.Mutating,
		"callsModelRuntime":    false,
		"executesGatewayTool":  false,
		"importsForgeK":        false,
	}
}

func syscallRefs(req domain.SyscallRequest) []string {
	refs := make([]string, 0)
	add := func(values ...string) {
		for _, value := range values {
			value = strings.TrimSpace(value)
			if value != "" && safeFacadeRef(value) {
				refs = append(refs, value)
			}
		}
	}
	add(readString(req.Payload, "id"))
	add(readString(req.Payload, "sourceId"))
	add(readString(req.Payload, "targetId"))
	add(readString(req.Payload, "oldObjectId"))
	add(readString(req.Payload, "newObjectId"))
	add(readString(req.Payload, "noteId"))
	add(readString(req.Payload, "loopId"))
	add(readString(req.Payload, "modelId"))
	add(readStringSlice(req.Payload, "derivedFrom")...)
	add(readStringSlice(req.Payload, "relatedNotes")...)
	add(readStringSlice(req.Payload, "source_refs")...)
	add(readStringSlice(req.Payload, "derived_refs")...)
	add(readStringSlice(req.Payload, "provenance_refs")...)
	add(refIDsFromPayload(req.Payload["refs"])...)
	add(refIDsFromPayload(req.Payload["source_refs"])...)
	add(refIDsFromPayload(req.Payload["derived_refs"])...)
	add(refIDsFromPayload(req.Payload["provenance_refs"])...)

	seen := map[string]struct{}{}
	out := make([]string, 0, len(refs))
	for _, ref := range refs {
		if _, ok := seen[ref]; ok {
			continue
		}
		seen[ref] = struct{}{}
		out = append(out, ref)
	}
	sort.Strings(out)
	return out
}

func refIDsFromPayload(raw any) []string {
	items, ok := raw.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		out = append(out, readString(m, "ref_id"))
	}
	return out
}

func safeFacadeRef(ref string) bool {
	if len(ref) > 256 {
		return false
	}
	lower := strings.ToLower(ref)
	for _, term := range []string{"secret", "token", "password", "apikey", "api_key", "authorization", "cookie", "bearer"} {
		if strings.Contains(lower, term) {
			return false
		}
	}
	return true
}

func boolMap(in map[string]bool) map[string]any {
	if in == nil {
		return nil
	}
	out := make(map[string]any, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}
