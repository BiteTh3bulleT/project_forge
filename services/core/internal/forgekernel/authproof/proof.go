// Package authproof defines the production FORGE-K authorization evidence
// contract. It binds an authenticated actor, an immutable action-registry
// definition, a scoped capability grant, and an explicit approval decision to
// one semantic request.
//
// The package is deliberately pure: a production AuthorizationPort must load
// these records from trusted authority sources, while the Kernel independently
// validates their shape and binding. Hashes make persisted evidence
// tamper-evident; they do not turn caller-provided claims into authority.
package authproof

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"forge/projectforge/services/core/internal/aios/domain"
	"forge/projectforge/services/core/internal/forgekernel/court"
)

const (
	ProofVersion              = "forge_k.authorization_proof.v1"
	RequestFingerprintVersion = "forge_k.authorization_request.v1"
	BindingFingerprintVersion = "forge_k.authorization_binding.v1"
	RecordHashVersion         = "forge_k.authorization_record.v1"

	StatusActive      = "active"
	ApprovalApproved  = "approved"
	ApprovalNotNeeded = "not_required"

	MutationAlways           = "always"
	MutationNever            = "never"
	MutationRequestDependent = "request_dependent"
)

var (
	ErrInvalidRequest = errors.New("invalid authorization request")
	ErrInvalidProof   = errors.New("invalid authorization proof")
	ErrProofMismatch  = errors.New("authorization proof evidence mismatch")
)

// EvidenceError identifies the exact authorization evidence field that failed.
type EvidenceError struct {
	Cause error
	Field string
	Issue string
}

func (e *EvidenceError) Error() string {
	if e == nil {
		return ""
	}
	return fmt.Sprintf("%v: %s: %s", e.Cause, e.Field, e.Issue)
}

