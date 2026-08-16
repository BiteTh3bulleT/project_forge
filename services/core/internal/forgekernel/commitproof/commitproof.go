// Package commitproof defines the production FORGE-K commit-integrity
// contract. It seals the exact semantic request and prepared mutation plan,
// then validates the durable evidence returned by a commit adapter.
//
// This package deliberately has no dependency on the FORGE-K simulator or on
// a concrete store. It is a pure deterministic boundary suitable for use by
// the live kernel before and after one atomic durable commit.
package commitproof

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"forge/projectforge/services/core/internal/aios/domain"
	"forge/projectforge/services/core/internal/forgekernel/court"
	forgejournal "forge/projectforge/services/core/internal/forgekernel/journal"
	"forge/projectforge/services/core/internal/forgekernel/semanticdiff"
)

const (
	RequestFingerprintVersion = "forge_k.request_fingerprint.v1"
	PreparedPlanVersion       = "forge_k.prepared_plan.v1"
	CommitReceiptVersion      = "forge_k.commit_receipt.v1"
	IdempotencyVersion        = "forge_k.idempotency_fingerprint.v1"
	JournalSource             = "forge_kernel"
	LegacyCommittedBy         = "forge_kernel"
	DefaultDurableAdapter     = "aios.controllane.sqlite"
)

var (
	ErrInvalidRequest      = errors.New("invalid commit-proof request")
	ErrInvalidPreparedPlan = errors.New("invalid prepared commit plan")
	ErrSealMismatch        = errors.New("prepared commit plan seal mismatch")
	ErrInvalidReceipt      = errors.New("invalid commit receipt")
	ErrReceiptMismatch     = errors.New("commit receipt evidence mismatch")
)

// EvidenceError identifies the precise proof field that failed validation.
// Cause supports errors.Is without forcing callers to parse error text.
type EvidenceError struct {
	Cause error
	Field string
	Issue string
}

func (e *EvidenceError) Error() string {
	if e == nil {
		return ""
	}
	if strings.TrimSpace(e.Field) == "" {
		return fmt.Sprintf("%v: %s", e.Cause, e.Issue)
	}
	return fmt.Sprintf("%v: %s: %s", e.Cause, e.Field, e.Issue)
}

