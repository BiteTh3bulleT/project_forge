package hostbridge

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

const maxHostBridgeCommandOutputBytes = 64 << 10

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
	stdout := newBoundedCommandBuffer(maxHostBridgeCommandOutputBytes)
	stderr := newBoundedCommandBuffer(maxHostBridgeCommandOutputBytes)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if runCtx.Err() != nil {
		err = runCtx.Err()
	}
	if outputErr := oversizedCommandOutputError(stdout.overflow, stderr.overflow); outputErr != nil {
		if err != nil {
			err = fmt.Errorf("%w; %v", err, outputErr)
		} else {
			err = outputErr
		}
	}
	return CommandResult{
		Stdout: strings.TrimSpace(stdout.String()),
		Stderr: strings.TrimSpace(stderr.String()),
	}, err
}

type boundedCommandBuffer struct {
	buf      bytes.Buffer
	limit    int
	overflow bool
}

func newBoundedCommandBuffer(limit int) boundedCommandBuffer {
	return boundedCommandBuffer{limit: limit}
}

func (b *boundedCommandBuffer) Write(p []byte) (int, error) {
	if b.limit <= 0 {
		b.overflow = true
		return len(p), nil
	}
	remaining := b.limit - b.buf.Len()
	if remaining > 0 {
		chunk := p
		if len(chunk) > remaining {
			chunk = chunk[:remaining]
		}
		_, _ = b.buf.Write(chunk)
	}
	if len(p) > remaining {
		b.overflow = true
	}
	return len(p), nil
}

func (b *boundedCommandBuffer) String() string {
	return b.buf.String()
}

func oversizedCommandOutputError(stdoutOverflow, stderrOverflow bool) error {
	if !stdoutOverflow && !stderrOverflow {
		return nil
	}
	parts := []string{}
	if stdoutOverflow {
		parts = append(parts, "stdout too large")
	}
	if stderrOverflow {
		parts = append(parts, "stderr too large")
	}
	return fmt.Errorf("hostbridge command output too large: %s", strings.Join(parts, ", "))
}
