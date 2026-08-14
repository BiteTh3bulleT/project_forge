package controllane

import (
	"context"
	"testing"

	"forge/projectforge/services/core/internal/aios/domain"
	"forge/projectforge/services/core/internal/memory/vsaprojection"
)

func TestMemoryAccelerationRegistryCapabilityAndCommitPlan(t *testing.T) {
	def, ok := NewStaticActionRegistry().Get(domain.ActionRebuildMemoryAcceleration)
	if !ok || !def.Mutating || def.SupportsDryRun || def.Capability != CapMemoryAccelerationRebuild {
		t.Fatalf("registry definition = %+v ok=%v", def, ok)
	}
	req := validMemoryAccelerationRequest()
	plan, err := buildPreparedCommitPlan(req, def, NewInMemorySemanticStore())
	if err != nil {
		t.Fatal(err)
	}
	if !plan.Mutating || !containsString(plan.ExpectedObjectIDs, readString(req.Payload, "expectedManifestHash")) {
		t.Fatalf("plan object identities = %+v", plan.ExpectedObjectIDs)
	}
	if plan.Details["requestedAtMs"] != req.RequestedAt || plan.Details["laneId"] != req.Scope.LaneID {
		t.Fatalf("plan details do not bind scope/time: %+v", plan.Details)
	}
	capability := NewStaticCapabilityService()
	allowed, _, err := capability.HasCapability(context.Background(), req, CapMemoryAccelerationRebuild)
	if err != nil || !allowed {
		t.Fatalf("system capability = %v err=%v", allowed, err)
	}
	req.Source = domain.SourceAdapter
	allowed, _, err = capability.HasCapability(context.Background(), req, CapMemoryAccelerationRebuild)
	if err != nil || allowed {
		t.Fatalf("adapter capability = %v err=%v", allowed, err)
	}
}

func TestMemoryAccelerationValidationBindsScopeManifestAlgorithmAndTime(t *testing.T) {
	validator := NewDeterministicValidator()
	def, _ := NewStaticActionRegistry().Get(domain.ActionRebuildMemoryAcceleration)
	valid := validMemoryAccelerationRequest()
	if issues := validator.ValidatePayload(valid, def, NewInMemorySemanticStore()); len(issues) != 0 {
		t.Fatalf("valid request issues = %+v", issues)
	}
	tests := []struct {
		name   string
		mutate func(*domain.SyscallRequest)
		field  string
	}{
		{name: "lane", field: "scope.laneId", mutate: func(req *domain.SyscallRequest) { req.Scope.LaneID = "" }},
		{name: "manifest", field: "payload.expectedManifestHash", mutate: func(req *domain.SyscallRequest) { req.Payload["expectedManifestHash"] = "legacy" }},
		{name: "algorithm", field: "payload.algorithmVersion", mutate: func(req *domain.SyscallRequest) { req.Payload["algorithmVersion"] = "latest" }},
		{name: "timestamp", field: "payload.requestedAtMs", mutate: func(req *domain.SyscallRequest) { req.Payload["requestedAtMs"] = req.RequestedAt + 1 }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := validMemoryAccelerationRequest()
			test.mutate(&req)
			issues := validator.ValidatePayload(req, def, NewInMemorySemanticStore())
			if !hasIssueField(issues, test.field) {
				t.Fatalf("issues = %+v, want field %s", issues, test.field)
			}
		})
	}
}

func validMemoryAccelerationRequest() domain.SyscallRequest {
	hash := "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	return domain.SyscallRequest{
		ID: "memory-acceleration-1", Action: domain.ActionRebuildMemoryAcceleration,
		Actor: domain.ActorIdentity{ID: "forge-core", Kind: "system"}, Source: domain.SourceSystem,
		Scope: domain.ForgeScope{WorkspaceID: "workspace-a", LaneID: "lane-a"},
		Payload: map[string]any{
			"algorithmName": vsaprojection.AlgorithmName, "algorithmVersion": vsaprojection.AlgorithmVersion,
			"dimensions": vsaprojection.DefaultDims, "seed": int(vsaprojection.DefaultSeed),
			"expectedManifestHash": hash, "expectedPriorManifestHash": "", "requestedAtMs": int64(100),
		},
		Provenance:    domain.Provenance{Actor: "forge-core", ActorType: "system"},
		CorrelationID: "corr-1", TraceID: "trace-1", RequestedAt: 100,
		RequiredCapability: CapMemoryAccelerationRebuild,
	}
}

func hasIssueField(issues []domain.SyscallError, field string) bool {
	for _, issue := range issues {
		if issue.Field == field {
			return true
		}
	}
	return false
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
