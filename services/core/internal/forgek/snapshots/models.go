package snapshots

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
	"time"
)

type SnapshotInput struct {
	SnapshotID            string
	SnapshotType          SnapshotType
	WorkspaceID           string
	CaseID                string
	SourceObjectRefs      []string
	SourceRefs            []string
	PalaceRouteRefs       []string
	SubmittedObjectRefs   []string
	AdmittedObjectRefs    []string
	RejectedObjectRefs    []string
	SemanticOperationRefs []string
	ContradictionRefs     []string
	SupersessionRefs      []string
	DerivedObjectRefs     []string
	ContextBlockRefs      []string
	TokenHashRefs         []string
	KVManifestRefs        []string
	Summary               string
	Metadata              map[string]any
	Seal                  bool
	CreatedBy             string
	CreatedAt             time.Time
}

type Snapshot struct {
	SnapshotID            string         `json:"snapshot_id"`
	SnapshotType          SnapshotType   `json:"snapshot_type"`
	WorkspaceID           string         `json:"workspace_id"`
	CaseID                string         `json:"case_id,omitempty"`
	Status                SnapshotStatus `json:"status"`
	SourceObjectRefs      []string       `json:"source_object_refs,omitempty"`
	SourceRefs            []string       `json:"source_refs,omitempty"`
	PalaceRouteRefs       []string       `json:"palace_route_refs,omitempty"`
	SubmittedObjectRefs   []string       `json:"submitted_object_refs,omitempty"`
	AdmittedObjectRefs    []string       `json:"admitted_object_refs,omitempty"`
	RejectedObjectRefs    []string       `json:"rejected_object_refs,omitempty"`
	SemanticOperationRefs []string       `json:"semantic_operation_refs,omitempty"`
	ContradictionRefs     []string       `json:"contradiction_refs,omitempty"`
	SupersessionRefs      []string       `json:"supersession_refs,omitempty"`
	DerivedObjectRefs     []string       `json:"derived_object_refs,omitempty"`
	ContextBlockRefs      []string       `json:"context_block_refs,omitempty"`
	TokenHashRefs         []string       `json:"token_hash_refs,omitempty"`
	KVManifestRefs        []string       `json:"kv_manifest_refs,omitempty"`
	Summary               string         `json:"summary,omitempty"`
	ShapeHash             string         `json:"shape_hash"`
	SourceHash            string         `json:"source_hash"`
	CreatedBy             string         `json:"created_by"`
	CreatedAt             time.Time      `json:"created_at"`
	SealedAt              *time.Time     `json:"sealed_at,omitempty"`
	SupersededBy          string         `json:"superseded_by,omitempty"`
	Supersedes            []string       `json:"supersedes,omitempty"`
	ExpiredAt             *time.Time     `json:"expired_at,omitempty"`
	JournalRefs           []string       `json:"journal_refs,omitempty"`
	Metadata              map[string]any `json:"metadata,omitempty"`
}

type SnapshotSupersessionResult struct {
	Superseded  Snapshot `json:"superseded"`
	Superseding Snapshot `json:"superseding"`
}

