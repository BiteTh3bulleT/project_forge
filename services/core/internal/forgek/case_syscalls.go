package forgek

import "fmt"

func validateCaseOpen(request SyscallRequest) error {
	if request.ActorID == "" || request.WorkspaceID == "" {
		return ErrInvalidInput
	}
	if stringInput(request.Input, "user_intent") == "" {
		return ErrInvalidInput
	}
	return nil
}

func validateCaseID(request SyscallRequest) error {
	if request.ActorID == "" || request.WorkspaceID == "" || stringInput(request.Input, "case_id") == "" {
		return ErrInvalidInput
	}
	return nil
}

func handleCaseOpen(kernel *Kernel, request SyscallRequest, capabilityRefs []string) SyscallResult {
	now := kernel.clock.Now()
	caseID := kernel.ids.NextID("case")
	cp := CasePacket{
		CaseID:      caseID,
		WorkspaceID: request.WorkspaceID,
		UserIntent:  stringInput(request.Input, "user_intent"),
		Summary:     stringInput(request.Input, "summary"),
		OpenedAt:    now,
		Status:      CaseStatusOpen,
	}
	obj := KernelObject{
		ObjectID:        caseID,
		ObjectType:      ObjectTypeCasePacket,
		WorkspaceID:     request.WorkspaceID,
		OwnerID:         request.ActorID,
		AuthorityLevel:  AuthorityCommitted,
		State:           caseState(cp),
		CapabilityScope: append([]string(nil), capabilityRefs...),
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	event, err := kernel.journal.Append(JournalEvent{
		EventType:      JournalEventCaseOpened,
		Timestamp:      now,
		WorkspaceID:    request.WorkspaceID,
		CaseID:         caseID,
		ActorID:        request.ActorID,
		SyscallName:    request.Name,
		InputHash:      hashValue(request.Input),
		OutputHash:     hashValue(cp),
		ObjectRefs:     []string{caseID},
		CapabilityRefs: capabilityRefs,
		Result:         SyscallResultCommitted,
	})
	if err != nil {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: err}
	}
	cp.JournalRefs = []string{event.EventID}
	obj.JournalRefs = []string{event.EventID}
	kernel.objects.putCase(obj, cp)

	return SyscallResult{Success: true, SyscallName: request.Name, ObjectID: caseID, JournalEvent: event.EventID, Output: cp}
}

func handleCaseUpdate(kernel *Kernel, request SyscallRequest, capabilityRefs []string) SyscallResult {
	caseID := stringInput(request.Input, "case_id")
	cp, ok := kernel.objects.getCase(caseID)
	if !ok {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: ErrObjectNotFound}
	}
	if cp.WorkspaceID != request.WorkspaceID {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: ErrObjectNotFound}
	}
	if cp.Status == CaseStatusClosed {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: ErrInvalidStateTransition}
	}

	now := kernel.clock.Now()
	if summary := stringInput(request.Input, "summary"); summary != "" {
		cp.Summary = summary
	}
	if refs, ok := stringSliceInput(request.Input, "object_refs"); ok {
		cp.ObjectRefs = refs
	}
	cp.Status = CaseStatusUpdated

	event, err := kernel.journal.Append(JournalEvent{
		EventType:      JournalEventCaseUpdated,
		Timestamp:      now,
		WorkspaceID:    request.WorkspaceID,
		CaseID:         caseID,
		ActorID:        request.ActorID,
		SyscallName:    request.Name,
		InputHash:      hashValue(request.Input),
		OutputHash:     hashValue(cp),
		ObjectRefs:     append([]string{caseID}, cp.ObjectRefs...),
		CapabilityRefs: capabilityRefs,
		Result:         SyscallResultCommitted,
	})
	if err != nil {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: err}
	}
	cp.JournalRefs = append(cp.JournalRefs, event.EventID)

	obj, _ := kernel.objects.GetObject(caseID)
	obj.State = caseState(cp)
	obj.UpdatedAt = now
	obj.JournalRefs = append(obj.JournalRefs, event.EventID)
	kernel.objects.putCase(obj, cp)

	return SyscallResult{Success: true, SyscallName: request.Name, ObjectID: caseID, JournalEvent: event.EventID, Output: cp}
}

