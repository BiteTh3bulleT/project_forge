package hostbridge

import (
	"bytes"
	"context"
	"os/exec"
	"strings"
	"time"
)

type CommandResult struct {
	Stdout string
	Stderr string
}

type CommandRunner interface {
	LookPath(name string) (string, error)
	Run(ctx context.Context, name string, args ...string) (CommandResult, error)
}

type execRunner struct {
	timeout time.Duration
}

func newExecRunner(timeout time.Duration) execRunner {
	if timeout <= 0 {
		timeout = 1500 * time.Millisecond
	}
	return execRunner{timeout: timeout}
}

func (r execRunner) LookPath(name string) (string, error) {
	return exec.LookPath(name)
}

func (r execRunner) Run(ctx context.Context, name string, args ...string) (CommandResult, error) {
	runCtx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()
	cmd := exec.CommandContext(runCtx, name, args...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if runCtx.Err() != nil {
		err = runCtx.Err()
	}
	return CommandResult{
		Stdout: strings.TrimSpace(stdout.String()),
		Stderr: strings.TrimSpace(stderr.String()),
	}, err
}
