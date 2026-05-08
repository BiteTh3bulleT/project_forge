package forgeh

import (
	"encoding/json"
	"testing"
	"time"

	"forge/projectforge/services/core/internal/hostbridge"
)

func TestRAMPressureClassification(t *testing.T) {
	tests := []struct {
		name      string
		total     uint64
		available uint64
		want      string
	}{
		{name: "normal at thirty percent", total: 100, available: 30, want: ResourcePressureNormal},
		{name: "elevated below thirty", total: 10000, available: 2999, want: ResourcePressureElevated},
		{name: "elevated at fifteen", total: 100, available: 15, want: ResourcePressureElevated},
		{name: "constrained below fifteen", total: 10000, available: 1499, want: ResourcePressureConstrained},
		{name: "constrained at seven", total: 100, available: 7, want: ResourcePressureConstrained},
		{name: "critical below seven", total: 10000, available: 699, want: ResourcePressureCritical},
		{name: "unavailable zero total", total: 0, available: 0, want: ResourcePressureUnavailable},
		{name: "unavailable impossible available", total: 100, available: 101, want: ResourcePressureUnavailable},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ClassifyRAMPressure(tt.total, tt.available); got != tt.want {
				t.Fatalf("got %q want %q", got, tt.want)
			}
		})
	}
}

func TestSwapPressureClassification(t *testing.T) {
	tests := []struct {
		name  string
		total uint64
		free  uint64
		want  string
	}{
		{name: "no swap is normal", total: 0, free: 0, want: ResourcePressureNormal},
		{name: "normal at twenty five used", total: 100, free: 75, want: ResourcePressureNormal},
		{name: "elevated above twenty five used", total: 10000, free: 7499, want: ResourcePressureElevated},
		{name: "elevated at fifty used", total: 100, free: 50, want: ResourcePressureElevated},
		{name: "constrained above fifty used", total: 10000, free: 4999, want: ResourcePressureConstrained},
		{name: "constrained at eighty used", total: 100, free: 20, want: ResourcePressureConstrained},
		{name: "critical above eighty used", total: 10000, free: 1999, want: ResourcePressureCritical},
		{name: "unavailable impossible free", total: 100, free: 101, want: ResourcePressureUnavailable},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ClassifySwapPressure(tt.total, tt.free); got != tt.want {
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
		{name: "normal at twenty percent free", total: 100, free: 20, want: ResourcePressureNormal},
		{name: "elevated below twenty free", total: 10000, free: 1999, want: ResourcePressureElevated},
		{name: "elevated at ten free", total: 100, free: 10, want: ResourcePressureElevated},
		{name: "constrained below ten free", total: 10000, free: 999, want: ResourcePressureConstrained},
		{name: "constrained at five free", total: 100, free: 5, want: ResourcePressureConstrained},
		{name: "critical below five free", total: 10000, free: 499, want: ResourcePressureCritical},
		{name: "unavailable zero total", total: 0, free: 0, want: ResourcePressureUnavailable},
		{name: "unavailable impossible free", total: 100, free: 101, want: ResourcePressureUnavailable},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ClassifyDiskPressure(tt.total, tt.free); got != tt.want {
				t.Fatalf("got %q want %q", got, tt.want)
			}
		})
	}
}

func TestVRAMPressureClassification(t *testing.T) {
	tests := []struct {
		name string
		gpu  hostbridge.GPUDiagnostics
		want string
	}{
		{name: "unavailable", gpu: hostbridge.GPUDiagnostics{}, want: ResourcePressureUnavailable},
		{name: "normal at thirty five free", gpu: gpuWithFree(35), want: ResourcePressureNormal},
		{name: "elevated below thirty five free", gpu: gpuWithFree(34.99), want: ResourcePressureElevated},
		{name: "elevated at twenty free", gpu: gpuWithFree(20), want: ResourcePressureElevated},
		{name: "constrained below twenty free", gpu: gpuWithFree(19.99), want: ResourcePressureConstrained},
		{name: "constrained at ten free", gpu: gpuWithFree(10), want: ResourcePressureConstrained},
		{name: "critical below ten free", gpu: gpuWithFree(9.99), want: ResourcePressureCritical},
		{name: "multi gpu worst device", gpu: hostbridge.GPUDiagnostics{Available: true, Devices: []hostbridge.GPUDeviceDiagnostics{{MemoryTotalMiB: 100, MemoryFreeMiB: 80}, {MemoryTotalMiB: 100, MemoryFreeMiB: 4}}}, want: ResourcePressureCritical},
		{name: "invalid total", gpu: hostbridge.GPUDiagnostics{Available: true, Devices: []hostbridge.GPUDeviceDiagnostics{{MemoryTotalMiB: 0}}}, want: ResourcePressureUnavailable},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ClassifyVRAMPressure(tt.gpu); got != tt.want {
				t.Fatalf("got %q want %q", got, tt.want)
			}
		})
	}
}

