package gateway

import (
	"context"
	"os"
	"os/exec"
	"strings"
)

var gatewayCommandSecretKeyMarkers = []string{
	"API_KEY",
	"AUTH",
	"CREDENTIAL",
	"ENCRYPTION_KEY",
	"PASSWORD",
	"PRIVATE_KEY",
	"SECRET",
	"TOKEN",
}

func newGatewayCommand(ctx context.Context, dir, bin string, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, bin, args...)
	if strings.TrimSpace(dir) != "" {
		cmd.Dir = dir
	}
	cmd.Env = scrubGatewayCommandEnv(os.Environ())
	return cmd
}

func newGatewayDetachedCommand(dir, bin string, args ...string) *exec.Cmd {
	cmd := exec.Command(bin, args...)
	if strings.TrimSpace(dir) != "" {
		cmd.Dir = dir
	}
	cmd.Env = scrubGatewayCommandEnv(os.Environ())
	return cmd
}

func scrubGatewayCommandEnv(env []string) []string {
	out := make([]string, 0, len(env))
	for _, item := range env {
		key, _, ok := strings.Cut(item, "=")
		if !ok {
			continue
		}
		if gatewayCommandEnvKeyIsSecret(key) {
			continue
		}
		out = append(out, item)
	}
	return out
}

func gatewayCommandEnvKeyIsSecret(key string) bool {
	normalized := strings.ToUpper(strings.TrimSpace(key))
	for _, marker := range gatewayCommandSecretKeyMarkers {
		if strings.Contains(normalized, marker) {
			return true
		}
	}
	return false
}
