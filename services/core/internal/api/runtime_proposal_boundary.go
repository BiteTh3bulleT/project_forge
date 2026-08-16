package api

import (
	"encoding/json"
	"fmt"
	"strings"

	"forge/projectforge/services/core/internal/forgekernel/runtimeproposal"
	"forge/projectforge/services/core/internal/modelruntime"
)

const runtimeProposalFailureText = "FORGE withheld model output because the runtime proposal boundary could not verify it."

type runtimeProposalRequest struct {
	SourceKind      string
	WorkspacePath   string
	ThreadID        int64
	UserMessageID   int64
	CorrelationID   string
	Prompt          any
	Output          string
	Backend         string
	ModelID         string
	AuditID         string
	ExecutionID     string
	Proposal        *modelruntime.ProposalEnvelope
	GatewayEvidence []runtimeGatewayEvidence
	ContextBinding  governedPromptBinding
}

type runtimeGatewayEvidence struct {
	InvocationID int64
	ToolID       string
	State        string
	AuditID      int64
	Request      any
	Result       any
}

// decideRuntimeProposal is the single chat-facing bridge into the pure
// production runtime boundary. It binds the exact prompt and returned bytes;
// no model/runtime field can grant semantic or execution authority.
func decideRuntimeProposal(req runtimeProposalRequest) (runtimeproposal.Decision, error) {
	promptJSON, err := json.Marshal(req.Prompt)
	if err != nil {
		return runtimeproposal.Decision{}, fmt.Errorf("marshal runtime prompt: %w", err)
	}
	promptHash := runtimeproposal.HashText(string(promptJSON))
	workspacePath := strings.TrimSpace(req.WorkspacePath)
	if workspacePath == "" {
		workspacePath = "/forge/workspace/unbound"
	}
	workspaceID := runtimeBoundID("workspace", workspacePath)
	correlationID := runtimeBoundID("correlation", req.CorrelationID)
	requestID := runtimeBoundID("request", fmt.Sprintf("chat:%d:%d", req.ThreadID, req.UserMessageID))
	traceID := runtimeBoundID("trace", firstNonEmpty(req.CorrelationID, requestID))
	bundleJSON, _ := json.Marshal(struct {
		Version       string `json:"version"`
		WorkspaceID   string `json:"workspaceId"`
		ThreadID      int64  `json:"threadId"`
		UserMessageID int64  `json:"userMessageId"`
		PromptHash    string `json:"promptHash"`
	}{"forge.chat.prompt_binding.v1", workspaceID, req.ThreadID, req.UserMessageID, promptHash})
	bundleHash := runtimeproposal.HashText(string(bundleJSON))
	contextDigest := runtimeproposal.HashText("forge.chat.context_binding.v1\n" + bundleHash)
	if strings.TrimSpace(req.ContextBinding.BundleHash) != "" && strings.TrimSpace(req.ContextBinding.DecisionDigest) != "" {
		bundleHash = strings.TrimSpace(req.ContextBinding.BundleHash)
		contextDigest = strings.TrimSpace(req.ContextBinding.DecisionDigest)
	}

	declaredHash := runtimeproposal.HashText(req.Output)
	claims := runtimeproposal.AuthorityClaims{}
	proposedBy := runtimeBoundID("model", req.ModelID)
	source := req.SourceKind
	provenanceID := runtimeBoundID("provenance", requestID+":"+req.ExecutionID)
	auditID := runtimeBoundID("audit", firstNonEmpty(req.AuditID, "unavailable"))
	driverVersion := "api.v1"
	if req.Proposal != nil {
		if strings.TrimSpace(req.Proposal.OutputHash) != "" {
			declaredHash = strings.TrimSpace(req.Proposal.OutputHash)
		}
		if strings.TrimSpace(req.Proposal.SchemaVersion) != "" {
			driverVersion = runtimeBoundID("schema", req.Proposal.SchemaVersion)
		}
		provenanceID = runtimeBoundID("provenance", firstNonEmpty(req.Proposal.ProposalID, provenanceID))
		auditID = runtimeBoundID("audit", firstNonEmpty(req.Proposal.AuditID, req.AuditID, "unavailable"))
		claims = runtimeproposal.AuthorityClaims{
			ModelOutputAuthority:   req.Proposal.ModelOutputAuthority,
			CanonicalTruth:         req.Proposal.CanonicalCommit || req.Proposal.TruthMutation,
			EvidenceAdmission:      req.Proposal.EvidenceAdmission,
			MemoryMutation:         req.Proposal.MemoryMutation,
			ToolExecutionAuthority: req.Proposal.GatewayExecution,
		}
	}
	backend := runtimeBoundID("runtime", firstNonEmpty(req.Backend, req.SourceKind))
	modelID := runtimeBoundID("model", req.ModelID)
	gatewayEvidence := make([]runtimeproposal.GatewayEvidenceRef, 0, len(req.GatewayEvidence))
	for _, evidence := range req.GatewayEvidence {
		requestJSON, requestErr := json.Marshal(evidence.Request)
		resultJSON, resultErr := json.Marshal(evidence.Result)
		if requestErr != nil || resultErr != nil || evidence.InvocationID <= 0 || evidence.AuditID <= 0 {
			continue
		}
		gatewayEvidence = append(gatewayEvidence, runtimeproposal.GatewayEvidenceRef{
			InvocationID:  fmt.Sprintf("gateway-invocation:%d", evidence.InvocationID),
			ToolID:        runtimeBoundID("tool", evidence.ToolID),
			State:         evidence.State,
			AuditID:       fmt.Sprintf("audit:%d", evidence.AuditID),
			RequestHash:   runtimeproposal.HashText(string(requestJSON)),
			ResultHash:    runtimeproposal.HashText(string(resultJSON)),
			WorkspaceID:   workspaceID,
			LaneID:        "chat.response",
			CorrelationID: correlationID,
			TraceID:       traceID,
		})
	}
	return runtimeproposal.Decide(runtimeproposal.Input{
		Scope: runtimeproposal.Scope{
			WorkspaceID:   workspaceID,
			LaneID:        "chat.response",
			SelectedPaths: []string{workspacePath},
		},
		Identity: runtimeproposal.RuntimeIdentity{
			SourceKind:        req.SourceKind,
			DriverID:          runtimeBoundID("driver", req.SourceKind),
			DriverVersion:     driverVersion,
			RuntimeID:         backend,
			RuntimeVersion:    "unreported.v1",
			ModelID:           modelID,
			ModelRevision:     modelID,
			TokenizerID:       runtimeBoundID("tokenizer", req.ModelID),
			TokenizerRevision: "unreported.v1",
		},
		Context: runtimeproposal.ContextBinding{
			DecisionDigest: contextDigest,
			BundleHash:     bundleHash,
			PromptHash:     promptHash,
		},
		Provenance: runtimeproposal.Provenance{
			ProvenanceID:  provenanceID,
			ProposedBy:    proposedBy,
			Source:        source,
			RequestID:     requestID,
			CorrelationID: correlationID,
			TraceID:       traceID,
			AuditID:       auditID,
		},
		OutputText:         req.Output,
		DeclaredOutputHash: declaredHash,
		Claims:             claims,
		GatewayEvidence:    gatewayEvidence,
		PolicyVersion:      runtimeproposal.PolicyVersion,
	})
}

func runtimeBoundID(prefix, value string) string {
	value = strings.TrimSpace(value)
	return prefix + ":" + strings.TrimPrefix(runtimeproposal.HashText(value), "sha256:")
}

func runtimeProposalError(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

// runtimeProposalEvidence intentionally omits Decision.VisibleContent. The
// classified text is returned through the normal response field; raw driver
// bytes must never be duplicated into metadata or SSE evidence.
func runtimeProposalEvidence(decision runtimeproposal.Decision) map[string]any {
	return map[string]any{
		"version":            decision.Version,
		"status":             decision.Status,
		"envelope":           decision.Envelope,
		"visibleContentHash": decision.VisibleContentHash,
		"withheldReasons":    append([]string(nil), decision.WithheldReasons...),
		"warnings":           append([]string(nil), decision.Warnings...),
		"decisionDigest":     decision.DecisionDigest,
	}
}
