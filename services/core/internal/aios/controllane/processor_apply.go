package controllane

import (
	"context"
	"fmt"
	"strings"

	"forge/projectforge/services/core/internal/aios/domain"
	"forge/projectforge/services/core/internal/forgekernel/contextcompile"
	"forge/projectforge/services/core/internal/forgekernel/court"
)

func (p *Processor) normalize(req domain.SyscallRequest) domain.SyscallRequest {
	now := p.nowMillis()
	if strings.TrimSpace(req.ID) == "" {
		req.ID = fmt.Sprintf("syscall-%d", now)
	}
	if req.RequestedAt <= 0 {
		req.RequestedAt = now
	}
	if strings.TrimSpace(req.CorrelationID) == "" {
		req.CorrelationID = "corr-" + req.ID
	}
	if strings.TrimSpace(req.TraceID) == "" {
		req.TraceID = req.CorrelationID
	}
	if strings.TrimSpace(string(req.Source)) == "" {
		req.Source = domain.ActionSource(strings.TrimSpace(req.Actor.Kind))
	}
	if req.Payload == nil {
		req.Payload = map[string]any{}
	}
	if req.Metadata == nil {
		req.Metadata = map[string]any{}
	}
	return req
}

func (p *Processor) apply(ctx context.Context, store SemanticStore, req domain.SyscallRequest, def ActionDefinition) ([]string, map[string]any, []string, []domain.SyscallError) {
	switch def.Action {
	case domain.ActionCreateNote:
		return applyCreateNote(store, req)
	case domain.ActionCreateLink:
		return applyCreateLink(store, req)
	case domain.ActionUpdateState:
		return applyUpdateState(store, req)
	case domain.ActionOpenLoop:
		return applyOpenLoop(store, req)
	case domain.ActionCloseLoop:
		return applyCloseLoop(store, req)
	case domain.ActionMarkSuperseded:
		return applyMarkSuperseded(store, req)
	case domain.ActionRegisterContradict:
		return applyRegisterContradiction(store, req)
	case domain.ActionDeriveModel:
		return applyDeriveModel(store, req)
	case domain.ActionArchiveNote:
		return applyArchiveNote(store, req)
	case domain.ActionCompileContext:
		return applyCompileContext(ctx, store, req, p.ruleEngine)
	case domain.ActionValidateKVIdentity:
		return applyValidateKVIdentity(req)
	case domain.ActionValidateRefShape:
		return applyValidateRefShape(req)
	case domain.ActionCompareRefShape:
		return applyCompareRefShape(req)
	case domain.ActionValidateSourceObject:
		return applyValidateSourceObjectAuthority(store, req)
	case domain.ActionValidateSemanticOperation:
		return applyValidateSemanticOperation(req)
	case domain.ActionValidateAdmissionCandidate:
		return applyValidateAdmissionCandidate(req)
	case domain.ActionValidateContextAttribution:
		return applyValidateContextAttribution(req)
	case domain.ActionAdmitEvidence, domain.ActionAppealRuling:
		return applyCourtDecision(store, req)
	case domain.ActionRecordRetrievalEvidence:
		return applyRecordRetrievalEvidence(ctx, store, req)
	case domain.ActionRebuildMemoryAcceleration:
		return applyRebuildMemoryAcceleration(ctx, store, req)
	case domain.ActionRecordRetrievalUsefulness:
		return applyRecordRetrievalUsefulness(ctx, store, req)
	case domain.ActionRecordRestoreOutcomeFeedback:
		return applyRecordRestoreOutcomeFeedback(ctx, store, req)
	case domain.ActionMaterializeAdmittedEvidence, domain.ActionReviseMemoryEvidence:
		return applyMemoryEvidenceAction(ctx, store, req)
	case domain.ActionComputeSemanticDiff:
		return applyComputeSemanticDiff(ctx, store, req)
	default:
		return nil, nil, nil, []domain.SyscallError{{Code: domain.ErrUnsupportedAction, Field: "action", Message: "unsupported action"}}
	}
}

