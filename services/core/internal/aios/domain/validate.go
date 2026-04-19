package domain

import "strings"

func (r SyscallRequest) Validate() []SyscallError {
	var issues []SyscallError
	if strings.TrimSpace(r.ID) == "" {
		issues = append(issues, SyscallError{Code: ErrMissingRequiredField, Field: "id", Message: "id is required"})
	}
	if strings.TrimSpace(string(r.Action)) == "" {
		issues = append(issues, SyscallError{Code: ErrMissingRequiredField, Field: "action", Message: "action is required"})
	}
	if strings.TrimSpace(r.Actor.ID) == "" {
		issues = append(issues, SyscallError{Code: ErrMissingRequiredField, Field: "actor.id", Message: "actor.id is required"})
	}
	if strings.TrimSpace(r.Actor.Kind) == "" {
		issues = append(issues, SyscallError{Code: ErrMissingRequiredField, Field: "actor.kind", Message: "actor.kind is required"})
	}
	if strings.TrimSpace(string(r.Source)) == "" {
		issues = append(issues, SyscallError{Code: ErrMissingRequiredField, Field: "source", Message: "source is required"})
	}
	if strings.TrimSpace(r.Scope.WorkspaceID) == "" {
		issues = append(issues, SyscallError{Code: ErrInvalidScope, Field: "scope.workspaceId", Message: "scope.workspaceId is required"})
	}
	if strings.TrimSpace(r.Provenance.Actor) == "" {
		issues = append(issues, SyscallError{Code: ErrInvalidProvenance, Field: "provenance.actor", Message: "provenance.actor is required"})
	}
	if strings.TrimSpace(r.Provenance.ActorType) == "" {
		issues = append(issues, SyscallError{Code: ErrInvalidProvenance, Field: "provenance.actorType", Message: "provenance.actorType is required"})
	}
	if r.RequestedAt <= 0 {
		issues = append(issues, SyscallError{Code: ErrMissingRequiredField, Field: "requestedAt", Message: "requestedAt must be a positive timestamp"})
	}
	return issues
}

func (p ContextPacket) Validate() []SyscallError {
	var issues []SyscallError
	if strings.TrimSpace(p.ID) == "" {
		issues = append(issues, SyscallError{Code: ErrMissingRequiredField, Field: "id", Message: "id is required"})
	}
	if strings.TrimSpace(p.Query) == "" {
		issues = append(issues, SyscallError{Code: ErrMissingRequiredField, Field: "query", Message: "query is required"})
	}
	if strings.TrimSpace(p.Scope.WorkspaceID) == "" {
		issues = append(issues, SyscallError{Code: ErrInvalidScope, Field: "scope.workspaceId", Message: "scope.workspaceId is required"})
	}
	if p.Budget.MaxTokens <= 0 {
		issues = append(issues, SyscallError{Code: ErrInvalidPayload, Field: "budget.maxTokens", Message: "budget.maxTokens must be positive"})
	}
	if p.Budget.MaxEvents <= 0 {
		issues = append(issues, SyscallError{Code: ErrInvalidPayload, Field: "budget.maxEvents", Message: "budget.maxEvents must be positive"})
	}
	if p.Budget.MaxNotes <= 0 {
		issues = append(issues, SyscallError{Code: ErrInvalidPayload, Field: "budget.maxNotes", Message: "budget.maxNotes must be positive"})
	}
	return issues
}
