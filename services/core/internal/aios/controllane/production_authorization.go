package controllane

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"forge/projectforge/services/core/internal/aios/domain"
	"forge/projectforge/services/core/internal/forgekernel"
	"forge/projectforge/services/core/internal/forgekernel/authproof"
)

const (
	productionRegistryVersion   = "forge_k.production_action_registry.v1"
	productionCapabilityVersion = "forge_k.production_capability_policy.v1"
	productionApprovalVersion   = "forge_k.production_approval_policy.v1"
	forgeCorePrincipalRecordID  = "service_principal:forge.core"
)

var (
	ErrInvalidServicePrincipal = errors.New("invalid forge.core service principal")
	ErrAuthorizationDenied     = errors.New("production authorization denied")
	ErrApprovalProofRequired   = errors.New("durable approval proof required")
)

// ForgeCoreServicePrincipal is deliberately opaque. Production construction
// creates exactly one instance at daemon assembly and passes that instance to
// the authorization service; request envelopes cannot manufacture it.
type ForgeCoreServicePrincipal struct {
	record authproof.PrincipalRecord
	guard  *byte
}

func NewForgeCoreServicePrincipal() *ForgeCoreServicePrincipal {
	return &ForgeCoreServicePrincipal{
		guard: new(byte),
		record: authproof.PrincipalRecord{
			RecordID: forgeCorePrincipalRecordID, Version: "forge.core.service_identity.v1",
			SubjectID: "forge.core", SubjectKind: "service", Source: domain.SourceSystem,
			Issuer:                "forge.core.bootstrap",
			CredentialFingerprint: authproof.CredentialFingerprint("forge.core.service_principal.v1"),
			Status:                authproof.StatusActive, AuthenticatedAt: 1,
		},
	}
}

type ProductionAuthorizationOptions struct {
	Registry         ActionRegistry
	DB               *sql.DB
	ServicePrincipal *ForgeCoreServicePrincipal
}

// ProductionAuthorizationService resolves evidence independently of the
// legacy StaticCapabilityService and StaticApprovalGate used by legacy_v1.
type ProductionAuthorizationService struct {
	registry  ActionRegistry
	db        *sql.DB
	principal *ForgeCoreServicePrincipal
}

var _ forgekernel.AuthorizationPort = (*ProductionAuthorizationService)(nil)

func NewProductionAuthorizationService(opts ProductionAuthorizationOptions) (*ProductionAuthorizationService, error) {
	if opts.Registry == nil || opts.DB == nil || !validForgeCorePrincipal(opts.ServicePrincipal) {
		return nil, ErrInvalidServicePrincipal
	}
	return &ProductionAuthorizationService{registry: opts.Registry, db: opts.DB, principal: opts.ServicePrincipal}, nil
}

func validForgeCorePrincipal(principal *ForgeCoreServicePrincipal) bool {
	if principal == nil || principal.guard == nil {
		return false
	}
	r := principal.record
	return r.RecordID == forgeCorePrincipalRecordID && r.SubjectID == "forge.core" && r.SubjectKind == "service" &&
		r.Source == domain.SourceSystem && r.Issuer == "forge.core.bootstrap" && r.Status == authproof.StatusActive &&
		strings.HasPrefix(r.CredentialFingerprint, "sha256:")
}