func (e *EvidenceError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

// PreparedPlan is the deterministic mutation intent produced after policy and
// admission checks but before the durable transaction begins. Expected IDs are
// sealed as sets: their ordering is not semantically significant.
type PreparedPlan struct {
	Action                        domain.SemanticActionType `json:"action"`
	Capability                    string                    `json:"capability"`
	TargetObjectType              string                    `json:"targetObjectType"`
	Mutating                      bool                      `json:"mutating"`
	JournalEventType              string                    `json:"journalEventType"`
	ExpectedObjectIDs             []string                  `json:"expectedObjectIds"`
	ExpectedProvenanceIDs         []string                  `json:"expectedProvenanceIds"`
	ExpectedTransactionID         string                    `json:"expectedTransactionId"`
	ExpectedJournalEventID        string                    `json:"expectedJournalEventId"`
	ExpectedAuditOutboxID         string                    `json:"expectedAuditOutboxId"`
	ExpectedJournalSource         string                    `json:"expectedJournalSource"`
	ExpectedJournalPayloadHash    string                    `json:"expectedJournalPayloadHash"`
	ExpectedJournalProvenanceHash string                    `json:"expectedJournalProvenanceHash"`
	ExpectedJournalMetadataHash   string                    `json:"expectedJournalMetadataHash"`
	ExpectedJournalCommittedBy    string                    `json:"expectedJournalCommittedBy"`
	Details                       map[string]any            `json:"details,omitempty"`
}

// PreparedPlanSeal binds a normalized semantic request to its prepared plan.
// The seal detects any mutation between preparation and commit.
type PreparedPlanSeal struct {
	Version            string `json:"version"`
	RequestFingerprint string `json:"requestFingerprint"`
	PlanFingerprint    string `json:"planFingerprint"`
	SealDigest         string `json:"sealDigest"`
}

// CommitReceipt is the typed evidence that one durable transaction must
// return. AuditOutboxID refers to an audit record staged atomically with the
// canonical mutation; delivery/observation may occur after commit.
type CommitReceipt struct {
	Version                string             `json:"version"`
	RequestFingerprint     string             `json:"requestFingerprint"`
	PreparedPlanSeal       string             `json:"preparedPlanSeal"`
	TransactionID          string             `json:"transactionId"`
	JournalEventID         string             `json:"journalEventId"`
	JournalEventHash       string             `json:"journalEventHash"`
	ObjectIDs              []string           `json:"objectIds"`
	ProvenanceIDs          []string           `json:"provenanceIds"`
	AuditOutboxID          string             `json:"auditOutboxId"`
	IdempotencyFingerprint string             `json:"idempotencyFingerprint"`
	JournalEntry           forgejournal.Entry `json:"journalEntry"`
}

// BindPreparedPlan fills the production-derived portion of a plan after all
// deterministic policy layers (including the Courthouse) have finalized the
// request. Existing derived fields must either match or be empty, so rebinding
// a persisted/tampered plan cannot silently repair it.
func BindPreparedPlan(req domain.SyscallRequest, plan PreparedPlan) (PreparedPlan, error) {
	if _, err := FingerprintRequest(req); err != nil {
		return PreparedPlan{}, err
	}
	plan.Action = domain.SemanticActionType(strings.TrimSpace(string(plan.Action)))
	plan.Capability = strings.TrimSpace(plan.Capability)
	plan.TargetObjectType = strings.TrimSpace(plan.TargetObjectType)
	plan.JournalEventType = strings.TrimSpace(plan.JournalEventType)
	if plan.Action == "" || plan.Action != req.Action || plan.Capability == "" || plan.TargetObjectType == "" || plan.JournalEventType == "" {
		return PreparedPlan{}, evidenceError(ErrInvalidPreparedPlan, "plan", "action, capability, target object type, and journal event type must match the request")
	}
	var err error
	plan.ExpectedObjectIDs, err = normalizedIDs(plan.ExpectedObjectIDs, "plan.expectedObjectIds", false, ErrInvalidPreparedPlan)
	if err != nil {
		return PreparedPlan{}, err
	}
	plan.ExpectedProvenanceIDs, err = normalizedIDs(plan.ExpectedProvenanceIDs, "plan.expectedProvenanceIds", true, ErrInvalidPreparedPlan)
	if err != nil {
		return PreparedPlan{}, err
	}
	if len(plan.ExpectedProvenanceIDs) != 1 {
		return PreparedPlan{}, evidenceError(ErrInvalidPreparedPlan, "plan.expectedProvenanceIds", "exactly one journal provenance id is required")
	}

	expectedJournalID := req.ID + ":journal_event"
	if plan.Mutating && !containsExact(plan.ExpectedObjectIDs, expectedJournalID) {
		plan.ExpectedObjectIDs = append(plan.ExpectedObjectIDs, expectedJournalID)
		sort.Strings(plan.ExpectedObjectIDs)
	}
	if !plan.Mutating && containsExact(plan.ExpectedObjectIDs, expectedJournalID) {
		return PreparedPlan{}, evidenceError(ErrInvalidPreparedPlan, "plan.expectedObjectIds", "nonmutating plans must expose the journal only through the receipt")
	}

	derived := PreparedPlan{
		ExpectedTransactionID:      req.ID + ":transaction",
		ExpectedJournalEventID:     expectedJournalID,
		ExpectedAuditOutboxID:      req.ID + ":audit_outbox",
		ExpectedJournalSource:      JournalSource,
		ExpectedJournalCommittedBy: requestCommittedBy(req),
	}
	for _, check := range []struct {
		field    string
		existing string
		expected string
	}{
		{"plan.expectedTransactionId", plan.ExpectedTransactionID, derived.ExpectedTransactionID},
		{"plan.expectedJournalEventId", plan.ExpectedJournalEventID, derived.ExpectedJournalEventID},
		{"plan.expectedAuditOutboxId", plan.ExpectedAuditOutboxID, derived.ExpectedAuditOutboxID},
		{"plan.expectedJournalSource", plan.ExpectedJournalSource, derived.ExpectedJournalSource},
		{"plan.expectedJournalCommittedBy", plan.ExpectedJournalCommittedBy, derived.ExpectedJournalCommittedBy},
	} {
		if existing := strings.TrimSpace(check.existing); existing != "" && existing != check.expected {
			return PreparedPlan{}, evidenceError(ErrInvalidPreparedPlan, check.field, "conflicts with deterministic request binding")
		}
	}
	plan.ExpectedTransactionID = derived.ExpectedTransactionID
	plan.ExpectedJournalEventID = derived.ExpectedJournalEventID
	plan.ExpectedAuditOutboxID = derived.ExpectedAuditOutboxID
	plan.ExpectedJournalSource = derived.ExpectedJournalSource
	plan.ExpectedJournalCommittedBy = derived.ExpectedJournalCommittedBy

	payload, err := journalPayload(req, plan)
	if err != nil {
		return PreparedPlan{}, err
	}
	payloadHash, err := hashJSONValue(payload)
	if err != nil {
		return PreparedPlan{}, err
	}
	provenanceHash, err := hashJSONValue(BuildJournalProvenance(req))
	if err != nil {
		return PreparedPlan{}, err
	}
	metadataHash, err := hashJSONValue(BuildJournalMetadata())
	if err != nil {
		return PreparedPlan{}, err
	}
	for _, check := range []struct {
		field    string
		existing string
		expected string
	}{
		{"plan.expectedJournalPayloadHash", plan.ExpectedJournalPayloadHash, payloadHash},
		{"plan.expectedJournalProvenanceHash", plan.ExpectedJournalProvenanceHash, provenanceHash},
		{"plan.expectedJournalMetadataHash", plan.ExpectedJournalMetadataHash, metadataHash},
	} {
		if existing := strings.TrimSpace(check.existing); existing != "" && existing != check.expected {
			return PreparedPlan{}, evidenceError(ErrInvalidPreparedPlan, check.field, "conflicts with deterministic journal content")
		}
	}
	plan.ExpectedJournalPayloadHash = payloadHash
	plan.ExpectedJournalProvenanceHash = provenanceHash
	plan.ExpectedJournalMetadataHash = metadataHash
	return normalizePlan(req, plan)
}

// BuildJournalPayload returns the exact Kernel-owned journal payload. The
// prepared-plan seal is intentionally absent because the plan seals this
// payload hash; including it would introduce a circular hash dependency.
func BuildJournalPayload(req domain.SyscallRequest, plan PreparedPlan) (map[string]any, error) {
	normalized, err := normalizePlan(req, plan)
	if err != nil {
		return nil, err
	}
	return journalPayload(req, normalized)
}

func journalPayload(req domain.SyscallRequest, plan PreparedPlan) (map[string]any, error) {
	requestFingerprint, err := FingerprintRequest(req)
	if err != nil {
		return nil, err
	}
	semanticIDs := make([]string, 0, len(plan.ExpectedObjectIDs))
	for _, id := range plan.ExpectedObjectIDs {
		if id != plan.ExpectedJournalEventID {
			semanticIDs = append(semanticIDs, id)
		}
	}
	return map[string]any{
		"action":               req.Action,
		"committedObjectIds":   semanticIDs,
		"dryRun":               false,
		"kernelAuthorityOwner": plan.ExpectedJournalCommittedBy,
		"durableCommitAdapter": requestMetadataString(req, "durableCommitAdapter", DefaultDurableAdapter),
		"requestFingerprint":   requestFingerprint,
		"transactionId":        plan.ExpectedTransactionID,
		"auditOutboxId":        plan.ExpectedAuditOutboxID,
	}, nil
}

// BuildJournalProvenance mirrors the normalized provenance persisted by the
// journal adapter.
func BuildJournalProvenance(req domain.SyscallRequest) domain.Provenance {
	provenance := req.Provenance
	if strings.TrimSpace(provenance.Source) == "" {
		provenance.Source = string(req.Source)
	}
	if strings.TrimSpace(provenance.TraceID) == "" {
		provenance.TraceID = req.TraceID
	}
	return provenance
}

// BuildJournalMetadata is the immutable v1 journal metadata document.
func BuildJournalMetadata() map[string]any { return map[string]any{} }

// FingerprintRequest hashes the complete semantic syscall envelope, including
// action, actor, scope, payload, provenance, capability, and metadata. JSON map
// keys are deterministically ordered by encoding/json.
func FingerprintRequest(req domain.SyscallRequest) (string, error) {
	if strings.TrimSpace(req.ID) == "" {
		return "", evidenceError(ErrInvalidRequest, "id", "is required")
	}
	if strings.TrimSpace(string(req.Action)) == "" {
		return "", evidenceError(ErrInvalidRequest, "action", "is required")
	}
	if strings.TrimSpace(req.Actor.ID) == "" || strings.TrimSpace(req.Actor.Kind) == "" {
		return "", evidenceError(ErrInvalidRequest, "actor", "id and kind are required")
	}
	if strings.TrimSpace(req.Scope.WorkspaceID) == "" {
		return "", evidenceError(ErrInvalidRequest, "scope.workspaceId", "is required")
	}
	if strings.TrimSpace(req.Provenance.Actor) == "" || strings.TrimSpace(req.Provenance.ActorType) == "" {
		return "", evidenceError(ErrInvalidRequest, "provenance", "actor and actorType are required")
	}
	return digest(RequestFingerprintVersion, req)
}

// SealPreparedPlan validates and seals a prepared plan against the exact
// request that was authorized.
func SealPreparedPlan(req domain.SyscallRequest, plan PreparedPlan) (PreparedPlanSeal, error) {
	requestFingerprint, err := FingerprintRequest(req)
	if err != nil {
		return PreparedPlanSeal{}, err
	}
	normalized, err := normalizePlan(req, plan)
	if err != nil {
		return PreparedPlanSeal{}, err
	}
	planFingerprint, err := digest(PreparedPlanVersion+".body", normalized)
	if err != nil {
		return PreparedPlanSeal{}, err
	}
	sealDigest, err := digest(PreparedPlanVersion+".seal", struct {
		Version            string `json:"version"`
		RequestFingerprint string `json:"requestFingerprint"`
		PlanFingerprint    string `json:"planFingerprint"`
	}{PreparedPlanVersion, requestFingerprint, planFingerprint})
	if err != nil {
		return PreparedPlanSeal{}, err
	}
	return PreparedPlanSeal{
		Version:            PreparedPlanVersion,
		RequestFingerprint: requestFingerprint,
		PlanFingerprint:    planFingerprint,
		SealDigest:         sealDigest,
	}, nil
}

// VerifyPreparedPlan recomputes every component of the seal. Any request,
// plan, version, or digest mutation fails closed.
func VerifyPreparedPlan(req domain.SyscallRequest, plan PreparedPlan, seal PreparedPlanSeal) error {
	if seal.Version != PreparedPlanVersion {
		return evidenceError(ErrSealMismatch, "seal.version", "unsupported or missing version")
	}
	if !validDigest(seal.RequestFingerprint) {
		return evidenceError(ErrSealMismatch, "seal.requestFingerprint", "missing or malformed digest")
	}
	if !validDigest(seal.PlanFingerprint) {
		return evidenceError(ErrSealMismatch, "seal.planFingerprint", "missing or malformed digest")
	}
	if !validDigest(seal.SealDigest) {
		return evidenceError(ErrSealMismatch, "seal.sealDigest", "missing or malformed digest")
	}
	expected, err := SealPreparedPlan(req, plan)
	if err != nil {
		return err
	}
	if seal.RequestFingerprint != expected.RequestFingerprint {
		return evidenceError(ErrSealMismatch, "seal.requestFingerprint", "does not match request")
	}
	if seal.PlanFingerprint != expected.PlanFingerprint {
		return evidenceError(ErrSealMismatch, "seal.planFingerprint", "does not match prepared plan")
	}
	if seal.SealDigest != expected.SealDigest {
		return evidenceError(ErrSealMismatch, "seal.sealDigest", "does not bind request and plan")
	}
	return nil
}

// IdempotencyFingerprint binds the caller's idempotency key (including an
// intentionally empty key) to the semantic request while excluding only
// retry-local transport identity and deterministic Kernel decision metadata.
func IdempotencyFingerprint(req domain.SyscallRequest) (string, error) {
	if _, err := FingerprintRequest(req); err != nil {
		return "", err
	}
	// Kernel decisions are deterministic production-derived metadata added
	// after Prepare. They remain bound by the prepared-plan seal, but must not
	// turn an otherwise identical retry into a different idempotency identity.
	// Request-local transport identity is also excluded so a retry may carry a
	// fresh request/correlation/trace envelope. No other semantic field or
	// metadata key is excluded.
	metadata := make(map[string]any, len(req.Metadata))
	for key, value := range req.Metadata {
		if key != court.MetadataDecisionKey && key != semanticdiff.MetadataDecisionKey {
			metadata[key] = value
		}
	}
	return digest(IdempotencyVersion, struct {
		Action             domain.SemanticActionType `json:"action"`
		Actor              domain.ActorIdentity      `json:"actor"`
		Source             domain.ActionSource       `json:"source"`
		Scope              domain.ForgeScope         `json:"scope"`
		Payload            map[string]any            `json:"payload"`
		Provenance         domain.Provenance         `json:"provenance"`
		IdempotencyKey     string                    `json:"idempotencyKey"`
		DryRun             bool                      `json:"dryRun"`
		RequiredCapability string                    `json:"requiredCapability"`
		CapabilityHints    []string                  `json:"capabilityHints"`
		Metadata           map[string]any            `json:"metadata"`
	}{
		Action: req.Action, Actor: req.Actor, Source: req.Source, Scope: req.Scope,
		Payload: req.Payload,
		Provenance: domain.Provenance{
			Actor: req.Provenance.Actor, ActorType: req.Provenance.ActorType, Source: req.Provenance.Source,
		},
		IdempotencyKey: strings.TrimSpace(req.IdempotencyKey), DryRun: req.DryRun,
		RequiredCapability: strings.TrimSpace(req.RequiredCapability),
		CapabilityHints:    append([]string(nil), req.CapabilityHints...), Metadata: metadata,
	})
}

// ValidateCommitReceipt verifies the prepared-plan seal and then requires a
// complete, internally consistent receipt. It never treats partial commit
// evidence as success.
func ValidateCommitReceipt(req domain.SyscallRequest, plan PreparedPlan, seal PreparedPlanSeal, receipt CommitReceipt, result domain.SyscallResult) error {
	if err := VerifyPreparedPlan(req, plan, seal); err != nil {
		return err
	}
	for _, mismatch := range []struct {
		field string
		bad   bool
	}{
		{"result.success", !result.Success},
		{"result.action", result.Action != req.Action},
		{"result.requestId", result.RequestID != req.ID},
		{"result.correlationId", result.CorrelationID != req.CorrelationID},
		{"result.traceId", result.TraceID != req.TraceID},
		{"result.idempotencyKey", result.IdempotencyKey != req.IdempotencyKey},
		{"result.dryRun", result.DryRun != req.DryRun},
	} {
		if mismatch.bad {
			return evidenceError(ErrReceiptMismatch, mismatch.field, "does not match sealed request")
		}
	}
	if receipt.Version != CommitReceiptVersion {
		return evidenceError(ErrInvalidReceipt, "receipt.version", "unsupported or missing version")
	}
	requiredDigests := []struct {
		field string
		value string
	}{
		{"receipt.requestFingerprint", receipt.RequestFingerprint},
		{"receipt.preparedPlanSeal", receipt.PreparedPlanSeal},
		{"receipt.journalEventHash", receipt.JournalEventHash},
		{"receipt.idempotencyFingerprint", receipt.IdempotencyFingerprint},
	}
	for _, item := range requiredDigests {
		if !validDigest(item.value) {
			return evidenceError(ErrInvalidReceipt, item.field, "missing or malformed digest")
		}
	}
	for _, item := range []struct {
		field string
		value string
	}{
		{"receipt.transactionId", receipt.TransactionID},
		{"receipt.journalEventId", receipt.JournalEventID},
		{"receipt.auditOutboxId", receipt.AuditOutboxID},
	} {
		if strings.TrimSpace(item.value) == "" {
			return evidenceError(ErrInvalidReceipt, item.field, "is required")
		}
	}

	normalized, err := normalizePlan(req, plan)
	if err != nil {
		return err
	}
	objects, err := normalizedIDs(receipt.ObjectIDs, "receipt.objectIds", normalized.Mutating, ErrInvalidReceipt)
	if err != nil {
		return err
	}
	provenance, err := normalizedIDs(receipt.ProvenanceIDs, "receipt.provenanceIds", true, ErrInvalidReceipt)
	if err != nil {
		return err
	}

	if receipt.RequestFingerprint != seal.RequestFingerprint {
		return evidenceError(ErrReceiptMismatch, "receipt.requestFingerprint", "does not match sealed request")
	}
	if receipt.PreparedPlanSeal != seal.SealDigest {
		return evidenceError(ErrReceiptMismatch, "receipt.preparedPlanSeal", "does not match prepared plan seal")
	}
	expectedIdempotency, err := IdempotencyFingerprint(req)
	if err != nil {
		return err
	}
	if receipt.IdempotencyFingerprint != expectedIdempotency {
		return evidenceError(ErrReceiptMismatch, "receipt.idempotencyFingerprint", "does not match request")
	}
	for _, check := range []struct {
		field    string
		actual   string
		expected string
	}{
		{"receipt.transactionId", receipt.TransactionID, normalized.ExpectedTransactionID},
		{"receipt.journalEventId", receipt.JournalEventID, normalized.ExpectedJournalEventID},
		{"receipt.auditOutboxId", receipt.AuditOutboxID, normalized.ExpectedAuditOutboxID},
	} {
		if check.actual != check.expected {
			return evidenceError(ErrReceiptMismatch, check.field, "does not match prepared plan")
		}
	}
	if err := validateJournalEntry(req, normalized, receipt); err != nil {
		return err
	}
	if !equalStrings(objects, normalized.ExpectedObjectIDs) {
		return evidenceError(ErrReceiptMismatch, "receipt.objectIds", "do not match prepared plan")
	}
	reportedObjects, err := normalizedIDs(result.CommittedObjectIDs, "result.committedObjectIds", normalized.Mutating, ErrReceiptMismatch)
	if err != nil {
		return err
	}
	if !equalStrings(reportedObjects, objects) {
		return evidenceError(ErrReceiptMismatch, "result.committedObjectIds", "do not match commit receipt")
	}
	if !equalStrings(provenance, normalized.ExpectedProvenanceIDs) {
		return evidenceError(ErrReceiptMismatch, "receipt.provenanceIds", "do not match prepared plan")
	}
	return nil
}

func normalizePlan(req domain.SyscallRequest, plan PreparedPlan) (PreparedPlan, error) {
	plan.Action = domain.SemanticActionType(strings.TrimSpace(string(plan.Action)))
	plan.Capability = strings.TrimSpace(plan.Capability)
	plan.TargetObjectType = strings.TrimSpace(plan.TargetObjectType)
	plan.JournalEventType = strings.TrimSpace(plan.JournalEventType)
	plan.ExpectedTransactionID = strings.TrimSpace(plan.ExpectedTransactionID)
	plan.ExpectedJournalEventID = strings.TrimSpace(plan.ExpectedJournalEventID)
	plan.ExpectedAuditOutboxID = strings.TrimSpace(plan.ExpectedAuditOutboxID)
	plan.ExpectedJournalSource = strings.TrimSpace(plan.ExpectedJournalSource)
	plan.ExpectedJournalPayloadHash = strings.TrimSpace(plan.ExpectedJournalPayloadHash)
	plan.ExpectedJournalProvenanceHash = strings.TrimSpace(plan.ExpectedJournalProvenanceHash)
	plan.ExpectedJournalMetadataHash = strings.TrimSpace(plan.ExpectedJournalMetadataHash)
	plan.ExpectedJournalCommittedBy = strings.TrimSpace(plan.ExpectedJournalCommittedBy)
	if plan.Action == "" {
		return PreparedPlan{}, evidenceError(ErrInvalidPreparedPlan, "plan.action", "is required")
	}
	if plan.Action != req.Action {
		return PreparedPlan{}, evidenceError(ErrInvalidPreparedPlan, "plan.action", "does not match request")
	}
	if plan.Capability == "" {
		return PreparedPlan{}, evidenceError(ErrInvalidPreparedPlan, "plan.capability", "is required")
	}
	if plan.TargetObjectType == "" {
		return PreparedPlan{}, evidenceError(ErrInvalidPreparedPlan, "plan.targetObjectType", "is required")
	}
	if plan.JournalEventType == "" {
		return PreparedPlan{}, evidenceError(ErrInvalidPreparedPlan, "plan.journalEventType", "is required")
	}
	var err error
	plan.ExpectedObjectIDs, err = normalizedIDs(plan.ExpectedObjectIDs, "plan.expectedObjectIds", plan.Mutating, ErrInvalidPreparedPlan)
	if err != nil {
		return PreparedPlan{}, err
	}
	plan.ExpectedProvenanceIDs, err = normalizedIDs(plan.ExpectedProvenanceIDs, "plan.expectedProvenanceIds", true, ErrInvalidPreparedPlan)
	if err != nil {
		return PreparedPlan{}, err
	}
	if len(plan.ExpectedProvenanceIDs) != 1 {
		return PreparedPlan{}, evidenceError(ErrInvalidPreparedPlan, "plan.expectedProvenanceIds", "exactly one journal provenance id is required")
	}
	for _, check := range []struct {
		field    string
		actual   string
		expected string
	}{
		{"plan.expectedTransactionId", plan.ExpectedTransactionID, req.ID + ":transaction"},
		{"plan.expectedJournalEventId", plan.ExpectedJournalEventID, req.ID + ":journal_event"},
		{"plan.expectedAuditOutboxId", plan.ExpectedAuditOutboxID, req.ID + ":audit_outbox"},
		{"plan.expectedJournalSource", plan.ExpectedJournalSource, JournalSource},
		{"plan.expectedJournalCommittedBy", plan.ExpectedJournalCommittedBy, requestCommittedBy(req)},
	} {
		if check.actual == "" || check.actual != check.expected {
			return PreparedPlan{}, evidenceError(ErrInvalidPreparedPlan, check.field, "is missing or inconsistent with the request")
		}
	}
	for _, check := range []struct {
		field string
		value string
	}{
		{"plan.expectedJournalPayloadHash", plan.ExpectedJournalPayloadHash},
		{"plan.expectedJournalProvenanceHash", plan.ExpectedJournalProvenanceHash},
		{"plan.expectedJournalMetadataHash", plan.ExpectedJournalMetadataHash},
	} {
		if !validDigest(check.value) {
			return PreparedPlan{}, evidenceError(ErrInvalidPreparedPlan, check.field, "missing or malformed digest")
		}
	}
	expectedPayload, err := journalPayload(req, plan)
	if err != nil {
		return PreparedPlan{}, err
	}
	expectedPayloadHash, err := hashJSONValue(expectedPayload)
	if err != nil {
		return PreparedPlan{}, err
	}
	expectedProvenanceHash, err := hashJSONValue(BuildJournalProvenance(req))
	if err != nil {
		return PreparedPlan{}, err
	}
	expectedMetadataHash, err := hashJSONValue(BuildJournalMetadata())
	if err != nil {
		return PreparedPlan{}, err
	}
	for _, check := range []struct {
		field    string
		actual   string
		expected string
	}{
		{"plan.expectedJournalPayloadHash", plan.ExpectedJournalPayloadHash, expectedPayloadHash},
		{"plan.expectedJournalProvenanceHash", plan.ExpectedJournalProvenanceHash, expectedProvenanceHash},
		{"plan.expectedJournalMetadataHash", plan.ExpectedJournalMetadataHash, expectedMetadataHash},
	} {
		if check.actual != check.expected {
			return PreparedPlan{}, evidenceError(ErrInvalidPreparedPlan, check.field, "does not match deterministic journal content")
		}
	}
	if plan.Mutating && !containsExact(plan.ExpectedObjectIDs, plan.ExpectedJournalEventID) {
		return PreparedPlan{}, evidenceError(ErrInvalidPreparedPlan, "plan.expectedObjectIds", "mutating plan must include its journal event id")
	}
	if !plan.Mutating && containsExact(plan.ExpectedObjectIDs, plan.ExpectedJournalEventID) {
		return PreparedPlan{}, evidenceError(ErrInvalidPreparedPlan, "plan.expectedObjectIds", "nonmutating plan must not report its journal as a semantic object")
	}
	return plan, nil
}

func validateJournalEntry(req domain.SyscallRequest, plan PreparedPlan, receipt CommitReceipt) error {
	entry := receipt.JournalEntry
	recomputed, err := forgejournal.HashEntry(entry)
	if err != nil {
		return evidenceError(ErrInvalidReceipt, "receipt.journalEntry", err.Error())
	}
	if entry.Hash != recomputed || receipt.JournalEventHash != recomputed {
		return evidenceError(ErrReceiptMismatch, "receipt.journalEventHash", "does not match canonical journal entry")
	}
	expectedTrace := BuildJournalProvenance(req).TraceID
	checks := []struct {
		field    string
		actual   string
		expected string
	}{
		{"receipt.journalEntry.schemaVersion", entry.SchemaVersion, forgejournal.SchemaVersion},
		{"receipt.journalEntry.eventId", entry.EventID, plan.ExpectedJournalEventID},
		{"receipt.journalEntry.eventType", entry.EventType, plan.JournalEventType},
		{"receipt.journalEntry.source", entry.Source, plan.ExpectedJournalSource},
		{"receipt.journalEntry.actor", entry.Actor, req.Provenance.Actor},
		{"receipt.journalEntry.workspaceId", entry.WorkspaceID, req.Scope.WorkspaceID},
		{"receipt.journalEntry.laneId", entry.LaneID, req.Scope.LaneID},
		{"receipt.journalEntry.correlationId", entry.CorrelationID, req.CorrelationID},
		{"receipt.journalEntry.traceId", entry.TraceID, expectedTrace},
		{"receipt.journalEntry.provenanceId", entry.ProvenanceID, plan.ExpectedProvenanceIDs[0]},
		{"receipt.journalEntry.provenanceHash", entry.ProvenanceHash, plan.ExpectedJournalProvenanceHash},
		{"receipt.journalEntry.payloadHash", entry.PayloadHash, plan.ExpectedJournalPayloadHash},
		{"receipt.journalEntry.metadataHash", entry.MetadataHash, plan.ExpectedJournalMetadataHash},
		{"receipt.journalEntry.proposedBy", entry.ProposedBy, string(req.Source)},
		{"receipt.journalEntry.committedBy", entry.CommittedBy, plan.ExpectedJournalCommittedBy},
		{"receipt.journalEntry.syscallId", entry.SyscallID, req.ID},
		{"receipt.journalEntry.auditId", entry.AuditID, ""},
	}
	for _, check := range checks {
		if check.actual != check.expected {
			return evidenceError(ErrReceiptMismatch, check.field, "does not match sealed request or plan")
		}
	}
	if entry.CreatedAt != req.RequestedAt {
		return evidenceError(ErrReceiptMismatch, "receipt.journalEntry.createdAt", "does not match sealed request")
	}
	if !equalStringsExact(entry.SelectedPaths, req.Scope.SelectedPaths) {
		return evidenceError(ErrReceiptMismatch, "receipt.journalEntry.selectedPaths", "do not match sealed request order")
	}
	return nil
}

func normalizedIDs(ids []string, field string, required bool, cause error) ([]string, error) {
	if required && len(ids) == 0 {
		return nil, evidenceError(cause, field, "at least one id is required")
	}
	out := make([]string, len(ids))
	seen := make(map[string]struct{}, len(ids))
	for i, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			return nil, evidenceError(cause, field, "contains an empty id")
		}
		if _, exists := seen[id]; exists {
			return nil, evidenceError(cause, field, "contains duplicate ids")
		}
		seen[id] = struct{}{}
		out[i] = id
	}
	sort.Strings(out)
	return out, nil
}

