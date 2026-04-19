package controllane

import (
	"fmt"
	"sort"

	"forge/projectforge/services/core/internal/aios/domain"
)

const (
	CapMemoryNoteCreate       = "memory.note.create"
	CapMemoryNoteArchive      = "memory.note.archive"
	CapMemoryLinkCreate       = "memory.link.create"
	CapStateUpdate            = "state.update"
	CapLoopOpen               = "loop.open"
	CapLoopClose              = "loop.close"
	CapMemoryContradictionReg = "memory.contradiction.register"
	CapMemorySupersessionMark = "memory.supersession.mark"
	CapModelDerive            = "model.derive"
	CapContextCompile         = "context.compile"
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
