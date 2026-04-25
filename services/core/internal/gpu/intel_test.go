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
	svc := NewIntel(IntelOptions{Enabled: true, ZEInfoPath: filepath.Join(t.TempDir(), "missing-ze-info"), Timeout: 10 * time.Millisecond})
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
