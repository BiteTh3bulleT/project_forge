package gateway

import (
	"context"
	"os"
	"strings"
)

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

func desktopOpenTarget(_ context.Context, target string) (pid int, output string, err error) {
	pid, err = runDetachedCmd("", "explorer.exe", strings.TrimSpace(target))
	return pid, "", err
}

func desktopLaunchApp(command string, args []string) (int, error) {
	command = strings.TrimSpace(command)
	if command == "" {
		return 0, os.ErrInvalid
	}
	if desktopLooksLikeURL(command) || strings.HasSuffix(strings.ToLower(command), ":") || strings.HasPrefix(strings.ToLower(command), "shell:") {
		return runDetachedCmd("", "explorer.exe", command)
	}
	return runDetachedCmd("", append([]string{command}, args...)...)
}

func desktopPlatformLaunchCandidates(normalized string) [][]string {
	normalized = strings.TrimSpace(strings.ToLower(normalized))
	switch {
	case normalized == "minecraft" || strings.Contains(normalized, "minecraft"):
		return [][]string{
			{"minecraft:"},
			{"MinecraftLauncher.exe"},
			{"explorer.exe", `shell:AppsFolder\Microsoft.4297127D64EC6_8wekyb3d8bbwe!Minecraft`},
		}
	case strings.Contains(normalized, "terminal") || strings.Contains(normalized, "powershell"):
		return [][]string{
			{"wt.exe"},
			{"powershell.exe"},
			{"cmd.exe"},
		}
	case strings.Contains(normalized, "notepad"):
		return [][]string{{"notepad.exe"}}
	default:
		return nil
	}
}
