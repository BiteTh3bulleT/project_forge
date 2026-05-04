package snapshots

import "time"

type SnapshotDiff struct {
	DiffID          string         `json:"diff_id"`
	LeftSnapshotID  string         `json:"left_snapshot_id"`
	RightSnapshotID string         `json:"right_snapshot_id"`
	WorkspaceID     string         `json:"workspace_id"`
	AddedRefs       []string       `json:"added_refs,omitempty"`
	RemovedRefs     []string       `json:"removed_refs,omitempty"`
	UnchangedRefs   []string       `json:"unchanged_refs,omitempty"`
	ChangedFields   []string       `json:"changed_fields,omitempty"`
	Summary         string         `json:"summary,omitempty"`
	CreatedAt       time.Time      `json:"created_at"`
	Metadata        map[string]any `json:"metadata,omitempty"`
}

func DiffSnapshots(diffID string, left, right Snapshot, createdAt time.Time, metadata map[string]any) (SnapshotDiff, error) {
	if diffID == "" || left.SnapshotID == "" || right.SnapshotID == "" {
		return SnapshotDiff{}, ErrInvalidSnapshotDiff
	}
	if left.WorkspaceID == "" || left.WorkspaceID != right.WorkspaceID {
		return SnapshotDiff{}, ErrWorkspaceMismatch
	}
	leftSet := refSet(left.AllRefs())
	rightSet := refSet(right.AllRefs())

	added := make([]string, 0)
	removed := make([]string, 0)
	unchanged := make([]string, 0)
	for ref := range rightSet {
		if _, ok := leftSet[ref]; !ok {
			added = append(added, ref)
		} else {
			unchanged = append(unchanged, ref)
		}
	}
	for ref := range leftSet {
		if _, ok := rightSet[ref]; !ok {
			removed = append(removed, ref)
		}
	}
	added = normalizeRefs(added)
	removed = normalizeRefs(removed)
	unchanged = normalizeRefs(unchanged)

	return SnapshotDiff{
		DiffID:          diffID,
		LeftSnapshotID:  left.SnapshotID,
		RightSnapshotID: right.SnapshotID,
		WorkspaceID:     left.WorkspaceID,
		AddedRefs:       added,
		RemovedRefs:     removed,
		UnchangedRefs:   unchanged,
		ChangedFields:   changedFields(left, right),
		Summary:         "deterministic snapshot shape diff",
		CreatedAt:       createdAt,
		Metadata:        cloneMap(metadata),
	}, nil
}

func changedFields(left, right Snapshot) []string {
	fields := make([]string, 0)
	addChanged := func(name string, changed bool) {
		if changed {
			fields = append(fields, name)
		}
	}
	addChanged("snapshot_type", left.SnapshotType != right.SnapshotType)
	addChanged("workspace_id", left.WorkspaceID != right.WorkspaceID)
	addChanged("case_id", left.CaseID != right.CaseID)
	addChanged("status", left.Status != right.Status)
	addChanged("summary", left.Summary != right.Summary)
	addChanged("shape_hash", left.ShapeHash != right.ShapeHash)
	addChanged("source_hash", left.SourceHash != right.SourceHash)
	addChanged("source_object_refs", StableJSON(left.SourceObjectRefs) != StableJSON(right.SourceObjectRefs))
	addChanged("source_refs", StableJSON(left.SourceRefs) != StableJSON(right.SourceRefs))
	addChanged("palace_route_refs", StableJSON(left.PalaceRouteRefs) != StableJSON(right.PalaceRouteRefs))
	addChanged("submitted_object_refs", StableJSON(left.SubmittedObjectRefs) != StableJSON(right.SubmittedObjectRefs))
	addChanged("admitted_object_refs", StableJSON(left.AdmittedObjectRefs) != StableJSON(right.AdmittedObjectRefs))
	addChanged("rejected_object_refs", StableJSON(left.RejectedObjectRefs) != StableJSON(right.RejectedObjectRefs))
	addChanged("semantic_operation_refs", StableJSON(left.SemanticOperationRefs) != StableJSON(right.SemanticOperationRefs))
	addChanged("contradiction_refs", StableJSON(left.ContradictionRefs) != StableJSON(right.ContradictionRefs))
	addChanged("supersession_refs", StableJSON(left.SupersessionRefs) != StableJSON(right.SupersessionRefs))
	addChanged("derived_object_refs", StableJSON(left.DerivedObjectRefs) != StableJSON(right.DerivedObjectRefs))
	addChanged("context_block_refs", StableJSON(left.ContextBlockRefs) != StableJSON(right.ContextBlockRefs))
	addChanged("token_hash_refs", StableJSON(left.TokenHashRefs) != StableJSON(right.TokenHashRefs))
	addChanged("kv_manifest_refs", StableJSON(left.KVManifestRefs) != StableJSON(right.KVManifestRefs))
	return normalizeRefs(fields)
}

func refSet(values []string) map[string]struct{} {
	out := make(map[string]struct{}, len(values))
	for _, value := range values {
		out[value] = struct{}{}
	}
	return out
}
