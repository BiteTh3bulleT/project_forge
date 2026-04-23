package jobs

import "testing"

func TestEnrichGatewayActionInputDesktopOpenUsesUserRequest(t *testing.T) {
	got := enrichGatewayActionInput("desktop.open", map[string]any{
		"application": "konsole",
	}, "open konsole and ping 10.100.1.5")
	if got["query"] != "open konsole and ping 10.100.1.5" {
		t.Fatalf("expected query to be filled from user request, got %#v", got["query"])
	}
}

func TestEnrichGatewayActionInputDesktopOpenSkipsSyntheticUserRequest(t *testing.T) {
	got := enrichGatewayActionInput("desktop.open", map[string]any{
		"application": "konsole",
	}, "Chat gateway: desktop.open (correlation chat-tools-1)")
	if _, ok := got["query"]; ok {
		t.Fatalf("expected synthetic gateway userRequest to be ignored, got %#v", got["query"])
	}
}

func TestEnrichGatewayActionInputNonDesktopOpenUnchanged(t *testing.T) {
	got := enrichGatewayActionInput("proc.run", map[string]any{
		"command": "ping 10.100.1.5",
	}, "open konsole and ping 10.100.1.5")
	if _, ok := got["query"]; ok {
		t.Fatalf("expected non-desktop tool input to remain unchanged, got %#v", got["query"])
	}
}
