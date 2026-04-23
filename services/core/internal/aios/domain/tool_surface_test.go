package domain

import "testing"

func TestToolRequestValidation(t *testing.T) {
	t.Parallel()
	req := ToolRequest{
		ID:      "",
		ToolID:  "",
		Actor:   ActorIdentity{},
		Source:  SourceUser,
		Scope:   ForgeScope{},
		Payload: map[string]any{},
	}
	issues := req.Validate()
	if len(issues) == 0 {
		t.Fatalf("expected validation issues for incomplete tool request")
	}
}

func TestToolCapabilityValidation(t *testing.T) {
	t.Parallel()
	capability := ToolCapability{
		ID:          "filesystem.read_file",
		Domain:      "filesystem",
		Name:        "read_file",
		Description: "read",
		Status:      ToolCapabilityActive,
		Lane:        ToolLaneIO,
		Effect:      []ToolEffect{ToolEffectRead},
		Risk:        ToolRiskLow,
	}
	issues := capability.Validate()
	if len(issues) != 0 {
		t.Fatalf("expected valid capability, got %v", issues)
	}
}

func TestIsKnownToolCapabilityStatusRejectsUnknownValues(t *testing.T) {
	t.Parallel()
	if IsKnownToolCapabilityStatus(ToolCapabilityStatus("unknown")) {
		t.Fatalf("expected unknown status to be rejected")
	}
	if !IsKnownToolCapabilityStatus(ToolCapabilityStatus("ACTIVE")) {
		t.Fatalf("expected known status to be accepted case-insensitively")
	}
}
