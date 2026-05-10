package gateway

import (
	"errors"
	"strings"
	"testing"
)

func TestNormalizeAlertRuleNameIsBounded(t *testing.T) {
	t.Parallel()
	name, err := normalizeAlertRuleName("  cpu.pressure  ")
	if err != nil {
		t.Fatalf("expected valid alert name, got %v", err)
	}
	if name != "cpu.pressure" {
		t.Fatalf("unexpected normalized alert name %q", name)
	}
	if _, err := normalizeAlertRuleName(strings.Repeat("n", maxAlertRuleNameBytes+1)); !errors.Is(err, errAlertRuleNameTooLarge) {
		t.Fatalf("expected alert name size rejection, got %v", err)
	}
}

func TestNormalizeAlertRuleExpressionIsBounded(t *testing.T) {
	t.Parallel()
	expression, err := normalizeAlertRuleExpression("  alloc > 0  ")
	if err != nil {
		t.Fatalf("expected valid alert expression, got %v", err)
	}
	if expression != "alloc > 0" {
		t.Fatalf("unexpected normalized expression %q", expression)
	}
	if _, err := normalizeAlertRuleExpression(strings.Repeat("e", maxAlertRuleExpressionBytes+1)); !errors.Is(err, errAlertRuleExpressionTooLarge) {
		t.Fatalf("expected alert expression size rejection, got %v", err)
	}
}
