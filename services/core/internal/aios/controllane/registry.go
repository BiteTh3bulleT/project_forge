package controllane

import (
	"fmt"
	"sort"

	"forge/projectforge/services/core/internal/aios/domain"
)

const (
	CapMemoryNoteCreate           = "memory.note.create"
	CapMemoryNoteArchive          = "memory.note.archive"
	CapMemoryLinkCreate           = "memory.link.create"
	CapStateUpdate                = "state.update"
	CapLoopOpen                   = "loop.open"
	CapLoopClose                  = "loop.close"
	CapMemoryContradictionReg     = "memory.contradiction.register"
	CapMemorySupersessionMark     = "memory.supersession.mark"
	CapModelDerive                = "model.derive"
	CapContextCompile             = "context.compile"
	CapKVIdentityValidate         = "kv.identity.validate"
	CapRefShapeValidate           = "ref.shape.validate"
	CapRefShapeCompare            = "ref.shape.compare"
	CapSourceObjectValidate       = "source.object.authority.validate"
	CapSemanticOperationValidate  = "semantic.operation.validate"
	CapAdmissionCandidateValidate = "admission.candidate.validate"
	CapContextAttributionValidate = "context.attribution.validate"
	CapEvidenceAdmit              = "court.evidence.admit"
	CapRulingAppeal               = "court.ruling.appeal"
	CapRetrievalEvidenceRecord    = "retrieval.evidence.record"
	CapMemoryAccelerationRebuild  = "memory.acceleration.rebuild"
	CapRetrievalUsefulnessRecord  = "retrieval.usefulness.record"
	CapRestoreOutcomeFeedback     = "context.restore.outcome.feedback.record"
	CapMemoryEvidenceMaterialize  = "memory.evidence.materialize"
	CapMemoryEvidenceRevise       = "memory.evidence.revise"
	CapSemanticDiffCompute        = "semantic.diff.compute"
)

type ActionDefinition struct {
	Action           domain.SemanticActionType `json:"action"`
	Capability       string                    `json:"capability"`
	Mutating         bool                      `json:"mutating"`
	SupportsDryRun   bool                      `json:"supportsDryRun"`
	ApprovalPossible bool                      `json:"approvalPossible"`
	TargetObjectType string                    `json:"targetObjectType"`
	AuditEventName   string                    `json:"auditEventName"`
}

type ActionRegistry interface {
	Get(action domain.SemanticActionType) (ActionDefinition, bool)
	List() []ActionDefinition
}

type StaticActionRegistry struct {
	definitions map[domain.SemanticActionType]ActionDefinition
}

