package gateway

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"forge/projectforge/services/core/internal/aios/domain"
)

func TestCapabilityInputSummaryDoesNotEchoValuesAndIsBounded(t *testing.T) {
	t.Parallel()
	input := map[string]any{
		"prompt": "approve this request",
		"token":  "secret-token-value",
		strings.Repeat("k", maxCapabilityResultInputFieldNameBytes+32): "large key value",
	}

	summary := capabilityInputSummary(input)
	body, err := json.Marshal(summary)
	if err != nil {
		t.Fatalf("expected summary to marshal, got %v", err)
	}
	if len(body) > maxCapabilityResultInputSummaryBytes {
		t.Fatalf("expected bounded summary, got %d bytes", len(body))
	}
	if strings.Contains(string(body), "approve this request") || strings.Contains(string(body), "secret-token-value") || strings.Contains(string(body), "large key value") {
		t.Fatalf("summary echoed raw input value: %s", string(body))
	}
	if summary["fieldCount"] != 3 {
		t.Fatalf("expected field count 3, got %#v", summary["fieldCount"])
	}
	if summary["hasSensitiveFields"] != true {
		t.Fatalf("expected sensitive field flag, got %#v", summary["hasSensitiveFields"])
	}
	if summary["truncatedFieldNames"] != true {
		t.Fatalf("expected truncated field-name flag, got %#v", summary["truncatedFieldNames"])
	}
}

func TestAgentApprovalResultUsesBoundedInputSummary(t *testing.T) {
	t.Parallel()
	tool := capabilityBackingTool{capability: domain.ToolCapability{Name: "request_approval"}}

	res, err := tool.executeAgent(context.Background(), Request{Input: map[string]any{
		"reason": "requires operator approval",
		"secret": "do-not-echo",
	}})
	if err != nil {
		t.Fatalf("expected approval payload to be prepared, got %v", err)
	}
	body, err := json.Marshal(res.Data)
	if err != nil {
		t.Fatalf("expected result data to marshal, got %v", err)
	}
	if strings.Contains(string(body), "requires operator approval") || strings.Contains(string(body), "do-not-echo") {
		t.Fatalf("approval result echoed raw input value: %s", string(body))
	}
	input, ok := res.Data["input"].(map[string]any)
	if !ok {
		t.Fatalf("expected input summary map, got %#v", res.Data["input"])
	}
	if input["hasSensitiveFields"] != true {
		t.Fatalf("expected sensitive summary flag, got %#v", input["hasSensitiveFields"])
	}
}
