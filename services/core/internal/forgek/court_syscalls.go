package forgek

import (
	"errors"

	"forge/projectforge/services/core/internal/forgek/court"
)

func (k *Kernel) registerCourtSyscalls(register func(SyscallDefinition)) {
	for _, definition := range []SyscallDefinition{
		{Name: SyscallCourtSubmit, Handler: handleCourtSubmit},
		{Name: SyscallCourtAdmit, Handler: handleCourtAdmit},
		{Name: SyscallCourtReject, Handler: handleCourtReject},
		{Name: SyscallCourtRule, Handler: handleCourtRule},
		{Name: SyscallCourtRegisterContradiction, Handler: handleCourtRegisterContradiction},
		{Name: SyscallCourtRegisterSupersession, Handler: handleCourtRegisterSupersession},
	} {
		definition.Version = "v1"
		definition.AllowedLanes = []string{"arterial"}
		definition.Deterministic = true
		definition.SideEffects = true
		definition.JournalRequired = true
		definition.Replayable = true
		register(definition)
	}
	register(SyscallDefinition{Name: SyscallCourtListExhibits, Version: "v1", AllowedLanes: []string{"arterial"}, Deterministic: true, Replayable: true, Handler: handleCourtListExhibits})
	register(SyscallDefinition{Name: SyscallCourtListRulings, Version: "v1", AllowedLanes: []string{"arterial"}, Deterministic: true, Replayable: true, Handler: handleCourtListRulings})
	register(SyscallDefinition{Name: SyscallCourtListContradictions, Version: "v1", AllowedLanes: []string{"arterial"}, Deterministic: true, Replayable: true, Handler: handleCourtListContradictions})
}

func handleCourtSubmit(kernel *Kernel, request SyscallRequest, capabilityRefs []string) SyscallResult {
	cp, err := kernel.requireOpenCase(request)
	if err != nil {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: err}
	}
	now := kernel.clock.Now()
	exhibit, err := court.NewExhibit(court.ExhibitInput{
		ExhibitID:      kernel.ids.NextID("exhibit"),
		CaseID:         cp.CaseID,
		WorkspaceID:    request.WorkspaceID,
		SourceObjectID: stringInput(request.Input, "source_object_id"),
		SubmittedBy:    request.ActorID,
		SourceType:     stringInput(request.Input, "source_type"),
		SourceRefs:     stringSliceInputDefault(request.Input, "source_refs"),
		ClaimRefs:      stringSliceInputDefault(request.Input, "claim_refs"),
		ContentSummary: stringInput(request.Input, "content_summary"),
		RawRef:         stringInput(request.Input, "raw_ref"),
		CreatedAt:      now,
		Metadata:       mapInput(request.Input, "metadata"),
	})
	if err != nil {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: ErrInvalidInput}
	}
	event, err := kernel.appendCourtEvent(request, JournalEventExhibitSubmitted, exhibit.ExhibitID, cp.CaseID, capabilityRefs, exhibit)
	if err != nil {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: err}
	}
	exhibit.JournalRefs = []string{event.EventID}
	kernel.court.SubmitExhibit(exhibit)
	kernel.objects.putObject(exhibitObject(exhibit, request.ActorID, capabilityRefs))
	cp.SubmittedExhibitRefs = appendUnique(cp.SubmittedExhibitRefs, exhibit.ExhibitID)
	cp.JournalRefs = append(cp.JournalRefs, event.EventID)
	kernel.putUpdatedCase(cp, event.EventID)
	return SyscallResult{Success: true, SyscallName: request.Name, ObjectID: exhibit.ExhibitID, JournalEvent: event.EventID, Output: exhibit}
}

