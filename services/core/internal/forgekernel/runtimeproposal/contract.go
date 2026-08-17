// Package runtimeproposal defines the pure production FORGE-K boundary for
// model-driver output. It performs no I/O and has no clock, database, model,
// gateway, cache, domain, or simulator dependency.
package runtimeproposal

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	ContractVersion = "forge.runtime_proposal.decision.v2"
	PolicyVersion   = "forge.runtime_proposal.policy.v2"

	SourceModelRuntime = "modelruntime"
	SourceNativeOllama = "native_ollama"

	StatusAccepted = "accepted_proposal"
	StatusWithheld = "withheld"

	MaxOutputBytes        = 4 << 20
	MaxSelectedPaths      = 64
	MaxGatewayEvidence    = 32
	MaxIdentifierBytes    = 4096
	withheldVisibleText   = "FORGE withheld an unverified model proposal before final visibility."
	OutputHashAlgorithm   = "sha256"
	ProposalAuthority     = "proposal_only"
	NonCanonicalEvidence  = "NONCANONICAL_MODEL_PROPOSAL"
	ContextAuthorityOwner = "forge_k.kernel"
)

var ErrInvalidInput = errors.New("invalid runtime proposal input")

type ValidationError struct {
	Field  string
	Reason string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("%v: %s: %s", ErrInvalidInput, e.Field, e.Reason)
}

func (e *ValidationError) Unwrap() error { return ErrInvalidInput }

func invalid(field, reason string) error {
	return &ValidationError{Field: field, Reason: reason}
}

type Scope struct {
	WorkspaceID   string   `json:"workspaceId"`
	LaneID        string   `json:"laneId"`
	SelectedPaths []string `json:"selectedPaths"`
}

type RuntimeIdentity struct {
	SourceKind        string `json:"sourceKind"`
	DriverID          string `json:"driverId"`
	DriverVersion     string `json:"driverVersion"`
	RuntimeID         string `json:"runtimeId"`
	RuntimeVersion    string `json:"runtimeVersion"`
	ModelID           string `json:"modelId"`
	ModelRevision     string `json:"modelRevision"`
	TokenizerID       string `json:"tokenizerId"`
	TokenizerRevision string `json:"tokenizerRevision"`
}

type ContextBinding struct {
	PacketID                 string `json:"packetId"`
	DecisionDigest           string `json:"decisionDigest"`
	BundleHash               string `json:"bundleHash"`
	PromptHash               string `json:"promptHash"`
	AuthorityOwner           string `json:"authorityOwner"`
	TransactionID            string `json:"transactionId"`
	JournalEventID           string `json:"journalEventId"`
	PreparedPlanSeal         string `json:"preparedPlanSeal"`
	AuthorizationFingerprint string `json:"authorizationFingerprint"`
}

type Provenance struct {
	ProvenanceID  string `json:"provenanceId"`
	ProposedBy    string `json:"proposedBy"`
	Source        string `json:"source"`
	RequestID     string `json:"requestId"`
	CorrelationID string `json:"correlationId"`
	TraceID       string `json:"traceId"`
	AuditID       string `json:"auditId"`
}

// AuthorityClaims are untrusted claims attached to driver output. Every
// authority claim is forbidden. ActionCompletion is different: it may be
// displayed only when exact gateway evidence is present and scope-bound.
type AuthorityClaims struct {
	ModelOutputAuthority   bool `json:"modelOutputAuthority"`
	CanonicalTruth         bool `json:"canonicalTruth"`
	EvidenceAdmission      bool `json:"evidenceAdmission"`
	MemoryMutation         bool `json:"memoryMutation"`
	ToolSelectionAuthority bool `json:"toolSelectionAuthority"`
	ToolExecutionAuthority bool `json:"toolExecutionAuthority"`
	ActionCompletion       bool `json:"actionCompletion"`
}

type GatewayEvidenceRef struct {
	InvocationID  string `json:"invocationId"`
	ToolID        string `json:"toolId"`
	State         string `json:"state"`
	AuditID       string `json:"auditId"`
	RequestHash   string `json:"requestHash"`
	ResultHash    string `json:"resultHash"`
	WorkspaceID   string `json:"workspaceId"`
	LaneID        string `json:"laneId"`
	CorrelationID string `json:"correlationId"`
	TraceID       string `json:"traceId"`
}