func NewSnapshot(input SnapshotInput) (Snapshot, error) {
	if input.SnapshotID == "" || input.WorkspaceID == "" || input.CreatedBy == "" {
		return Snapshot{}, ErrInvalidSnapshot
	}
	if !ValidSnapshotType(input.SnapshotType) {
		return Snapshot{}, ErrInvalidSnapshotType
	}
	if containsRawContent(input.Metadata) {
		return Snapshot{}, ErrSnapshotContentRejected
	}

	normalized := input
	normalizeSnapshotInput(&normalized)
	if !hasShapeRefs(normalized) && normalized.SnapshotType != SnapshotTypeWorkspaceSnapshot {
		return Snapshot{}, ErrInvalidSnapshot
	}

	status := StatusDraft
	var sealedAt *time.Time
	if normalized.Seal {
		status = StatusSealed
		sealed := normalized.CreatedAt
		sealedAt = &sealed
	}
	snapshot := Snapshot{
		SnapshotID:            normalized.SnapshotID,
		SnapshotType:          normalized.SnapshotType,
		WorkspaceID:           normalized.WorkspaceID,
		CaseID:                normalized.CaseID,
		Status:                status,
		SourceObjectRefs:      normalized.SourceObjectRefs,
		SourceRefs:            normalized.SourceRefs,
		PalaceRouteRefs:       normalized.PalaceRouteRefs,
		SubmittedObjectRefs:   normalized.SubmittedObjectRefs,
		AdmittedObjectRefs:    normalized.AdmittedObjectRefs,
		RejectedObjectRefs:    normalized.RejectedObjectRefs,
		SemanticOperationRefs: normalized.SemanticOperationRefs,
		ContradictionRefs:     normalized.ContradictionRefs,
		SupersessionRefs:      normalized.SupersessionRefs,
		DerivedObjectRefs:     normalized.DerivedObjectRefs,
		ContextBlockRefs:      normalized.ContextBlockRefs,
		TokenHashRefs:         normalized.TokenHashRefs,
		KVManifestRefs:        normalized.KVManifestRefs,
		Summary:               normalized.Summary,
		CreatedBy:             normalized.CreatedBy,
		CreatedAt:             normalized.CreatedAt,
		SealedAt:              sealedAt,
		Metadata:              cloneMap(normalized.Metadata),
	}
	snapshot.SourceHash = SourceHash(snapshot)
	snapshot.ShapeHash = ShapeHash(snapshot)
	return snapshot, nil
}

func (s Snapshot) Clone() Snapshot {
	s.SourceObjectRefs = cloneStrings(s.SourceObjectRefs)
	s.SourceRefs = cloneStrings(s.SourceRefs)
	s.PalaceRouteRefs = cloneStrings(s.PalaceRouteRefs)
	s.SubmittedObjectRefs = cloneStrings(s.SubmittedObjectRefs)
	s.AdmittedObjectRefs = cloneStrings(s.AdmittedObjectRefs)
	s.RejectedObjectRefs = cloneStrings(s.RejectedObjectRefs)
	s.SemanticOperationRefs = cloneStrings(s.SemanticOperationRefs)
	s.ContradictionRefs = cloneStrings(s.ContradictionRefs)
	s.SupersessionRefs = cloneStrings(s.SupersessionRefs)
	s.DerivedObjectRefs = cloneStrings(s.DerivedObjectRefs)
	s.ContextBlockRefs = cloneStrings(s.ContextBlockRefs)
	s.TokenHashRefs = cloneStrings(s.TokenHashRefs)
	s.KVManifestRefs = cloneStrings(s.KVManifestRefs)
	s.Supersedes = cloneStrings(s.Supersedes)
	s.JournalRefs = cloneStrings(s.JournalRefs)
	s.Metadata = cloneMap(s.Metadata)
	if s.SealedAt != nil {
		sealed := *s.SealedAt
		s.SealedAt = &sealed
	}
	if s.ExpiredAt != nil {
		expired := *s.ExpiredAt
		s.ExpiredAt = &expired
	}
	return s
}

func (s Snapshot) AllRefs() []string {
	refs := make([]string, 0)
	for _, values := range [][]string{
		s.SourceObjectRefs,
		s.SourceRefs,
		s.PalaceRouteRefs,
		s.SubmittedObjectRefs,
		s.AdmittedObjectRefs,
		s.RejectedObjectRefs,
		s.SemanticOperationRefs,
		s.ContradictionRefs,
		s.SupersessionRefs,
		s.DerivedObjectRefs,
		s.ContextBlockRefs,
		s.TokenHashRefs,
		s.KVManifestRefs,
	} {
		refs = append(refs, values...)
	}
	return normalizeRefs(refs)
}

func (s Snapshot) IsCanonicalTruth() bool {
	return false
}

