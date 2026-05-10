package gpu

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestIntelLevelZeroUnavailableReturnsDegraded(t *testing.T) {
	dir := t.TempDir()
	svc := NewIntel(IntelOptions{
		Enabled:     true,
		ZEInfoPath:  filepath.Join(dir, "missing-ze-info"),
		IntelGPUTop: filepath.Join(dir, "missing-intel-gpu-top"),
		Timeout:     10 * time.Millisecond,
	})
	snap := svc.Snapshot(context.Background())
	if snap.Healthy || snap.State != "degraded" {
		t.Fatalf("expected degraded Intel Level Zero telemetry, got %+v", snap)
	}
	if !snap.BackgroundAdmissionOK {
		t.Fatalf("missing ze_info should not block background admission by itself")
	}
}

func TestIntelLevelZeroParsesZEInfo(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script fixture is unix-only")
	}
	dir := t.TempDir()
	ze := filepath.Join(dir, "ze_info")
	if err := os.WriteFile(ze, []byte("#!/bin/sh\ncat <<'EOF'\nDevice Name : Intel(R) Iris(R) Xe Graphics\nUUID : GPU-intel-0\nEOF\n"), 0o755); err != nil {
		t.Fatalf("write ze_info fixture: %v", err)
	}
	svc := NewIntel(IntelOptions{Enabled: true, ZEInfoPath: ze, Timeout: time.Second})
	snap := svc.Snapshot(context.Background())
	if !snap.Healthy || !snap.Available || snap.State != "available" {
		t.Fatalf("expected available Intel Level Zero telemetry, got %+v", snap)
	}
	if len(snap.Devices) != 1 || snap.Devices[0].Index != "Intel(R) Iris(R) Xe Graphics" || snap.Devices[0].UUID != "GPU-intel-0" {
		t.Fatalf("unexpected device parsing: %+v", snap.Devices)
	}
}

func TestIntelGPUTopCanProvideTelemetryWithoutZEInfo(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script fixture is unix-only")
	}
	dir := t.TempDir()
	top := filepath.Join(dir, "intel_gpu_top")
	if err := os.WriteFile(top, []byte("#!/bin/sh\ncat <<'EOF'\n{\"engines\":{\"Render/3D\":{\"busy\":12.5}}}\nEOF\n"), 0o755); err != nil {
		t.Fatalf("write intel_gpu_top fixture: %v", err)
	}
	svc := NewIntel(IntelOptions{
		Enabled:     true,
		ZEInfoPath:  filepath.Join(dir, "missing-ze-info"),
		IntelGPUTop: top,
		Timeout:     time.Second,
	})
	snap := svc.Snapshot(context.Background())
	if !snap.Healthy || !snap.Available || snap.State != "available" {
		t.Fatalf("expected available Intel GPU telemetry through intel_gpu_top, got %+v", snap)
	}
	if len(snap.Devices) == 0 || snap.Devices[0].GPUUtilization != 12.5 {
		t.Fatalf("expected intel_gpu_top utilization sample, got %+v", snap.Devices)
	}
	if len(snap.Warnings) == 0 {
		t.Fatalf("expected warning preserving missing ze_info diagnostic")
	}
}

func TestIntelRunToolTruncatesLargeOutput(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script fixture is unix-only")
	}
	dir := t.TempDir()
	tool := filepath.Join(dir, "large-output")
	if err := os.WriteFile(tool, []byte("#!/bin/sh\nyes x | head -c 1052672\n"), 0o755); err != nil {
		t.Fatalf("write large-output fixture: %v", err)
	}
	out, err := runTool(context.Background(), time.Second, tool)
	if err != nil {
		t.Fatalf("runTool returned error: %v", err)
	}
	if !strings.Contains(out, "Intel telemetry tool output truncated") {
		t.Fatalf("expected truncation marker in output")
	}
	if len(out) > intelToolOutputLimit+128 {
		t.Fatalf("bounded output too large: got %d, limit %d", len(out), intelToolOutputLimit)
	}
}

func TestIntelGPUTopBoundsStderrOnDecodeError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script fixture is unix-only")
	}
	dir := t.TempDir()
	tool := filepath.Join(dir, "intel_gpu_top")
	if err := os.WriteFile(tool, []byte("#!/bin/sh\nprintf '{'\nhead -c 1052672 /dev/zero | tr '\\0' e >&2\n"), 0o755); err != nil {
		t.Fatalf("write intel_gpu_top fixture: %v", err)
	}
	_, err := runIntelGPUTop(context.Background(), time.Second, tool)
	if err == nil {
		t.Fatal("expected decode error")
	}
	msg := err.Error()
	if !strings.Contains(msg, "Intel telemetry tool output truncated") {
		t.Fatalf("expected bounded stderr truncation marker in error, got %q", msg)
	}
	if len(msg) > intelToolOutputLimit+512 {
		t.Fatalf("bounded stderr error too large: got %d, limit %d", len(msg), intelToolOutputLimit)
	}
}

func TestIntelGPUTopRejectsOversizeStdout(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script fixture is unix-only")
	}
	dir := t.TempDir()
	tool := filepath.Join(dir, "intel_gpu_top")
	if err := os.WriteFile(tool, []byte("#!/bin/sh\nprintf '{\"sample\":\"'\nhead -c 1052672 /dev/zero | tr '\\0' x\nprintf '\"}'\n"), 0o755); err != nil {
		t.Fatalf("write intel_gpu_top fixture: %v", err)
	}
	_, err := runIntelGPUTop(context.Background(), time.Second, tool)
	if err == nil || !strings.Contains(err.Error(), "intel_gpu_top output too large") {
		t.Fatalf("runIntelGPUTop error = %v, want stdout size error", err)
	}
}
