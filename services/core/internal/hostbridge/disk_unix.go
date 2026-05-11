//go:build !windows

package hostbridge

import "syscall"

func (s *Service) readDisk(snapshot *Snapshot) DiskDiagnostics {
	out := DiskDiagnostics{Path: s.storageRoot, PressureLevel: PressureUnavailable}
	var stat syscall.Statfs_t
	if err := syscall.Statfs(s.storageRoot, &stat); err != nil {
		addSourceError(snapshot, "disk.statfs", err)
		return out
	}
	total := stat.Blocks * uint64(stat.Bsize)
	free := stat.Bavail * uint64(stat.Bsize)
	used := uint64(0)
	if total >= free {
		used = total - free
	}
	out.TotalBytes = total
	out.FreeBytes = free
	out.UsedBytes = used
	out.PressureLevel = ClassifyDiskPressure(total, free)
	return out
}
