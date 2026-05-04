package forgek

import (
	"errors"
	"sort"

	"forge/projectforge/services/core/internal/forgek/snapshots"
)

func (k *Kernel) registerSnapshotSyscalls(register func(SyscallDefinition)) {
	for _, definition := range []SyscallDefinition{
		{Name: SyscallSnapshotCreate, Handler: handleSnapshotCreate},
		{Name: SyscallSnapshotSeal, Handler: handleSnapshotSeal},
		{Name: SyscallSnapshotSupersede, Handler: handleSnapshotSupersede},
		{Name: SyscallSnapshotExpire, Handler: handleSnapshotExpire},
		{Name: SyscallSnapshotRestoreSeed, Handler: handleSnapshotRestoreSeed},
	} {
		definition.Version = "v1"
		definition.AllowedLanes = []string{"arterial"}
		definition.Deterministic = true
		definition.SideEffects = true
		definition.JournalRequired = true
		definition.Replayable = true
		register(definition)
	}
	register(SyscallDefinition{Name: SyscallSnapshotGet, Version: "v1", AllowedLanes: []string{"arterial"}, Deterministic: true, Replayable: true, Handler: handleSnapshotGet})
	register(SyscallDefinition{Name: SyscallSnapshotList, Version: "v1", AllowedLanes: []string{"arterial"}, Deterministic: true, Replayable: true, Handler: handleSnapshotList})
	register(SyscallDefinition{Name: SyscallSnapshotDiff, Version: "v1", AllowedLanes: []string{"arterial"}, Deterministic: true, Replayable: true, Handler: handleSnapshotDiff})
}

func handleSnapshotCreate(kernel *Kernel, request SyscallRequest, capabilityRefs []string) SyscallResult {
	input, err := snapshotInputFromRequest(kernel, request)
	if err != nil {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: err}
	}
	snapshot, err := snapshots.NewSnapshot(input)
	if err != nil {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: mapSnapshotError(err)}
	}
	event, err := kernel.appendSnapshotEvent(request, JournalEventSnapshotCreated, snapshot.SnapshotID, snapshot.CaseID, capabilityRefs, snapshot, snapshot.AllRefs())
	if err != nil {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: err}
	}
	snapshot.JournalRefs = []string{event.EventID}
	kernel.snapshots.StoreSnapshot(snapshot)
	kernel.objects.putObject(snapshotObject(snapshot, request.ActorID, capabilityRefs))
	return SyscallResult{Success: true, SyscallName: request.Name, ObjectID: snapshot.SnapshotID, JournalEvent: event.EventID, Output: snapshot}
}

func handleSnapshotGet(kernel *Kernel, request SyscallRequest, _ []string) SyscallResult {
	snapshotID := stringInput(request.Input, "snapshot_id")
	snapshot, ok := kernel.snapshots.GetSnapshot(snapshotID)
	if !ok || snapshot.WorkspaceID != request.WorkspaceID {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: ErrObjectNotFound}
	}
	return SyscallResult{Success: true, SyscallName: request.Name, ObjectID: snapshotID, Output: snapshot}
}

func handleSnapshotList(kernel *Kernel, request SyscallRequest, _ []string) SyscallResult {
	filter := snapshots.ListFilter{
		WorkspaceID:  request.WorkspaceID,
		CaseID:       stringInput(request.Input, "case_id"),
		SnapshotType: snapshots.SnapshotType(stringInput(request.Input, "snapshot_type")),
		Status:       snapshots.SnapshotStatus(stringInput(request.Input, "status")),
	}
	return SyscallResult{Success: true, SyscallName: request.Name, Output: kernel.snapshots.ListSnapshots(filter)}
}

func handleSnapshotSeal(kernel *Kernel, request SyscallRequest, capabilityRefs []string) SyscallResult {
	snapshotID := stringInput(request.Input, "snapshot_id")
	snapshot, ok := kernel.snapshots.GetSnapshot(snapshotID)
	if !ok || snapshot.WorkspaceID != request.WorkspaceID {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: ErrObjectNotFound}
	}
	if snapshot.Status != snapshots.StatusDraft {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: ErrInvalidStateTransition}
	}
	sealed := snapshot.Clone()
	now := kernel.clock.Now()
	sealed.Status = snapshots.StatusSealed
	sealed.SealedAt = &now
	event, err := kernel.appendSnapshotEvent(request, JournalEventSnapshotSealed, sealed.SnapshotID, sealed.CaseID, capabilityRefs, sealed, sealed.AllRefs())
	if err != nil {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: err}
	}
	sealed.JournalRefs = append(sealed.JournalRefs, event.EventID)
	kernel.snapshots.StoreSnapshot(sealed)
	kernel.objects.putObject(snapshotObject(sealed, sealed.CreatedBy, capabilityRefs))
	return SyscallResult{Success: true, SyscallName: request.Name, ObjectID: snapshotID, JournalEvent: event.EventID, Output: sealed}
}