func TestThermalPressureClassification(t *testing.T) {
	tests := []struct {
		name    string
		thermal hostbridge.ThermalDiagnostics
		want    string
	}{
		{name: "unavailable", thermal: hostbridge.ThermalDiagnostics{}, want: ResourcePressureUnavailable},
		{name: "normal", thermal: thermalWithTemp(69), want: ResourcePressureNormal},
		{name: "elevated", thermal: thermalWithTemp(70), want: ResourcePressureElevated},
		{name: "constrained", thermal: thermalWithTemp(80), want: ResourcePressureConstrained},
		{name: "critical", thermal: thermalWithTemp(90), want: ResourcePressureCritical},
		{name: "worst sensor", thermal: hostbridge.ThermalDiagnostics{Available: true, Sensors: []hostbridge.ThermalSensor{{Label: "a", TemperatureC: 45}, {Label: "b", TemperatureC: 92}}}, want: ResourcePressureCritical},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ClassifyThermalPressure(tt.thermal); got != tt.want {
				t.Fatalf("got %q want %q", got, tt.want)
			}
		})
	}
}

func TestOverallPosture(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*hostbridge.Snapshot)
		want   string
	}{
		{name: "normal with optional unavailable", want: ResourcePostureNormal},
		{name: "degraded on elevated required resource", mutate: func(s *hostbridge.Snapshot) { s.Memory.AvailableBytes = 25 }, want: ResourcePostureDegraded},
		{name: "constrained on constrained required resource", mutate: func(s *hostbridge.Snapshot) { s.Memory.AvailableBytes = 10 }, want: ResourcePostureConstrained},
		{name: "critical on critical resource", mutate: func(s *hostbridge.Snapshot) { s.Disk.FreeBytes = 4 }, want: ResourcePostureCritical},
		{name: "degraded on source errors", mutate: func(s *hostbridge.Snapshot) {
			s.SourceErrors = []hostbridge.SourceError{{Source: "proc.meminfo", Error: "missing"}}
		}, want: ResourcePostureNormal},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			snapshot := normalSnapshot()
			if tt.mutate != nil {
				tt.mutate(&snapshot)
			}
			got := New(Options{Now: fixedNow}).Evaluate(snapshot)
			if got.OverallPosture != tt.want {
				t.Fatalf("got %q want %q: %#v", got.OverallPosture, tt.want, got)
			}
		})
	}
}

func TestLaneDecisions(t *testing.T) {
	tests := []struct {
		name     string
		mutate   func(*hostbridge.Snapshot)
		lane     string
		decision string
	}{
		{name: "interactive normal", lane: WorkloadLaneInteractive, decision: PolicyDecisionAllow},
		{name: "interactive constrained warns", mutate: func(s *hostbridge.Snapshot) { s.Memory.AvailableBytes = 10 }, lane: WorkloadLaneInteractive, decision: PolicyDecisionAllowWithWarning},
		{name: "interactive critical denies", mutate: func(s *hostbridge.Snapshot) { s.Memory.AvailableBytes = 1 }, lane: WorkloadLaneInteractive, decision: PolicyDecisionDeny},
		{name: "background elevated defers", mutate: func(s *hostbridge.Snapshot) { s.Memory.AvailableBytes = 25 }, lane: WorkloadLaneBackgroundIngest, decision: PolicyDecisionDefer},
		{name: "background critical denies", mutate: func(s *hostbridge.Snapshot) { s.Disk.FreeBytes = 1 }, lane: WorkloadLaneBackgroundIngest, decision: PolicyDecisionDeny},
		{name: "embedding elevated defers", mutate: func(s *hostbridge.Snapshot) { s.Disk.FreeBytes = 15 }, lane: WorkloadLaneEmbedding, decision: PolicyDecisionDefer},
		{name: "embedding critical denies", mutate: func(s *hostbridge.Snapshot) { s.Memory.AvailableBytes = 1 }, lane: WorkloadLaneEmbedding, decision: PolicyDecisionDeny},
		{name: "model load elevated warns", mutate: func(s *hostbridge.Snapshot) { s.GPU = gpuWithFree(30) }, lane: WorkloadLaneModelLoad, decision: PolicyDecisionAllowWithWarning},
		{name: "model load constrained defers", mutate: func(s *hostbridge.Snapshot) { s.GPU = gpuWithFree(12) }, lane: WorkloadLaneModelLoad, decision: PolicyDecisionDefer},
		{name: "model load critical denies", mutate: func(s *hostbridge.Snapshot) { s.GPU = gpuWithFree(4) }, lane: WorkloadLaneModelLoad, decision: PolicyDecisionDeny},
		{name: "maintenance constrained defers", mutate: func(s *hostbridge.Snapshot) { s.Memory.AvailableBytes = 10 }, lane: WorkloadLaneMaintenance, decision: PolicyDecisionDefer},
		{name: "desktop ui follows interactive", mutate: func(s *hostbridge.Snapshot) { s.Memory.AvailableBytes = 10 }, lane: WorkloadLaneDesktopUI, decision: PolicyDecisionAllowWithWarning},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			snapshot := normalSnapshot()
			if tt.mutate != nil {
				tt.mutate(&snapshot)
			}
			policy := New(Options{Now: fixedNow}).Evaluate(snapshot)
			if got := policy.LaneDecisions[tt.lane].Decision; got != tt.decision {
				t.Fatalf("lane %s got %q want %q in policy %#v", tt.lane, got, tt.decision, policy)
			}
		})
	}
}