func applyCourtDecision(store SemanticStore, req domain.SyscallRequest) ([]string, map[string]any, []string, []domain.SyscallError) {
	if ingress, _ := req.Metadata["forgeKIngressAuthority"].(bool); !ingress {
		return nil, nil, nil, []domain.SyscallError{{Code: domain.ErrUnauthorized, Field: "metadata.forgeKIngressAuthority", Message: "Courthouse mutation requires production FORGE-K ingress"}}
	}
	decision, ok := court.DecisionFromMetadata(req.Metadata)
	if !ok || decision.Action != req.Action {
		return nil, nil, nil, []domain.SyscallError{{Code: domain.ErrUnauthorized, Field: "metadata." + court.MetadataDecisionKey, Message: "missing production FORGE-K Courthouse decision"}}
	}
	proposedBy := nonEmpty(req.Actor.ID, string(req.Source))
	committedBy := nonEmpty(readString(req.Metadata, "kernelAuthorityOwner"), "forge_k.kernel")
	exhibit := court.Exhibit{
		ID: decision.ExhibitID, CaseID: decision.CaseID, Scope: req.Scope,
		SourceType: readString(req.Payload, "sourceType"), SourceRefs: append([]string{}, decision.InputRefs...),
		ContentSummary: readString(req.Payload, "contentSummary"), RawRef: readString(req.Payload, "rawRef"),
		ContentHash: decision.ContentHash, Status: decision.Decision,
		CurrentRulingID: decision.RulingID, CreatedAt: req.RequestedAt, UpdatedAt: req.RequestedAt,
		Provenance: req.Provenance, SyscallID: req.ID, CorrelationID: req.CorrelationID, TraceID: req.TraceID,
		ProposedBy: proposedBy, CommittedBy: committedBy,
	}
	ruling := court.Ruling{
		ID: decision.RulingID, CaseID: decision.CaseID, ExhibitID: decision.ExhibitID,
		AppealID: decision.AppealID, PriorRulingID: decision.PriorRulingID, Scope: req.Scope,
		Decision: decision.Decision, ReasonCode: decision.ReasonCode, Reason: decision.Reason,
		PolicyVersion: decision.PolicyVersion, PolicyRefs: append([]string{}, decision.PolicyRefs...),
		InputRefs: append([]string{}, decision.InputRefs...), ContentHash: decision.ContentHash,
		CreatedAt: req.RequestedAt, Provenance: req.Provenance, SyscallID: req.ID,
		CorrelationID: req.CorrelationID, TraceID: req.TraceID, ProposedBy: proposedBy, CommittedBy: committedBy,
	}
	var appeal *court.Appeal
	if req.Action == domain.ActionAppealRuling {
		current, found := store.FindCourtExhibit(decision.ExhibitID, req.Scope)
		if !found {
			return nil, nil, nil, []domain.SyscallError{{Code: domain.ErrNotFound, Field: "payload.exhibitId", Message: "court exhibit not found at commit"}}
		}
		exhibit = current
		exhibit.Status = decision.Decision
		exhibit.CurrentRulingID = decision.RulingID
		exhibit.UpdatedAt = req.RequestedAt
		exhibit.SyscallID = req.ID
		exhibit.CorrelationID = req.CorrelationID
		exhibit.TraceID = req.TraceID
		exhibit.ProposedBy = proposedBy
		exhibit.CommittedBy = committedBy
		appeal = &court.Appeal{
			ID: decision.AppealID, CaseID: decision.CaseID, ExhibitID: decision.ExhibitID,
			PriorRulingID: decision.PriorRulingID, NewRulingID: decision.RulingID, Scope: req.Scope,
			Grounds: readString(req.Payload, "grounds"), NewSourceRefs: append([]string{}, decision.InputRefs...),
			NewContentHash: decision.ContentHash, CreatedAt: req.RequestedAt,
			Provenance: req.Provenance, SyscallID: req.ID, CorrelationID: req.CorrelationID, TraceID: req.TraceID,
			ProposedBy: proposedBy, CommittedBy: committedBy,
		}
	}
	if err := store.CreateCourtDecision(exhibit, ruling, appeal); err != nil {
		return nil, nil, nil, []domain.SyscallError{{Code: domain.ErrConflict, Field: "court", Message: err.Error()}}
	}
	ids := []string{decision.ExhibitID, decision.RulingID}
	if appeal != nil {
		ids = []string{decision.AppealID, decision.RulingID, decision.ExhibitID}
	}
	return ids, map[string]any{
		"courthouse": map[string]any{
			"caseId": decision.CaseID, "exhibitId": decision.ExhibitID, "rulingId": decision.RulingID,
			"appealId": decision.AppealID, "priorRulingId": decision.PriorRulingID,
			"decision": decision.Decision, "reasonCode": decision.ReasonCode,
			"policyVersion": decision.PolicyVersion, "kernelDecisionAuthority": true,
			"modelAdmissionAuthority": false,
		},
	}, nil, nil
}

func applyValidateKVIdentity(req domain.SyscallRequest) ([]string, map[string]any, []string, []domain.SyscallError) {
	decision := EnforceKVIdentity(req)
	if !decision.Accepted {
		return nil, nil, nil, []domain.SyscallError{decision.ToSyscallError()}
	}
	return nil, decision.ToStateSummary(), decision.Warnings, nil
}

func applyValidateRefShape(req domain.SyscallRequest) ([]string, map[string]any, []string, []domain.SyscallError) {
	decision := EnforceRefShape(req)
	if !decision.Accepted {
		return nil, nil, nil, []domain.SyscallError{decision.ToSyscallError()}
	}
	return nil, decision.ToStateSummary(), decision.Warnings, nil
}

func applyCompareRefShape(req domain.SyscallRequest) ([]string, map[string]any, []string, []domain.SyscallError) {
	decision := EnforceRefShapeComparison(req)
	if !decision.Accepted {
		return nil, nil, nil, []domain.SyscallError{decision.ToSyscallError()}
	}
	return nil, decision.ToStateSummary(), decision.Warnings, nil
}

