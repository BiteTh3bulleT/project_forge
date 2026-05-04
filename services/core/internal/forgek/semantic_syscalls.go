package forgek

import (
	"errors"

	"forge/projectforge/services/core/internal/forgek/court"
	"forge/projectforge/services/core/internal/forgek/semantic"
)

func (k *Kernel) registerSemanticSyscalls(register func(SyscallDefinition)) {
	for _, definition := range []SyscallDefinition{
		{Name: SyscallSemanticApply, Handler: handleSemanticApply},
		{Name: SyscallSemanticMerge, Handler: handleSemanticMerge},
		{Name: SyscallSemanticDiff, Handler: handleSemanticDiff},
		{Name: SyscallSemanticIntersect, Handler: handleSemanticIntersect},
		{Name: SyscallSemanticContradict, Handler: handleSemanticContradict},
		{Name: SyscallSemanticSupersede, Handler: handleSemanticSupersede},
		{Name: SyscallSemanticCompress, Handler: handleSemanticCompress},
		{Name: SyscallSemanticDerive, Handler: handleSemanticDerive},
		{Name: SyscallSemanticPromote, Handler: handleSemanticPromote},
		{Name: SyscallSemanticDemote, Handler: handleSemanticDemote},
		{Name: SyscallSemanticExpire, Handler: handleSemanticExpire},
	} {
		definition.Version = "v1"
		definition.AllowedLanes = []string{"arterial"}
		definition.Deterministic = true
		definition.SideEffects = true
		definition.JournalRequired = true
		definition.Replayable = true
		register(definition)
	}
	register(SyscallDefinition{Name: SyscallSemanticListOperations, Version: "v1", AllowedLanes: []string{"arterial"}, Deterministic: true, Replayable: true, Handler: handleSemanticListOperations})
	register(SyscallDefinition{Name: SyscallSemanticGetOperation, Version: "v1", AllowedLanes: []string{"arterial"}, Deterministic: true, Replayable: true, Handler: handleSemanticGetOperation})
}

func handleSemanticApply(kernel *Kernel, request SyscallRequest, capabilityRefs []string) SyscallResult {
	operationType := stringInput(request.Input, "operation_type")
	if operationType == "" {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: ErrInvalidInput}
	}
	return kernel.applySemanticOperation(request, capabilityRefs, operationType, JournalEventSemanticOperationApplied)
}

func handleSemanticMerge(kernel *Kernel, request SyscallRequest, capabilityRefs []string) SyscallResult {
	return kernel.applySemanticOperation(request, capabilityRefs, semantic.OperationMerge, JournalEventSemanticMergeApplied)
}

func handleSemanticDiff(kernel *Kernel, request SyscallRequest, capabilityRefs []string) SyscallResult {
	return kernel.applySemanticOperation(request, capabilityRefs, semantic.OperationDiff, JournalEventSemanticDiffApplied)
}

func handleSemanticIntersect(kernel *Kernel, request SyscallRequest, capabilityRefs []string) SyscallResult {
	return kernel.applySemanticOperation(request, capabilityRefs, semantic.OperationIntersect, JournalEventSemanticIntersectApplied)
}

func handleSemanticContradict(kernel *Kernel, request SyscallRequest, capabilityRefs []string) SyscallResult {
	return kernel.applySemanticOperation(request, capabilityRefs, semantic.OperationContradict, JournalEventSemanticContradictionApplied)
}

func handleSemanticSupersede(kernel *Kernel, request SyscallRequest, capabilityRefs []string) SyscallResult {
	return kernel.applySemanticOperation(request, capabilityRefs, semantic.OperationSupersede, JournalEventSemanticSupersessionApplied)
}

func handleSemanticCompress(kernel *Kernel, request SyscallRequest, capabilityRefs []string) SyscallResult {
	return kernel.applySemanticOperation(request, capabilityRefs, semantic.OperationCompress, JournalEventSemanticCompressApplied)
}