type Input struct {
	Scope              Scope                `json:"scope"`
	Identity           RuntimeIdentity      `json:"identity"`
	Context            ContextBinding       `json:"context"`
	Provenance         Provenance           `json:"provenance"`
	OutputText         string               `json:"outputText"`
	DeclaredOutputHash string               `json:"declaredOutputHash"`
	Claims             AuthorityClaims      `json:"claims"`
	GatewayEvidence    []GatewayEvidenceRef `json:"gatewayEvidence"`
	PolicyVersion      string               `json:"policyVersion"`
}

type Envelope struct {
	Version                   string          `json:"version"`
	PolicyVersion             string          `json:"policyVersion"`
	ProposalID                string          `json:"proposalId"`
	ObjectClass               string          `json:"objectClass"`
	AuthorityLevel            string          `json:"authorityLevel"`
	Scope                     Scope           `json:"scope"`
	Identity                  RuntimeIdentity `json:"identity"`
	ContextDecisionDigest     string          `json:"contextDecisionDigest"`
	ContextBundleHash         string          `json:"contextBundleHash"`
	ContextPacketID           string          `json:"contextPacketId"`
	ContextAuthorityOwner     string          `json:"contextAuthorityOwner"`
	ContextTransactionID      string          `json:"contextTransactionId"`
	ContextJournalEventID     string          `json:"contextJournalEventId"`
	ContextPreparedPlanSeal   string          `json:"contextPreparedPlanSeal"`
	ContextAuthorizationProof string          `json:"contextAuthorizationProof"`
	PromptHash                string          `json:"promptHash"`
	DeclaredOutputHash        string          `json:"declaredOutputHash"`
	OutputHash                string          `json:"outputHash"`
	OutputHashVerified        bool            `json:"outputHashVerified"`
	OutputBytes               int             `json:"outputBytes"`
	ProvenanceHash            string          `json:"provenanceHash"`
	RuntimeIdentityHash       string          `json:"runtimeIdentityHash"`
	ScopeHash                 string          `json:"scopeHash"`
	GatewayEvidenceCommitment string          `json:"gatewayEvidenceCommitment"`
	GatewayEvidenceCount      int             `json:"gatewayEvidenceCount"`
	GatewayExecutionObserved  bool            `json:"gatewayExecutionObserved"`
	ProposalOnly              bool            `json:"proposalOnly"`
	CanonicalTruth            bool            `json:"canonicalTruth"`
	EvidenceAdmission         bool            `json:"evidenceAdmission"`
	MemoryMutation            bool            `json:"memoryMutation"`
	ToolSelectionAuthority    bool            `json:"toolSelectionAuthority"`
	ToolExecutionAuthority    bool            `json:"toolExecutionAuthority"`
	RequiresKernelCommit      bool            `json:"requiresKernelCommit"`
}

type Decision struct {
	Version            string   `json:"version"`
	Status             string   `json:"status"`
	Envelope           Envelope `json:"envelope"`
	VisibleContent     string   `json:"visibleContent"`
	VisibleContentHash string   `json:"visibleContentHash"`
	WithheldReasons    []string `json:"withheldReasons"`
	Warnings           []string `json:"warnings"`
	DecisionDigest     string   `json:"decisionDigest"`
}