func applyValidateSourceObjectAuthority(store SemanticReadStore, req domain.SyscallRequest) ([]string, map[string]any, []string, []domain.SyscallError) {
	decision := EnforceSourceObjectAuthority(req, store)
	if !decision.Accepted {
		return nil, nil, nil, []domain.SyscallError{decision.ToSyscallError()}
	}
	return nil, decision.ToStateSummary(), decision.Warnings, nil
}

func applyValidateSemanticOperation(req domain.SyscallRequest) ([]string, map[string]any, []string, []domain.SyscallError) {
	decision := EnforceSemanticOperation(req)
	if !decision.Accepted {
		return nil, nil, nil, []domain.SyscallError{decision.ToSyscallError()}
	}
	return nil, decision.ToStateSummary(), decision.Warnings, nil
}

func applyValidateAdmissionCandidate(req domain.SyscallRequest) ([]string, map[string]any, []string, []domain.SyscallError) {
	decision := EnforceAdmissionCandidate(req)
	if !decision.Accepted {
		return nil, nil, nil, []domain.SyscallError{decision.ToSyscallError()}
	}
	return nil, decision.ToStateSummary(), decision.Warnings, nil
}

func applyValidateContextAttribution(req domain.SyscallRequest) ([]string, map[string]any, []string, []domain.SyscallError) {
	decision := EnforceContextAttribution(req)
	if !decision.Accepted {
		return nil, nil, nil, []domain.SyscallError{decision.ToSyscallError()}
	}
	return nil, decision.ToStateSummary(), decision.Warnings, nil
}

func applyCreateNote(store SemanticStore, req domain.SyscallRequest) ([]string, map[string]any, []string, []domain.SyscallError) {
	id := nonEmpty(readString(req.Payload, "id"), req.ID+":note")
	note := domain.MemoryNote{
		ID:         id,
		Type:       domain.MemoryNoteType(readString(req.Payload, "type")),
		Title:      readString(req.Payload, "title"),
		Content:    readString(req.Payload, "content"),
		Scope:      req.Scope,
		Confidence: readFloat(req.Payload, "confidence", 0.7),
		Status:     domain.MemoryNoteStatus(nonEmpty(readString(req.Payload, "status"), string(domain.NoteActive))),
		CreatedAt:  req.RequestedAt,
		UpdatedAt:  req.RequestedAt,
		Provenance: req.Provenance,
	}
	if err := store.CreateNote(note); err != nil {
		return nil, nil, nil, []domain.SyscallError{{Code: domain.ErrConflict, Field: "payload.id", Message: err.Error()}}
	}
	return []string{id}, map[string]any{"noteId": id, "noteStatus": note.Status}, nil, nil
}

func applyCreateLink(store SemanticStore, req domain.SyscallRequest) ([]string, map[string]any, []string, []domain.SyscallError) {
	id := nonEmpty(readString(req.Payload, "id"), req.ID+":link")
	link := domain.SemanticLink{
		ID:         id,
		Type:       domain.SemanticLinkType(readString(req.Payload, "type")),
		SourceID:   readString(req.Payload, "sourceId"),
		TargetID:   readString(req.Payload, "targetId"),
		Scope:      req.Scope,
		Confidence: readFloat(req.Payload, "confidence", 0.7),
		Provenance: req.Provenance,
		CreatedAt:  req.RequestedAt,
	}
	if err := store.CreateLink(link); err != nil {
		return nil, nil, nil, []domain.SyscallError{{Code: domain.ErrConflict, Field: "payload.id", Message: err.Error()}}
	}
	return []string{id}, map[string]any{"linkId": id, "linkType": link.Type}, nil, nil
}

func applyUpdateState(store SemanticStore, req domain.SyscallRequest) ([]string, map[string]any, []string, []domain.SyscallError) {
	id := nonEmpty(readString(req.Payload, "id"), req.ID+":state")
	key := readString(req.Payload, "key")
	item := domain.StateItem{
		ID:          id,
		Key:         key,
		Value:       payloadValue(req.Payload, "value"),
		Scope:       req.Scope,
		Status:      domain.StateItemStatus(nonEmpty(readString(req.Payload, "status"), string(domain.StateActive))),
		DerivedFrom: readStringSlice(req.Payload, "derivedFrom"),
		UpdatedAt:   req.RequestedAt,
	}
	if err := store.CreateState(item); err != nil {
		return nil, nil, nil, []domain.SyscallError{{Code: domain.ErrConflict, Field: "payload.id", Message: err.Error()}}
	}
	if persisted, ok := store.FindStateByScopeKey(req.Scope, key); ok {
		return []string{persisted.ID}, map[string]any{"stateId": persisted.ID, "key": key}, nil, nil
	}
	if persisted, ok := store.FindStateByKey(key); ok {
		return []string{persisted.ID}, map[string]any{"stateId": persisted.ID, "key": key}, nil, nil
	}
	return []string{id}, map[string]any{"stateId": id, "key": key}, nil, nil
}