func handleSemanticDerive(kernel *Kernel, request SyscallRequest, capabilityRefs []string) SyscallResult {
	return kernel.applySemanticOperation(request, capabilityRefs, semantic.OperationDerive, JournalEventSemanticDeriveApplied)
}

func handleSemanticPromote(kernel *Kernel, request SyscallRequest, capabilityRefs []string) SyscallResult {
	return kernel.applySemanticOperation(request, capabilityRefs, semantic.OperationPromote, JournalEventSemanticPromoteApplied)
}

func handleSemanticDemote(kernel *Kernel, request SyscallRequest, capabilityRefs []string) SyscallResult {
	return kernel.applySemanticOperation(request, capabilityRefs, semantic.OperationDemote, JournalEventSemanticDemoteApplied)
}

func handleSemanticExpire(kernel *Kernel, request SyscallRequest, capabilityRefs []string) SyscallResult {
	return kernel.applySemanticOperation(request, capabilityRefs, semantic.OperationExpire, JournalEventSemanticExpireApplied)
}

func (k *Kernel) applySemanticOperation(request SyscallRequest, capabilityRefs []string, operationType, eventType string) SyscallResult {
	inputObjects, err := k.semanticObjectsInput(request)
	if err != nil {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: err}
	}
	operationID := k.ids.NextID("semantic-operation")
	resultID := k.ids.NextID("semantic-result")
	operation, result, err := k.semantic.ApplyOperation(semantic.OperationRequest{
		OperationID:   operationID,
		ResultID:      resultID,
		OperationType: operationType,
		WorkspaceID:   request.WorkspaceID,
		CaseID:        semanticCaseID(request),
		InputObjects:  inputObjects,
		Parameters:    mapInput(request.Input, "parameters"),
		CreatedBy:     request.ActorID,
		CreatedAt:     k.clock.Now(),
		NextObjectID: func() string {
			return k.ids.NextID("semantic-object")
		},
	})
	if err != nil {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: mapSemanticError(err)}
	}
	event, err := k.appendSemanticEvent(request, eventType, operation.OperationID, operation.CaseID, capabilityRefs, result)
	if err != nil {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: err}
	}
	operation.JournalRef = event.EventID
	k.semantic.StoreOperation(operation)
	k.objects.putObject(semanticOperationObject(operation, capabilityRefs))
	for _, object := range result.OutputObjects {
		object.JournalRefs = appendUnique(object.JournalRefs, event.EventID)
		k.semantic.StoreObject(object)
		k.objects.putObject(semanticObjectObject(object, capabilityRefs))
	}
	return SyscallResult{Success: true, SyscallName: request.Name, ObjectID: operation.OperationID, JournalEvent: event.EventID, Output: result}
}

func handleSemanticListOperations(kernel *Kernel, request SyscallRequest, _ []string) SyscallResult {
	return SyscallResult{Success: true, SyscallName: request.Name, Output: kernel.semantic.ListOperations(request.WorkspaceID)}
}

func handleSemanticGetOperation(kernel *Kernel, request SyscallRequest, _ []string) SyscallResult {
	operation, ok := kernel.semantic.GetOperation(stringInput(request.Input, "operation_id"))
	if !ok || operation.WorkspaceID != request.WorkspaceID {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: ErrObjectNotFound}
	}
	return SyscallResult{Success: true, SyscallName: request.Name, ObjectID: operation.OperationID, Output: operation}
}

func (k *Kernel) semanticObjectsInput(request SyscallRequest) ([]semantic.SemanticObject, error) {
	value, ok := request.Input["objects"]
	if !ok {
		return nil, ErrInvalidInput
	}
	var objects []semantic.SemanticObject
	switch typed := value.(type) {
	case []semantic.SemanticObject:
		objects = append(objects, typed...)
	case []any:
		for _, item := range typed {
			object, ok := item.(semantic.SemanticObject)
			if !ok {
				return nil, ErrInvalidInput
			}
			objects = append(objects, object)
		}
	default:
		return nil, ErrInvalidInput
	}
	out := make([]semantic.SemanticObject, 0, len(objects))
	for _, object := range objects {
		if object.WorkspaceID != request.WorkspaceID {
			return nil, ErrInvalidInput
		}
		if k.semanticObjectReferencesRejectedEvidence(object) {
			return nil, mapSemanticError(semantic.ErrRejectedEvidenceInput)
		}
		out = append(out, object.Clone())
	}
	return out, nil
}

