package gateway

import "os"

func fileOwnerIDs(_ os.FileInfo) (uid, gid uint32, ok bool) {
	return 0, 0, false
}

func signalProcess(pid int, _ string) error {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	return proc.Kill()
}