func applyOpenLoop(store SemanticStore, req domain.SyscallRequest) ([]string, map[string]any, []string, []domain.SyscallError) {
	id := nonEmpty(readString(req.Payload, "id"), req.ID+":loop")
	if existing, ok := store.FindLoop(id); ok {
		next := existing
		requestedState := readString(req.Payload, "state")
		targetState := existing.State
		if strings.TrimSpace(requestedState) != "" {
			targetState = domain.OpenLoopState(requestedState)
		}
		if targetState != existing.State {
			if !IsValidOpenLoopTransition(existing.State, targetState) {
				return nil, nil, nil, []domain.SyscallError{{Code: domain.ErrInvalidStateTransition, Field: "payload.state", Message: "invalid loop transition"}}
			}
			next.State = targetState
		}
		if v := readString(req.Payload, "title"); v != "" {
			next.Title = v
		}
		if v := readString(req.Payload, "priority"); v != "" {
			next.Priority = strings.ToLower(v)
		}
		if v := readString(req.Payload, "owner"); v != "" {
			next.Owner = v
		}
		if v := readString(req.Payload, "blocker"); v != "" {
			next.Blocker = v
		}
		if v := readString(req.Payload, "nextAction"); v != "" {
			next.NextAction = v
		}
		if related := readStringSlice(req.Payload, "relatedNotes"); len(related) > 0 {
			next.RelatedNotes = related
		}
		if reason := readString(req.Payload, "reason"); reason != "" && (next.State == domain.LoopResolved || next.State == domain.LoopArchived) {
			next.NextAction = "resolved: " + reason
		}
		next.UpdatedAt = req.RequestedAt
		if err := store.UpdateLoop(next); err != nil {
			return nil, nil, nil, []domain.SyscallError{{Code: domain.ErrInternal, Field: "payload.id", Message: err.Error()}}
		}
		return []string{id}, map[string]any{"loopId": id, "loopState": next.State}, nil, nil
	}
	loop := domain.OpenLoop{
		ID:           id,
		Title:        readString(req.Payload, "title"),
		State:        domain.OpenLoopState(nonEmpty(readString(req.Payload, "state"), string(domain.LoopOpen))),
		Scope:        req.Scope,
		Priority:     strings.ToLower(nonEmpty(readString(req.Payload, "priority"), "medium")),
		Owner:        nonEmpty(readString(req.Payload, "owner"), req.Actor.ID),
		Blocker:      readString(req.Payload, "blocker"),
		NextAction:   readString(req.Payload, "nextAction"),
		RelatedNotes: readStringSlice(req.Payload, "relatedNotes"),
		CreatedFrom:  nonEmpty(readString(req.Payload, "createdFrom"), req.CorrelationID),
		CreatedAt:    req.RequestedAt,
		UpdatedAt:    req.RequestedAt,
	}
	if err := store.CreateLoop(loop); err != nil {
		return nil, nil, nil, []domain.SyscallError{{Code: domain.ErrConflict, Field: "payload.id", Message: err.Error()}}
	}
	return []string{id}, map[string]any{"loopId": id, "loopState": loop.State}, nil, nil
}

func applyCloseLoop(store SemanticStore, req domain.SyscallRequest) ([]string, map[string]any, []string, []domain.SyscallError) {
	loopID := readString(req.Payload, "loopId")
	loop, ok := store.FindLoop(loopID)
	if !ok {
		return nil, nil, nil, []domain.SyscallError{{Code: domain.ErrNotFound, Field: "payload.loopId", Message: "loop not found"}}
	}
	if !IsValidOpenLoopTransition(loop.State, domain.LoopResolved) {
		return nil, nil, nil, []domain.SyscallError{{Code: domain.ErrInvalidStateTransition, Field: "payload.loopId", Message: "invalid loop transition"}}
	}
	loop.State = domain.LoopResolved
	loop.UpdatedAt = req.RequestedAt
	if reason := readString(req.Payload, "reason"); reason != "" {
		loop.NextAction = "resolved: " + reason
	}
	if err := store.UpdateLoop(loop); err != nil {
		return nil, nil, nil, []domain.SyscallError{{Code: domain.ErrInternal, Field: "payload.loopId", Message: err.Error()}}
	}
	return []string{loopID}, map[string]any{"loopId": loopID, "loopState": loop.State}, nil, nil
}

