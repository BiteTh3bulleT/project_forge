package gateway

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestMarshalGatewayInvocationInputPreservesSmallInput(t *testing.T) {
	t.Parallel()
	body, err := marshalGatewayInvocationInput(map[string]any{"query": "status"}, map[string]any{"capabilityId": "cap.read"})
	if err != nil {
		t.Fatalf("expected invocation input to marshal, got %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(body, &decoded); err != nil {
		t.Fatalf("expected invocation input JSON, got %v", err)
	}
	if decoded["query"] != "status" {
		t.Fatalf("expected raw small input to be preserved, got %#v", decoded)
	}
	meta, ok := decoded["_metadata"].(map[string]any)
	if !ok || meta["capabilityId"] != "cap.read" {
		t.Fatalf("expected metadata to be preserved, got %#v", decoded["_metadata"])
	}
}

func TestMarshalGatewayInvocationInputSummarizesOversizeInput(t *testing.T) {
	t.Parallel()
	large := strings.Repeat("x", maxGatewayInvocationInputJSONBytes+1)
	body, err := marshalGatewayInvocationInput(map[string]any{"token": "secret-token", "payload": large}, map[string]any{"note": large})
	if err != nil {
		t.Fatalf("expected oversized invocation input to summarize, got %v", err)
	}
	if len(body) > maxGatewayInvocationInputJSONBytes {
		t.Fatalf("expected bounded invocation input, got %d bytes", len(body))
	}
	if strings.Contains(string(body), "secret-token") || strings.Contains(string(body), large) {
		t.Fatalf("expected oversized invocation input to omit raw values")
	}
	var decoded map[string]any
	if err := json.Unmarshal(body, &decoded); err != nil {
		t.Fatalf("expected summary JSON, got %v", err)
	}
	if decoded["_inputOmitted"] != true {
		t.Fatalf("expected omitted marker, got %#v", decoded)
	}
	if _, ok := decoded["_inputSummary"].(map[string]any); !ok {
		t.Fatalf("expected input summary, got %#v", decoded["_inputSummary"])
	}
	if _, ok := decoded["_metadataSummary"].(map[string]any); !ok {
		t.Fatalf("expected metadata summary, got %#v", decoded["_metadataSummary"])
	}
}