func handleCourtAdmit(kernel *Kernel, request SyscallRequest, capabilityRefs []string) SyscallResult {
	cp, err := kernel.requireOpenCase(request)
	if err != nil {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: err}
	}
	exhibitID := stringInput(request.Input, "exhibit_id")
	reason := stringInput(request.Input, "admission_reason")
	exhibit, ok := kernel.court.GetExhibit(exhibitID)
	if !ok || exhibit.CaseID != cp.CaseID {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: ErrObjectNotFound}
	}
	if exhibit.AdmissibilityStatus != court.StatusSubmitted {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: ErrInvalidStateTransition}
	}
	exhibit.AdmissibilityStatus = court.StatusAdmitted
	exhibit.AdmissionReason = reason
	exhibit.UpdatedAt = kernel.clock.Now()
	event, err := kernel.appendCourtEvent(request, JournalEventExhibitAdmitted, exhibit.ExhibitID, cp.CaseID, capabilityRefs, exhibit)
	if err != nil {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: err}
	}
	exhibit.JournalRefs = append(exhibit.JournalRefs, event.EventID)
	kernel.court.UpdateExhibit(exhibit)
	kernel.objects.putObject(exhibitObject(exhibit, request.ActorID, capabilityRefs))
	cp.AdmittedExhibitRefs = appendUnique(cp.AdmittedExhibitRefs, exhibit.ExhibitID)
	cp.JournalRefs = append(cp.JournalRefs, event.EventID)
	kernel.putUpdatedCase(cp, event.EventID)
	return SyscallResult{Success: true, SyscallName: request.Name, ObjectID: exhibit.ExhibitID, JournalEvent: event.EventID, Output: exhibit}
}

func handleCourtReject(kernel *Kernel, request SyscallRequest, capabilityRefs []string) SyscallResult {
	cp, err := kernel.requireOpenCase(request)
	if err != nil {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: err}
	}
	reason := stringInput(request.Input, "rejection_reason")
	if reason == "" {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: ErrInvalidInput}
	}
	exhibitID := stringInput(request.Input, "exhibit_id")
	exhibit, ok := kernel.court.GetExhibit(exhibitID)
	if !ok || exhibit.CaseID != cp.CaseID {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: ErrObjectNotFound}
	}
	if exhibit.AdmissibilityStatus != court.StatusSubmitted {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: ErrInvalidStateTransition}
	}
	exhibit.AdmissibilityStatus = court.StatusRejected
	exhibit.RejectionReason = reason
	exhibit.UpdatedAt = kernel.clock.Now()
	event, err := kernel.appendCourtEvent(request, JournalEventExhibitRejected, exhibit.ExhibitID, cp.CaseID, capabilityRefs, exhibit)
	if err != nil {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: err}
	}
	exhibit.JournalRefs = append(exhibit.JournalRefs, event.EventID)
	kernel.court.UpdateExhibit(exhibit)
	kernel.objects.putObject(exhibitObject(exhibit, request.ActorID, capabilityRefs))
	cp.RejectedExhibitRefs = appendUnique(cp.RejectedExhibitRefs, exhibit.ExhibitID)
	cp.JournalRefs = append(cp.JournalRefs, event.EventID)
	kernel.putUpdatedCase(cp, event.EventID)
	return SyscallResult{Success: true, SyscallName: request.Name, ObjectID: exhibit.ExhibitID, JournalEvent: event.EventID, Output: exhibit}
}