func applyMarkSuperseded(store SemanticStore, req domain.SyscallRequest) ([]string, map[string]any, []string, []domain.SyscallError) {
	oldID := readString(req.Payload, "oldObjectId")
	newID := readString(req.Payload, "newObjectId")
	oldKindDetected, oldScope, oldKnown := detectStoreObject(store, oldID)
	newKindDetected, newScope, newKnown := detectStoreObject(store, newID)
	if oldKnown && !scopeMatchesRequest(req.Scope, oldScope) {
		return nil, nil, nil, []domain.SyscallError{{Code: domain.ErrInvalidScope, Field: "payload.oldObjectId", Message: "old object outside request scope"}}
	}
	if newKnown && !scopeMatchesRequest(req.Scope, newScope) {
		return nil, nil, nil, []domain.SyscallError{{Code: domain.ErrInvalidScope, Field: "payload.newObjectId", Message: "new object outside request scope"}}
	}
	if oldKnown && newKnown && oldKindDetected != newKindDetected {
		return nil, nil, nil, []domain.SyscallError{{Code: domain.ErrInvalidPayload, Field: "payload.newObjectKind", Message: "supersession requires compatible object kinds"}}
	}
	recordID := req.ID + ":supersession"
	record := SupersessionRecord{
		ID:            recordID,
		OldID:         oldID,
		OldKind:       nonEmpty(readString(req.Payload, "oldObjectKind"), "object"),
		NewID:         newID,
		NewKind:       nonEmpty(readString(req.Payload, "newObjectKind"), "object"),
		Reason:        readString(req.Payload, "reason"),
		WorkspaceID:   req.Scope.WorkspaceID,
		LaneID:        req.Scope.LaneID,
		CorrelationID: req.CorrelationID,
		TraceID:       req.TraceID,
		SyscallID:     req.ID,
		ProposedBy:    string(req.Source),
		CommittedBy:   "forge_kernel",
		Metadata:      cloneMap(req.Metadata),
		CreatedAt:     req.RequestedAt,
		Provenance:    req.Provenance,
	}
	if err := store.CreateSupersession(record); err != nil {
		return nil, nil, nil, []domain.SyscallError{{Code: domain.ErrConflict, Field: "payload.oldObjectId", Message: err.Error()}}
	}

	linkID := req.ID + ":supersedes_link"
	link := domain.SemanticLink{
		ID:         linkID,
		Type:       domain.LinkSupersedes,
		SourceID:   newID,
		TargetID:   oldID,
		Scope:      req.Scope,
		Confidence: 1.0,
		Provenance: req.Provenance,
		CreatedAt:  req.RequestedAt,
	}
	if err := store.CreateLink(link); err != nil {
		return nil, nil, nil, []domain.SyscallError{{Code: domain.ErrConflict, Field: "payload.oldObjectId", Message: err.Error()}}
	}
	if oldNote, ok := store.FindNote(oldID); ok && IsValidNoteTransition(oldNote.Status, domain.NoteSuperseded) {
		oldNote.Status = domain.NoteSuperseded
		oldNote.UpdatedAt = req.RequestedAt
		_ = store.UpdateNote(oldNote)
	}
	return []string{recordID, linkID}, map[string]any{"oldObjectId": oldID, "newObjectId": newID}, nil, nil
}

func applyRegisterContradiction(store SemanticStore, req domain.SyscallRequest) ([]string, map[string]any, []string, []domain.SyscallError) {
	leftID := readString(req.Payload, "leftObjectId")
	rightID := readString(req.Payload, "rightObjectId")
	recordID := req.ID + ":contradiction"
	record := ContradictionRecord{
		ID:            recordID,
		LeftID:        leftID,
		LeftKind:      nonEmpty(readString(req.Payload, "leftObjectKind"), "object"),
		RightID:       rightID,
		RightKind:     nonEmpty(readString(req.Payload, "rightObjectKind"), "object"),
		Reason:        readString(req.Payload, "reason"),
		Severity:      nonEmpty(readString(req.Payload, "severity"), "medium"),
		Confidence:    readFloat(req.Payload, "confidence", 0.7),
		WorkspaceID:   req.Scope.WorkspaceID,
		LaneID:        req.Scope.LaneID,
		CorrelationID: req.CorrelationID,
		TraceID:       req.TraceID,
		SyscallID:     req.ID,
		ProposedBy:    string(req.Source),
		CommittedBy:   "forge_kernel",
		Metadata:      cloneMap(req.Metadata),
		CreatedAt:     req.RequestedAt,
		Provenance:    req.Provenance,
	}
	if err := store.CreateContradiction(record); err != nil {
		return nil, nil, nil, []domain.SyscallError{{Code: domain.ErrConflict, Field: "payload.leftObjectId", Message: err.Error()}}
	}
	linkID := req.ID + ":contradiction_link"
	link := domain.SemanticLink{
		ID:         linkID,
		Type:       domain.LinkContradicts,
		SourceID:   leftID,
		TargetID:   rightID,
		Scope:      req.Scope,
		Confidence: record.Confidence,
		Provenance: req.Provenance,
		CreatedAt:  req.RequestedAt,
	}
	if err := store.CreateLink(link); err != nil {
		return nil, nil, nil, []domain.SyscallError{{Code: domain.ErrConflict, Field: "payload.leftObjectId", Message: err.Error()}}
	}
	return []string{recordID, linkID}, map[string]any{"leftObjectId": leftID, "rightObjectId": rightID}, nil, nil
}

