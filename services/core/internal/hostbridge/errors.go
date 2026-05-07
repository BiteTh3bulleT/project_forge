package hostbridge

import "errors"

var (
	ErrSnapshotNil       = errors.New("host diagnostic snapshot is nil")
	ErrReportDirRequired = errors.New("host diagnostic report directory is required")
)