func handleSnapshotSupersede(kernel *Kernel, request SyscallRequest, capabilityRefs []string) SyscallResult {
	oldID := stringInput(request.Input, "old_snapshot_id")
	newID := stringInput(request.Input, "new_snapshot_id")
	oldSnapshot, ok := kernel.snapshots.GetSnapshot(oldID)
	if !ok || oldSnapshot.WorkspaceID != request.WorkspaceID {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: ErrObjectNotFound}
	}
	newSnapshot, ok := kernel.snapshots.GetSnapshot(newID)
	if !ok || newSnapshot.WorkspaceID != request.WorkspaceID {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: ErrObjectNotFound}
	}
	if oldSnapshot.Status == snapshots.StatusExpired || oldSnapshot.Status == snapshots.StatusSuperseded {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: ErrInvalidStateTransition}
	}
	result := snapshots.SnapshotSupersessionResult{
		Superseded:  oldSnapshot.Clone(),
		Superseding: newSnapshot.Clone(),
	}
	result.Superseded.Status = snapshots.StatusSuperseded
	result.Superseded.SupersededBy = newID
	result.Superseding.Supersedes = appendUnique(result.Superseding.Supersedes, oldID)
	event, err := kernel.appendSnapshotEvent(request, JournalEventSnapshotSuperseded, oldID, result.Superseded.CaseID, capabilityRefs, result, []string{oldID, newID})
	if err != nil {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: err}
	}
	result.Superseded.JournalRefs = append(result.Superseded.JournalRefs, event.EventID)
	result.Superseding.JournalRefs = append(result.Superseding.JournalRefs, event.EventID)
	kernel.snapshots.StoreSnapshot(result.Superseded)
	kernel.snapshots.StoreSnapshot(result.Superseding)
	kernel.objects.putObject(snapshotObject(result.Superseded, result.Superseded.CreatedBy, capabilityRefs))
	kernel.objects.putObject(snapshotObject(result.Superseding, result.Superseding.CreatedBy, capabilityRefs))
	return SyscallResult{Success: true, SyscallName: request.Name, ObjectID: oldID, JournalEvent: event.EventID, Output: result}
}

func handleSnapshotExpire(kernel *Kernel, request SyscallRequest, capabilityRefs []string) SyscallResult {
	snapshotID := stringInput(request.Input, "snapshot_id")
	snapshot, ok := kernel.snapshots.GetSnapshot(snapshotID)
	if !ok || snapshot.WorkspaceID != request.WorkspaceID {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: ErrObjectNotFound}
	}
	if snapshot.Status == snapshots.StatusExpired || snapshot.Status == snapshots.StatusSuperseded {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: ErrInvalidStateTransition}
	}
	expired := snapshot.Clone()
	now := kernel.clock.Now()
	expired.Status = snapshots.StatusExpired
	expired.ExpiredAt = &now
	event, err := kernel.appendSnapshotEvent(request, JournalEventSnapshotExpired, expired.SnapshotID, expired.CaseID, capabilityRefs, expired, expired.AllRefs())
	if err != nil {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: err}
	}
	expired.JournalRefs = append(expired.JournalRefs, event.EventID)
	kernel.snapshots.StoreSnapshot(expired)
	kernel.objects.putObject(snapshotObject(expired, expired.CreatedBy, capabilityRefs))
	return SyscallResult{Success: true, SyscallName: request.Name, ObjectID: snapshotID, JournalEvent: event.EventID, Output: expired}
}

func handleSnapshotDiff(kernel *Kernel, request SyscallRequest, _ []string) SyscallResult {
	leftID := stringInput(request.Input, "left_snapshot_id")
	rightID := stringInput(request.Input, "right_snapshot_id")
	diff, err := kernel.snapshots.DiffSnapshots(leftID, rightID, kernel.ids.NextID("snapshot-diff"), kernel.clock.Now(), mapInput(request.Input, "metadata"))
	if err != nil {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: mapSnapshotError(err)}
	}
	if diff.WorkspaceID != request.WorkspaceID {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: ErrObjectNotFound}
	}
	return SyscallResult{Success: true, SyscallName: request.Name, ObjectID: diff.DiffID, Output: diff}
}