func applyDeriveModel(store SemanticStore, req domain.SyscallRequest) ([]string, map[string]any, []string, []domain.SyscallError) {
	id := nonEmpty(readString(req.Payload, "id"), req.ID+":model")
	if existing, ok := store.FindModel(id); ok {
		nextStatusRaw := strings.TrimSpace(readString(req.Payload, "status"))
		if nextStatusRaw == "" {
			return nil, nil, nil, []domain.SyscallError{{Code: domain.ErrConflict, Field: "payload.id", Message: "model already exists; include status transition for lifecycle update"}}
		}
		nextStatus := domain.AdaptivePolicyModelStatus(nextStatusRaw)
		if !IsValidModelTransition(existing.Status, nextStatus) {
			return nil, nil, nil, []domain.SyscallError{{Code: domain.ErrInvalidStateTransition, Field: "payload.status", Message: "invalid model lifecycle transition"}}
		}
		model := existing
		model.Status = nextStatus
		if expr, ok := req.Payload["expression"]; ok {
			model.Expression = expr
		}
		if derived := readStringSlice(req.Payload, "derivedFrom"); len(derived) > 0 {
			model.DerivedFrom = derived
			model.SupportCount = len(derived)
		}
		if conf := readFloat(req.Payload, "confidence", model.Confidence); conf >= 0 && conf <= 1 {
			model.Confidence = conf
		}
		model.CreatedAt = req.RequestedAt
		if err := store.UpdateModel(model); err != nil {
			return nil, nil, nil, []domain.SyscallError{{Code: domain.ErrInternal, Field: "payload.id", Message: err.Error()}}
		}
		return []string{id}, map[string]any{"modelId": id, "modelStatus": nextStatus}, nil, nil
	}
	derived := readStringSlice(req.Payload, "derivedFrom")
	model := domain.AdaptivePolicyModel{
		ID:           id,
		Type:         readString(req.Payload, "type"),
		Expression:   req.Payload["expression"],
		DerivedFrom:  derived,
		SupportCount: readInt(req.Payload, "supportCount", len(derived)),
		Confidence:   readFloat(req.Payload, "confidence", 0.7),
		Status:       domain.ModelProvisional,
		Scope:        req.Scope,
		CreatedAt:    req.RequestedAt,
	}
	if err := store.CreateModel(model); err != nil {
		return nil, nil, nil, []domain.SyscallError{{Code: domain.ErrConflict, Field: "payload.id", Message: err.Error()}}
	}
	return []string{id}, map[string]any{"modelId": id, "modelStatus": model.Status}, nil, nil
}

func applyArchiveNote(store SemanticStore, req domain.SyscallRequest) ([]string, map[string]any, []string, []domain.SyscallError) {
	noteID := readString(req.Payload, "noteId")
	note, ok := store.FindNote(noteID)
	if !ok {
		return nil, nil, nil, []domain.SyscallError{{Code: domain.ErrNotFound, Field: "payload.noteId", Message: "note not found"}}
	}
	if !IsValidNoteTransition(note.Status, domain.NoteArchived) {
		return nil, nil, nil, []domain.SyscallError{{Code: domain.ErrInvalidStateTransition, Field: "payload.noteId", Message: "invalid note transition"}}
	}
	note.Status = domain.NoteArchived
	note.UpdatedAt = req.RequestedAt
	if err := store.UpdateNote(note); err != nil {
		return nil, nil, nil, []domain.SyscallError{{Code: domain.ErrInternal, Field: "payload.noteId", Message: err.Error()}}
	}
	return []string{noteID}, map[string]any{"noteId": noteID, "noteStatus": note.Status}, nil, nil
}

func applyCompileContext(ctx context.Context, store SemanticStore, req domain.SyscallRequest, engine RuleEngine) ([]string, map[string]any, []string, []domain.SyscallError) {
	_ = ctx
	_ = engine
	input, inputOK := contextcompile.InputFromMetadata(req.Metadata)
	decision, decisionOK := contextcompile.DecisionFromMetadata(req.Metadata)
	if !inputOK || !decisionOK {
		return nil, nil, nil, []domain.SyscallError{{Code: domain.ErrUnauthorized, Field: "metadata.forgeKContextCompileDecision", Message: "missing production FORGE-K Context Compiler decision"}}
	}
	bundle, err := buildGovernedContextBundle(req, input, decision, store)
	if err != nil {
		return nil, nil, nil, []domain.SyscallError{{Code: domain.ErrConflict, Field: "contextCompile", Message: err.Error()}}
	}
	if err := store.CreateGovernedContextBundle(bundle); err != nil {
		return nil, nil, nil, []domain.SyscallError{{Code: domain.ErrConflict, Field: "contextCompile.head", Message: err.Error()}}
	}
	return []string{decision.PacketID}, map[string]any{
		"contextPacketId":           decision.PacketID,
		"contextSnapshotId":         decision.SnapshotID,
		"contextDecisionDigest":     decision.DecisionDigest,
		"contextPacketCommitment":   decision.PacketCommitment,
		"contextSnapshotCommitment": decision.SnapshotCommitment,
		"sourceManifestHash":        decision.SourceManifestHash,
		"selectedEvidenceIds":       append([]string(nil), decision.SelectedEvidenceIDs...),
		"restoreDecision":           decision.Selection.Mode,
		"governedSources":           cloneIntegrityValue(bundle.Sources),
		"kernelContextCompiler":     true,
		"legacyContextInputs":       false,
	}, []string{"context bundle is immutable governed evidence, not canonical truth"}, nil
}

