package hostbridge

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type fakeRunner struct {
	paths map[string]bool
	out   map[string]CommandResult
	err   map[string]error
	calls []string
}

func (r *fakeRunner) LookPath(name string) (string, error) {
	if r.paths != nil && r.paths[name] {
		return "/usr/bin/" + name, nil
	}
	return "", os.ErrNotExist
}

func (r *fakeRunner) Run(_ context.Context, name string, args ...string) (CommandResult, error) {
	call := name + " " + strings.Join(args, " ")
	r.calls = append(r.calls, call)
	if r.err != nil && r.err[name] != nil {
		return CommandResult{}, r.err[name]
	}
	if r.out != nil {
		return r.out[name], nil
	}
	return CommandResult{}, nil
}

func TestRedactBootParams(t *testing.T) {
	params, redactions := redactBootParameters("quiet password=hunter2 token=abc root=/dev/sda api_key=xyz endpoint=https://user:pass@example.test/path bearer=Bearer abc")
	joined := strings.Join(params, " ")
	for _, unsafe := range []string{"hunter2", "token=abc", "api_key=xyz", "user:pass", "Bearer abc"} {
		if strings.Contains(joined, unsafe) {
			t.Fatalf("boot params leaked %q in %q", unsafe, joined)
		}
	}
	if !strings.Contains(joined, "password=[REDACTED]") || !strings.Contains(joined, "root=/dev/sda") {
		t.Fatalf("unexpected redacted params: %q", joined)
	}
	if len(redactions) != 5 {
		t.Fatalf("expected 5 redactions, got %d: %#v", len(redactions), redactions)
	}
}

func TestSnapshotSourceFailureIsolation(t *testing.T) {
	root := t.TempDir()
	proc := filepath.Join(root, "proc")
	sys := filepath.Join(root, "sys")
	mustWrite(t, filepath.Join(proc, "version"), "Linux version test\n")
	mustWrite(t, filepath.Join(proc, "cmdline"), "quiet token=abc\n")
	mustWrite(t, filepath.Join(proc, "loadavg"), "0.10 0.20 0.30 1/2 3\n")
	mustWrite(t, filepath.Join(proc, "stat"), "cpu  10 0 10 80 0 0 0 0 0 0\n")
	mustWrite(t, filepath.Join(proc, "modules"), "loop 123 0 - Live 0x0\n")
	mustWrite(t, filepath.Join(sys, "class", "thermal", "thermal_zone0", "type"), "x86_pkg_temp\n")
	mustWrite(t, filepath.Join(sys, "class", "thermal", "thermal_zone0", "temp"), "42000\n")

	service := New(Options{
		ProcRoot:    proc,
		SysRoot:     sys,
		StorageRoot: filepath.Join(root, "missing-storage"),
		Now:         fixedNow,
		Runner:      &fakeRunner{},
	})
	snapshot := service.Snapshot(context.Background())
	if snapshot.SnapshotID == "" {
		t.Fatal("snapshot id is required")
	}
	if !snapshot.Degraded {
		t.Fatal("snapshot should be degraded when sources are missing")
	}
	if snapshot.Memory.PressureLevel != PressureUnavailable {
		t.Fatalf("expected unavailable memory pressure, got %q", snapshot.Memory.PressureLevel)
	}
	if len(snapshot.SourceErrors) == 0 {
		t.Fatal("expected source errors for missing sources")
	}
	if len(snapshot.Redactions) != 1 {
		t.Fatalf("expected boot token redaction, got %#v", snapshot.Redactions)
	}
}

func TestHostProbeReadsRejectOversizeFiles(t *testing.T) {
	path := filepath.Join(t.TempDir(), "oversize")
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create oversized probe file: %v", err)
	}
	if err := f.Truncate(maxHostProbeFileBytes + 1); err != nil {
		_ = f.Close()
		t.Fatalf("truncate oversized probe file: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("close oversized probe file: %v", err)
	}

	if _, err := readHostProbeFile(path); err == nil || !strings.Contains(err.Error(), "host probe file too large") {
		t.Fatalf("readHostProbeFile error = %v, want size error", err)
	}
}

func TestExecRunnerRejectsOversizeCommandOutput(t *testing.T) {
	runner := newExecRunner(2 * time.Second)
	result, err := runner.Run(context.Background(), "sh", "-c", "i=0; while [ $i -lt 70000 ]; do printf x; i=$((i+1)); done")
	if err == nil || !strings.Contains(err.Error(), "stdout too large") {
		t.Fatalf("Run error = %v, want stdout size error", err)
	}
	if len(result.Stdout) > maxHostBridgeCommandOutputBytes {
		t.Fatalf("stdout length = %d, want <= %d", len(result.Stdout), maxHostBridgeCommandOutputBytes)
	}
}

