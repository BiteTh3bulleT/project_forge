package snapshots

import "time"

type RestoreSeed struct {
	RestoreSeedID            string         `json:"restore_seed_id"`
	SnapshotID               string         `json:"snapshot_id"`
	WorkspaceID              string         `json:"workspace_id"`
	CaseID                   string         `json:"case_id,omitempty"`
	SourceSnapshotType       SnapshotType   `json:"source_snapshot_type"`
	SourceShapeHash          string         `json:"source_shape_hash"`
	RecommendedSourceRefs    []string       `json:"recommended_source_refs,omitempty"`
	RecommendedOperationRefs []string       `json:"recommended_operation_refs,omitempty"`
	RecommendedCaseRefs      []string       `json:"recommended_case_refs,omitempty"`
	Summary                  string         `json:"summary,omitempty"`
	CreatedAt                time.Time      `json:"created_at"`
	JournalRefs              []string       `json:"journal_refs,omitempty"`
	Metadata                 map[string]any `json:"metadata,omitempty"`
}

func NewRestoreSeed(seedID string, snapshot Snapshot, createdAt time.Time, metadata map[string]any) (RestoreSeed, error) {
	if seedID == "" || snapshot.SnapshotID == "" || snapshot.WorkspaceID == "" || snapshot.ShapeHash == "" {
		return RestoreSeed{}, ErrInvalidRestoreSeed
	}
	caseRefs := []string(nil)
	if snapshot.CaseID != "" {
		caseRefs = []string{snapshot.CaseID}
	}
	return RestoreSeed{
		RestoreSeedID:            seedID,
		SnapshotID:               snapshot.SnapshotID,
		WorkspaceID:              snapshot.WorkspaceID,
		CaseID:                   snapshot.CaseID,
		SourceSnapshotType:       snapshot.SnapshotType,
		SourceShapeHash:          snapshot.ShapeHash,
		RecommendedSourceRefs:    restoreSourceRefs(snapshot),
		RecommendedOperationRefs: restoreOperationRefs(snapshot),
		RecommendedCaseRefs:      caseRefs,
		Summary:                  snapshot.Summary,
		CreatedAt:                createdAt,
		Metadata:                 cloneMap(metadata),
	}, nil
}

func (s RestoreSeed) Clone() RestoreSeed {
	s.RecommendedSourceRefs = cloneStrings(s.RecommendedSourceRefs)
	s.RecommendedOperationRefs = cloneStrings(s.RecommendedOperationRefs)
	s.RecommendedCaseRefs = cloneStrings(s.RecommendedCaseRefs)
	s.JournalRefs = cloneStrings(s.JournalRefs)
	s.Metadata = cloneMap(s.Metadata)
	return s
}

func (s RestoreSeed) IsCanonicalTruth() bool {
	return false
}

func (s RestoreSeed) IsContextBlock() bool {
	return false
}

func restoreSourceRefs(snapshot Snapshot) []string {
	refs := make([]string, 0)
	refs = append(refs, snapshot.SourceObjectRefs...)
	refs = append(refs, snapshot.SourceRefs...)
	refs = append(refs, snapshot.PalaceRouteRefs...)
	refs = append(refs, snapshot.SubmittedObjectRefs...)
	refs = append(refs, snapshot.AdmittedObjectRefs...)
	refs = append(refs, snapshot.RejectedObjectRefs...)
	refs = append(refs, snapshot.DerivedObjectRefs...)
	return normalizeRefs(refs)
}

func restoreOperationRefs(snapshot Snapshot) []string {
	refs := make([]string, 0)
	refs = append(refs, snapshot.SemanticOperationRefs...)
	refs = append(refs, snapshot.ContradictionRefs...)
	refs = append(refs, snapshot.SupersessionRefs...)
	return normalizeRefs(refs)
}
