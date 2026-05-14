package gateway

import (
	"strings"
	"testing"

	"forge/projectforge/services/core/internal/aios/domain"
)

func TestModelRuntimeCapabilitiesDeclareGatewayModelruntimeRouting(t *testing.T) {
	t.Parallel()
	reg := NewToolCapabilityRegistry()

	for _, id := range []string{"model.chat", "model.generate"} {
		id := id
		t.Run(id, func(t *testing.T) {
			t.Parallel()
			capability, ok := reg.Get(id)
			if !ok {
				t.Fatalf("missing capability %s", id)
			}
			if capability.Status != domain.ToolCapabilityApprovalOnly {
				t.Fatalf("expected approval-only runtime capability, got %s", capability.Status)
			}
			if capability.Risk != domain.ToolRiskHigh {
				t.Fatalf("expected high risk model execution capability, got %s", capability.Risk)
			}
			if got := metadataString(capability.Metadata, "runtimeAuthority"); got != "modelruntime_api" {
				t.Fatalf("expected modelruntime API authority marker, got %q", got)
			}
			if got := metadataString(capability.Metadata, "gatewayToolId"); got != id {
				t.Fatalf("expected explicit gateway tool id %q, got %q", id, got)
			}
			if !strings.HasPrefix(capability.AdapterID, "gateway.model_") {
				t.Fatalf("expected gateway model adapter marker, got %q", capability.AdapterID)
			}
		})
	}
}

func TestModelDeleteFileCapabilityRemainsDeferredGatewayCapability(t *testing.T) {
	t.Parallel()
	reg := NewToolCapabilityRegistry()

	capability, ok := reg.Get("model.delete_file")
	if !ok {
		t.Fatalf("missing model.delete_file capability")
	}
	if capability.Status != domain.ToolCapabilityDeferred {
		t.Fatalf("destructive model file deletion gateway capability must remain deferred, got %s", capability.Status)
	}
	if !capability.RequiresApprovalByDefault {
		t.Fatalf("destructive model file deletion must remain approval-required")
	}
	if capability.Risk != domain.ToolRiskCritical {
		t.Fatalf("expected critical risk for destructive model file deletion, got %s", capability.Risk)
	}
	if got := metadataString(capability.Metadata, "runtimeAuthority"); got != "modelruntime_api" {
		t.Fatalf("expected modelruntime API authority marker, got %q", got)
	}
	deferredReason := strings.ToLower(metadataString(capability.Metadata, "deferredReason"))
	if !strings.Contains(deferredReason, "approval") {
		t.Fatalf("expected approval-oriented deferred reason, got %q", deferredReason)
	}
}