// Decide normalizes and binds one external driver result. Structural errors
// return ErrInvalidInput. Untrusted or unproved authority/action claims return
// a valid withheld decision whose visible content never contains driver text.
func Decide(input Input) (Decision, error) {
	normalized, err := normalizeAndValidate(input)
	if err != nil {
		return Decision{}, err
	}

	outputHash := HashText(normalized.OutputText)
	scopeHash, _ := hashValue(normalized.Scope)
	identityHash, _ := hashValue(normalized.Identity)
	provenanceHash, _ := hashValue(normalized.Provenance)
	evidenceCommitment, _ := hashValue(normalized.GatewayEvidence)
	proposalBindingHash, _ := hashValue(normalized)

	reasons := make([]string, 0, 8)
	if normalized.DeclaredOutputHash != outputHash {
		reasons = append(reasons, "output_hash_mismatch")
	}
	if normalized.Claims.ModelOutputAuthority {
		reasons = append(reasons, "model_output_authority_forbidden")
	}
	if normalized.Claims.CanonicalTruth {
		reasons = append(reasons, "canonical_truth_claim_forbidden")
	}
	if normalized.Claims.EvidenceAdmission {
		reasons = append(reasons, "evidence_admission_claim_forbidden")
	}
	if normalized.Claims.MemoryMutation {
		reasons = append(reasons, "memory_mutation_claim_forbidden")
	}
	if normalized.Claims.ToolSelectionAuthority {
		reasons = append(reasons, "tool_selection_authority_forbidden")
	}
	if normalized.Claims.ToolExecutionAuthority {
		reasons = append(reasons, "tool_execution_authority_forbidden")
	}

	validGatewayCompletion, evidenceReasons := verifyGatewayEvidence(normalized)
	reasons = append(reasons, evidenceReasons...)
	actionLanguage := containsActionCompletionLanguage(normalized.OutputText)
	if (normalized.Claims.ActionCompletion || actionLanguage) && !validGatewayCompletion {
		reasons = append(reasons, "unbound_action_completion_claim")
	}
	reasons = uniqueSorted(reasons)

	envelope := Envelope{
		Version: ContractVersion, PolicyVersion: PolicyVersion,
		ProposalID:  "runtime-proposal:" + strings.TrimPrefix(proposalBindingHash, OutputHashAlgorithm+":"),
		ObjectClass: NonCanonicalEvidence, AuthorityLevel: ProposalAuthority,
		Scope: normalized.Scope, Identity: normalized.Identity,
		ContextDecisionDigest:     normalized.Context.DecisionDigest,
		ContextBundleHash:         normalized.Context.BundleHash,
		ContextPacketID:           normalized.Context.PacketID,
		ContextAuthorityOwner:     normalized.Context.AuthorityOwner,
		ContextTransactionID:      normalized.Context.TransactionID,
		ContextJournalEventID:     normalized.Context.JournalEventID,
		ContextPreparedPlanSeal:   normalized.Context.PreparedPlanSeal,
		ContextAuthorizationProof: normalized.Context.AuthorizationFingerprint,
		PromptHash:                normalized.Context.PromptHash,
		DeclaredOutputHash:        normalized.DeclaredOutputHash,
		OutputHash:                outputHash,
		OutputHashVerified:        normalized.DeclaredOutputHash == outputHash,
		OutputBytes:               len([]byte(normalized.OutputText)),
		ProvenanceHash:            provenanceHash, RuntimeIdentityHash: identityHash,
		ScopeHash: scopeHash, GatewayEvidenceCommitment: evidenceCommitment,
		GatewayEvidenceCount:     len(normalized.GatewayEvidence),
		GatewayExecutionObserved: validGatewayCompletion,
		ProposalOnly:             true, CanonicalTruth: false, EvidenceAdmission: false,
		MemoryMutation: false, ToolSelectionAuthority: false,
		ToolExecutionAuthority: false, RequiresKernelCommit: true,
	}

	decision := Decision{
		Version: ContractVersion, Status: StatusAccepted, Envelope: envelope,
		VisibleContent: normalized.OutputText, WithheldReasons: []string{},
		Warnings: []string{"model_driver_output_is_proposal_only"},
	}
	if len(reasons) > 0 {
		decision.Status = StatusWithheld
		decision.VisibleContent = withheldVisibleText
		decision.WithheldReasons = reasons
		decision.Warnings = []string{"unverified_model_output_withheld_before_visibility"}
	}
	decision.VisibleContentHash = HashText(decision.VisibleContent)
	decision.DecisionDigest, _ = decisionDigest(decision)
	return decision, nil
}

// VerifyDecision reconstructs the decision from the original driver input and
// rejects any envelope, classification, visible-content, or digest tampering.
func VerifyDecision(input Input, decision Decision) error {
	expected, err := Decide(input)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(expected, decision) {
		return invalid("decision", "decision differs from deterministic runtime proposal contract")
	}
	return nil
}

// HashText is the exact byte commitment used for driver and visible content.
func HashText(content string) string {
	sum := sha256.Sum256([]byte(content))
	return OutputHashAlgorithm + ":" + hex.EncodeToString(sum[:])
}