func (e *EvidenceError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

// PrincipalRecord is an attestation loaded from a trusted identity source. The
// credential fingerprint is an identifier/hash only; secret credential
// material must never enter the proof or journal.
type PrincipalRecord struct {
	RecordID              string              `json:"recordId"`
	Version               string              `json:"version"`
	SubjectID             string              `json:"subjectId"`
	SubjectKind           string              `json:"subjectKind"`
	Source                domain.ActionSource `json:"source"`
	Issuer                string              `json:"issuer"`
	CredentialFingerprint string              `json:"credentialFingerprint"`
	Status                string              `json:"status"`
	AuthenticatedAt       int64               `json:"authenticatedAt"`
	ExpiresAt             int64               `json:"expiresAt,omitempty"`
	RecordHash            string              `json:"recordHash"`
}

// RegistryRecord is the exact action definition the authority resolved.
type RegistryRecord struct {
	RecordID           string                    `json:"recordId"`
	Version            string                    `json:"version"`
	Authority          string                    `json:"authority"`
	Action             domain.SemanticActionType `json:"action"`
	Capability         string                    `json:"capability"`
	TargetObjectType   string                    `json:"targetObjectType"`
	Mutating           bool                      `json:"mutating"`
	MutationPolicy     string                    `json:"mutationPolicy"`
	AuthorizedMutating bool                      `json:"authorizedMutating"`
	SupportsDryRun     bool                      `json:"supportsDryRun"`
	ApprovalPossible   bool                      `json:"approvalPossible"`
	JournalEventType   string                    `json:"journalEventType"`
	RecordHash         string                    `json:"recordHash"`
}

// CapabilityRecord is one durable, actor-scoped authorization grant.
type CapabilityRecord struct {
	RecordID    string                    `json:"recordId"`
	Version     string                    `json:"version"`
	Authority   string                    `json:"authority"`
	SubjectID   string                    `json:"subjectId"`
	SubjectKind string                    `json:"subjectKind"`
	Source      domain.ActionSource       `json:"source"`
	Action      domain.SemanticActionType `json:"action"`
	Capability  string                    `json:"capability"`
	Scope       domain.ForgeScope         `json:"scope"`
	Status      string                    `json:"status"`
	GrantedAt   int64                     `json:"grantedAt"`
	ExpiresAt   int64                     `json:"expiresAt,omitempty"`
	RecordHash  string                    `json:"recordHash"`
}

// ApprovalRecord is explicit even when policy says approval is not required.
// Required approvals must name the durable request and decision records and
// bind the full retry-stable semantic request fingerprint.
type ApprovalRecord struct {
	PolicyRecordID     string `json:"policyRecordId"`
	PolicyVersion      string `json:"policyVersion"`
	Authority          string `json:"authority"`
	Required           bool   `json:"required"`
	Status             string `json:"status"`
	RequestID          string `json:"requestId,omitempty"`
	DecisionID         string `json:"decisionId,omitempty"`
	DecidedBy          string `json:"decidedBy,omitempty"`
	DecisionAt         int64  `json:"decisionAt,omitempty"`
	ExpiresAt          int64  `json:"expiresAt,omitempty"`
	RequestFingerprint string `json:"requestFingerprint"`
	RecordHash         string `json:"recordHash"`
}

// Proof is self-contained evidence persisted with an idempotency result. The
// authority port must re-resolve/verify the referenced records during replay.
type Proof struct {
	Version                  string           `json:"version"`
	EvidenceSnapshotID       string           `json:"evidenceSnapshotId"`
	RequestFingerprint       string           `json:"requestFingerprint"`
	AuthorizationFingerprint string           `json:"authorizationFingerprint"`
	ServicePrincipal         PrincipalRecord  `json:"servicePrincipal"`
	Origin                   *PrincipalRecord `json:"origin,omitempty"`
	Registry                 RegistryRecord   `json:"registry"`
	Capability               CapabilityRecord `json:"capability"`
	Approval                 ApprovalRecord   `json:"approval"`
}

// PlanBinding is the authorization-relevant portion of a prepared commit plan.
// It avoids coupling authproof to the commitproof package.
type PlanBinding struct {
	Action           domain.SemanticActionType
	Capability       string
	TargetObjectType string
	Mutating         bool
	JournalEventType string
}

// BuildProof normalizes authority-loaded records, calculates all record
// hashes, and binds the resulting proof to req.
func BuildProof(req domain.SyscallRequest, proof Proof) (Proof, error) {
	requestFingerprint, err := RequestFingerprint(req)
	if err != nil {
		return Proof{}, err
	}
	proof.Version = ProofVersion
	proof.EvidenceSnapshotID = strings.TrimSpace(proof.EvidenceSnapshotID)
	normalizeRecords(&proof)
	proof.RequestFingerprint = requestFingerprint
	proof.Approval.RequestFingerprint = requestFingerprint
	if err := validateSemantics(req, proof); err != nil {
		return Proof{}, err
	}
	proof.ServicePrincipal.RecordHash, err = hashRecord("service_principal", principalRecordBody(proof.ServicePrincipal))
	if err != nil {
		return Proof{}, err
	}
	if proof.Origin != nil {
		proof.Origin.RecordHash, err = hashRecord("origin", principalRecordBody(*proof.Origin))
		if err != nil {
			return Proof{}, err
		}
	}
	proof.Registry.RecordHash, err = hashRecord("registry", registryRecordBody(proof.Registry))
	if err != nil {
		return Proof{}, err
	}
	proof.Capability.RecordHash, err = hashRecord("capability", capabilityRecordBody(proof.Capability))
	if err != nil {
		return Proof{}, err
	}
	proof.Approval.RecordHash, err = hashRecord("approval", approvalRecordBody(proof.Approval))
	if err != nil {
		return Proof{}, err
	}
	proof.AuthorizationFingerprint, err = authorizationFingerprint(proof)
	if err != nil {
		return Proof{}, err
	}
	return proof, nil
}

// VerifyProof recomputes every derived field and fails closed on missing,
// inconsistent, expired-at-request-time, or tampered evidence.
func VerifyProof(req domain.SyscallRequest, proof Proof) error {
	if proof.Version != ProofVersion {
		return evidenceError(ErrInvalidProof, "proof.version", "unsupported or missing version")
	}
	for _, item := range []struct {
		field string
		value string
	}{
		{"proof.requestFingerprint", proof.RequestFingerprint},
		{"proof.authorizationFingerprint", proof.AuthorizationFingerprint},
		{"proof.servicePrincipal.recordHash", proof.ServicePrincipal.RecordHash},
		{"proof.registry.recordHash", proof.Registry.RecordHash},
		{"proof.capability.recordHash", proof.Capability.RecordHash},
		{"proof.approval.recordHash", proof.Approval.RecordHash},
	} {
		if !validDigest(item.value) {
			return evidenceError(ErrInvalidProof, item.field, "missing or malformed digest")
		}
	}
	if proof.Origin != nil && !validDigest(proof.Origin.RecordHash) {
		return evidenceError(ErrInvalidProof, "proof.origin.recordHash", "missing or malformed digest")
	}
	original := proof
	rebuilt, err := BuildProof(req, proof)
	if err != nil {
		return err
	}
	checks := []struct {
		field    string
		actual   string
		expected string
	}{
		{"proof.requestFingerprint", original.RequestFingerprint, rebuilt.RequestFingerprint},
		{"proof.authorizationFingerprint", original.AuthorizationFingerprint, rebuilt.AuthorizationFingerprint},
		{"proof.servicePrincipal.recordHash", original.ServicePrincipal.RecordHash, rebuilt.ServicePrincipal.RecordHash},
		{"proof.registry.recordHash", original.Registry.RecordHash, rebuilt.Registry.RecordHash},
		{"proof.capability.recordHash", original.Capability.RecordHash, rebuilt.Capability.RecordHash},
		{"proof.approval.recordHash", original.Approval.RecordHash, rebuilt.Approval.RecordHash},
	}
	for _, check := range checks {
		if check.actual != check.expected {
			return evidenceError(ErrProofMismatch, check.field, "does not match authoritative evidence and request")
		}
	}
	if (original.Origin == nil) != (rebuilt.Origin == nil) {
		return evidenceError(ErrProofMismatch, "proof.origin", "presence does not match authoritative evidence")
	}
	if original.Origin != nil && original.Origin.RecordHash != rebuilt.Origin.RecordHash {
		return evidenceError(ErrProofMismatch, "proof.origin.recordHash", "does not match authoritative evidence")
	}
	return nil
}

// VerifyPlanBinding prevents an adapter from preparing an action definition
// different from the one the Kernel authorized.
func VerifyPlanBinding(proof Proof, plan PlanBinding) error {
	checks := []struct {
		field string
		bad   bool
	}{
		{"plan.action", plan.Action != proof.Registry.Action},
		{"plan.capability", strings.TrimSpace(plan.Capability) != proof.Registry.Capability},
		{"plan.targetObjectType", strings.TrimSpace(plan.TargetObjectType) != proof.Registry.TargetObjectType},
		{"plan.mutating", plan.Mutating != proof.Registry.AuthorizedMutating},
		{"plan.journalEventType", strings.TrimSpace(plan.JournalEventType) != proof.Registry.JournalEventType},
	}
	for _, check := range checks {
		if check.bad {
			return evidenceError(ErrProofMismatch, check.field, "does not match authorized registry definition")
		}
	}
	return nil
}

// BindingFingerprint returns the retry-stable identity of the authority
// records. It is used to require a retry to present the same authenticated
// principal and authorization records as the stored original.
func BindingFingerprint(proof Proof) (string, error) {
	if !validDigest(proof.AuthorizationFingerprint) {
		return "", evidenceError(ErrInvalidProof, "proof.authorizationFingerprint", "missing or malformed digest")
	}
	return digest(BindingFingerprintVersion, struct {
		EvidenceSnapshotID       string `json:"evidenceSnapshotId"`
		ServicePrincipalRecordID string `json:"servicePrincipalRecordId"`
		ServicePrincipalHash     string `json:"servicePrincipalRecordHash"`
		OriginRecordID           string `json:"originRecordId,omitempty"`
		OriginRecordHash         string `json:"originRecordHash,omitempty"`
		RegistryRecordID         string `json:"registryRecordId"`
		RegistryRecordHash       string `json:"registryRecordHash"`
		CapabilityRecordID       string `json:"capabilityRecordId"`
		CapabilityHash           string `json:"capabilityRecordHash"`
		ApprovalRequestID        string `json:"approvalRequestId"`
		ApprovalDecisionID       string `json:"approvalDecisionId"`
		ApprovalRecordHash       string `json:"approvalRecordHash"`
	}{
		EvidenceSnapshotID:       proof.EvidenceSnapshotID,
		ServicePrincipalRecordID: proof.ServicePrincipal.RecordID,
		ServicePrincipalHash:     proof.ServicePrincipal.RecordHash,
		OriginRecordID:           principalRecordID(proof.Origin),
		OriginRecordHash:         principalRecordHash(proof.Origin),
		RegistryRecordID:         proof.Registry.RecordID,
		RegistryRecordHash:       proof.Registry.RecordHash,
		CapabilityRecordID:       proof.Capability.RecordID,
		CapabilityHash:           proof.Capability.RecordHash,
		ApprovalRequestID:        proof.Approval.RequestID,
		ApprovalDecisionID:       proof.Approval.DecisionID,
		ApprovalRecordHash:       proof.Approval.RecordHash,
	})
}

// SameAuthorization requires two proofs to resolve to the same authenticated
// principal, registry definition, grant, and approval decision/policy.
func SameAuthorization(a, b Proof) error {
	left, err := BindingFingerprint(a)
	if err != nil {
		return err
	}
	right, err := BindingFingerprint(b)
	if err != nil {
		return err
	}
	if left != right {
		return evidenceError(ErrProofMismatch, "proof.bindingFingerprint", "retry authorization differs from stored authorization")
	}
	return nil
}

// RequestFingerprint binds the semantic request while excluding retry-local
// transport identity, Kernel-derived metadata, and the durable approval row
// locator. The proof binds the typed approval request and decision records
// separately, avoiding a circular fingerprint while preserving authority.
func RequestFingerprint(req domain.SyscallRequest) (string, error) {
	if strings.TrimSpace(string(req.Action)) == "" {
		return "", evidenceError(ErrInvalidRequest, "request.action", "is required")
	}
	if strings.TrimSpace(req.Actor.ID) == "" || strings.TrimSpace(req.Actor.Kind) == "" {
		return "", evidenceError(ErrInvalidRequest, "request.actor", "id and kind are required")
	}
	if strings.TrimSpace(string(req.Source)) == "" {
		return "", evidenceError(ErrInvalidRequest, "request.source", "is required")
	}
	if strings.TrimSpace(req.Scope.WorkspaceID) == "" {
		return "", evidenceError(ErrInvalidRequest, "request.scope.workspaceId", "is required")
	}
	if req.RequestedAt <= 0 {
		return "", evidenceError(ErrInvalidRequest, "request.requestedAt", "must be positive")
	}
	if strings.TrimSpace(req.Provenance.Actor) == "" || strings.TrimSpace(req.Provenance.ActorType) == "" {
		return "", evidenceError(ErrInvalidRequest, "request.provenance", "actor and actorType are required")
	}
	metadata := make(map[string]any, len(req.Metadata))
	for key, value := range req.Metadata {
		switch key {
		case court.MetadataDecisionKey, "forgeKIngressAuthority", "kernelAuthorityOwner", "durableCommitAdapter", "forgeKAuthorizationProof",
			"approvalRequestId", "approvalId":
			continue
		default:
			metadata[key] = value
		}
	}
	return digest(RequestFingerprintVersion, struct {
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
		req.Action, req.Actor, req.Source, req.Scope, req.Payload,
		domain.Provenance{Actor: req.Provenance.Actor, ActorType: req.Provenance.ActorType, Source: req.Provenance.Source},
		strings.TrimSpace(req.IdempotencyKey), req.DryRun, strings.TrimSpace(req.RequiredCapability),
		append([]string(nil), req.CapabilityHints...), metadata,
	})
}

func validateSemantics(req domain.SyscallRequest, proof Proof) error {
	if proof.EvidenceSnapshotID == "" {
		return evidenceError(ErrInvalidProof, "proof.evidenceSnapshotId", "is required")
	}
	if proof.ServicePrincipal.RecordID == "" || proof.ServicePrincipal.Version == "" || proof.ServicePrincipal.Issuer == "" {
		return evidenceError(ErrInvalidProof, "proof.servicePrincipal", "record id, version, and issuer are required")
	}
	if !validDigest(proof.ServicePrincipal.CredentialFingerprint) {
		return evidenceError(ErrInvalidProof, "proof.servicePrincipal.credentialFingerprint", "missing or malformed digest")
	}
	if proof.ServicePrincipal.Status != StatusActive {
		return evidenceError(ErrInvalidProof, "proof.servicePrincipal.status", "service principal record is not active")
	}
	if proof.ServicePrincipal.SubjectID == "" || proof.ServicePrincipal.SubjectKind == "" {
		return evidenceError(ErrInvalidProof, "proof.servicePrincipal.subject", "id and kind are required")
	}
	if strings.TrimSpace(req.Provenance.Actor) != strings.TrimSpace(req.Actor.ID) {
		return evidenceError(ErrProofMismatch, "request.provenance.actor", "does not match logical request actor")
	}
	if err := validAt("proof.servicePrincipal", proof.ServicePrincipal.AuthenticatedAt, proof.ServicePrincipal.ExpiresAt, req.RequestedAt); err != nil {
		return err
	}
	authorityPrincipal := proof.ServicePrincipal
	requiresOrigin := req.Source == domain.SourceUser || req.Source == domain.SourceAdapter || req.Source == domain.SourceFutureIRIS
	if requiresOrigin {
		if proof.Origin == nil {
			return evidenceError(ErrInvalidProof, "proof.origin", "authenticated origin is required for caller/proposer source")
		}
		if err := validateOrigin(req, *proof.Origin); err != nil {
			return err
		}
		authorityPrincipal = *proof.Origin
		if strings.TrimSpace(req.Provenance.ActorType) != strings.TrimSpace(req.Actor.Kind) {
			return evidenceError(ErrProofMismatch, "request.provenance.actorType", "origin-backed actor taxonomy does not match authenticated actor kind")
		}
	} else if proof.Origin != nil {
		if err := validateOrigin(req, *proof.Origin); err != nil {
			return err
		}
	}
	if proof.Registry.RecordID == "" || proof.Registry.Version == "" || proof.Registry.Authority == "" {
		return evidenceError(ErrInvalidProof, "proof.registry", "record id, version, and authority are required")
	}
	if proof.Registry.Action != req.Action || proof.Registry.Capability == "" || proof.Registry.TargetObjectType == "" || proof.Registry.JournalEventType == "" {
		return evidenceError(ErrProofMismatch, "proof.registry", "does not define the requested action")
	}
	expectedMutating, err := authorizedMutating(req, proof.Registry)
	if err != nil {
		return err
	}
	if proof.Registry.AuthorizedMutating != expectedMutating {
		return evidenceError(ErrProofMismatch, "proof.registry.authorizedMutating", "does not match deterministic mutation policy")
	}
	if req.DryRun && !proof.Registry.SupportsDryRun {
		return evidenceError(ErrProofMismatch, "proof.registry.supportsDryRun", "requested dry-run is not authorized")
	}
	if required := strings.TrimSpace(req.RequiredCapability); required != "" && required != proof.Registry.Capability {
		return evidenceError(ErrProofMismatch, "request.requiredCapability", "does not match registry capability")
	}
	if proof.Capability.RecordID == "" || proof.Capability.Version == "" || proof.Capability.Authority == "" {
		return evidenceError(ErrInvalidProof, "proof.capability", "record id, version, and authority are required")
	}
	if proof.Capability.Status != StatusActive {
		return evidenceError(ErrInvalidProof, "proof.capability.status", "capability grant is not active")
	}
	if proof.Capability.SubjectID != authorityPrincipal.SubjectID || proof.Capability.SubjectKind != authorityPrincipal.SubjectKind || proof.Capability.Source != req.Source {
		return evidenceError(ErrProofMismatch, "proof.capability.subject", "does not match authenticated authority principal")
	}
	if proof.Capability.Action != req.Action || proof.Capability.Capability != proof.Registry.Capability {
		return evidenceError(ErrProofMismatch, "proof.capability", "does not grant the requested action capability")
	}
	if !equalScope(proof.Capability.Scope, req.Scope) {
		return evidenceError(ErrProofMismatch, "proof.capability.scope", "does not exactly match request scope")
	}
	if err := validAt("proof.capability", proof.Capability.GrantedAt, proof.Capability.ExpiresAt, req.RequestedAt); err != nil {
		return err
	}
	if proof.Approval.PolicyRecordID == "" || proof.Approval.PolicyVersion == "" || proof.Approval.Authority == "" {
		return evidenceError(ErrInvalidProof, "proof.approval", "policy record id, version, and authority are required")
	}
	if proof.Approval.Required && !proof.Registry.ApprovalPossible {
		return evidenceError(ErrProofMismatch, "proof.approval.required", "registry does not permit approval for this action")
	}
	if proof.Registry.Mutating && (req.Source == domain.SourceAdapter || req.Source == domain.SourceFutureIRIS) && !proof.Approval.Required {
		return evidenceError(ErrProofMismatch, "proof.approval.required", "mutating proposer-source actions require approval")
	}
	if proof.Approval.Required {
		if proof.Approval.Status != ApprovalApproved {
			return evidenceError(ErrInvalidProof, "proof.approval.status", "required approval is not approved")
		}
		if proof.Approval.RequestID == "" || proof.Approval.DecisionID == "" || proof.Approval.DecidedBy == "" {
			return evidenceError(ErrInvalidProof, "proof.approval", "durable request, decision, and decision actor are required")
		}
		if proof.Approval.DecidedBy == authorityPrincipal.SubjectID || proof.Approval.DecidedBy == strings.TrimSpace(req.Actor.ID) {
			return evidenceError(ErrProofMismatch, "proof.approval.decidedBy", "requesting actor cannot approve its own mutation")
		}
		if err := validAt("proof.approval", proof.Approval.DecisionAt, proof.Approval.ExpiresAt, req.RequestedAt); err != nil {
			return err
		}
	} else {
		if proof.Approval.Status != ApprovalNotNeeded {
			return evidenceError(ErrInvalidProof, "proof.approval.status", "non-required approval needs explicit not_required evidence")
		}
		if proof.Approval.RequestID != "" || proof.Approval.DecisionID != "" || proof.Approval.DecidedBy != "" || proof.Approval.DecisionAt != 0 || proof.Approval.ExpiresAt != 0 {
			return evidenceError(ErrProofMismatch, "proof.approval", "not_required evidence cannot carry a decision")
		}
	}
	if proof.Approval.RequestFingerprint != proof.RequestFingerprint {
		return evidenceError(ErrProofMismatch, "proof.approval.requestFingerprint", "does not match semantic request")
	}
	return nil
}

func validAt(field string, issuedAt, expiresAt, requestedAt int64) error {
	if issuedAt <= 0 || issuedAt > requestedAt {
		return evidenceError(ErrInvalidProof, field+".issuedAt", "must be positive and no later than request time")
	}
	if expiresAt != 0 && expiresAt < requestedAt {
		return evidenceError(ErrInvalidProof, field+".expiresAt", "evidence expired before request time")
	}
	return nil
}

func normalizeRecords(proof *Proof) {
	normalizePrincipal(&proof.ServicePrincipal)
	if proof.Origin != nil {
		normalizePrincipal(proof.Origin)
	}
	proof.Registry.RecordID = strings.TrimSpace(proof.Registry.RecordID)
	proof.Registry.Version = strings.TrimSpace(proof.Registry.Version)
	proof.Registry.Authority = strings.TrimSpace(proof.Registry.Authority)
	proof.Registry.Capability = strings.TrimSpace(proof.Registry.Capability)
	proof.Registry.TargetObjectType = strings.TrimSpace(proof.Registry.TargetObjectType)
	proof.Registry.JournalEventType = strings.TrimSpace(proof.Registry.JournalEventType)
	proof.Registry.MutationPolicy = strings.ToLower(strings.TrimSpace(proof.Registry.MutationPolicy))
	proof.Capability.RecordID = strings.TrimSpace(proof.Capability.RecordID)
	proof.Capability.Version = strings.TrimSpace(proof.Capability.Version)
	proof.Capability.Authority = strings.TrimSpace(proof.Capability.Authority)
	proof.Capability.SubjectID = strings.TrimSpace(proof.Capability.SubjectID)
	proof.Capability.SubjectKind = strings.TrimSpace(proof.Capability.SubjectKind)
	proof.Capability.Capability = strings.TrimSpace(proof.Capability.Capability)
	proof.Capability.Status = strings.ToLower(strings.TrimSpace(proof.Capability.Status))
	proof.Approval.PolicyRecordID = strings.TrimSpace(proof.Approval.PolicyRecordID)
	proof.Approval.PolicyVersion = strings.TrimSpace(proof.Approval.PolicyVersion)
	proof.Approval.Authority = strings.TrimSpace(proof.Approval.Authority)
	proof.Approval.Status = strings.ToLower(strings.TrimSpace(proof.Approval.Status))
	proof.Approval.RequestID = strings.TrimSpace(proof.Approval.RequestID)
	proof.Approval.DecisionID = strings.TrimSpace(proof.Approval.DecisionID)
	proof.Approval.DecidedBy = strings.TrimSpace(proof.Approval.DecidedBy)
	proof.Approval.RequestFingerprint = strings.TrimSpace(proof.Approval.RequestFingerprint)
}

func principalRecordBody(record PrincipalRecord) any {
	record.RecordHash = ""
	return record
}

func normalizePrincipal(record *PrincipalRecord) {
	record.RecordID = strings.TrimSpace(record.RecordID)
	record.Version = strings.TrimSpace(record.Version)
	record.SubjectID = strings.TrimSpace(record.SubjectID)
	record.SubjectKind = strings.TrimSpace(record.SubjectKind)
	record.Issuer = strings.TrimSpace(record.Issuer)
	record.CredentialFingerprint = strings.TrimSpace(record.CredentialFingerprint)
	record.Status = strings.ToLower(strings.TrimSpace(record.Status))
}

func validateOrigin(req domain.SyscallRequest, origin PrincipalRecord) error {
	if origin.RecordID == "" || origin.Version == "" || origin.Issuer == "" {
		return evidenceError(ErrInvalidProof, "proof.origin", "record id, version, and issuer are required")
	}
	if !validDigest(origin.CredentialFingerprint) {
		return evidenceError(ErrInvalidProof, "proof.origin.credentialFingerprint", "missing or malformed digest")
	}
	if origin.Status != StatusActive {
		return evidenceError(ErrInvalidProof, "proof.origin.status", "origin record is not active")
	}
	if origin.SubjectID != strings.TrimSpace(req.Actor.ID) || origin.SubjectKind != strings.TrimSpace(req.Actor.Kind) || origin.Source != req.Source {
		return evidenceError(ErrProofMismatch, "proof.origin", "does not match request actor and source")
	}
	return validAt("proof.origin", origin.AuthenticatedAt, origin.ExpiresAt, req.RequestedAt)
}

func authorizedMutating(req domain.SyscallRequest, registry RegistryRecord) (bool, error) {
	switch registry.MutationPolicy {
	case MutationAlways:
		if !registry.Mutating {
			return false, evidenceError(ErrInvalidProof, "proof.registry.mutating", "always mutation policy requires mutating action definition")
		}
		return true, nil
	case MutationNever:
		if registry.Mutating {
			return false, evidenceError(ErrInvalidProof, "proof.registry.mutating", "never mutation policy conflicts with mutating action definition")
		}
		return false, nil
	case MutationRequestDependent:
		if registry.Mutating {
			return false, evidenceError(ErrInvalidProof, "proof.registry.mutating", "request-dependent policy must not claim unconditional mutation")
		}
		if req.Action != domain.ActionCompileContext {
			return false, evidenceError(ErrInvalidProof, "proof.registry.mutationPolicy", "request-dependent mutation is unsupported for this action")
		}
		return compileContextPersists(req.Payload), nil
	default:
		return false, evidenceError(ErrInvalidProof, "proof.registry.mutationPolicy", "unsupported or missing mutation policy")
	}
}

func compileContextPersists(payload map[string]any) bool {
	persist := false
	apply := func(values map[string]any) {
		if values == nil {
			return
		}
		if value, ok := values["persistSnapshot"].(bool); ok {
			persist = value
		}
	}
	apply(payload)
	if nested, ok := payload["restoreSnapshot"].(map[string]any); ok {
		apply(nested)
	}
	if nested, ok := payload["compileOptions"].(map[string]any); ok {
		apply(nested)
	}
	return persist
}

func principalRecordID(record *PrincipalRecord) string {
	if record == nil {
		return ""
	}
	return record.RecordID
}

func principalRecordHash(record *PrincipalRecord) string {
	if record == nil {
		return ""
	}
	return record.RecordHash
}

func registryRecordBody(record RegistryRecord) any {
	record.RecordHash = ""
	return record
}

func capabilityRecordBody(record CapabilityRecord) any {
	record.RecordHash = ""
	return record
}

func approvalRecordBody(record ApprovalRecord) any {
	record.RecordHash = ""
	return record
}

func authorizationFingerprint(proof Proof) (string, error) {
	proof.AuthorizationFingerprint = ""
	return digest(ProofVersion+".body", proof)
}

func hashRecord(kind string, value any) (string, error) {
	return digest(RecordHashVersion+"."+kind, value)
}

func digest(namespace string, value any) (string, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return "", evidenceError(ErrInvalidProof, "canonical_json", err.Error())
	}
	sum := sha256.Sum256(append([]byte(namespace+"\n"), raw...))
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

func validDigest(value string) bool {
	if !strings.HasPrefix(value, "sha256:") || len(value) != len("sha256:")+sha256.Size*2 {
		return false
	}
	_, err := hex.DecodeString(strings.TrimPrefix(value, "sha256:"))
	return err == nil
}

func equalScope(a, b domain.ForgeScope) bool {
	if a.WorkspaceID != b.WorkspaceID || a.LaneID != b.LaneID || len(a.SelectedPaths) != len(b.SelectedPaths) {
		return false
	}
	for i := range a.SelectedPaths {
		if a.SelectedPaths[i] != b.SelectedPaths[i] {
			return false
		}
	}
	return true
}

func evidenceError(cause error, field, issue string) error {
	return &EvidenceError{Cause: cause, Field: field, Issue: issue}
}
