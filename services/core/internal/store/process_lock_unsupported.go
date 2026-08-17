//go:build !linux

package store

import "errors"

var (
	ErrDataDirLocked              = errors.New("FORGE data directory is locked by a running daemon or recovery process")
	ErrOfflineRecoveryUnsupported = errors.New("offline FORGE-K recovery is supported only on Linux/NixOS")
)

type ProcessLock struct{}

// Normal daemon startup remains portable. Offline replacement is rejected on
// platforms where this build cannot provide the required process-scoped lock.
func acquireDaemonProcessLock(string) (*ProcessLock, error) { return &ProcessLock{}, nil }

func AcquireOfflineRecoveryLock(string) (*ProcessLock, error) {
	return nil, ErrOfflineRecoveryUnsupported
}

func (l *ProcessLock) Close() error { return nil }
