package hostbridge

import "time"

const (
	PressureNormal      = "normal"
	PressureElevated    = "elevated"
	PressureCritical    = "critical"
	PressureUnavailable = "unavailable"
)

type Snapshot struct {
	SnapshotID   string                  `json:"snapshot_id"`
	CapturedAt   time.Time               `json:"captured_at"`
	Host         HostDiagnostics         `json:"host"`
	Kernel       KernelDiagnostics       `json:"kernel"`
	Boot         BootDiagnostics         `json:"boot"`
	CPU          CPUDiagnostics          `json:"cpu"`
	Memory       MemoryDiagnostics       `json:"memory"`
	Disk         DiskDiagnostics         `json:"disk"`
	GPU          GPUDiagnostics          `json:"gpu"`
	Thermal      ThermalDiagnostics      `json:"thermal"`
	Services     ServiceDiagnostics      `json:"services"`
	ModelRuntime ModelRuntimeDiagnostics `json:"modelruntime"`
	Degraded     bool                    `json:"degraded"`
	Warnings     []string                `json:"warnings,omitempty"`
	Redactions   []RedactionRecord       `json:"redactions,omitempty"`
	SourceErrors []SourceError           `json:"source_errors,omitempty"`
}

type HostDiagnostics struct {
	Hostname     string `json:"hostname,omitempty"`
	Architecture string `json:"architecture"`
	OSRelease    string `json:"os_release,omitempty"`
}

type KernelDiagnostics struct {
	Version string              `json:"version,omitempty"`
	Modules []ModuleDiagnostics `json:"modules,omitempty"`
}

type BootDiagnostics struct {
	Parameters []string `json:"parameters,omitempty"`
}

type ModuleDiagnostics struct {
	Name     string `json:"name"`
	RefCount int    `json:"refcount,omitempty"`
	State    string `json:"state,omitempty"`
}

type CPUDiagnostics struct {
	Count               int       `json:"count"`
	LoadAverage         []float64 `json:"load_average,omitempty"`
	UtilizationEstimate float64   `json:"utilization_estimate,omitempty"`
}

type MemoryDiagnostics struct {
	TotalBytes     uint64 `json:"total_bytes,omitempty"`
	AvailableBytes uint64 `json:"available_bytes,omitempty"`
	SwapTotalBytes uint64 `json:"swap_total_bytes,omitempty"`
	SwapFreeBytes  uint64 `json:"swap_free_bytes,omitempty"`
	PressureLevel  string `json:"pressure_level"`
}

type DiskDiagnostics struct {
	Path          string `json:"path"`
	TotalBytes    uint64 `json:"total_bytes,omitempty"`
	FreeBytes     uint64 `json:"free_bytes,omitempty"`
	UsedBytes     uint64 `json:"used_bytes,omitempty"`
	PressureLevel string `json:"pressure_level"`
}

type GPUDeviceDiagnostics struct {
	Name           string  `json:"name,omitempty"`
	DriverVersion  string  `json:"driver_version,omitempty"`
	MemoryTotalMiB float64 `json:"memory_total_mib,omitempty"`
	MemoryFreeMiB  float64 `json:"memory_free_mib,omitempty"`
	MemoryUsedMiB  float64 `json:"memory_used_mib,omitempty"`
}

type GPUDiagnostics struct {
	Available bool                   `json:"available"`
	Vendor    string                 `json:"vendor,omitempty"`
	Devices   []GPUDeviceDiagnostics `json:"devices,omitempty"`
	Warnings  []string               `json:"warnings,omitempty"`
}

type ThermalSensor struct {
	Label        string  `json:"label"`
	TemperatureC float64 `json:"temperature_c"`
}

type ThermalDiagnostics struct {
	Available bool            `json:"available"`
	Sensors   []ThermalSensor `json:"sensors,omitempty"`
	Warnings  []string        `json:"warnings,omitempty"`
}

type UnitState struct {
	Name      string `json:"name"`
	LoadState string `json:"load_state,omitempty"`
	Active    string `json:"active,omitempty"`
	SubState  string `json:"sub_state,omitempty"`
}

type ServiceDiagnostics struct {
	Available bool        `json:"available"`
	Units     []UnitState `json:"units,omitempty"`
	Failed    []UnitState `json:"failed,omitempty"`
	Warnings  []string    `json:"warnings,omitempty"`
}

type ModelRuntimeDiagnostics struct {
	Available bool     `json:"available"`
	State     string   `json:"state"`
	Warnings  []string `json:"warnings,omitempty"`
}

type RedactionRecord struct {
	Source string `json:"source"`
	Field  string `json:"field"`
	Reason string `json:"reason"`
}

type SourceError struct {
	Source string `json:"source"`
	Error  string `json:"error"`
}