func (s *ProductionAuthorizationService) ResolveAuthorization(ctx context.Context, req domain.SyscallRequest) (authproof.Proof, error) {
	if s == nil || !validForgeCorePrincipal(s.principal) || s.registry == nil || s.db == nil {
		return authproof.Proof{}, ErrInvalidServicePrincipal
	}
	def, ok := s.registry.Get(req.Action)
	if !ok {
		return authproof.Proof{}, fmt.Errorf("%w: unsupported action %q", ErrAuthorizationDenied, req.Action)
	}
	requestFingerprint, err := authproof.RequestFingerprint(req)
	if err != nil {
		return authproof.Proof{}, err
	}
	mutationPolicy, authorizedMutating := productionMutationPolicy(req, def)
	proof := authproof.Proof{
		EvidenceSnapshotID: "authorization:" + strings.TrimPrefix(requestFingerprint, "sha256:"),
		ServicePrincipal:   s.principal.record,
		Registry: authproof.RegistryRecord{
			RecordID: "action_definition:" + string(req.Action), Version: productionRegistryVersion,
			Authority: "forge_k.production_registry", Action: req.Action,
			Capability: def.Capability, TargetObjectType: def.TargetObjectType,
			Mutating: def.Mutating, MutationPolicy: mutationPolicy, AuthorizedMutating: authorizedMutating,
			SupportsDryRun: def.SupportsDryRun, ApprovalPossible: def.ApprovalPossible,
			JournalEventType: "semantic_syscall." + strings.ToLower(string(req.Action)),
		},
		Approval: authproof.ApprovalRecord{
			PolicyRecordID: "approval_policy:" + string(req.Source), PolicyVersion: productionApprovalVersion,
			Authority: "forge.approvals.sqlite", Status: authproof.ApprovalNotNeeded,
		},
	}

	authoritySubject := proof.ServicePrincipal
	if requiresAuthenticatedOrigin(req.Source) {
		origin, present := authproof.TrustedOriginFromContext(ctx)
		if !present {
			return authproof.Proof{}, fmt.Errorf("%w: authenticated origin missing for source %q", ErrAuthorizationDenied, req.Source)
		}
		proof.Origin = &origin
		authoritySubject = origin
	} else if req.Source != domain.SourceSystem && req.Source != domain.SourceInternal {
		return authproof.Proof{}, fmt.Errorf("%w: source %q is not a production principal class", ErrAuthorizationDenied, req.Source)
	}

	policyID, allowed := productionCapabilityPolicy(req.Source, req.Action)
	if !allowed {
		return authproof.Proof{}, fmt.Errorf("%w: no explicit capability grant for source %q action %q", ErrAuthorizationDenied, req.Source, req.Action)
	}
	proof.Capability = authproof.CapabilityRecord{
		RecordID: "capability_grant:" + policyID + ":" + strings.TrimPrefix(requestFingerprint, "sha256:"),
		Version:  productionCapabilityVersion, Authority: "forge_k.production_capabilities",
		SubjectID: authoritySubject.SubjectID, SubjectKind: authoritySubject.SubjectKind,
		Source: req.Source, Action: req.Action, Capability: def.Capability, Scope: req.Scope,
		Status: authproof.StatusActive, GrantedAt: 1,
	}

	if authorizedMutating && (req.Source == domain.SourceAdapter || req.Source == domain.SourceFutureIRIS) {
		approval, approvalErr := s.resolveDurableApproval(ctx, req, requestFingerprint)
		if approvalErr != nil {
			return authproof.Proof{}, approvalErr
		}
		proof.Approval = approval
	}
	return authproof.BuildProof(req, proof)
}

func requiresAuthenticatedOrigin(source domain.ActionSource) bool {
	return source == domain.SourceUser || source == domain.SourceAdapter || source == domain.SourceFutureIRIS
}

func productionMutationPolicy(req domain.SyscallRequest, def ActionDefinition) (string, bool) {
	if def.Mutating {
		return authproof.MutationAlways, true
	}
	if req.Action == domain.ActionCompileContext {
		return authproof.MutationRequestDependent, mergeCompileContextOptions(req.Payload).PersistSnapshot
	}
	return authproof.MutationNever, false
}

func productionCapabilityPolicy(source domain.ActionSource, action domain.SemanticActionType) (string, bool) {
	switch source {
	case domain.SourceUser:
		switch action {
		case domain.ActionMaterializeAdmittedEvidence, domain.ActionReviseMemoryEvidence:
			return "authenticated_user_scoped_admitted_evidence", true
		case domain.ActionRebuildMemoryAcceleration:
			return "authenticated_user_scoped_acceleration_rebuild", true
		case domain.ActionRecordRetrievalUsefulness, domain.ActionRecordRestoreOutcomeFeedback:
			return "authenticated_user_utility_evidence", true
		default:
			return "authenticated_user", true
		}
	case domain.SourceSystem, domain.SourceInternal:
		return "forge_core_service", true
	case domain.SourceAdapter:
		switch action {
		case domain.ActionCreateNote, domain.ActionCreateLink, domain.ActionCompileContext,
			domain.ActionValidateKVIdentity, domain.ActionValidateRefShape, domain.ActionCompareRefShape,
			domain.ActionValidateSourceObject, domain.ActionValidateSemanticOperation,
			domain.ActionValidateAdmissionCandidate, domain.ActionValidateContextAttribution:
			return "bounded_adapter_proposal", true
		}
	case domain.SourceFutureIRIS:
		switch action {
		case domain.ActionCreateNote, domain.ActionCreateLink, domain.ActionRegisterContradict,
			domain.ActionDeriveModel, domain.ActionCompileContext:
			return "bounded_iris_proposal", true
		}
	}
	return "", false
}

