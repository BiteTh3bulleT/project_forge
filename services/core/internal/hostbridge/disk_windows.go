//go:build windows

package hostbridge

import "fmt"

func (s *Service) readDisk(snapshot *Snapshot) DiskDiagnostics {
	out := DiskDiagnostics{Path: s.storageRoot, PressureLevel: PressureUnavailable}
	addSourceError(snapshot, "disk.statfs", fmt.Errorf("disk statfs unavailable on windows hostbridge build"))
	return out
}