func handleCourtRule(kernel *Kernel, request SyscallRequest, capabilityRefs []string) SyscallResult {
	cp, err := kernel.requireOpenCase(request)
	if err != nil {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: err}
	}
	now := kernel.clock.Now()
	ruling, err := court.NewRuling(court.RulingInput{
		RulingID:            kernel.ids.NextID("ruling"),
		CaseID:              cp.CaseID,
		WorkspaceID:         request.WorkspaceID,
		RulingType:          stringInput(request.Input, "ruling_type"),
		AdmittedExhibitRefs: stringSliceInputDefault(request.Input, "admitted_exhibit_refs"),
		RejectedExhibitRefs: stringSliceInputDefault(request.Input, "rejected_exhibit_refs"),
		ContradictionRefs:   stringSliceInputDefault(request.Input, "contradiction_refs"),
		SupersessionRefs:    stringSliceInputDefault(request.Input, "supersession_refs"),
		ReasoningSummary:    stringInput(request.Input, "reasoning_summary"),
		PolicyRefs:          stringSliceInputDefault(request.Input, "policy_refs"),
		CreatedBy:           request.ActorID,
		CreatedAt:           now,
		Metadata:            mapInput(request.Input, "metadata"),
	})
	if err != nil {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: ErrInvalidInput}
	}
	event, err := kernel.appendCourtEvent(request, JournalEventRulingCreated, ruling.RulingID, cp.CaseID, capabilityRefs, ruling)
	if err != nil {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: err}
	}
	ruling.JournalRef = event.EventID
	kernel.court.CreateRuling(ruling)
	kernel.objects.putObject(rulingObject(ruling, capabilityRefs))
	cp.RulingRefs = appendUnique(cp.RulingRefs, ruling.RulingID)
	cp.JournalRefs = append(cp.JournalRefs, event.EventID)
	kernel.putUpdatedCase(cp, event.EventID)
	return SyscallResult{Success: true, SyscallName: request.Name, ObjectID: ruling.RulingID, JournalEvent: event.EventID, Output: ruling}
}

func handleCourtRegisterContradiction(kernel *Kernel, request SyscallRequest, capabilityRefs []string) SyscallResult {
	cp, err := kernel.requireOpenCase(request)
	if err != nil {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: err}
	}
	a := stringInput(request.Input, "exhibit_a_id")
	b := stringInput(request.Input, "exhibit_b_id")
	if !kernel.exhibitExistsInCase(a, cp.CaseID) || !kernel.exhibitExistsInCase(b, cp.CaseID) {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: ErrObjectNotFound}
	}
	contradiction, err := court.NewContradiction(court.ContradictionInput{
		ContradictionID:   kernel.ids.NextID("contradiction"),
		CaseID:            cp.CaseID,
		WorkspaceID:       request.WorkspaceID,
		ExhibitAID:        a,
		ExhibitBID:        b,
		ClaimAID:          stringInput(request.Input, "claim_a_id"),
		ClaimBID:          stringInput(request.Input, "claim_b_id"),
		ContradictionType: stringInput(request.Input, "contradiction_type"),
		Description:       stringInput(request.Input, "description"),
		Severity:          stringInput(request.Input, "severity"),
		CreatedAt:         kernel.clock.Now(),
	})
	if err != nil {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: ErrInvalidInput}
	}
	event, err := kernel.appendCourtEvent(request, JournalEventContradictionRegistered, contradiction.ContradictionID, cp.CaseID, capabilityRefs, contradiction)
	if err != nil {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: err}
	}
	contradiction.JournalRefs = []string{event.EventID}
	kernel.court.RegisterContradiction(contradiction)
	kernel.objects.putObject(contradictionObject(contradiction, capabilityRefs))
	cp.ContradictionRefs = appendUnique(cp.ContradictionRefs, contradiction.ContradictionID)
	cp.JournalRefs = append(cp.JournalRefs, event.EventID)
	kernel.putUpdatedCase(cp, event.EventID)
	return SyscallResult{Success: true, SyscallName: request.Name, ObjectID: contradiction.ContradictionID, JournalEvent: event.EventID, Output: contradiction}
}

