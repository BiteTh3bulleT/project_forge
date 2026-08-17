//go:build linux

package store

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

var ErrDataDirLocked = errors.New("FORGE data directory is locked by a running daemon or recovery process")

type ProcessLock struct {
	file *os.File
}

func acquireDaemonProcessLock(dataDir string) (*ProcessLock, error) {
	return AcquireOfflineRecoveryLock(dataDir)
}

// AcquireOfflineRecoveryLock excludes the daemon and all other recovery
// processes for the complete validate/stage/swap operation. Linux flock is
// process-scoped and releases automatically after a crash.
func AcquireOfflineRecoveryLock(dataDir string) (*ProcessLock, error) {
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return nil, fmt.Errorf("mkdir data dir: %w", err)
	}
	lockPath := filepath.Join(dataDir, "forge.sqlite.lock")
	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open recovery lock: %w", err)
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		_ = f.Close()
		if errors.Is(err, syscall.EWOULDBLOCK) || errors.Is(err, syscall.EAGAIN) {
			return nil, ErrDataDirLocked
		}
		return nil, fmt.Errorf("acquire recovery lock: %w", err)
	}
	if err := f.Truncate(0); err == nil {
		_, _ = fmt.Fprintf(f, "pid=%d\n", os.Getpid())
		_ = f.Sync()
	}
	return &ProcessLock{file: f}, nil
}

func (l *ProcessLock) Close() error {
	if l == nil || l.file == nil {
		return nil
	}
	err := syscall.Flock(int(l.file.Fd()), syscall.LOCK_UN)
	err = errors.Join(err, l.file.Close())
	l.file = nil
	return err
}