func NewStaticActionRegistry() *StaticActionRegistry {
	defs := map[domain.SemanticActionType]ActionDefinition{
		domain.ActionCreateNote: {
			Action:           domain.ActionCreateNote,
			Capability:       CapMemoryNoteCreate,
			Mutating:         true,
			SupportsDryRun:   true,
			ApprovalPossible: true,
			TargetObjectType: "memory_note",
			AuditEventName:   "semantic_syscall.create_note",
		},
		domain.ActionCreateLink: {
			Action:           domain.ActionCreateLink,
			Capability:       CapMemoryLinkCreate,
			Mutating:         true,
			SupportsDryRun:   true,
			ApprovalPossible: true,
			TargetObjectType: "semantic_link",
			AuditEventName:   "semantic_syscall.create_link",
		},
		domain.ActionUpdateState: {
			Action:           domain.ActionUpdateState,
			Capability:       CapStateUpdate,
			Mutating:         true,
			SupportsDryRun:   true,
			ApprovalPossible: true,
			TargetObjectType: "state_item",
			AuditEventName:   "semantic_syscall.update_state",
		},
		domain.ActionOpenLoop: {
			Action:           domain.ActionOpenLoop,
			Capability:       CapLoopOpen,
			Mutating:         true,
			SupportsDryRun:   true,
			ApprovalPossible: true,
			TargetObjectType: "open_loop",
			AuditEventName:   "semantic_syscall.open_loop",
		},
		domain.ActionCloseLoop: {
			Action:           domain.ActionCloseLoop,
			Capability:       CapLoopClose,
			Mutating:         true,
			SupportsDryRun:   true,
			ApprovalPossible: true,
			TargetObjectType: "open_loop",
			AuditEventName:   "semantic_syscall.close_loop",
		},
		domain.ActionMarkSuperseded: {
			Action:           domain.ActionMarkSuperseded,
			Capability:       CapMemorySupersessionMark,
			Mutating:         true,
			SupportsDryRun:   true,
			ApprovalPossible: true,
			TargetObjectType: "supersession",
			AuditEventName:   "semantic_syscall.mark_superseded",
		},
		domain.ActionRegisterContradict: {
			Action:           domain.ActionRegisterContradict,
			Capability:       CapMemoryContradictionReg,
			Mutating:         true,
			SupportsDryRun:   true,
			ApprovalPossible: true,
			TargetObjectType: "contradiction",
			AuditEventName:   "semantic_syscall.register_contradiction",
		},
		domain.ActionDeriveModel: {
			Action:           domain.ActionDeriveModel,
			Capability:       CapModelDerive,
			Mutating:         true,
			SupportsDryRun:   true,
			ApprovalPossible: true,
			TargetObjectType: "derived_model",
			AuditEventName:   "semantic_syscall.derive_model",
		},
		domain.ActionArchiveNote: {
			Action:           domain.ActionArchiveNote,
			Capability:       CapMemoryNoteArchive,
			Mutating:         true,
			SupportsDryRun:   true,
			ApprovalPossible: true,
			TargetObjectType: "memory_note",
			AuditEventName:   "semantic_syscall.archive_note",
		},
		domain.ActionCompileContext: {
			Action:           domain.ActionCompileContext,
			Capability:       CapContextCompile,
			Mutating:         false,
			SupportsDryRun:   true,
			ApprovalPossible: false,
			TargetObjectType: "context_packet",
			AuditEventName:   "semantic_syscall.compile_context",
		},
		domain.ActionValidateKVIdentity: {
			Action:           domain.ActionValidateKVIdentity,
			Capability:       CapKVIdentityValidate,
			Mutating:         false,
			SupportsDryRun:   true,
			ApprovalPossible: false,
			TargetObjectType: "kv_identity_validation",
			AuditEventName:   "semantic_syscall.validate_kv_identity",
		},
		domain.ActionValidateRefShape: {
			Action:           domain.ActionValidateRefShape,
			Capability:       CapRefShapeValidate,
			Mutating:         false,
			SupportsDryRun:   true,
			ApprovalPossible: false,
			TargetObjectType: "ref_shape_validation",
			AuditEventName:   "semantic_syscall.validate_ref_shape",
		},
		domain.ActionCompareRefShape: {
			Action:           domain.ActionCompareRefShape,
			Capability:       CapRefShapeCompare,
			Mutating:         false,
			SupportsDryRun:   true,
			ApprovalPossible: false,
			TargetObjectType: "ref_shape_comparison",
			AuditEventName:   "semantic_syscall.compare_ref_shape",
		},
		domain.ActionValidateSourceObject: {
			Action:           domain.ActionValidateSourceObject,
			Capability:       CapSourceObjectValidate,
			Mutating:         false,
			SupportsDryRun:   true,
			ApprovalPossible: false,
			TargetObjectType: "source_object_authority_validation",
			AuditEventName:   "semantic_syscall.validate_source_object_authority",
		},
		domain.ActionValidateSemanticOperation: {
			Action:           domain.ActionValidateSemanticOperation,
			Capability:       CapSemanticOperationValidate,
			Mutating:         false,
			SupportsDryRun:   true,
			ApprovalPossible: false,
			TargetObjectType: "semantic_operation_validation",
			AuditEventName:   "semantic_syscall.validate_semantic_operation",
		},
		domain.ActionValidateAdmissionCandidate: {
			Action:           domain.ActionValidateAdmissionCandidate,
			Capability:       CapAdmissionCandidateValidate,
			Mutating:         false,
			SupportsDryRun:   true,
			ApprovalPossible: false,
			TargetObjectType: "admission_candidate_validation",
			AuditEventName:   "semantic_syscall.validate_admission_candidate",
		},
		domain.ActionValidateContextAttribution: {
			Action:           domain.ActionValidateContextAttribution,
			Capability:       CapContextAttributionValidate,
			Mutating:         false,
			SupportsDryRun:   true,
			ApprovalPossible: false,
			TargetObjectType: "context_attribution_validation",
			AuditEventName:   "semantic_syscall.validate_context_attribution",
		},
		domain.ActionAdmitEvidence: {
			Action:           domain.ActionAdmitEvidence,
			Capability:       CapEvidenceAdmit,
			Mutating:         true,
			SupportsDryRun:   true,
			ApprovalPossible: true,
			TargetObjectType: "court_exhibit_ruling",
			AuditEventName:   "semantic_syscall.admit_evidence",
		},
		domain.ActionAppealRuling: {
			Action:           domain.ActionAppealRuling,
			Capability:       CapRulingAppeal,
			Mutating:         true,
			SupportsDryRun:   true,
			ApprovalPossible: true,
			TargetObjectType: "court_appeal_ruling",
			AuditEventName:   "semantic_syscall.appeal_ruling",
		},
		domain.ActionRecordRetrievalEvidence: {
			Action:           domain.ActionRecordRetrievalEvidence,
			Capability:       CapRetrievalEvidenceRecord,
			Mutating:         true,
			SupportsDryRun:   true,
			ApprovalPossible: false,
			TargetObjectType: "retrieval_evidence_bundle",
			AuditEventName:   "semantic_syscall.record_retrieval_evidence",
		},
		domain.ActionRebuildMemoryAcceleration: {
			Action:           domain.ActionRebuildMemoryAcceleration,
			Capability:       CapMemoryAccelerationRebuild,
			Mutating:         true,
			SupportsDryRun:   false,
			ApprovalPossible: true,
			TargetObjectType: "memory_acceleration_projection",
			AuditEventName:   "semantic_syscall.rebuild_memory_acceleration",
		},
		domain.ActionRecordRetrievalUsefulness: {
			Action:           domain.ActionRecordRetrievalUsefulness,
			Capability:       CapRetrievalUsefulnessRecord,
			Mutating:         true,
			SupportsDryRun:   true,
			ApprovalPossible: false,
			TargetObjectType: "retrieval_usefulness_evidence",
			AuditEventName:   "semantic_syscall.record_retrieval_usefulness",
		},
		domain.ActionRecordRestoreOutcomeFeedback: {
			Action:           domain.ActionRecordRestoreOutcomeFeedback,
			Capability:       CapRestoreOutcomeFeedback,
			Mutating:         true,
			SupportsDryRun:   true,
			ApprovalPossible: false,
			TargetObjectType: "restore_outcome_feedback_evidence",
			AuditEventName:   "semantic_syscall.record_restore_outcome_feedback",
		},
		domain.ActionMaterializeAdmittedEvidence: {
			Action: domain.ActionMaterializeAdmittedEvidence, Capability: CapMemoryEvidenceMaterialize,
			Mutating: true, SupportsDryRun: true, ApprovalPossible: false,
			TargetObjectType: "forge_k_memory_evidence", AuditEventName: "semantic_syscall.materialize_admitted_evidence",
		},
		domain.ActionReviseMemoryEvidence: {
			Action: domain.ActionReviseMemoryEvidence, Capability: CapMemoryEvidenceRevise,
			Mutating: true, SupportsDryRun: true, ApprovalPossible: false,
			TargetObjectType: "forge_k_memory_evidence_revision", AuditEventName: "semantic_syscall.revise_memory_evidence",
		},
		domain.ActionComputeSemanticDiff: {
			Action: domain.ActionComputeSemanticDiff, Capability: CapSemanticDiffCompute,
			Mutating: true, SupportsDryRun: false, ApprovalPossible: false,
			TargetObjectType: "semantic_diff_operation", AuditEventName: "semantic_syscall.compute_semantic_diff",
		},
	}
	return &StaticActionRegistry{definitions: defs}
}

func (r *StaticActionRegistry) Get(action domain.SemanticActionType) (ActionDefinition, bool) {
	if r == nil {
		return ActionDefinition{}, false
	}
	def, ok := r.definitions[action]
	return def, ok
}

func (r *StaticActionRegistry) List() []ActionDefinition {
	if r == nil {
		return nil
	}
	out := make([]ActionDefinition, 0, len(r.definitions))
	for _, def := range r.definitions {
		out = append(out, def)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Action < out[j].Action })
	return out
}

func RequireActionDefinition(reg ActionRegistry, action domain.SemanticActionType) (ActionDefinition, error) {
	def, ok := reg.Get(action)
	if !ok {
		return ActionDefinition{}, fmt.Errorf("unsupported action %q", action)
	}
	return def, nil
}