func (k *Kernel) semanticObjectReferencesRejectedEvidence(object semantic.SemanticObject) bool {
	if object.ObjectType != semantic.ObjectTypeExhibitRef {
		return false
	}
	for _, ref := range object.SourceObjectRefs {
		exhibit, ok := k.court.GetExhibit(ref)
		if ok && exhibit.AdmissibilityStatus == court.StatusRejected {
			return true
		}
	}
	return false
}

func semanticCaseID(request SyscallRequest) string {
	caseID := stringInput(request.Input, "case_id")
	if caseID == "" {
		caseID = request.CaseID
	}
	return caseID
}

func (k *Kernel) appendSemanticEvent(request SyscallRequest, eventType, objectID, caseID string, capabilityRefs []string, output any) (JournalEvent, error) {
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

func semanticOperationObject(operation semantic.SemanticOperation, capabilityRefs []string) KernelObject {
	return KernelObject{
		ObjectID:       operation.OperationID,
		ObjectType:     ObjectTypeSemanticOperation,
		WorkspaceID:    operation.WorkspaceID,
		OwnerID:        operation.CreatedBy,
		AuthorityLevel: AuthorityCommitted,
		State: map[string]any{
			"operation_type":     operation.OperationType,
			"case_id":            operation.CaseID,
			"input_object_refs":  append([]string(nil), operation.InputObjectRefs...),
			"output_object_refs": append([]string(nil), operation.OutputObjectRefs...),
			"operator_version":   operation.OperatorVersion,
			"reasoning_summary":  operation.ReasoningSummary,
		},
		SourceRefs:      append([]string(nil), operation.ProvenanceRefs...),
		CapabilityScope: append([]string(nil), capabilityRefs...),
		CreatedAt:       operation.CreatedAt,
		UpdatedAt:       operation.CreatedAt,
		JournalRefs:     []string{operation.JournalRef},
	}
}

func semanticObjectObject(object semantic.SemanticObject, capabilityRefs []string) KernelObject {
	return KernelObject{
		ObjectID:       object.SemanticObjectID,
		ObjectType:     ObjectTypeSemanticObject,
		WorkspaceID:    object.WorkspaceID,
		AuthorityLevel: AuthorityProposal,
		State: map[string]any{
			"object_type":          object.ObjectType,
			"content_summary":      object.ContentSummary,
			"normalized_content":   object.NormalizedContent,
			"semantic_authority":   object.AuthorityLevel,
			"admissibility_status": object.AdmissibilityStatus,
			"superseded_by":        append([]string(nil), object.SupersededBy...),
			"contradicted_by":      append([]string(nil), object.ContradictedBy...),
		},
		SourceRefs:      append([]string(nil), object.SourceRefs...),
		CapabilityScope: append([]string(nil), capabilityRefs...),
		CreatedAt:       object.CreatedAt,
		UpdatedAt:       object.UpdatedAt,
		JournalRefs:     append([]string(nil), object.JournalRefs...),
	}
}

func mapSemanticError(err error) error {
	switch {
	case errors.Is(err, semantic.ErrRejectedEvidenceInput), errors.Is(err, semantic.ErrContradictionMerge), errors.Is(err, semantic.ErrInvalidOperation), errors.Is(err, semantic.ErrInvalidSemanticObject), errors.Is(err, semantic.ErrUnknownOperator):
		return ErrInvalidInput
	case errors.Is(err, semantic.ErrSemanticObjectNotFound), errors.Is(err, semantic.ErrSemanticOperationNotFound):
		return ErrObjectNotFound
	default:
		return err
	}
}
