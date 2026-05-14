package authoritymatrix

import "testing"

func TestModelRuntimeRoutesAreCovered(t *testing.T) {
	rows := ByID(DefaultRows())
	for _, id := range []string{
		"model.list",
		"model.import",
		"model.scan",
		"model.get",
		"model.compatibility",
		"model.verify",
		"model.enable",
		"model.disable",
		"model.archive",
		"model.remove_registration",
		"model.delete_file",
		"model.load",
		"model.unload",
		"model.chat",
		"modelruntime.backends",
		"modelruntime.usage",
		"modelruntime.health",
		"modelruntime.queue",
		"modelruntime.loaded",
		"openai.models",
		"model.generate",
	} {
		row, ok := rows[id]
		if !ok {
			t.Fatalf("missing authority row %q", id)
		}
		if row.AuthorityOwner != OwnerModelRuntime {
			t.Fatalf("%s owner=%q, want %q", id, row.AuthorityOwner, OwnerModelRuntime)
		}
		if row.LiveAuthority != true {
			t.Fatalf("%s liveAuthority=false, want true", id)
		}
	}
}

func TestDeleteFileRequiresApproval(t *testing.T) {
	row, ok := ByID(DefaultRows())["model.delete_file"]
	if !ok {
		t.Fatal("missing model.delete_file row")
	}
	if !row.Mutating || !row.Destructive || !row.RequiresApproval {
		t.Fatalf("delete-file governance flags wrong: %#v", row)
	}
	if row.ApprovalMechanism != ApprovalModelRuntimeManagement {
		t.Fatalf("approval mechanism=%q", row.ApprovalMechanism)
	}
	if row.Status != StatusReal || !row.LiveAuthority {
		t.Fatalf("model.delete_file must be a live modelruntime route, got %#v", row)
	}
	if row.Route != "/forge/models/{id}/delete-file" || !row.ModelRuntimeMutation {
		t.Fatalf("model.delete_file must claim the approval-governed runtime mutation route: %#v", row)
	}
	if row.GatewayCapabilityStatus != GatewayStatusApprovalOnly {
		t.Fatalf("gateway capability status=%q", row.GatewayCapabilityStatus)
	}
	if row.HostMutation || row.SemanticMemoryWrite {
		t.Fatalf("delete-file mutation boundaries wrong: %#v", row)
	}
}

func TestRemoveRegistrationIsNotDeleteFile(t *testing.T) {
	row, ok := ByID(DefaultRows())["model.remove_registration"]
	if !ok {
		t.Fatal("missing model.remove_registration row")
	}
	if row.Route != "/forge/models/{id}/remove" || row.Action != "model.remove_registration" {
		t.Fatalf("remove-registration route/action mismatch: %#v", row)
	}
	if row.Destructive {
		t.Fatalf("remove-registration must not be represented as destructive file deletion: %#v", row)
	}
	if !row.Mutating || !row.RequiresApproval || !row.ModelRuntimeMutation {
		t.Fatalf("remove-registration modelruntime governance flags wrong: %#v", row)
	}
}

func TestChatAndGenerateHaveExplicitModelRuntimeOwnership(t *testing.T) {
	rows := ByID(DefaultRows())
	for _, id := range []string{"model.chat", "model.generate"} {
		row, ok := rows[id]
		if !ok {
			t.Fatalf("missing %s row", id)
		}
		if row.AuthorityOwner != OwnerModelRuntime {
			t.Fatalf("%s owner=%q", id, row.AuthorityOwner)
		}
		if row.CapabilityID != id {
			t.Fatalf("%s capability=%q", id, row.CapabilityID)
		}
		if row.GatewayCapabilityStatus != GatewayStatusNotApplicable {
			t.Fatalf("%s incorrectly implies Gateway capability status %q", id, row.GatewayCapabilityStatus)
		}
		if row.Mutating || row.Destructive || row.RequiresApproval || row.SemanticMemoryWrite {
			t.Fatalf("%s should be inference/execution governance only, got %#v", id, row)
		}
	}
}

func TestGatewayInvokeOwnsToolExecution(t *testing.T) {
	row, ok := ByID(DefaultRows())["gateway.invoke"]
	if !ok {
		t.Fatal("missing gateway.invoke row")
	}
	if row.Method != "POST" || row.Route != "/api/gateway/invoke" {
		t.Fatalf("gateway route mismatch: %#v", row)
	}
	if row.AuthorityOwner != OwnerGateway {
		t.Fatalf("gateway invoke owner=%q", row.AuthorityOwner)
	}
	if row.CapabilityID != "gateway.tool.execute" {
		t.Fatalf("gateway invoke capability=%q", row.CapabilityID)
	}
	if !row.Mutating || row.ApprovalMechanism != ApprovalGatewayPolicy {
		t.Fatalf("gateway invoke execution governance wrong: %#v", row)
	}
	if row.ForgeKAuthority || row.ModelRuntimeMutation || row.SemanticMemoryWrite {
		t.Fatalf("gateway invoke expanded authority: %#v", row)
	}
}

func TestPartialValidationRowsDoNotClaimForgeKLiveAuthority(t *testing.T) {
	for _, row := range DefaultRows() {
		if row.ForgeKAuthority {
			t.Fatalf("%s claims FORGE-K live authority; M5A allows shared deterministic validation contracts only: %#v", row.ID, row)
		}
	}
}

func TestDiagnosticsDoNotMutateHost(t *testing.T) {
	for _, id := range []string{"hostbridge.diagnostics", "forgeh.posture", "forgeh.proposals", "system.status"} {
		row, ok := ByID(DefaultRows())[id]
		if !ok {
			t.Fatalf("missing diagnostic row %q", id)
		}
		if row.HostMutation {
			t.Fatalf("%s mutates host: %#v", id, row)
		}
		if row.SemanticMemoryWrite || row.ModelRuntimeMutation {
			t.Fatalf("%s mutates forbidden live state: %#v", id, row)
		}
	}
}

func TestRequiredMatrixFieldsArePopulated(t *testing.T) {
	for _, row := range DefaultRows() {
		if row.ID == "" || row.Surface == "" || row.Method == "" || row.Route == "" ||
			row.Action == "" || row.AuthorityOwner == "" || row.CapabilityID == "" ||
			row.GatewayCapabilityStatus == "" || row.ApprovalMechanism == "" ||
			row.AuditCategory == "" || row.AuditAction == "" || row.ResponseVisibility == "" ||
			row.Status == "" || row.Notes == "" {
			t.Fatalf("row has unpopulated required field: %#v", row)
		}
	}
}
