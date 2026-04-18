package adapters

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

type Codex struct {
	workspaceDir string
}

func NewCodex(workspaceDir string) Codex {
	return Codex{workspaceDir: workspaceDir}
}

func (c Codex) Info(ctx context.Context) AdapterInfo {
	_ = ctx
	agentsPath := filepath.Join(c.workspaceDir, "AGENTS.md")
	return AdapterInfo{
		ID:          "codex",
		DisplayName: "Codex",
		Status:      StatusReady,
		Detail:      "Structured handoff is wired. Execution remains explicit external workflow with operator control.",
		Capabilities: []string{
			"prepare_handoff",
			"request_execution",
			"import_execution_result",
		},
		Config: map[string]any{
			"guidanceFile":  agentsPath,
			"executionMode": "external_only",
		},
	}
}

func (c Codex) Invoke(ctx context.Context, req InvokeRequest) (InvokeResult, error) {
	_ = ctx
	if req.AdapterID != "" && req.AdapterID != "codex" {
		return InvokeResult{OK: false, FailureCode: "validation", Message: "adapterId mismatch for codex", Data: map[string]any{}}, nil
	}

	switch strings.TrimSpace(req.Capability) {
	case "prepare_handoff":
		if req.TaskPacketRef == nil {
			return InvokeResult{OK: false, FailureCode: "validation", Message: "taskPacketRef is required", Data: map[string]any{}}, nil
		}
		handoff := c.buildHandoff(req)
		return InvokeResult{OK: true, Message: "Codex handoff prepared", Data: map[string]any{
			"phase":             "prepared",
			"handoffMarkdown":   handoff,
			"deliverableType":   readString(req.Input, "expectedDeliverable"),
			"executionBoundary": "external_only",
		}}, nil
	case "request_execution":
		return InvokeResult{OK: false, FailureCode: "adapter_unavailable", Message: "Codex execution is external-only in this build; prepare handoff and run operator-mediated execution.", Data: map[string]any{
			"phase": "requested_execution",
		}}, nil
	case "import_execution_result":
		summary := readString(req.Input, "summary")
		if strings.TrimSpace(summary) == "" {
			summary = "Execution result imported without summary text."
		}
		return InvokeResult{OK: true, Message: "Codex execution result import accepted", Data: map[string]any{
			"phase":   "imported_result",
			"summary": summary,
		}}, nil
	default:
		return InvokeResult{OK: false, FailureCode: "validation", Message: fmt.Sprintf("unsupported capability %q for codex", req.Capability), Data: map[string]any{}}, nil
	}
}

func (c Codex) buildHandoff(req InvokeRequest) string {
	now := time.Now().UTC().Format(time.RFC3339)
	deliverable := readString(req.Input, "expectedDeliverable")
	if strings.TrimSpace(deliverable) == "" {
		deliverable = "code_change_summary"
	}
	objective := readString(req.Input, "objective")
	if strings.TrimSpace(objective) == "" {
		objective = "Execute task packet with bounded scope and explicit result reporting."
	}

	return fmt.Sprintf(`# Codex Handoff Packet

Prepared: %s
Task packet ref: %d
Correlation id: %s

## Objective
%s

## Scope
Allowed paths:
%s
Forbidden paths:
%s
Selected paths:
%s

## Intent
- Write intent: %t
- Dry run: %t
- Expected deliverable: %s
- Guidance file: %s
- Execution mode: external_only

## Contract
- Do not touch paths outside allowed scope.
- Do not execute hidden commands.
- Return explicit summary, files touched, and outstanding risks.
`,
		now,
		valueOrZero(req.TaskPacketRef),
		req.CorrelationID,
		objective,
		bulletLines(req.Scope.AllowedPaths),
		bulletLines(req.Scope.ForbiddenPaths),
		bulletLines(req.Scope.SelectedPaths),
		req.WriteIntent,
		req.DryRun,
		deliverable,
		filepath.Join(c.workspaceDir, "AGENTS.md"),
	)
}

func valueOrZero(v *int64) int64 {
	if v == nil {
		return 0
	}
	return *v
}

func bulletLines(items []string) string {
	if len(items) == 0 {
		return "- (none)"
	}
	var b strings.Builder
	for _, it := range items {
		trim := strings.TrimSpace(it)
		if trim == "" {
			continue
		}
		b.WriteString("- ")
		b.WriteString(trim)
		b.WriteString("\n")
	}
	out := strings.TrimSpace(b.String())
	if out == "" {
		return "- (none)"
	}
	return out
}
