package gateway

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestScrubGatewayCommandEnvRemovesSecretBearingKeys(t *testing.T) {
	t.Parallel()
	env := scrubGatewayCommandEnv([]string{
		"PATH=/usr/bin",
		"FORGE_API_TOKEN=secret",
		"OPENAI_API_KEY=secret",
		"DB_PASSWORD=secret",
		"FORGE_ENCRYPTION_KEY_PATH=secret",
		"HOME=/tmp/forge",
	})
	joined := "\n" + strings.Join(env, "\n") + "\n"
	for _, forbidden := range []string{"FORGE_API_TOKEN=", "OPENAI_API_KEY=", "DB_PASSWORD=", "FORGE_ENCRYPTION_KEY_PATH="} {
		if strings.Contains(joined, "\n"+forbidden) {
			t.Fatalf("secret-bearing env key %s was not scrubbed from %#v", forbidden, env)
		}
	}
	for _, allowed := range []string{"PATH=/usr/bin", "HOME=/tmp/forge"} {
		if !strings.Contains(joined, "\n"+allowed+"\n") {
			t.Fatalf("expected non-secret env key %s to be preserved in %#v", allowed, env)
		}
	}
}

func TestProcessAndPatchCodeUseGatewayCommandFactory(t *testing.T) {
	t.Parallel()
	for _, path := range []string{
		filepath.Join("service_process.go"),
		filepath.Join("capability_backing_execute.go"),
	} {
		body, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		if strings.Contains(string(body), "exec.CommandContext") {
			t.Fatalf("%s must use newGatewayCommand so subprocess env scrubbing stays centralized", path)
		}
	}
}