func normalizeSnapshotInput(input *SnapshotInput) {
	input.SourceObjectRefs = normalizeRefs(input.SourceObjectRefs)
	input.SourceRefs = normalizeRefs(input.SourceRefs)
	input.PalaceRouteRefs = normalizeRefs(input.PalaceRouteRefs)
	input.SubmittedObjectRefs = normalizeRefs(input.SubmittedObjectRefs)
	input.AdmittedObjectRefs = normalizeRefs(input.AdmittedObjectRefs)
	input.RejectedObjectRefs = normalizeRefs(input.RejectedObjectRefs)
	input.SemanticOperationRefs = normalizeRefs(input.SemanticOperationRefs)
	input.ContradictionRefs = normalizeRefs(input.ContradictionRefs)
	input.SupersessionRefs = normalizeRefs(input.SupersessionRefs)
	input.DerivedObjectRefs = normalizeRefs(input.DerivedObjectRefs)
	input.ContextBlockRefs = normalizeRefs(input.ContextBlockRefs)
	input.TokenHashRefs = normalizeRefs(input.TokenHashRefs)
	input.KVManifestRefs = normalizeRefs(input.KVManifestRefs)
	input.Metadata = cloneMap(input.Metadata)
}

func hasShapeRefs(input SnapshotInput) bool {
	return len(input.SourceObjectRefs)+len(input.SourceRefs)+len(input.PalaceRouteRefs)+
		len(input.SubmittedObjectRefs)+len(input.AdmittedObjectRefs)+len(input.RejectedObjectRefs)+
		len(input.SemanticOperationRefs)+len(input.ContradictionRefs)+len(input.SupersessionRefs)+
		len(input.DerivedObjectRefs)+len(input.ContextBlockRefs)+len(input.TokenHashRefs)+
		len(input.KVManifestRefs) > 0
}

func SourceHash(snapshot Snapshot) string {
	return stableHash(map[string]any{
		"source_object_refs": normalizeRefs(snapshot.SourceObjectRefs),
		"source_refs":        normalizeRefs(snapshot.SourceRefs),
	})
}

func ShapeHash(snapshot Snapshot) string {
	return stableHash(map[string]any{
		"snapshot_type":           snapshot.SnapshotType,
		"workspace_id":            snapshot.WorkspaceID,
		"case_id":                 snapshot.CaseID,
		"source_hash":             SourceHash(snapshot),
		"source_object_refs":      normalizeRefs(snapshot.SourceObjectRefs),
		"source_refs":             normalizeRefs(snapshot.SourceRefs),
		"palace_route_refs":       normalizeRefs(snapshot.PalaceRouteRefs),
		"submitted_object_refs":   normalizeRefs(snapshot.SubmittedObjectRefs),
		"admitted_object_refs":    normalizeRefs(snapshot.AdmittedObjectRefs),
		"rejected_object_refs":    normalizeRefs(snapshot.RejectedObjectRefs),
		"semantic_operation_refs": normalizeRefs(snapshot.SemanticOperationRefs),
		"contradiction_refs":      normalizeRefs(snapshot.ContradictionRefs),
		"supersession_refs":       normalizeRefs(snapshot.SupersessionRefs),
		"derived_object_refs":     normalizeRefs(snapshot.DerivedObjectRefs),
		"context_block_refs":      normalizeRefs(snapshot.ContextBlockRefs),
		"token_hash_refs":         normalizeRefs(snapshot.TokenHashRefs),
		"kv_manifest_refs":        normalizeRefs(snapshot.KVManifestRefs),
		"summary":                 snapshot.Summary,
		"metadata":                cloneMap(snapshot.Metadata),
	})
}

func StableJSON(value any) string {
	encoded, _ := json.Marshal(value)
	return string(encoded)
}

func stableHash(value any) string {
	encoded, _ := json.Marshal(value)
	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:])
}

func normalizeRefs(values []string) []string {
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

func cloneStrings(values []string) []string {
	return append([]string(nil), values...)
}

func cloneMap(in map[string]any) map[string]any {
	if in == nil {
		return nil
	}
	out := make(map[string]any, len(in))
	for key, value := range in {
		switch typed := value.(type) {
		case []string:
			out[key] = cloneStrings(typed)
		case map[string]string:
			nested := make(map[string]string, len(typed))
			for nestedKey, nestedValue := range typed {
				nested[nestedKey] = nestedValue
			}
			out[key] = nested
		case map[string]any:
			out[key] = cloneMap(typed)
		default:
			out[key] = value
		}
	}
	return out
}

func containsRawContent(metadata map[string]any) bool {
	for key := range metadata {
		switch key {
		case "raw_content", "canonical_content", "full_content", "content_blob":
			return true
		}
	}
	return false
}