func handleSnapshotRestoreSeed(kernel *Kernel, request SyscallRequest, capabilityRefs []string) SyscallResult {
	snapshotID := stringInput(request.Input, "snapshot_id")
	snapshot, ok := kernel.snapshots.GetSnapshot(snapshotID)
	if !ok || snapshot.WorkspaceID != request.WorkspaceID {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: ErrObjectNotFound}
	}
	if snapshot.Status == snapshots.StatusExpired || snapshot.Status == snapshots.StatusSuperseded {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: ErrInvalidStateTransition}
	}
	seed, err := snapshots.NewRestoreSeed(kernel.ids.NextID("restore-seed"), snapshot, kernel.clock.Now(), mapInput(request.Input, "metadata"))
	if err != nil {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: mapSnapshotError(err)}
	}
	event, err := kernel.appendSnapshotEvent(request, JournalEventSnapshotRestoreSeedCreated, seed.RestoreSeedID, seed.CaseID, capabilityRefs, seed, []string{seed.SnapshotID})
	if err != nil {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: err}
	}
	seed.JournalRefs = []string{event.EventID}
	updatedSnapshot := snapshot.Clone()
	updatedSnapshot.Status = snapshots.StatusRestoreSeedCreated
	updatedSnapshot.JournalRefs = append(updatedSnapshot.JournalRefs, event.EventID)
	kernel.snapshots.StoreRestoreSeed(seed)
	kernel.snapshots.StoreSnapshot(updatedSnapshot)
	kernel.objects.putObject(snapshotObject(updatedSnapshot, updatedSnapshot.CreatedBy, capabilityRefs))
	kernel.objects.putObject(restoreSeedObject(seed, request.ActorID, capabilityRefs))
	return SyscallResult{Success: true, SyscallName: request.Name, ObjectID: seed.RestoreSeedID, JournalEvent: event.EventID, Output: seed}
}

func snapshotInputFromRequest(kernel *Kernel, request SyscallRequest) (snapshots.SnapshotInput, error) {
	if request.ActorID == "" || request.WorkspaceID == "" {
		return snapshots.SnapshotInput{}, ErrInvalidInput
	}
	if inputWorkspaceID := stringInput(request.Input, "workspace_id"); inputWorkspaceID != "" && inputWorkspaceID != request.WorkspaceID {
		return snapshots.SnapshotInput{}, ErrInvalidInput
	}
	caseID := stringInput(request.Input, "case_id")
	if caseID == "" {
		caseID = request.CaseID
	}
	return snapshots.SnapshotInput{
		SnapshotID:            kernel.ids.NextID("snapshot"),
		SnapshotType:          snapshots.SnapshotType(stringInput(request.Input, "snapshot_type")),
		WorkspaceID:           request.WorkspaceID,
		CaseID:                caseID,
		SourceObjectRefs:      stringSliceInputDefault(request.Input, "source_object_refs"),
		SourceRefs:            stringSliceInputDefault(request.Input, "source_refs"),
		PalaceRouteRefs:       stringSliceInputDefault(request.Input, "palace_route_refs"),
		SubmittedObjectRefs:   stringSliceInputDefault(request.Input, "submitted_object_refs"),
		AdmittedObjectRefs:    stringSliceInputDefault(request.Input, "admitted_object_refs"),
		RejectedObjectRefs:    stringSliceInputDefault(request.Input, "rejected_object_refs"),
		SemanticOperationRefs: stringSliceInputDefault(request.Input, "semantic_operation_refs"),
		ContradictionRefs:     stringSliceInputDefault(request.Input, "contradiction_refs"),
		SupersessionRefs:      stringSliceInputDefault(request.Input, "supersession_refs"),
		DerivedObjectRefs:     stringSliceInputDefault(request.Input, "derived_object_refs"),
		ContextBlockRefs:      stringSliceInputDefault(request.Input, "context_block_refs"),
		TokenHashRefs:         stringSliceInputDefault(request.Input, "token_hash_refs"),
		KVManifestRefs:        stringSliceInputDefault(request.Input, "kv_manifest_refs"),
		Summary:               stringInput(request.Input, "summary"),
		Metadata:              mapInput(request.Input, "metadata"),
		Seal:                  boolInput(request.Input, "seal"),
		CreatedBy:             request.ActorID,
		CreatedAt:             kernel.clock.Now(),
	}, nil
}

func (k *Kernel) appendSnapshotEvent(request SyscallRequest, eventType, objectID, caseID string, capabilityRefs []string, output any, objectRefs []string) (JournalEvent, error) {
	refs := append([]string{objectID}, objectRefs...)
	return k.journal.Append(JournalEvent{
		EventType:      eventType,
		Timestamp:      k.clock.Now(),
		WorkspaceID:    request.WorkspaceID,
		CaseID:         caseID,
		ActorID:        request.ActorID,
		SyscallName:    request.Name,
		InputHash:      hashValue(request.Input),
		OutputHash:     hashValue(output),
		ObjectRefs:     snapshotsNormalizeRefs(refs),
		CapabilityRefs: capabilityRefs,
		Result:         SyscallResultCommitted,
	})
}