func handleCaseClose(kernel *Kernel, request SyscallRequest, capabilityRefs []string) SyscallResult {
	caseID := stringInput(request.Input, "case_id")
	cp, ok := kernel.objects.getCase(caseID)
	if !ok {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: ErrObjectNotFound}
	}
	if cp.WorkspaceID != request.WorkspaceID {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: ErrObjectNotFound}
	}
	if cp.Status == CaseStatusClosed {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: ErrInvalidStateTransition}
	}

	now := kernel.clock.Now()
	closedAt := now
	cp.ClosedAt = &closedAt
	cp.Status = CaseStatusClosed

	event, err := kernel.journal.Append(JournalEvent{
		EventType:      JournalEventCaseClosed,
		Timestamp:      now,
		WorkspaceID:    request.WorkspaceID,
		CaseID:         caseID,
		ActorID:        request.ActorID,
		SyscallName:    request.Name,
		InputHash:      hashValue(request.Input),
		OutputHash:     hashValue(cp),
		ObjectRefs:     []string{caseID},
		CapabilityRefs: capabilityRefs,
		Result:         SyscallResultCommitted,
	})
	if err != nil {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: err}
	}
	cp.JournalRefs = append(cp.JournalRefs, event.EventID)

	obj, _ := kernel.objects.GetObject(caseID)
	obj.State = caseState(cp)
	obj.UpdatedAt = now
	obj.JournalRefs = append(obj.JournalRefs, event.EventID)
	kernel.objects.putCase(obj, cp)

	return SyscallResult{Success: true, SyscallName: request.Name, ObjectID: caseID, JournalEvent: event.EventID, Output: cp}
}

func handleObjectGet(kernel *Kernel, request SyscallRequest, _ []string) SyscallResult {
	objectID := stringInput(request.Input, "object_id")
	if objectID == "" {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: ErrInvalidInput}
	}
	obj, ok := kernel.objects.GetObject(objectID)
	if !ok || obj.WorkspaceID != request.WorkspaceID {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: ErrObjectNotFound}
	}
	return SyscallResult{Success: true, SyscallName: request.Name, ObjectID: objectID, Output: obj}
}

func handleObjectList(kernel *Kernel, request SyscallRequest, _ []string) SyscallResult {
	all := kernel.objects.ListObjects()
	filtered := make([]KernelObject, 0, len(all))
	for _, obj := range all {
		if obj.WorkspaceID == request.WorkspaceID {
			filtered = append(filtered, obj)
		}
	}
	return SyscallResult{Success: true, SyscallName: request.Name, Output: filtered}
}

func handleCapabilityGrant(kernel *Kernel, request SyscallRequest, capabilityRefs []string) SyscallResult {
	raw, ok := request.Input["capability"].(Capability)
	if !ok {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: ErrInvalidInput}
	}
	if err := kernel.capabilities.Grant(raw); err != nil {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: err}
	}
	now := kernel.clock.Now()
	event, err := kernel.journal.Append(JournalEvent{
		EventType:      JournalEventCapabilityGranted,
		Timestamp:      now,
		WorkspaceID:    request.WorkspaceID,
		ActorID:        request.ActorID,
		SyscallName:    request.Name,
		InputHash:      hashValue(request.Input),
		OutputHash:     hashValue(raw),
		ObjectRefs:     []string{raw.CapabilityID},
		CapabilityRefs: capabilityRefs,
		Result:         SyscallResultCommitted,
	})
	if err != nil {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: err}
	}
	return SyscallResult{Success: true, SyscallName: request.Name, ObjectID: raw.CapabilityID, JournalEvent: event.EventID, Output: raw}
}

func stringInput(input map[string]any, key string) string {
	value, _ := input[key].(string)
	return value
}

func stringSliceInput(input map[string]any, key string) ([]string, bool) {
	value, ok := input[key]
	if !ok {
		return nil, false
	}
	switch refs := value.(type) {
	case []string:
		return append([]string(nil), refs...), true
	case []any:
		out := make([]string, 0, len(refs))
		for _, ref := range refs {
			text, ok := ref.(string)
			if !ok {
				return nil, false
			}
			out = append(out, text)
		}
		return out, true
	default:
		return nil, false
	}
}

func formatError(base error, detail string) error {
	return fmt.Errorf("%w: %s", base, detail)
}

func caseState(cp CasePacket) map[string]any {
	return map[string]any{
		"status":                cp.Status,
		"summary":               cp.Summary,
		"submitted_exhibits":    append([]string(nil), cp.SubmittedExhibitRefs...),
		"admitted_exhibits":     append([]string(nil), cp.AdmittedExhibitRefs...),
		"rejected_exhibits":     append([]string(nil), cp.RejectedExhibitRefs...),
		"ruling_refs":           append([]string(nil), cp.RulingRefs...),
		"contradiction_refs":    append([]string(nil), cp.ContradictionRefs...),
		"supersession_refs":     append([]string(nil), cp.SupersessionRefs...),
		"palace_route_refs":     append([]string(nil), cp.PalaceRouteRefs...),
		"candidate_object_refs": append([]string(nil), cp.CandidateObjectRefs...),
		"retrieval_summary":     cp.RetrievalSummary,
	}
}