func normalizeAndValidate(in Input) (Input, error) {
	in.PolicyVersion = strings.TrimSpace(in.PolicyVersion)
	if in.PolicyVersion != PolicyVersion {
		return Input{}, invalid("policyVersion", "unsupported or missing policy")
	}
	var err error
	in.Scope, err = normalizeScope(in.Scope)
	if err != nil {
		return Input{}, err
	}
	in.Identity = normalizeIdentity(in.Identity)
	if err := validateIdentity(in.Identity); err != nil {
		return Input{}, err
	}
	in.Context = normalizeContextBinding(in.Context)
	if err := ValidateContextBinding(in.Context); err != nil {
		return Input{}, err
	}
	in.Provenance = normalizeProvenance(in.Provenance)
	if err := validateProvenance(in.Provenance); err != nil {
		return Input{}, err
	}
	if in.OutputText == "" || len([]byte(in.OutputText)) > MaxOutputBytes || !utf8.ValidString(in.OutputText) || strings.ContainsRune(in.OutputText, 0) {
		return Input{}, invalid("outputText", "output is empty, invalid UTF-8, contains NUL, or exceeds bound")
	}
	in.DeclaredOutputHash = strings.TrimSpace(in.DeclaredOutputHash)
	if !validHash(in.DeclaredOutputHash) {
		return Input{}, invalid("declaredOutputHash", "sha256 commitment is required")
	}
	if len(in.GatewayEvidence) > MaxGatewayEvidence {
		return Input{}, invalid("gatewayEvidence", "evidence count exceeds bound")
	}
	seen := map[string]struct{}{}
	for i := range in.GatewayEvidence {
		in.GatewayEvidence[i] = normalizeGatewayEvidence(in.GatewayEvidence[i])
		if err := validateGatewayEvidenceShape(in.GatewayEvidence[i]); err != nil {
			return Input{}, invalid(fmt.Sprintf("gatewayEvidence[%d]", i), err.Error())
		}
		if _, ok := seen[in.GatewayEvidence[i].InvocationID]; ok {
			return Input{}, invalid("gatewayEvidence", "duplicate invocation identity")
		}
		seen[in.GatewayEvidence[i].InvocationID] = struct{}{}
	}
	sort.Slice(in.GatewayEvidence, func(i, j int) bool {
		if in.GatewayEvidence[i].InvocationID != in.GatewayEvidence[j].InvocationID {
			return in.GatewayEvidence[i].InvocationID < in.GatewayEvidence[j].InvocationID
		}
		return in.GatewayEvidence[i].ToolID < in.GatewayEvidence[j].ToolID
	})
	return in, nil
}

// ValidateContextBinding requires the immutable identity and commit proof
// emitted by one successful production Context Compiler transaction. It does
// not accept caller-synthesized digest/bundle pairs as authority evidence.
func ValidateContextBinding(binding ContextBinding) error {
	binding = normalizeContextBinding(binding)
	if err := ValidateContextCompilerIssuance(binding); err != nil {
		return err
	}
	if !validHash(binding.PromptHash) {
		return invalid("context.promptHash", "sha256 commitment is required")
	}
	return nil
}

// ValidateContextCompilerIssuance validates the authority and durable receipt
// portion of a binding before exact prompt bytes are available.
func ValidateContextCompilerIssuance(binding ContextBinding) error {
	binding = normalizeContextBinding(binding)
	if binding.AuthorityOwner != ContextAuthorityOwner {
		return invalid("context.authorityOwner", "production FORGE-K Context Compiler authority is required")
	}
	for _, item := range []struct {
		field string
		value string
	}{
		{"packetId", binding.PacketID},
		{"transactionId", binding.TransactionID},
		{"journalEventId", binding.JournalEventID},
	} {
		if !validID(item.value) {
			return invalid("context."+item.field, "live Context Compiler commit identity is required")
		}
	}
	for _, item := range []struct {
		field string
		value string
	}{
		{"decisionDigest", binding.DecisionDigest},
		{"bundleHash", binding.BundleHash},
		{"preparedPlanSeal", binding.PreparedPlanSeal},
		{"authorizationFingerprint", binding.AuthorizationFingerprint},
	} {
		if !validHash(item.value) {
			return invalid("context."+item.field, "sha256 commitment is required")
		}
	}
	return nil
}