func snapshotObject(snapshot snapshots.Snapshot, ownerID string, capabilityRefs []string) KernelObject {
	return KernelObject{
		ObjectID:       snapshot.SnapshotID,
		ObjectType:     ObjectTypeSnapshot,
		WorkspaceID:    snapshot.WorkspaceID,
		OwnerID:        ownerID,
		AuthorityLevel: AuthorityShape,
		State: map[string]any{
			"snapshot_type":             string(snapshot.SnapshotType),
			"status":                    string(snapshot.Status),
			"case_id":                   snapshot.CaseID,
			"shape_hash":                snapshot.ShapeHash,
			"source_hash":               snapshot.SourceHash,
			"source_object_refs":        append([]string(nil), snapshot.SourceObjectRefs...),
			"source_refs":               append([]string(nil), snapshot.SourceRefs...),
			"palace_route_refs":         append([]string(nil), snapshot.PalaceRouteRefs...),
			"submitted_object_refs":     append([]string(nil), snapshot.SubmittedObjectRefs...),
			"admitted_object_refs":      append([]string(nil), snapshot.AdmittedObjectRefs...),
			"rejected_object_refs":      append([]string(nil), snapshot.RejectedObjectRefs...),
			"semantic_operation_refs":   append([]string(nil), snapshot.SemanticOperationRefs...),
			"contradiction_refs":        append([]string(nil), snapshot.ContradictionRefs...),
			"supersession_refs":         append([]string(nil), snapshot.SupersessionRefs...),
			"derived_object_refs":       append([]string(nil), snapshot.DerivedObjectRefs...),
			"context_block_refs":        append([]string(nil), snapshot.ContextBlockRefs...),
			"token_hash_refs":           append([]string(nil), snapshot.TokenHashRefs...),
			"kv_manifest_refs":          append([]string(nil), snapshot.KVManifestRefs...),
			"superseded_by":             snapshot.SupersededBy,
			"supersedes":                append([]string(nil), snapshot.Supersedes...),
			"is_canonical_truth":        false,
			"is_context_block":          false,
			"is_deterministic_kv_cache": false,
		},
		SourceRefs:      append([]string(nil), snapshot.SourceRefs...),
		CapabilityScope: append([]string(nil), capabilityRefs...),
		CreatedAt:       snapshot.CreatedAt,
		UpdatedAt:       snapshot.CreatedAt,
		JournalRefs:     append([]string(nil), snapshot.JournalRefs...),
	}
}

func restoreSeedObject(seed snapshots.RestoreSeed, ownerID string, capabilityRefs []string) KernelObject {
	return KernelObject{
		ObjectID:       seed.RestoreSeedID,
		ObjectType:     ObjectTypeRestoreSeed,
		WorkspaceID:    seed.WorkspaceID,
		OwnerID:        ownerID,
		AuthorityLevel: AuthorityProposal,
		State: map[string]any{
			"snapshot_id":                seed.SnapshotID,
			"source_snapshot_type":       string(seed.SourceSnapshotType),
			"source_shape_hash":          seed.SourceShapeHash,
			"recommended_source_refs":    append([]string(nil), seed.RecommendedSourceRefs...),
			"recommended_operation_refs": append([]string(nil), seed.RecommendedOperationRefs...),
			"recommended_case_refs":      append([]string(nil), seed.RecommendedCaseRefs...),
			"is_canonical_truth":         false,
			"is_context_block":           false,
		},
		SourceRefs:      append([]string{seed.SnapshotID}, seed.RecommendedSourceRefs...),
		CapabilityScope: append([]string(nil), capabilityRefs...),
		CreatedAt:       seed.CreatedAt,
		UpdatedAt:       seed.CreatedAt,
		JournalRefs:     append([]string(nil), seed.JournalRefs...),
	}
}

func mapSnapshotError(err error) error {
	switch {
	case errors.Is(err, snapshots.ErrSnapshotNotFound), errors.Is(err, snapshots.ErrRestoreSeedNotFound):
		return ErrObjectNotFound
	case errors.Is(err, snapshots.ErrInvalidStateTransition), errors.Is(err, snapshots.ErrImmutableSnapshot):
		return ErrInvalidStateTransition
	case errors.Is(err, snapshots.ErrWorkspaceMismatch):
		return ErrObjectNotFound
	case errors.Is(err, snapshots.ErrInvalidSnapshot),
		errors.Is(err, snapshots.ErrInvalidSnapshotType),
		errors.Is(err, snapshots.ErrInvalidSnapshotStatus),
		errors.Is(err, snapshots.ErrInvalidSnapshotDiff),
		errors.Is(err, snapshots.ErrInvalidRestoreSeed),
		errors.Is(err, snapshots.ErrSnapshotContentRejected):
		return ErrInvalidInput
	default:
		return err
	}
}

func boolInput(input map[string]any, key string) bool {
	value, _ := input[key].(bool)
	return value
}

func snapshotsNormalizeRefs(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}