func (s *ProductionAuthorizationService) resolveDurableApproval(ctx context.Context, req domain.SyscallRequest, requestFingerprint string) (authproof.ApprovalRecord, error) {
	requestID, ok := approvalRequestID(req.Metadata)
	if !ok {
		return authproof.ApprovalRecord{}, fmt.Errorf("%w: approvalRequestId metadata missing", ErrApprovalProofRequired)
	}
	row := s.db.QueryRowContext(ctx, `
SELECT ar.status,ar.requested_action,ar.scope_snapshot_json,ar.expires_at,
       ad.id,ad.created_at,ad.actor,ad.decision
FROM approval_requests ar
JOIN approval_decisions ad ON ad.request_id=ar.id
WHERE ar.id=?
ORDER BY ad.id DESC LIMIT 1`, requestID)
	var status, requestedAction, scopeJSON, decidedBy, decision string
	var expiresAt, decisionID, decisionAt int64
	if err := row.Scan(&status, &requestedAction, &scopeJSON, &expiresAt, &decisionID, &decisionAt, &decidedBy, &decision); err != nil {
		return authproof.ApprovalRecord{}, fmt.Errorf("%w: lookup request %d: %v", ErrApprovalProofRequired, requestID, err)
	}
	if status != "resolved" || decision != authproof.ApprovalApproved || decisionAt <= 0 || decisionAt > req.RequestedAt || (expiresAt > 0 && expiresAt < req.RequestedAt) {
		return authproof.ApprovalRecord{}, fmt.Errorf("%w: request %d is not a valid current approval", ErrApprovalProofRequired, requestID)
	}
	expectedAction := "semantic_syscall." + strings.ToLower(string(req.Action))
	if requestedAction != string(req.Action) && requestedAction != expectedAction {
		return authproof.ApprovalRecord{}, fmt.Errorf("%w: request %d action mismatch", ErrApprovalProofRequired, requestID)
	}
	var scope map[string]any
	if err := json.Unmarshal([]byte(scopeJSON), &scope); err != nil {
		return authproof.ApprovalRecord{}, fmt.Errorf("%w: request %d scope evidence invalid", ErrApprovalProofRequired, requestID)
	}
	storedFingerprint, _ := scope["authorizationRequestFingerprint"].(string)
	if storedFingerprint != requestFingerprint {
		return authproof.ApprovalRecord{}, fmt.Errorf("%w: request %d fingerprint mismatch", ErrApprovalProofRequired, requestID)
	}
	return authproof.ApprovalRecord{
		PolicyRecordID: "approval_policy:proposal_mutation", PolicyVersion: productionApprovalVersion,
		Authority: "forge.approvals.sqlite", Required: true, Status: authproof.ApprovalApproved,
		RequestID: strconv.FormatInt(requestID, 10), DecisionID: strconv.FormatInt(decisionID, 10),
		DecidedBy: decidedBy, DecisionAt: decisionAt, ExpiresAt: expiresAt,
	}, nil
}

func approvalRequestID(metadata map[string]any) (int64, bool) {
	for _, key := range []string{"approvalRequestId", "approvalId"} {
		value, present := metadata[key]
		if !present {
			continue
		}
		switch typed := value.(type) {
		case int:
			return int64(typed), typed > 0
		case int64:
			return typed, typed > 0
		case float64:
			return int64(typed), typed > 0 && typed == float64(int64(typed))
		case string:
			id, err := strconv.ParseInt(strings.TrimSpace(typed), 10, 64)
			return id, err == nil && id > 0
		}
	}
	return 0, false
}
