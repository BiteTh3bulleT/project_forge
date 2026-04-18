package adapters

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

type ClaudeCode struct {
	workspaceDir string
}

func NewClaudeCode(workspaceDir string) ClaudeCode {
	return ClaudeCode{workspaceDir: workspaceDir}
}

func (c ClaudeCode) Info(ctx context.Context) AdapterInfo {
	_ = ctx
	claudePath := filepath.Join(c.workspaceDir, "CLAUDE.md")
	return AdapterInfo{
		ID:          "claude_code",
		DisplayName: "Claude Code",
		Status:      StatusReady,
		Detail:      "Structured handoff is wired. Execution remains operator-mediated external workflow.",
		Capabilities: []string{
			"prepare_handoff",
			"request_execution",
			"import_execution_result",
		},
		Config: map[string]any{
			"guidanceFile":  claudePath,
			"executionMode": "external_only",
		},
	}
}

func (c ClaudeCode) Invoke(ctx context.Context, req InvokeRequest) (InvokeResult, error) {
	_ = ctx
	if req.AdapterID != "" && req.AdapterID != "claude_code" {
		return InvokeResult{OK: false, FailureCode: "validation", Message: "adapterId mismatch for claude_code", Data: map[string]any{}}, nil
	}

	switch strings.TrimSpace(req.Capability) {
	case "prepare_handoff":
		if req.TaskPacketRef == nil {
			return InvokeResult{OK: false, FailureCode: "validation", Message: "taskPacketRef is required", Data: map[string]any{}}, nil
		}
		handoff := c.buildHandoff(req)
		return InvokeResult{OK: true, Message: "Claude Code handoff prepared", Data: map[string]any{
			"phase":             "prepared",
			"handoffMarkdown":   handoff,
			"deliverableType":   readString(req.Input, "expectedDeliverable"),
			"executionBoundary": "external_only",
		}}, nil
	case "request_execution":
		return InvokeResult{OK: false, FailureCode: "adapter_unavailable", Message: "Claude Code execution is external-only in this build; use operator-mediated handoff.", Data: map[string]any{
			"phase": "requested_execution",
		}}, nil
	case "import_execution_result":
		summary := readString(req.Input, "summary")
		if strings.TrimSpace(summary) == "" {
			summary = "Execution result imported without summary text."
		}
		return InvokeResult{OK: true, Message: "Claude Code execution result import accepted", Data: map[string]any{
			"phase":   "imported_result",
			"summary": summary,
		}}, nil
	default:
		return InvokeResult{OK: false, FailureCode: "validation", Message: fmt.Sprintf("unsupported capability %q for claude_code", req.Capability), Data: map[string]any{}}, nil
	}
}

func (c ClaudeCode) buildHandoff(req InvokeRequest) string {
	now := time.Now().UTC().Format(time.RFC3339)
	deliverable := readString(req.Input, "expectedDeliverable")
	if strings.TrimSpace(deliverable) == "" {
		deliverable = "implementation_plan_or_patch"
	}
	objective := readString(req.Input, "objective")
	if strings.TrimSpace(objective) == "" {
		objective = "Execute task packet with strict path and intent boundaries."
	}

	return fmt.Sprintf(`# Claude Code Handoff Packet

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
- Restrict all operations to allowed scope.
- Preserve forbidden path boundaries.
- Report changed files, rationale, and unresolved risks.
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
		filepath.Join(c.workspaceDir, "CLAUDE.md"),
	)
}