func handleCourtRegisterSupersession(kernel *Kernel, request SyscallRequest, capabilityRefs []string) SyscallResult {
	cp, err := kernel.requireOpenCase(request)
	if err != nil {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: err}
	}
	supersession, err := court.NewSupersession(court.SupersessionInput{
		SupersessionID: kernel.ids.NextID("supersession"),
		CaseID:         cp.CaseID,
		WorkspaceID:    request.WorkspaceID,
		OldObjectID:    stringInput(request.Input, "old_object_id"),
		NewObjectID:    stringInput(request.Input, "new_object_id"),
		Reason:         stringInput(request.Input, "reason"),
		CreatedAt:      kernel.clock.Now(),
	})
	if err != nil {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: ErrInvalidInput}
	}
	if _, ok := kernel.objects.GetObject(supersession.OldObjectID); !ok {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: ErrObjectNotFound}
	}
	if _, ok := kernel.objects.GetObject(supersession.NewObjectID); !ok {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: ErrObjectNotFound}
	}
	event, err := kernel.appendCourtEvent(request, JournalEventSupersessionRegistered, supersession.SupersessionID, cp.CaseID, capabilityRefs, supersession)
	if err != nil {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: err}
	}
	supersession.JournalRef = event.EventID
	kernel.court.RegisterSupersession(supersession)
	kernel.objects.putObject(supersessionObject(supersession, capabilityRefs))
	if exhibit, ok := kernel.court.GetExhibit(supersession.OldObjectID); ok {
		exhibit.AdmissibilityStatus = court.StatusSuperseded
		exhibit.SupersessionRefs = appendUnique(exhibit.SupersessionRefs, supersession.SupersessionID)
		exhibit.JournalRefs = append(exhibit.JournalRefs, event.EventID)
		kernel.court.UpdateExhibit(exhibit)
		kernel.objects.putObject(exhibitObject(exhibit, request.ActorID, capabilityRefs))
	}
	cp.SupersessionRefs = appendUnique(cp.SupersessionRefs, supersession.SupersessionID)
	cp.JournalRefs = append(cp.JournalRefs, event.EventID)
	kernel.putUpdatedCase(cp, event.EventID)
	return SyscallResult{Success: true, SyscallName: request.Name, ObjectID: supersession.SupersessionID, JournalEvent: event.EventID, Output: supersession}
}

func handleCourtListExhibits(kernel *Kernel, request SyscallRequest, _ []string) SyscallResult {
	return SyscallResult{Success: true, SyscallName: request.Name, Output: kernel.court.ListCaseExhibits(stringInput(request.Input, "case_id"))}
}

func handleCourtListRulings(kernel *Kernel, request SyscallRequest, _ []string) SyscallResult {
	return SyscallResult{Success: true, SyscallName: request.Name, Output: kernel.court.ListCaseRulings(stringInput(request.Input, "case_id"))}
}

func handleCourtListContradictions(kernel *Kernel, request SyscallRequest, _ []string) SyscallResult {
	return SyscallResult{Success: true, SyscallName: request.Name, Output: kernel.court.ListCaseContradictions(stringInput(request.Input, "case_id"))}
}

func (k *Kernel) requireOpenCase(request SyscallRequest) (CasePacket, error) {
	caseID := stringInput(request.Input, "case_id")
	if caseID == "" {
		caseID = request.CaseID
	}
	cp, ok := k.objects.getCase(caseID)
	if !ok || cp.WorkspaceID != request.WorkspaceID {
		return CasePacket{}, ErrObjectNotFound
	}
	if cp.Status == CaseStatusClosed {
		return CasePacket{}, ErrInvalidStateTransition
	}
	return cp, nil
}

func (k *Kernel) putUpdatedCase(cp CasePacket, _ string) {
	obj, _ := k.objects.GetObject(cp.CaseID)
	obj.State = caseState(cp)
	obj.UpdatedAt = k.clock.Now()
	obj.JournalRefs = append([]string(nil), cp.JournalRefs...)
	k.objects.putCase(obj, cp)
}

func (k *Kernel) appendCourtEvent(request SyscallRequest, eventType, objectID, caseID string, capabilityRefs []string, output any) (JournalEvent, error) {
	return k.journal.Append(JournalEvent{
		EventType:      eventType,
		Timestamp:      k.clock.Now(),
		WorkspaceID:    request.WorkspaceID,
		CaseID:         caseID,
		ActorID:        request.ActorID,
		SyscallName:    request.Name,
		InputHash:      hashValue(request.Input),
		OutputHash:     hashValue(output),
		ObjectRefs:     []string{objectID},
		CapabilityRefs: capabilityRefs,
		Result:         SyscallResultCommitted,
	})
}

