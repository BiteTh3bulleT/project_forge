//go:build !linux

package offlinerecovery

import "fmt"

func prepareReplacementMetadata(_, _ string) error {
	return fmt.Errorf("offline recovery replacement is supported only on Linux/NixOS")
}
