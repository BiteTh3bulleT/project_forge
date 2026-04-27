//go:build !windows

package gateway

import (
	"context"
	"os"
	"strings"
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

func desktopOpenTarget(ctx context.Context, target string) (pid int, output string, err error) {
	out, err := runCmd(ctx, "", "xdg-open", strings.TrimSpace(target))
	return 0, out, err
}

func desktopLaunchApp(command string, args []string) (int, error) {
	return runDetachedCmd("", append([]string{command}, args...)...)
}

func desktopPlatformLaunchCandidates(_ string) [][]string {
	return nil
}