func TestSnapshotReportsOversizeProcSource(t *testing.T) {
	root := t.TempDir()
	proc := filepath.Join(root, "proc")
	sys := filepath.Join(root, "sys")
	mustWrite(t, filepath.Join(proc, "version"), "Linux version test\n")
	mustWrite(t, filepath.Join(proc, "cmdline"), "quiet\n")
	mustWrite(t, filepath.Join(proc, "loadavg"), "0.10 0.20 0.30 1/2 3\n")
	mustWrite(t, filepath.Join(proc, "stat"), "cpu  10 0 10 80 0 0 0 0 0 0\n")
	mustWrite(t, filepath.Join(proc, "modules"), "loop 123 0 - Live 0x0\n")
	meminfo := filepath.Join(proc, "meminfo")
	if err := os.MkdirAll(filepath.Dir(meminfo), 0o755); err != nil {
		t.Fatal(err)
	}
	f, err := os.Create(meminfo)
	if err != nil {
		t.Fatalf("create oversized meminfo: %v", err)
	}
	if err := f.Truncate(maxHostProbeFileBytes + 1); err != nil {
		_ = f.Close()
		t.Fatalf("truncate oversized meminfo: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("close oversized meminfo: %v", err)
	}

	snapshot := New(Options{
		ProcRoot:    proc,
		SysRoot:     sys,
		StorageRoot: root,
		Now:         fixedNow,
		Runner:      &fakeRunner{},
	}).Snapshot(context.Background())
	if snapshot.Memory.PressureLevel != PressureUnavailable {
		t.Fatalf("expected unavailable memory pressure, got %q", snapshot.Memory.PressureLevel)
	}
	if !sourceErrorsContain(snapshot.SourceErrors, "proc.meminfo", "too large") {
		t.Fatalf("expected oversized proc.meminfo source error, got %#v", snapshot.SourceErrors)
	}
}

func TestMemoryPressureClassification(t *testing.T) {
	tests := []struct {
		name      string
		total     uint64
		available uint64
		want      string
	}{
		{name: "unavailable", total: 0, available: 0, want: PressureUnavailable},
		{name: "critical", total: 100, available: 9, want: PressureCritical},
		{name: "elevated", total: 100, available: 19, want: PressureElevated},
		{name: "normal", total: 100, available: 20, want: PressureNormal},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ClassifyMemoryPressure(tt.total, tt.available); got != tt.want {
				t.Fatalf("got %q want %q", got, tt.want)
			}
		})
	}
}

func TestDiskPressureClassification(t *testing.T) {
	tests := []struct {
		name  string
		total uint64
		free  uint64
		want  string
	}{
		{name: "unavailable", total: 0, free: 0, want: PressureUnavailable},
		{name: "critical", total: 100, free: 10, want: PressureCritical},
		{name: "elevated", total: 100, free: 20, want: PressureElevated},
		{name: "normal", total: 100, free: 21, want: PressureNormal},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ClassifyDiskPressure(tt.total, tt.free); got != tt.want {
				t.Fatalf("got %q want %q", got, tt.want)
			}
		})
	}
}

func TestSnapshotJSONTopLevelStability(t *testing.T) {
	snapshot := New(Options{ProcRoot: t.TempDir(), SysRoot: t.TempDir(), StorageRoot: t.TempDir(), Now: fixedNow, Runner: &fakeRunner{}}).Snapshot(context.Background())
	body, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(body, &decoded); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"snapshot_id", "captured_at", "host", "kernel", "boot", "cpu", "memory", "disk", "gpu", "thermal", "services", "modelruntime", "degraded", "source_errors"} {
		if _, ok := decoded[key]; !ok {
			t.Fatalf("missing top-level key %q in %s", key, string(body))
		}
	}
}

func TestNoMutationCommands(t *testing.T) {
	runner := &fakeRunner{
		paths: map[string]bool{"systemctl": true, "nvidia-smi": true},
		out: map[string]CommandResult{
			"systemctl":  {Stdout: "Id=forge-core.service\nLoadState=loaded\nActiveState=active\nSubState=running\n"},
			"nvidia-smi": {Stdout: "Test GPU, 550.1, 1000, 900, 100\n"},
		},
	}
	root := t.TempDir()
	proc := filepath.Join(root, "proc")
	mustWrite(t, filepath.Join(proc, "version"), "Linux version test\n")
	mustWrite(t, filepath.Join(proc, "cmdline"), "quiet\n")
	mustWrite(t, filepath.Join(proc, "loadavg"), "0 0 0 0/0 0\n")
	mustWrite(t, filepath.Join(proc, "stat"), "cpu  10 0 10 80 0 0 0 0 0 0\n")
	mustWrite(t, filepath.Join(proc, "meminfo"), "MemTotal: 100 kB\nMemAvailable: 50 kB\nSwapTotal: 0 kB\nSwapFree: 0 kB\n")
	mustWrite(t, filepath.Join(proc, "modules"), "loop 123 0 - Live 0x0\n")

	_ = New(Options{ProcRoot: proc, SysRoot: filepath.Join(root, "sys"), StorageRoot: root, Now: fixedNow, Runner: runner}).Snapshot(context.Background())
	joined := strings.Join(runner.calls, "\n")
	for _, forbidden := range []string{"nixos-rebuild", "systemctl restart", "systemctl stop", "systemctl start", "modprobe", "rmmod", "zypper", "apt upgrade", "dnf upgrade", "pacman -Syu"} {
		if strings.Contains(joined, forbidden) {
			t.Fatalf("mutation command was invoked: %s", joined)
		}
	}
}

func TestWriteSnapshotRequiresReportDir(t *testing.T) {
	service := New(Options{})
	if _, err := service.WriteSnapshot(Snapshot{SnapshotID: "hostdiag_test"}); !errors.Is(err, ErrReportDirRequired) {
		t.Fatalf("expected ErrReportDirRequired, got %v", err)
	}
}

func TestWriteSnapshot(t *testing.T) {
	dir := t.TempDir()
	service := New(Options{ReportDir: dir})
	path, err := service.WriteSnapshot(Snapshot{SnapshotID: "hostdiag_test", CapturedAt: fixedNow()})
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Dir(path) != dir {
		t.Fatalf("snapshot wrote outside report dir: %s", path)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
}

func fixedNow() time.Time {
	return time.Date(2026, 5, 7, 12, 0, 0, 0, time.UTC)
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func sourceErrorsContain(errors []SourceError, source, text string) bool {
	for _, item := range errors {
		if item.Source == source && strings.Contains(item.Error, text) {
			return true
		}
	}
	return false
}
