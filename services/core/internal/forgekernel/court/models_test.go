package court

import (
	"strings"
	"testing"

	"forge/projectforge/services/core/internal/aios/domain"
)

func TestDecideIsDeterministicAndModelCannotRule(t *testing.T) {
	req := domain.SyscallRequest{
		ID: "sys-1", Action: domain.ActionAdmitEvidence,
		Actor: domain.ActorIdentity{ID: "operator", Kind: "user"},
		Payload: map[string]any{
			"caseId": "case-1", "exhibitId": "ex-1",
			"sourceRefs":  []any{"artifact:a", "artifact:a", "journal:j"},
			"contentHash": "sha256:" + strings.Repeat("a", 64), "policyRefs": []string{"policy:1"},
		},
	}
	first, issues := Decide(req)
	if len(issues) != 0 || first.Decision != DecisionAdmitted {
		t.Fatalf("unexpected ruling: %#v %#v", first, issues)
	}
	second, _ := Decide(req)
	if first.Reason != second.Reason || len(first.InputRefs) != 2 {
		t.Fatalf("decision was not deterministic/deduplicated: %#v %#v", first, second)
	}
	req.Actor.Kind = "llm_model"
	if _, issues = Decide(req); len(issues) != 1 || issues[0].Code != domain.ErrUnauthorized {
		t.Fatalf("model actor must fail closed: %#v", issues)
	}
}

func TestDecideRejectsMalformedContentHashAndNonStringRefs(t *testing.T) {
	req := domain.SyscallRequest{
		ID: "sys-invalid-hash", Action: domain.ActionAdmitEvidence,
		Actor: domain.ActorIdentity{ID: "operator", Kind: "user"},
		Payload: map[string]any{
			"caseId": "case-1", "sourceRefs": []any{map[string]any{"id": "not-a-string"}},
			"contentHash": "sha256:not-a-digest", "policyRefs": []string{"policy:1"},
		},
	}
	d, issues := Decide(req)
	if len(issues) != 0 || d.Decision != DecisionRejected || len(d.InputRefs) != 0 || !strings.Contains(d.Reason, "valid sha256 content hash") {
		t.Fatalf("malformed evidence identity was not rejected deterministically: %#v %#v", d, issues)
	}
}

func TestDecidePersistsPolicyRejectionShape(t *testing.T) {
	req := domain.SyscallRequest{
		ID: "sys-reject", Action: domain.ActionAdmitEvidence,
		Actor:   domain.ActorIdentity{ID: "operator", Kind: "user"},
		Payload: map[string]any{"caseId": "case-1", "exhibitId": "ex-1"},
	}
	d, issues := Decide(req)
	if len(issues) != 0 || d.Decision != DecisionRejected || d.ReasonCode != "policy_material_incomplete" {
		t.Fatalf("expected deterministic rejection ruling: %#v %#v", d, issues)
	}
}
