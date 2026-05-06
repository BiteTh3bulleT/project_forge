package gpu

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
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