func normalizeContextBinding(binding ContextBinding) ContextBinding {
	binding.PacketID = strings.TrimSpace(binding.PacketID)
	binding.DecisionDigest = strings.TrimSpace(binding.DecisionDigest)
	binding.BundleHash = strings.TrimSpace(binding.BundleHash)
	binding.PromptHash = strings.TrimSpace(binding.PromptHash)
	binding.AuthorityOwner = strings.TrimSpace(binding.AuthorityOwner)
	binding.TransactionID = strings.TrimSpace(binding.TransactionID)
	binding.JournalEventID = strings.TrimSpace(binding.JournalEventID)
	binding.PreparedPlanSeal = strings.TrimSpace(binding.PreparedPlanSeal)
	binding.AuthorizationFingerprint = strings.TrimSpace(binding.AuthorizationFingerprint)
	return binding
}

func normalizeScope(scope Scope) (Scope, error) {
	scope.WorkspaceID = strings.TrimSpace(scope.WorkspaceID)
	scope.LaneID = strings.TrimSpace(scope.LaneID)
	if !validID(scope.WorkspaceID) || !validID(scope.LaneID) {
		return Scope{}, invalid("scope", "workspace and lane are required")
	}
	if len(scope.SelectedPaths) < 1 || len(scope.SelectedPaths) > MaxSelectedPaths {
		return Scope{}, invalid("scope.selectedPaths", "selected path count outside bound")
	}
	paths := make([]string, len(scope.SelectedPaths))
	for i, path := range scope.SelectedPaths {
		path = strings.TrimSpace(path)
		if !validPath(path) {
			return Scope{}, invalid("scope.selectedPaths", "invalid selected path")
		}
		paths[i] = path
	}
	sort.Strings(paths)
	for i := 1; i < len(paths); i++ {
		if paths[i] == paths[i-1] {
			return Scope{}, invalid("scope.selectedPaths", "duplicate selected path")
		}
	}
	scope.SelectedPaths = paths
	return scope, nil
}

func normalizeIdentity(identity RuntimeIdentity) RuntimeIdentity {
	identity.SourceKind = strings.ToLower(strings.TrimSpace(identity.SourceKind))
	identity.DriverID = strings.TrimSpace(identity.DriverID)
	identity.DriverVersion = strings.TrimSpace(identity.DriverVersion)
	identity.RuntimeID = strings.TrimSpace(identity.RuntimeID)
	identity.RuntimeVersion = strings.TrimSpace(identity.RuntimeVersion)
	identity.ModelID = strings.TrimSpace(identity.ModelID)
	identity.ModelRevision = strings.TrimSpace(identity.ModelRevision)
	identity.TokenizerID = strings.TrimSpace(identity.TokenizerID)
	identity.TokenizerRevision = strings.TrimSpace(identity.TokenizerRevision)
	return identity
}

func validateIdentity(identity RuntimeIdentity) error {
	if identity.SourceKind != SourceModelRuntime && identity.SourceKind != SourceNativeOllama {
		return invalid("identity.sourceKind", "only governed modelruntime or native Ollama drivers are supported")
	}
	for _, item := range []struct {
		field string
		value string
	}{
		{"driverId", identity.DriverID},
		{"driverVersion", identity.DriverVersion},
		{"runtimeId", identity.RuntimeID},
		{"runtimeVersion", identity.RuntimeVersion},
		{"modelId", identity.ModelID},
		{"modelRevision", identity.ModelRevision},
		{"tokenizerId", identity.TokenizerID},
		{"tokenizerRevision", identity.TokenizerRevision},
	} {
		if !validID(item.value) {
			return invalid("identity."+item.field, "required stable identity is missing or invalid")
		}
	}
	return nil
}

func normalizeProvenance(provenance Provenance) Provenance {
	provenance.ProvenanceID = strings.TrimSpace(provenance.ProvenanceID)
	provenance.ProposedBy = strings.TrimSpace(provenance.ProposedBy)
	provenance.Source = strings.TrimSpace(provenance.Source)
	provenance.RequestID = strings.TrimSpace(provenance.RequestID)
	provenance.CorrelationID = strings.TrimSpace(provenance.CorrelationID)
	provenance.TraceID = strings.TrimSpace(provenance.TraceID)
	provenance.AuditID = strings.TrimSpace(provenance.AuditID)
	return provenance
}

func validateProvenance(provenance Provenance) error {
	for _, item := range []struct {
		field string
		value string
	}{
		{"provenanceId", provenance.ProvenanceID},
		{"proposedBy", provenance.ProposedBy},
		{"source", provenance.Source},
		{"requestId", provenance.RequestID},
		{"correlationId", provenance.CorrelationID},
		{"traceId", provenance.TraceID},
		{"auditId", provenance.AuditID},
	} {
		if !validID(item.value) {
			return invalid("provenance."+item.field, "required provenance identity is missing or invalid")
		}
	}
	return nil
}

