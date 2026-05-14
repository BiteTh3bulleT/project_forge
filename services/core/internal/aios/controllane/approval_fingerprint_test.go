package controllane

import (
	"reflect"
	"testing"

	"forge/projectforge/services/core/internal/aios/domain"
)

func TestApprovalFingerprintDeterministicAndStableMapOrdering(t *testing.T) {
	def := mustFingerprintActionDef(t, domain.ActionCreateNote)
	reqA := fingerprintBaseRequest(domain.ActionCreateNote)
	reqA.Payload = map[string]any{
		"content": "first raw body must not appear",
		"id":      "note-1",
		"nested": map[string]any{
			"b": true,
			"a": "secret-like value",
		},
		"title": "Title A",
	}
	reqB := fingerprintBaseRequest(domain.ActionCreateNote)
	reqB.Payload = map[string]any{
		"title":   "Title A",
		"nested":  map[string]any{"a": "different raw value", "b": true},
		"id":      "note-1",
		"content": "second raw body must not appear",
	}

	fpA := BuildApprovalFingerprint(ApprovalFingerprintInput{
		Request:           reqA,
		Definition:        def,
		RiskClass:         "medium",
		ApprovalRequestID: "approval-1",
		DecisionStatus:    domain.ApprovalRequired,
		CreatedAtMillis:   1760000000000,
		ExpiresAtMillis:   1760003600000,
	})
	fpB := BuildApprovalFingerprint(ApprovalFingerprintInput{
		Request:           reqB,
		Definition:        def,
		RiskClass:         "medium",
		ApprovalRequestID: "approval-1",
		DecisionStatus:    domain.ApprovalRequired,
		CreatedAtMillis:   1760000000000,
		ExpiresAtMillis:   1760003600000,
	})

	if !reflect.DeepEqual(fpA, fpB) {
		t.Fatalf("fingerprint should be deterministic and map-order stable:\nA=%+v\nB=%+v", fpA, fpB)
	}
	if fpA.PayloadShapeHash == "" {
		t.Fatalf("expected payload shape hash")
	}
	if containsAny(fpA.PayloadShapeHash, []string{"first raw body", "second raw body", "secret-like value", "different raw value"}) {
		t.Fatalf("payload shape hash must not expose raw content: %q", fpA.PayloadShapeHash)
	}
	if got, want := fpA.SafeTargetIdentifiers, []string{"id=note-1"}; !equalStrings(got, want) {
		t.Fatalf("safe target identifiers = %v, want %v", got, want)
	}
}

func TestApprovalFingerprintChangesOnAuthorityFields(t *testing.T) {
	def := mustFingerprintActionDef(t, domain.ActionCreateNote)
	baseReq := fingerprintBaseRequest(domain.ActionCreateNote)
	baseReq.Payload = map[string]any{"id": "note-1", "title": "Title", "content": "Body"}
	base := BuildApprovalFingerprint(ApprovalFingerprintInput{Request: baseReq, Definition: def})

	cases := []struct {
		name string
		edit func(domain.SyscallRequest) domain.SyscallRequest
	}{
		{name: "action", edit: func(req domain.SyscallRequest) domain.SyscallRequest {
			req.Action = domain.ActionOpenLoop
			return req
		}},
		{name: "source", edit: func(req domain.SyscallRequest) domain.SyscallRequest {
			req.Source = domain.SourceAdapter
			return req
		}},
		{name: "actor", edit: func(req domain.SyscallRequest) domain.SyscallRequest {
			req.Actor.ID = "operator-2"
			return req
		}},
		{name: "workspace", edit: func(req domain.SyscallRequest) domain.SyscallRequest {
			req.Scope.WorkspaceID = "ws-other"
			return req
		}},
		{name: "payload_shape", edit: func(req domain.SyscallRequest) domain.SyscallRequest {
			req.Payload["confidence"] = 0.8
			return req
		}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := tc.edit(fingerprintBaseRequest(domain.ActionCreateNote))
			req.Payload = map[string]any{"id": "note-1", "title": "Title", "content": "Body"}
			if tc.name == "payload_shape" {
				req.Payload["confidence"] = 0.8
			}
			changed := BuildApprovalFingerprint(ApprovalFingerprintInput{Request: req, Definition: def})
			if reflect.DeepEqual(changed, base) {
				t.Fatalf("fingerprint did not change for %s", tc.name)
			}
		})
	}
}

func TestApprovalFingerprintRepresentsValidationOnlyAction(t *testing.T) {
	def := mustFingerprintActionDef(t, domain.ActionValidateSemanticOperation)
	req := fingerprintBaseRequest(domain.ActionValidateSemanticOperation)
	req.Payload = map[string]any{
		"operation": map[string]any{
			"action":   string(domain.ActionCreateNote),
			"mutating": true,
		},
	}

	fp := BuildApprovalFingerprint(ApprovalFingerprintInput{Request: req, Definition: def})
	if fp.Mutating {
		t.Fatalf("validation-only fingerprint must not be mutating: %+v", fp)
	}
	if fp.ActionClass != ApprovalFingerprintActionValidationOnly {
		t.Fatalf("action class = %q, want %q", fp.ActionClass, ApprovalFingerprintActionValidationOnly)
	}
	if fp.Capability != CapSemanticOperationValidate {
		t.Fatalf("capability = %q", fp.Capability)
	}
	if fp.TargetObjectType != "semantic_operation_validation" {
		t.Fatalf("target object type = %q", fp.TargetObjectType)
	}
}

func fingerprintBaseRequest(action domain.SemanticActionType) domain.SyscallRequest {
	req := validBaseRequest(action)
	req.Actor.ID = "operator-1"
	req.Source = domain.SourceUser
	req.Scope.WorkspaceID = "ws-main"
	req.CorrelationID = "corr-1"
	req.TraceID = "trace-1"
	return req
}

func mustFingerprintActionDef(t *testing.T, action domain.SemanticActionType) ActionDefinition {
	t.Helper()
	def, ok := NewStaticActionRegistry().Get(action)
	if !ok {
		t.Fatalf("missing action definition for %s", action)
	}
	return def
}

func containsAny(s string, needles []string) bool {
	for _, needle := range needles {
		if needle != "" && len(s) >= len(needle) {
			for i := 0; i+len(needle) <= len(s); i++ {
				if s[i:i+len(needle)] == needle {
					return true
				}
			}
		}
	}
	return false
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