func TestModelLoadRecommendations(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*hostbridge.Snapshot)
		want   string
	}{
		{name: "normal", want: ModelLoadSmallLocalOK},
		{name: "gpu unavailable cpu safe", mutate: func(s *hostbridge.Snapshot) { s.GPU = hostbridge.GPUDiagnostics{} }, want: ModelLoadCPUOnlySafeMode},
		{name: "elevated", mutate: func(s *hostbridge.Snapshot) { s.GPU = gpuWithFree(30) }, want: ModelLoadCurrentModelOnly},
		{name: "constrained", mutate: func(s *hostbridge.Snapshot) { s.GPU = gpuWithFree(12) }, want: ModelLoadDeferLargeModel},
		{name: "critical", mutate: func(s *hostbridge.Snapshot) { s.Memory.AvailableBytes = 1 }, want: ModelLoadDenyNewModelLoad},
		{name: "unavailable required RAM", mutate: func(s *hostbridge.Snapshot) { s.Memory.TotalBytes = 0; s.Memory.AvailableBytes = 0 }, want: ModelLoadUnavailable},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			snapshot := normalSnapshot()
			if tt.mutate != nil {
				tt.mutate(&snapshot)
			}
			policy := New(Options{Now: fixedNow}).Evaluate(snapshot)
			if policy.ModelLoadRecommendation != tt.want {
				t.Fatalf("got %q want %q", policy.ModelLoadRecommendation, tt.want)
			}
		})
	}
}

func TestPolicyAdvisoryOnlyAndJSONStability(t *testing.T) {
	service := New(Options{Now: fixedNow})
	left := service.Evaluate(normalSnapshot())
	right := service.Evaluate(normalSnapshot())
	if !left.AdvisoryOnly {
		t.Fatal("policy must be advisory only")
	}
	if left.PolicyID == "" || left.PolicyID != right.PolicyID {
		t.Fatalf("policy id should be deterministic, got %q and %q", left.PolicyID, right.PolicyID)
	}
	body, err := json.Marshal(left)
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(body, &decoded); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"policy_id", "captured_at", "source_snapshot_id", "overall_posture", "ram_pressure", "swap_pressure", "disk_pressure", "vram_pressure", "thermal_pressure", "lane_decisions", "model_load_recommendation", "background_work_recommendation", "advisory_only"} {
		if _, ok := decoded[key]; !ok {
			t.Fatalf("missing key %q in %s", key, string(body))
		}
	}
}

func normalSnapshot() hostbridge.Snapshot {
	return hostbridge.Snapshot{
		SnapshotID: "hostdiag_test",
		CapturedAt: fixedNow(),
		Memory: hostbridge.MemoryDiagnostics{
			TotalBytes:     100,
			AvailableBytes: 40,
			SwapTotalBytes: 100,
			SwapFreeBytes:  100,
		},
		Disk: hostbridge.DiskDiagnostics{
			Path:       "/forge",
			TotalBytes: 100,
			FreeBytes:  40,
		},
		GPU:     gpuWithFree(50),
		Thermal: thermalWithTemp(50),
	}
}

func gpuWithFree(percent float64) hostbridge.GPUDiagnostics {
	return hostbridge.GPUDiagnostics{
		Available: true,
		Devices: []hostbridge.GPUDeviceDiagnostics{{
			Name:           "test-gpu",
			MemoryTotalMiB: 100,
			MemoryFreeMiB:  percent,
			MemoryUsedMiB:  100 - percent,
		}},
	}
}

func thermalWithTemp(temp float64) hostbridge.ThermalDiagnostics {
	return hostbridge.ThermalDiagnostics{
		Available: true,
		Sensors:   []hostbridge.ThermalSensor{{Label: "test", TemperatureC: temp}},
	}
}

func fixedNow() time.Time {
	return time.Date(2026, 5, 8, 12, 0, 0, 0, time.UTC)
}