func defaultBudget() domain.ContextBudget {
	return domain.ContextBudget{MaxTokens: 4000, MaxEvents: 50, MaxNotes: 50}
}

func listRestoreOutcomeSignals(ctx context.Context, store SemanticStore, scope domain.ForgeScope, query string, limit int) ([]RestoreOutcomeEvent, []string) {
	outcomeStore, ok := store.(RestoreOutcomeStore)
	if !ok {
		return nil, nil
	}
	events, err := outcomeStore.ListRestoreOutcomes(ctx, RestoreOutcomeFilter{
		WorkspaceID: scope.WorkspaceID,
		LaneID:      scope.LaneID,
		Query:       query,
		Limit:       limit,
	})
	if err != nil {
		return nil, []string{"restore outcome feedback lookup failed: " + err.Error()}
	}
	for index := range events {
		projection, found, projectionErr := store.GetRestoreOutcomeFeedbackProjection(ctx, events[index].ID, domain.ForgeScope{
			WorkspaceID:   events[index].WorkspaceID,
			LaneID:        events[index].LaneID,
			SelectedPaths: append([]string(nil), scope.SelectedPaths...),
		})
		if projectionErr != nil {
			return nil, []string{"restore outcome feedback projection lookup failed: " + projectionErr.Error()}
		}
		if found && exactUtilityScopeMatches(scope, projection.Scope) {
			events[index] = effectiveRestoreOutcomeUtilitySignal(events[index], &projection)
		} else {
			events[index] = effectiveRestoreOutcomeUtilitySignal(events[index], nil)
		}
	}
	return events, nil
}

func effectiveRestoreOutcomeUtilitySignal(original RestoreOutcomeEvent, projection *RestoreOutcomeFeedbackProjection) RestoreOutcomeEvent {
	effective := original
	effective.Metadata = cloneIntegrityValue(nonNilMap(original.Metadata))
	effective.Metadata["utilityOriginalOutcome"] = original.Outcome
	if projection != nil && projection.NonCanonical && projection.RestoreOutcomeID == original.ID &&
		strings.TrimSpace(projection.LatestEventID) != "" {
		effective.Outcome = projection.Outcome
		effective.OutcomeConfidence = projection.OutcomeConfidence
		effective.OperatorFeedback = projection.OperatorFeedback
		effective.CorrectionSummary = projection.CorrectionSummary
		effective.UpdatedAt = projection.UpdatedAt
		effective.Metadata["utilityEvidenceSource"] = "governed_feedback_projection"
		effective.Metadata["utilityProjectionEventId"] = projection.LatestEventID
		return effective
	}
	if hasLegacyMutableRestoreFeedback(original) {
		effective.Outcome = RestoreOutcomeUnknown
		effective.OutcomeConfidence = 0
		effective.OperatorFeedback = ""
		effective.CorrectionSummary = ""
		effective.Metadata["utilityEvidenceSource"] = "legacy_mutable_feedback_ignored"
		return effective
	}
	effective.Metadata["utilityEvidenceSource"] = "original_restore_outcome"
	return effective
}

func hasLegacyMutableRestoreFeedback(event RestoreOutcomeEvent) bool {
	if strings.TrimSpace(event.OperatorFeedback) != "" || strings.TrimSpace(event.CorrectionSummary) != "" {
		return true
	}
	if event.UpdatedAt > event.CreatedAt {
		return true
	}
	for _, key := range []string{"feedback_history", "last_feedback_by", "non_canonical_evidence"} {
		if _, exists := event.Metadata[key]; exists {
			return true
		}
	}
	return false
}

