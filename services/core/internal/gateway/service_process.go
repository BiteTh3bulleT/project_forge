package gateway

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

type processRunTool struct{ workspace string }

const (
	defaultProcessRunTimeoutMs = 30_000
	maxProcessRunTimeoutMs     = 120_000
	maxProcessRunOutputBytes   = 1 << 20
	maxProcessRunCommandBytes  = 64 << 10
)

var errProcessRunCommandTooLarge = errors.New("proc.run command too large")

type boundedOutputBuffer struct {
	buf       bytes.Buffer
	limit     int
	truncated bool
}

func newBoundedOutputBuffer(limit int) *boundedOutputBuffer {
	return &boundedOutputBuffer{limit: limit}
}

func (b *boundedOutputBuffer) Write(p []byte) (int, error) {
	if b.limit <= 0 {
		if len(p) > 0 {
			b.truncated = true
		}
		return len(p), nil
	}
	remaining := b.limit - b.buf.Len()
	if remaining > 0 {
		if len(p) > remaining {
			_, _ = b.buf.Write(p[:remaining])
			b.truncated = true
		} else {
			_, _ = b.buf.Write(p)
		}
	} else if len(p) > 0 {
		b.truncated = true
	}
	return len(p), nil
}

func (b *boundedOutputBuffer) String() string {
	return b.buf.String()
}

func (b *boundedOutputBuffer) Truncated() bool {
	return b.truncated
}

func normalizeProcessRunTimeoutMs(input map[string]any) int {
	timeoutMs := int(readFloat(input, "timeoutMs", defaultProcessRunTimeoutMs))
	if timeoutMs <= 0 {
		return defaultProcessRunTimeoutMs
	}
	if timeoutMs > maxProcessRunTimeoutMs {
		return maxProcessRunTimeoutMs
	}
	return timeoutMs
}

func normalizeProcessRunCommand(input map[string]any) (string, error) {
	command := strings.TrimSpace(inputString(input, "command"))
	if command == "" {
		return "", errors.New("proc.run requires input.command")
	}
	if len(command) > maxProcessRunCommandBytes {
		return "", fmt.Errorf("%w: %d > %d bytes", errProcessRunCommandTooLarge, len(command), maxProcessRunCommandBytes)
	}
	return command, nil
}

func (t *processRunTool) ID() string             { return "proc.run" }
func (t *processRunTool) Domain() string         { return "process" }
func (t *processRunTool) Action() string         { return "run_command" }
func (t *processRunTool) RiskClass() string      { return "scoped_execute" }
func (t *processRunTool) ExecutionLevel() string { return "L2" }
func (t *processRunTool) Executes() bool         { return true }
func (t *processRunTool) UsesNetwork() bool      { return false }
func (t *processRunTool) WriteIntent() bool      { return false }
func (t *processRunTool) Description() string {
	return "Run a command with timeout and captured stdout/stderr"
}
func (t *processRunTool) Execute(ctx context.Context, req Request) (Result, error) {
	command, err := normalizeProcessRunCommand(req.Input)
	if err != nil {
		return Result{}, err
	}
	timeoutMs := normalizeProcessRunTimeoutMs(req.Input)
	if runtime.GOOS == "windows" {
		return Result{Data: map[string]any{
			"command":     command,
			"timeoutMs":   timeoutMs,
			"ok":          false,
			"unsupported": true,
			"reason":      "proc.run shell execution requires a configured platform command runner on Windows",
		}, Message: "process execution unsupported on this platform"}, nil
	}
	runCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutMs)*time.Millisecond)
	defer cancel()
	cmd := exec.CommandContext(runCtx, "bash", "-lc", command)
	cwd, err := workspaceDirFromRequest(req.Paths, t.workspace)
	if err != nil {
		return Result{}, err
	}
	cmd.Dir = cwd
	stdout := newBoundedOutputBuffer(maxProcessRunOutputBytes)
	stderr := newBoundedOutputBuffer(maxProcessRunOutputBytes)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	startedAt := time.Now()
	err = cmd.Run()
	endedAt := time.Now()
	exitCode := 0
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = -1
		}
	}
	return Result{
		Data: map[string]any{
			"command":         command,
			"cwd":             cmd.Dir,
			"timeoutMs":       timeoutMs,
			"exitCode":        exitCode,
			"ok":              err == nil,
			"stdout":          stdout.String(),
			"stderr":          stderr.String(),
			"stdoutLimit":     maxProcessRunOutputBytes,
			"stderrLimit":     maxProcessRunOutputBytes,
			"stdoutTruncated": stdout.Truncated(),
			"stderrTruncated": stderr.Truncated(),
			"timedOut":        errors.Is(runCtx.Err(), context.DeadlineExceeded),
			"startedAtMs":     startedAt.UnixMilli(),
			"endedAtMs":       endedAt.UnixMilli(),
		},
		Message: "process execution completed",
	}, nil
}

type processTerminateTool struct{}

func normalizeTerminatePID(raw float64) (int, error) {
	if raw != float64(int(raw)) {
		return 0, errors.New("pid must be an integer")
	}
	pid := int(raw)
	if pid <= 0 {
		return 0, errors.New("pid must be positive")
	}
	if pid == 1 {
		return 0, errors.New("refusing to terminate pid 1")
	}
	if pid == os.Getpid() {
		return 0, errors.New("refusing to terminate current process")
	}
	return pid, nil
}

func (t *processTerminateTool) ID() string             { return "proc.terminate" }
func (t *processTerminateTool) Domain() string         { return "process" }
func (t *processTerminateTool) Action() string         { return "terminate_process" }
func (t *processTerminateTool) RiskClass() string      { return "privileged" }
func (t *processTerminateTool) ExecutionLevel() string { return "L3" }
func (t *processTerminateTool) Executes() bool         { return true }
func (t *processTerminateTool) UsesNetwork() bool      { return false }
func (t *processTerminateTool) WriteIntent() bool      { return true }
func (t *processTerminateTool) Description() string    { return "Terminate a process by PID (SIGTERM)" }
func (t *processTerminateTool) Execute(ctx context.Context, req Request) (Result, error) {
	pid, err := normalizeTerminatePID(readFloat(req.Input, "pid", 0))
	if err != nil {
		return Result{}, errors.New("proc.terminate requires input.pid")
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return Result{Data: map[string]any{"pid": pid, "ok": false, "error": err.Error()}, Message: "terminate attempted"}, nil
	}
	err = proc.Kill()
	data := map[string]any{"pid": pid, "ok": err == nil}
	if err != nil {
		data["error"] = err.Error()
	}
	return Result{Data: data, Message: "terminate attempted"}, nil
}
