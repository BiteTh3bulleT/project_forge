package gpu

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestDCGMUnavailableReturnsSafeDegradedTelemetry(t *testing.T) {
	svc := New(Options{Enabled: true, Endpoint: "http://127.0.0.1:1/metrics", Timeout: 10 * time.Millisecond})
	snap := svc.Snapshot(context.Background())
	if snap.Healthy || snap.State != "degraded" {
		t.Fatalf("expected degraded unavailable telemetry, got %+v", snap)
	}
	if !snap.BackgroundAdmissionOK {
		t.Fatalf("unreachable telemetry should not block background jobs by itself")
	}
	if err := svc.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	if err := svc.Close(); err != nil {
		t.Fatalf("second close: %v", err)
	}
}

func TestDCGMParsesExporterMetricsAndPressure(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`
DCGM_FI_DEV_GPU_UTIL{gpu="0",UUID="GPU-a"} 77
DCGM_FI_DEV_FB_USED{gpu="0",UUID="GPU-a"} 900
DCGM_FI_DEV_FB_FREE{gpu="0",UUID="GPU-a"} 100
DCGM_FI_DEV_POWER_USAGE{gpu="0",UUID="GPU-a"} 250
DCGM_FI_DEV_GPU_TEMP{gpu="0",UUID="GPU-a"} 72
`))
	}))
	defer ts.Close()

	svc := New(Options{Enabled: true, Endpoint: ts.URL, MemoryPressureThreshold: 0.80})
	snap := svc.Snapshot(context.Background())
	if !snap.Available || !snap.Healthy || snap.State != "pressure" {
		t.Fatalf("expected pressure telemetry state, got %+v", snap)
	}
	if snap.MemoryPressure != 0.9 || snap.BackgroundAdmissionOK {
		t.Fatalf("expected memory pressure to block background jobs, got %+v", snap)
	}
	if len(snap.Devices) != 1 || snap.Devices[0].GPUUtilization != 77 || snap.Devices[0].TemperatureC != 72 {
		t.Fatalf("unexpected parsed device telemetry: %+v", snap.Devices)
	}
}
