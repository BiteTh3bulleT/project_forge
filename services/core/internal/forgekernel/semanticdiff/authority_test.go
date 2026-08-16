package semanticdiff

import (
	"reflect"
	"testing"

	"forge/projectforge/services/core/internal/aios/domain"
)

func TestDecideBindsGovernedSourcesAndDeterministicOutput(t *testing.T) {
	req, input := validAuthorityFixture()
	decision, issues := Decide(req, input)
	if len(issues) != 0 {
		t.Fatalf("unexpected issues: %+v", issues)
	}
	if decision.Content != "beta" || decision.ObjectClass != DerivedObjectClass || decision.SourceManifestHash == "" {
		t.Fatalf("unexpected decision: %+v", decision)
	}
	if err := VerifyDecision(req, decision); err != nil {
		t.Fatal(err)
	}
	decoded, ok := DecisionFromMetadata(map[string]any{MetadataDecisionKey: map[string]any{
		"operatorVersion": decision.OperatorVersion,
		"left": map[string]any{
			"rowId": decision.Left.RowID, "evidenceId": decision.Left.EvidenceID, "scope": decision.Left.Scope,
			"content": decision.Left.Content, "materialHash": decision.Left.MaterialHash, "evidenceHash": decision.Left.EvidenceHash,
			"courtCaseId": decision.Left.CourtCaseID, "courtExhibitId": decision.Left.CourtExhibitID,
			"courtRulingId": decision.Left.CourtRulingID, "admissionSyscallId": decision.Left.AdmissionSyscallID,
			"sourceProvenanceId":          decision.Left.SourceProvenanceID,
			"materializationProvenanceId": decision.Left.MaterializationProvenanceID,
			"createdAt":                   decision.Left.CreatedAt, "committedBy": decision.Left.CommittedBy,
			"current": true, "admitted": true,
		},
		"right": decision.Right, "tokens": decision.Tokens, "content": decision.Content,
		"contentHash": decision.ContentHash, "sourceManifestHash": decision.SourceManifestHash,
		"objectClass": decision.ObjectClass,
	}})
	if !ok || !reflect.DeepEqual(decoded, decision) {
		t.Fatalf("metadata decode mismatch: ok=%t got=%+v want=%+v", ok, decoded, decision)
	}
}

func TestDecideFailsClosedOnAuthorityAndScopeTamper(t *testing.T) {
	tests := []struct {
		name string
		edit func(*domain.SyscallRequest, *AuthorityInput)
	}{
		{"wrong operator", func(r *domain.SyscallRequest, _ *AuthorityInput) { r.Payload["operatorVersion"] = "semantic.diff.v2" }},
		{"cross scope", func(_ *domain.SyscallRequest, in *AuthorityInput) { in.Right.Scope.WorkspaceID = "ws-other" }},
		{"superseded", func(_ *domain.SyscallRequest, in *AuthorityInput) { in.Left.Current = false }},
		{"not admitted", func(_ *domain.SyscallRequest, in *AuthorityInput) { in.Left.Admitted = false }},
		{"wrong committer", func(_ *domain.SyscallRequest, in *AuthorityInput) { in.Left.CommittedBy = "adapter" }},
		{"content tamper", func(_ *domain.SyscallRequest, in *AuthorityInput) { in.Left.Content += " tamper" }},
		{"same source", func(r *domain.SyscallRequest, in *AuthorityInput) {
			r.Payload["rightEvidenceId"] = in.Left.EvidenceID
			in.Right = in.Left
		}},
		{"predates source", func(r *domain.SyscallRequest, _ *AuthorityInput) { r.RequestedAt = 1 }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req, input := validAuthorityFixture()
			tc.edit(&req, &input)
			if _, issues := Decide(req, input); len(issues) == 0 {
				t.Fatal("expected deterministic rejection")
			}
		})
	}
}

func TestVerifyDecisionRejectsOutputAndSourceTamper(t *testing.T) {
	req, input := validAuthorityFixture()
	decision, issues := Decide(req, input)
	if len(issues) != 0 {
		t.Fatal(issues)
	}
	for _, mutate := range []func(*Decision){
		func(d *Decision) { d.Content = "forged" },
		func(d *Decision) { d.ContentHash = MaterialHash("forged") },
		func(d *Decision) { d.SourceManifestHash = MaterialHash("forged") },
		func(d *Decision) { d.Left.CourtRulingID = "ruling-forged" },
	} {
		copy := decision
		copy.Tokens = append([]string(nil), decision.Tokens...)
		mutate(&copy)
		if err := VerifyDecision(req, copy); err == nil {
			t.Fatalf("expected tampered decision rejection: %+v", copy)
		}
	}
}

func validAuthorityFixture() (domain.SyscallRequest, AuthorityInput) {
	scope := domain.ForgeScope{WorkspaceID: "ws-main", LaneID: "control.semantic", SelectedPaths: []string{"docs/b", "docs/a"}}
	req := domain.SyscallRequest{
		ID: "diff-1", Action: domain.ActionComputeSemanticDiff,
		Scope: scope, RequestedAt: 2000,
		Payload: map[string]any{
			"leftEvidenceId": "evidence-left", "rightEvidenceId": "evidence-right",
			"operatorVersion": OperatorVersion,
		},
	}
	makeEvidence := func(id, content string, row int64) Evidence {
		return Evidence{
			RowID: row, EvidenceID: id, Scope: scope, Content: content,
			MaterialHash: MaterialHash(content), EvidenceHash: MaterialHash("raw-" + id),
			CourtCaseID: "case-1", CourtExhibitID: "exhibit-" + id,
			CourtRulingID: "ruling-" + id, AdmissionSyscallID: "admit-" + id,
			SourceProvenanceID:          "source-prov-" + id,
			MaterializationProvenanceID: "materialize-prov-" + id,
			CreatedAt:                   1000, CommittedBy: KernelCommitter, Current: true, Admitted: true,
		}
	}
	return req, AuthorityInput{
		Left:  makeEvidence("evidence-left", "alpha beta", 1),
		Right: makeEvidence("evidence-right", "alpha", 2),
	}
}
