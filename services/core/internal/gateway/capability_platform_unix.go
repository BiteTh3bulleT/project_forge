//go:build !windows

package gateway

import (
	"os"
	"syscall"
)

func fileOwnerIDs(info os.FileInfo) (uid, gid uint32, ok bool) {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return 0, 0, false
	}
	return stat.Uid, stat.Gid, true
}

func signalProcess(pid int, sigName string) error {
	sig := syscall.SIGTERM
	if sigName == "KILL" {
		sig = syscall.SIGKILL
	}
	return syscall.Kill(pid, sig)
}