func buildRestoreOutcomeEvent(req domain.SyscallRequest, packet domain.ContextPacket, snapshot compiledContextSnapshot, selection compileContextRestoreSelection, selectedEvidence []string, selectedHeaderOnly bool) RestoreOutcomeEvent {
	outcome := RestoreOutcomeUnknown
	requiresFreshCompile := selection.Decision != "selected"
	switch selection.Decision {
	case "fresh_compile_no_candidates":
		outcome = RestoreOutcomeNoCandidate
	case "fresh_compile_below_threshold", "fresh_compile_forced":
		outcome = RestoreOutcomeFreshCompileRequired
	}
	selected := selection.selectedCandidate()
	restoreScore := selection.TopScore
	outcomeConfidence := 0.0
	selectedStateKeys := []string{}
	selectedLoopIDs := []string{}
	selectedArtifactIDs := []string{}
	if selected != nil {
		restoreScore = selected.Score.TotalScore
		outcomeConfidence = selected.Score.Confidence
		selectedStateKeys = stripGraphIDPrefix(dominantIDsByPrefix(selected.Snapshot.Graph, "state:"), "state:")
		selectedLoopIDs = stripGraphIDPrefix(dominantIDsByPrefix(selected.Snapshot.Graph, "loop:"), "loop:")
		selectedArtifactIDs = stripGraphIDPrefix(dominantIDsByPrefix(selected.Snapshot.Graph, "artifact:"), "artifact:")
	}
	if len(selectedArtifactIDs) == 0 {
		selectedArtifactIDs = stripGraphIDPrefix(dominantIDsByPrefix(snapshot.Graph, "artifact:"), "artifact:")
	}
	event := RestoreOutcomeEvent{
		ID:                   NewRestoreOutcomeID(req.ID, packet.ID, "compiled"),
		CreatedAt:            req.RequestedAt,
		UpdatedAt:            req.RequestedAt,
		WorkspaceID:          req.Scope.WorkspaceID,
		LaneID:               req.Scope.LaneID,
		Query:                packet.Query,
		ContextPacketID:      packet.ID,
		SnapshotID:           selection.selectedSnapshotID(),
		SnapshotKind:         snapshot.Header.SnapshotKind,
		RestoreScore:         clamp01(restoreScore),
		RequiresFreshCompile: requiresFreshCompile,
		SelectedEvidence:     selectedEvidence,
		SelectedStateKeys:    selectedStateKeys,
		SelectedLoopIDs:      selectedLoopIDs,
		SelectedArtifactIDs:  selectedArtifactIDs,
		Outcome:              outcome,
		OutcomeConfidence:    clamp01(outcomeConfidence),
		DownstreamActionType: "compile_context",
		DownstreamObjectID:   packet.ID,
		CorrelationID:        req.CorrelationID,
		TraceID:              req.TraceID,
		SyscallID:            req.ID,
		ProposedBy:           string(req.Source),
		CommittedBy:          "forge_kernel",
		Metadata: map[string]any{
			"restore_decision":        selection.Decision,
			"restore_decision_reason": selection.decisionReason(),
			"selected_header_only":    selectedHeaderOnly,
			"non_canonical_evidence":  true,
		},
	}
	if selection.RuleTrace != nil {
		event.Metadata["rule_trace"] = ruleTraceMap(*selection.RuleTrace)
	}
	return normalizeRestoreOutcomeEvent(event)
}

func stripGraphIDPrefix(ids []string, prefix string) []string {
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		out = append(out, strings.TrimPrefix(strings.TrimSpace(id), prefix))
	}
	return normalizeStringSet(out)
}

func restoreOutcomeSummary(event RestoreOutcomeEvent) map[string]any {
	event = normalizeRestoreOutcomeEvent(event)
	return map[string]any{
		"id":                   event.ID,
		"workspaceId":          event.WorkspaceID,
		"laneId":               event.LaneID,
		"contextPacketId":      event.ContextPacketID,
		"snapshotId":           event.SnapshotID,
		"snapshotKind":         event.SnapshotKind,
		"restoreScore":         event.RestoreScore,
		"requiresFreshCompile": event.RequiresFreshCompile,
		"selectedEvidence":     event.SelectedEvidence,
		"selectedStateKeys":    event.SelectedStateKeys,
		"selectedLoopIds":      event.SelectedLoopIDs,
		"selectedArtifactIds":  event.SelectedArtifactIDs,
		"outcome":              string(event.Outcome),
		"outcomeConfidence":    event.OutcomeConfidence,
		"correlationId":        event.CorrelationID,
		"traceId":              event.TraceID,
		"nonCanonicalEvidence": true,
		"metadata":             event.Metadata,
	}
}

func detectStoreObject(store SemanticStore, id string) (string, domain.ForgeScope, bool) {
	if strings.TrimSpace(id) == "" || store == nil {
		return "", domain.ForgeScope{}, false
	}
	if note, ok := store.FindNote(id); ok {
		return "memory_note", note.Scope, true
	}
	if loop, ok := store.FindLoop(id); ok {
		return "open_loop", loop.Scope, true
	}
	if model, ok := store.FindModel(id); ok {
		return "derived_model", model.Scope, true
	}
	return "", domain.ForgeScope{}, false
}

func scopeMatchesRequest(reqScope, objectScope domain.ForgeScope) bool {
	if strings.TrimSpace(reqScope.WorkspaceID) == "" {
		return false
	}
	if strings.TrimSpace(reqScope.WorkspaceID) != strings.TrimSpace(objectScope.WorkspaceID) {
		return false
	}
	reqLane := strings.TrimSpace(reqScope.LaneID)
	objLane := strings.TrimSpace(objectScope.LaneID)
	if reqLane == "" || objLane == "" {
		return true
	}
	return reqLane == objLane
}

func nonEmpty(v, fallback string) string {
	if strings.TrimSpace(v) != "" {
		return strings.TrimSpace(v)
	}
	return strings.TrimSpace(fallback)
}

func payloadValue(payload map[string]any, key string) map[string]any {
	if payload == nil {
		return map[string]any{}
	}
	v, ok := payload[key]
	if !ok || v == nil {
		return map[string]any{}
	}
	if m, ok := v.(map[string]any); ok {
		return m
	}
	return map[string]any{"value": v}
}
