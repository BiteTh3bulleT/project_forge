package refvalidation

import (
	"sort"
	"strings"
)

const (
	GateWorkspace = "workspace_present"
	GateRefs      = "refs_present"
	GateRefType   = "ref_type_allowed"
	GateRefID     = "ref_id_present"
	GateSafeRefID = "ref_id_safe"
	GateScope     = "ref_workspace_scope"
)

type ObjectRef struct {
	RefType     string `json:"ref_type"`
	RefID       string `json:"ref_id"`
	WorkspaceID string `json:"workspace_id,omitempty"`
	SourceRef   string `json:"source_ref,omitempty"`
}

type ValidationRequest struct {
	ResultID    string      `json:"result_id"`
	WorkspaceID string      `json:"workspace_id"`
	Refs        []ObjectRef `json:"refs"`
}

type ValidationFailure struct {
	Gate    string `json:"gate"`
	Field   string `json:"field"`
	Message string `json:"message"`
}

type ValidationResult struct {
	ResultID       string              `json:"result_id"`
	WorkspaceID    string              `json:"workspace_id"`
	Passed         bool                `json:"passed"`
	NormalizedRefs []ObjectRef         `json:"normalized_refs"`
	Failures       []ValidationFailure `json:"failures,omitempty"`
	Warnings       []string            `json:"warnings,omitempty"`
}

func ValidateRefs(req ValidationRequest) ValidationResult {
	resultID := strings.TrimSpace(req.ResultID)
	workspaceID := strings.TrimSpace(req.WorkspaceID)
	failures := make([]ValidationFailure, 0)
	if workspaceID == "" {
		failures = append(failures, ValidationFailure{Gate: GateWorkspace, Field: "workspace_id", Message: "workspace_id is required"})
	}
	if len(req.Refs) == 0 {
		failures = append(failures, ValidationFailure{Gate: GateRefs, Field: "refs", Message: "at least one ref is required"})
	}

	normalized := make([]ObjectRef, 0, len(req.Refs))
	seen := map[string]struct{}{}
	for _, raw := range req.Refs {
		ref := normalizeRef(raw, workspaceID)
		refValid := true
		if ref.RefType == "" || !allowedRefType(ref.RefType) {
			failures = append(failures, ValidationFailure{Gate: GateRefType, Field: "refs.ref_type", Message: "ref_type is not allowed"})
			refValid = false
		}
		if ref.RefID == "" {
			failures = append(failures, ValidationFailure{Gate: GateRefID, Field: "refs.ref_id", Message: "ref_id is required"})
			refValid = false
		} else if !safeRefID(ref.RefID) {
			failures = append(failures, ValidationFailure{Gate: GateSafeRefID, Field: "refs.ref_id", Message: "ref_id contains unsafe content"})
			refValid = false
		}
		if workspaceID != "" && ref.WorkspaceID != "" && ref.WorkspaceID != workspaceID {
			failures = append(failures, ValidationFailure{Gate: GateScope, Field: "refs.workspace_id", Message: "ref workspace does not match request workspace"})
			refValid = false
		}
		if !refValid {
			continue
		}
		key := ref.WorkspaceID + "\x00" + ref.RefType + "\x00" + ref.RefID
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		normalized = append(normalized, ref)
	}
	sort.SliceStable(normalized, func(i, j int) bool {
		if normalized[i].WorkspaceID != normalized[j].WorkspaceID {
			return normalized[i].WorkspaceID < normalized[j].WorkspaceID
		}
		if normalized[i].RefType != normalized[j].RefType {
			return normalized[i].RefType < normalized[j].RefType
		}
		return normalized[i].RefID < normalized[j].RefID
	})
	return ValidationResult{
		ResultID:       resultID,
		WorkspaceID:    workspaceID,
		Passed:         len(failures) == 0,
		NormalizedRefs: normalized,
		Failures:       failures,
		Warnings:       nil,
	}
}

func normalizeRef(raw ObjectRef, workspaceID string) ObjectRef {
	ref := ObjectRef{
		RefType:     strings.ToLower(strings.TrimSpace(raw.RefType)),
		RefID:       strings.TrimSpace(raw.RefID),
		WorkspaceID: strings.TrimSpace(raw.WorkspaceID),
		SourceRef:   strings.TrimSpace(raw.SourceRef),
	}
	if ref.WorkspaceID == "" {
		ref.WorkspaceID = workspaceID
	}
	return ref
}

func allowedRefType(refType string) bool {
	switch strings.TrimSpace(refType) {
	case "memory_note",
		"semantic_link",
		"state_item",
		"open_loop",
		"context_block",
		"context_bundle",
		"snapshot",
		"restore_seed",
		"kv_manifest",
		"runtime_manifest",
		"case_packet",
		"exhibit",
		"palace_route",
		"semantic_object",
		"semantic_operation",
		"diagnostic_report":
		return true
	default:
		return false
	}
}

func safeRefID(refID string) bool {
	if refID == "" || len(refID) > 256 {
		return false
	}
	lower := strings.ToLower(refID)
	for _, term := range []string{"secret", "token", "password", "apikey", "api_key", "authorization", "cookie", "bearer"} {
		if strings.Contains(lower, term) {
			return false
		}
	}
	for _, r := range refID {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' {
			continue
		}
		switch r {
		case '-', '_', ':', '.', '/':
			continue
		default:
			return false
		}
	}
	return true
}