func digest(namespace string, value any) (string, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return "", evidenceError(ErrInvalidRequest, "canonical_json", err.Error())
	}
	sum := sha256.Sum256(append([]byte(namespace+"\n"), encoded...))
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

func validDigest(value string) bool {
	if !strings.HasPrefix(value, "sha256:") || len(value) != len("sha256:")+sha256.Size*2 {
		return false
	}
	_, err := hex.DecodeString(strings.TrimPrefix(value, "sha256:"))
	return err == nil
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func equalStringsExact(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func containsExact(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func requestCommittedBy(req domain.SyscallRequest) string {
	return requestMetadataString(req, "kernelAuthorityOwner", LegacyCommittedBy)
}

func requestMetadataString(req domain.SyscallRequest, key, fallback string) string {
	if value, ok := req.Metadata[key].(string); ok && strings.TrimSpace(value) != "" {
		return strings.TrimSpace(value)
	}
	return fallback
}

func hashJSONValue(value any) (string, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return "", evidenceError(ErrInvalidPreparedPlan, "journal_content", err.Error())
	}
	hash, err := forgejournal.HashJSON(raw)
	if err != nil {
		return "", evidenceError(ErrInvalidPreparedPlan, "journal_content", err.Error())
	}
	return hash, nil
}

func evidenceError(cause error, field, issue string) error {
	return &EvidenceError{Cause: cause, Field: field, Issue: issue}
}
