//go:build linux

package offlinerecovery

import (
	"fmt"
	"os"
	"syscall"
)

func prepareReplacementMetadata(referencePath, replacementPath string) error {
	info, err := os.Stat(referencePath)
	if err != nil {
		return err
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return fmt.Errorf("Linux file ownership metadata unavailable")
	}
	if err := os.Chown(replacementPath, int(stat.Uid), int(stat.Gid)); err != nil {
		return err
	}
	return os.Chmod(replacementPath, info.Mode().Perm())
}
