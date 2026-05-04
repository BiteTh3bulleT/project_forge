package snapshots

type SnapshotType string

const (
	SnapshotTypeSemanticSnapshot       SnapshotType = "SEMANTIC_SNAPSHOT"
	SnapshotTypeCaseSnapshot           SnapshotType = "CASE_SNAPSHOT"
	SnapshotTypeContextRestoreSnapshot SnapshotType = "CONTEXT_RESTORE_SNAPSHOT"
	SnapshotTypePalaceRouteSnapshot    SnapshotType = "PALACE_ROUTE_SNAPSHOT"
	SnapshotTypeWorkspaceSnapshot      SnapshotType = "WORKSPACE_SNAPSHOT"
	SnapshotTypeDecisionSnapshot       SnapshotType = "DECISION_SNAPSHOT"
	SnapshotTypeKVShapeSnapshot        SnapshotType = "KV_SHAPE_SNAPSHOT"
	SnapshotTypeRuntimeSnapshot        SnapshotType = "RUNTIME_SNAPSHOT"
)

type SnapshotStatus string

const (
	StatusDraft              SnapshotStatus = "DRAFT"
	StatusSealed             SnapshotStatus = "SEALED"
	StatusSuperseded         SnapshotStatus = "SUPERSEDED"
	StatusExpired            SnapshotStatus = "EXPIRED"
	StatusRestoreSeedCreated SnapshotStatus = "RESTORE_SEED_CREATED"
)

func ValidSnapshotType(snapshotType SnapshotType) bool {
	switch snapshotType {
	case SnapshotTypeSemanticSnapshot,
		SnapshotTypeCaseSnapshot,
		SnapshotTypeContextRestoreSnapshot,
		SnapshotTypePalaceRouteSnapshot,
		SnapshotTypeWorkspaceSnapshot,
		SnapshotTypeDecisionSnapshot,
		SnapshotTypeKVShapeSnapshot,
		SnapshotTypeRuntimeSnapshot:
		return true
	default:
		return false
	}
}

func ValidSnapshotStatus(status SnapshotStatus) bool {
	switch status {
	case StatusDraft, StatusSealed, StatusSuperseded, StatusExpired, StatusRestoreSeedCreated:
		return true
	default:
		return false
	}
}