func normalizeGatewayEvidence(ref GatewayEvidenceRef) GatewayEvidenceRef {
	ref.InvocationID = strings.TrimSpace(ref.InvocationID)
	ref.ToolID = strings.TrimSpace(ref.ToolID)
	ref.State = strings.ToLower(strings.TrimSpace(ref.State))
	ref.AuditID = strings.TrimSpace(ref.AuditID)
	ref.RequestHash = strings.TrimSpace(ref.RequestHash)
	ref.ResultHash = strings.TrimSpace(ref.ResultHash)
	ref.WorkspaceID = strings.TrimSpace(ref.WorkspaceID)
	ref.LaneID = strings.TrimSpace(ref.LaneID)
	ref.CorrelationID = strings.TrimSpace(ref.CorrelationID)
	ref.TraceID = strings.TrimSpace(ref.TraceID)
	return ref
}

func validateGatewayEvidenceShape(ref GatewayEvidenceRef) error {
	for _, value := range []string{ref.InvocationID, ref.ToolID, ref.AuditID, ref.WorkspaceID, ref.LaneID, ref.CorrelationID, ref.TraceID} {
		if !validID(value) {
			return errors.New("required gateway identity is missing or invalid")
		}
	}
	if ref.State != "ok" && ref.State != "needs_approval" && ref.State != "denied" && ref.State != "error" {
		return errors.New("unsupported gateway state")
	}
	if !validHash(ref.RequestHash) || !validHash(ref.ResultHash) {
		return errors.New("gateway request and result commitments are required")
	}
	return nil
}

func verifyGatewayEvidence(in Input) (bool, []string) {
	if len(in.GatewayEvidence) == 0 {
		return false, nil
	}
	validCompletion := false
	reasons := []string{}
	for _, ref := range in.GatewayEvidence {
		if ref.WorkspaceID != in.Scope.WorkspaceID || ref.LaneID != in.Scope.LaneID {
			reasons = append(reasons, "gateway_evidence_scope_mismatch")
		}
		if ref.CorrelationID != in.Provenance.CorrelationID || ref.TraceID != in.Provenance.TraceID {
			reasons = append(reasons, "gateway_evidence_trace_mismatch")
		}
		if ref.State == "ok" && ref.WorkspaceID == in.Scope.WorkspaceID && ref.LaneID == in.Scope.LaneID && ref.CorrelationID == in.Provenance.CorrelationID && ref.TraceID == in.Provenance.TraceID {
			validCompletion = true
		}
	}
	return validCompletion, uniqueSorted(reasons)
}

func containsActionCompletionLanguage(content string) bool {
	normalized := " " + strings.ToLower(strings.Join(strings.Fields(content), " ")) + " "
	for _, phrase := range []string{
		" i deleted ", " i removed ", " i wrote ", " i created ", " i modified ",
		" i changed ", " i ran ", " i executed ", " i installed ", " i pushed ",
		" i committed ", " i shut down ", " i rebooted ", " was deleted ",
		" was written ", " was executed ",
	} {
		if strings.Contains(normalized, phrase) {
			return true
		}
	}
	return false
}

func validID(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" || len(value) > MaxIdentifierBytes || !utf8.ValidString(value) {
		return false
	}
	for _, r := range value {
		if unicode.IsControl(r) || unicode.IsSpace(r) {
			return false
		}
	}
	return true
}

func validPath(value string) bool {
	if value == "" || len(value) > MaxIdentifierBytes || !utf8.ValidString(value) || strings.ContainsRune(value, 0) {
		return false
	}
	for _, r := range value {
		if unicode.IsControl(r) {
			return false
		}
	}
	return true
}

func validHash(value string) bool {
	if len(value) != len("sha256:")+sha256.Size*2 || !strings.HasPrefix(value, "sha256:") {
		return false
	}
	decoded, err := hex.DecodeString(value[len("sha256:"):])
	return err == nil && len(decoded) == sha256.Size
}

func hashValue(value any) (string, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	return HashText(string(raw)), nil
}

func decisionDigest(decision Decision) (string, error) {
	decision.DecisionDigest = ""
	return hashValue(decision)
}

func uniqueSorted(values []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}