func (k *Kernel) exhibitExistsInCase(exhibitID, caseID string) bool {
	exhibit, ok := k.court.GetExhibit(exhibitID)
	return ok && exhibit.CaseID == caseID
}

func exhibitObject(exhibit court.Exhibit, ownerID string, capabilityRefs []string) KernelObject {
	return KernelObject{
		ObjectID:       exhibit.ExhibitID,
		ObjectType:     ObjectTypeExhibit,
		WorkspaceID:    exhibit.WorkspaceID,
		OwnerID:        ownerID,
		AuthorityLevel: AuthorityCommitted,
		State: map[string]any{
			"admissibility_status": exhibit.AdmissibilityStatus,
			"content_summary":      exhibit.ContentSummary,
			"admission_reason":     exhibit.AdmissionReason,
			"rejection_reason":     exhibit.RejectionReason,
			"contradiction_refs":   append([]string(nil), exhibit.ContradictionRefs...),
			"supersession_refs":    append([]string(nil), exhibit.SupersessionRefs...),
		},
		SourceRefs:      append([]string(nil), exhibit.SourceRefs...),
		CapabilityScope: append([]string(nil), capabilityRefs...),
		CreatedAt:       exhibit.CreatedAt,
		UpdatedAt:       exhibit.UpdatedAt,
		JournalRefs:     append([]string(nil), exhibit.JournalRefs...),
	}
}

func rulingObject(ruling court.Ruling, capabilityRefs []string) KernelObject {
	return KernelObject{ObjectID: ruling.RulingID, ObjectType: ObjectTypeRuling, WorkspaceID: ruling.WorkspaceID, OwnerID: ruling.CreatedBy, AuthorityLevel: AuthorityCommitted, State: map[string]any{"ruling_type": ruling.RulingType, "reasoning_summary": ruling.ReasoningSummary}, CapabilityScope: append([]string(nil), capabilityRefs...), CreatedAt: ruling.CreatedAt, UpdatedAt: ruling.CreatedAt, JournalRefs: []string{ruling.JournalRef}}
}

func contradictionObject(contradiction court.Contradiction, capabilityRefs []string) KernelObject {
	return KernelObject{ObjectID: contradiction.ContradictionID, ObjectType: ObjectTypeContradiction, WorkspaceID: contradiction.WorkspaceID, AuthorityLevel: AuthorityCommitted, State: map[string]any{"status": contradiction.Status, "exhibit_a_id": contradiction.ExhibitAID, "exhibit_b_id": contradiction.ExhibitBID}, CapabilityScope: append([]string(nil), capabilityRefs...), CreatedAt: contradiction.CreatedAt, UpdatedAt: contradiction.CreatedAt, JournalRefs: append([]string(nil), contradiction.JournalRefs...)}
}

func supersessionObject(supersession court.Supersession, capabilityRefs []string) KernelObject {
	return KernelObject{ObjectID: supersession.SupersessionID, ObjectType: ObjectTypeSupersession, WorkspaceID: supersession.WorkspaceID, AuthorityLevel: AuthorityCommitted, State: map[string]any{"old_object_id": supersession.OldObjectID, "new_object_id": supersession.NewObjectID, "reason": supersession.Reason}, CapabilityScope: append([]string(nil), capabilityRefs...), CreatedAt: supersession.CreatedAt, UpdatedAt: supersession.CreatedAt, JournalRefs: []string{supersession.JournalRef}}
}

func appendUnique(values []string, value string) []string {
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}

func stringSliceInputDefault(input map[string]any, key string) []string {
	values, ok := stringSliceInput(input, key)
	if !ok {
		return nil
	}
	return values
}

func mapInput(input map[string]any, key string) map[string]any {
	value, _ := input[key].(map[string]any)
	return value
}

func mapCourtError(err error) error {
	if errors.Is(err, court.ErrExhibitNotFound) {
		return ErrObjectNotFound
	}
	return err
}
