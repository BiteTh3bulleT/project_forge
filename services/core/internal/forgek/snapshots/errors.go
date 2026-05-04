package snapshots

import "errors"

var (
	ErrInvalidSnapshot         = errors.New("invalid snapshot")
	ErrInvalidSnapshotType     = errors.New("invalid snapshot type")
	ErrInvalidSnapshotStatus   = errors.New("invalid snapshot status")
	ErrSnapshotNotFound        = errors.New("snapshot not found")
	ErrRestoreSeedNotFound     = errors.New("restore seed not found")
	ErrImmutableSnapshot       = errors.New("snapshot is immutable")
	ErrInvalidStateTransition  = errors.New("invalid snapshot state transition")
	ErrWorkspaceMismatch       = errors.New("snapshot workspace mismatch")
	ErrInvalidSnapshotDiff     = errors.New("invalid snapshot diff")
	ErrInvalidRestoreSeed      = errors.New("invalid restore seed")
	ErrSnapshotContentRejected = errors.New("snapshot stores refs and summaries, not raw canonical content")
)
