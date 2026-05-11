package hostbridge

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	defaultProcRoot       = "/proc"
	defaultSysRoot        = "/sys"
	defaultStorageRoot    = "/forge"
	maxModules            = 256
	maxThermalSensors     = 64
	maxHostProbeFileBytes = 1 << 20
)

type Options struct {
	ProcRoot                   string
	SysRoot                    string
	StorageRoot                string
	ReportDir                  string
	Now                        func() time.Time
	Runner                     CommandRunner
	ModelRuntimeHealthProvider ModelRuntimeHealthProvider
}

type ModelRuntimeHealthProvider interface {
	HostBridgeModelRuntimeHealth(ctx context.Context) ModelRuntimeDiagnostics
}

type Service struct {
	procRoot     string
	sysRoot      string
	storageRoot  string
	reportDir    string
	now          func() time.Time
	runner       CommandRunner
	modelRuntime ModelRuntimeHealthProvider
}

func New(opts Options) *Service {
	now := opts.Now
	if now == nil {
		now = time.Now
	}
	procRoot := strings.TrimSpace(opts.ProcRoot)
	if procRoot == "" {
		procRoot = defaultProcRoot
	}
	sysRoot := strings.TrimSpace(opts.SysRoot)
	if sysRoot == "" {
		sysRoot = defaultSysRoot
	}
	storageRoot := strings.TrimSpace(opts.StorageRoot)
	if storageRoot == "" {
		storageRoot = defaultStorageRoot
	}
	runner := opts.Runner
	if runner == nil {
		runner = newExecRunner(1500 * time.Millisecond)
	}
	return &Service{
		procRoot:     procRoot,
		sysRoot:      sysRoot,
		storageRoot:  storageRoot,
		reportDir:    strings.TrimSpace(opts.ReportDir),
		now:          now,
		runner:       runner,
		modelRuntime: opts.ModelRuntimeHealthProvider,
	}
}

func (s *Service) Snapshot(ctx context.Context) Snapshot {
	if s == nil {
		s = New(Options{})
	}
	capturedAt := s.now().UTC()
	out := Snapshot{
		CapturedAt: capturedAt,
		Host: HostDiagnostics{
			Architecture: runtime.GOARCH,
		},
		CPU: CPUDiagnostics{
			Count: runtime.NumCPU(),
		},
		Disk: DiskDiagnostics{
			Path:          s.storageRoot,
			PressureLevel: PressureUnavailable,
		},
		Memory:       MemoryDiagnostics{PressureLevel: PressureUnavailable},
		GPU:          GPUDiagnostics{Available: false},
		Thermal:      ThermalDiagnostics{Available: false},
		Services:     ServiceDiagnostics{Available: false},
		ModelRuntime: ModelRuntimeDiagnostics{Available: false, State: "unavailable", Warnings: []string{"modelruntime health provider not configured"}},
	}

	out.Host.Hostname = hostname()
	out.Host.OSRelease = s.readOSRelease(&out)
	out.Kernel.Version = s.readTrimmed(&out, "proc.version", s.procPath("version"))
	out.Kernel.Modules = s.readModules(&out)

	if raw := s.readTrimmed(&out, "proc.cmdline", s.procPath("cmdline")); raw != "" {
		params, redactions := redactBootParameters(raw)
		out.Boot.Parameters = params
		out.Redactions = append(out.Redactions, redactions...)
	}
	out.CPU.LoadAverage = s.readLoadAverage(&out)
	out.CPU.UtilizationEstimate = s.readCPUUtilization(&out)
	out.Memory = s.readMemory(&out)
	out.Disk = s.readDisk(&out)
	out.GPU = s.readGPU(ctx, &out)
	out.Thermal = s.readThermal(&out)
	out.Services = s.readServices(ctx, &out)
	if s.modelRuntime != nil {
		out.ModelRuntime = s.modelRuntime.HostBridgeModelRuntimeHealth(ctx)
	}

	out.SourceErrors = sortSourceErrors(out.SourceErrors)
	sort.Strings(out.Warnings)
	out.Degraded = len(out.SourceErrors) > 0 || out.Memory.PressureLevel == PressureCritical || out.Disk.PressureLevel == PressureCritical
	out.SnapshotID = snapshotID(out)
	return out
}

func (s *Service) WriteSnapshot(snapshot Snapshot) (string, error) {
	if strings.TrimSpace(s.reportDir) == "" {
		return "", ErrReportDirRequired
	}
	if snapshot.SnapshotID == "" {
		return "", ErrSnapshotNil
	}
	if err := os.MkdirAll(s.reportDir, 0o750); err != nil {
		return "", err
	}
	path := filepath.Join(s.reportDir, snapshot.SnapshotID+".json")
	body, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(path, append(body, '\n'), 0o640); err != nil {
		return "", err
	}
	return path, nil
}

func (s *Service) procPath(name string) string {
	return filepath.Join(s.procRoot, name)
}

func (s *Service) sysPath(parts ...string) string {
	all := append([]string{s.sysRoot}, parts...)
	return filepath.Join(all...)
}

func (s *Service) readTrimmed(snapshot *Snapshot, source, path string) string {
	body, err := readHostProbeFile(path)
	if err != nil {
		addSourceError(snapshot, source, err)
		return ""
	}
	return strings.TrimSpace(string(body))
}

func (s *Service) readOSRelease(snapshot *Snapshot) string {
	candidates := []string{
		filepath.Join(filepath.Dir(s.procRoot), "etc", "os-release"),
		"/etc/os-release",
	}
	for _, path := range candidates {
		body, err := readHostProbeFile(path)
		if err != nil {
			continue
		}
		values := parseKeyValueLines(string(body))
		return firstNonEmpty(values["PRETTY_NAME"], values["NAME"])
	}
	addSourceError(snapshot, "etc.os-release", fs.ErrNotExist)
	return ""
}

func (s *Service) readModules(snapshot *Snapshot) []ModuleDiagnostics {
	raw := s.readTrimmed(snapshot, "proc.modules", s.procPath("modules"))
	if raw == "" {
		return nil
	}
	lines := strings.Split(raw, "\n")
	modules := make([]ModuleDiagnostics, 0, minInt(len(lines), maxModules))
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}
		refcount, _ := strconv.Atoi(fields[2])
		state := ""
		if len(fields) >= 5 {
			state = fields[4]
		}
		modules = append(modules, ModuleDiagnostics{Name: fields[0], RefCount: refcount, State: state})
		if len(modules) >= maxModules {
			snapshot.Warnings = append(snapshot.Warnings, "module list truncated")
			break
		}
	}
	sort.Slice(modules, func(i, j int) bool { return modules[i].Name < modules[j].Name })
	return modules
}

func (s *Service) readLoadAverage(snapshot *Snapshot) []float64 {
	raw := s.readTrimmed(snapshot, "proc.loadavg", s.procPath("loadavg"))
	if raw == "" {
		return nil
	}
	fields := strings.Fields(raw)
	out := make([]float64, 0, minInt(3, len(fields)))
	for i := 0; i < len(fields) && i < 3; i++ {
		value, err := strconv.ParseFloat(fields[i], 64)
		if err != nil {
			addSourceError(snapshot, "proc.loadavg", err)
			return nil
		}
		out = append(out, round3(value))
	}
	return out
}

func (s *Service) readCPUUtilization(snapshot *Snapshot) float64 {
	raw := s.readTrimmed(snapshot, "proc.stat", s.procPath("stat"))
	if raw == "" {
		return 0
	}
	for _, line := range strings.Split(raw, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 5 || fields[0] != "cpu" {
			continue
		}
		var total uint64
		var idle uint64
		for i, field := range fields[1:] {
			value, err := strconv.ParseUint(field, 10, 64)
			if err != nil {
				addSourceError(snapshot, "proc.stat", err)
				return 0
			}
			total += value
			if i == 3 || i == 4 {
				idle += value
			}
		}
		if total == 0 {
			return 0
		}
		return round3(float64(total-idle) / float64(total))
	}
	return 0
}

func (s *Service) readMemory(snapshot *Snapshot) MemoryDiagnostics {
	raw := s.readTrimmed(snapshot, "proc.meminfo", s.procPath("meminfo"))
	if raw == "" {
		return MemoryDiagnostics{PressureLevel: PressureUnavailable}
	}
	values := map[string]uint64{}
	for _, line := range strings.Split(raw, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		key := strings.TrimSuffix(fields[0], ":")
		value, err := strconv.ParseUint(fields[1], 10, 64)
		if err != nil {
			addSourceError(snapshot, "proc.meminfo", err)
			continue
		}
		values[key] = value * 1024
	}
	total := values["MemTotal"]
	available := values["MemAvailable"]
	return MemoryDiagnostics{
		TotalBytes:     total,
		AvailableBytes: available,
		SwapTotalBytes: values["SwapTotal"],
		SwapFreeBytes:  values["SwapFree"],
		PressureLevel:  ClassifyMemoryPressure(total, available),
	}
}

func (s *Service) readGPU(ctx context.Context, snapshot *Snapshot) GPUDiagnostics {
	if _, err := s.runner.LookPath("nvidia-smi"); err != nil {
		return GPUDiagnostics{Available: false, Warnings: []string{"nvidia-smi unavailable"}}
	}
	result, err := s.runner.Run(ctx, "nvidia-smi", "--query-gpu=name,driver_version,memory.total,memory.free,memory.used", "--format=csv,noheader,nounits")
	if err != nil {
		addSourceError(snapshot, "gpu.nvidia-smi", err)
		return GPUDiagnostics{Available: false, Vendor: "nvidia", Warnings: []string{"nvidia-smi failed"}}
	}
	devices := parseNvidiaSMI(result.Stdout)
	return GPUDiagnostics{Available: len(devices) > 0, Vendor: "nvidia", Devices: devices}
}

func (s *Service) readThermal(snapshot *Snapshot) ThermalDiagnostics {
	root := s.sysPath("class", "thermal")
	entries, err := os.ReadDir(root)
	if err != nil {
		addSourceError(snapshot, "sys.thermal", err)
		return ThermalDiagnostics{Available: false, Warnings: []string{"thermal sysfs unavailable"}}
	}
	sensors := []ThermalSensor{}
	for _, entry := range entries {
		if !entry.IsDir() || !strings.HasPrefix(entry.Name(), "thermal_zone") {
			continue
		}
		label := strings.TrimSpace(readFileString(filepath.Join(root, entry.Name(), "type")))
		if label == "" {
			label = entry.Name()
		}
		tempRaw := strings.TrimSpace(readFileString(filepath.Join(root, entry.Name(), "temp")))
		tempMilliC, err := strconv.ParseFloat(tempRaw, 64)
		if err != nil {
			continue
		}
		sensors = append(sensors, ThermalSensor{Label: label, TemperatureC: round3(tempMilliC / 1000)})
		if len(sensors) >= maxThermalSensors {
			snapshot.Warnings = append(snapshot.Warnings, "thermal sensor list truncated")
			break
		}
	}
	sort.Slice(sensors, func(i, j int) bool { return sensors[i].Label < sensors[j].Label })
	return ThermalDiagnostics{Available: len(sensors) > 0, Sensors: sensors}
}

func (s *Service) readServices(ctx context.Context, snapshot *Snapshot) ServiceDiagnostics {
	if _, err := s.runner.LookPath("systemctl"); err != nil {
		return ServiceDiagnostics{Available: false, Warnings: []string{"systemctl unavailable"}}
	}
	result, err := s.runner.Run(ctx, "systemctl", "show", "forge-core.service", "--property=Id,LoadState,ActiveState,SubState", "--no-pager")
	if err != nil {
		addSourceError(snapshot, "services.systemctl", err)
		return ServiceDiagnostics{Available: false, Warnings: []string{"systemctl show failed"}}
	}
	unit := parseSystemctlShow(result.Stdout)
	if unit.Name == "" {
		return ServiceDiagnostics{Available: true}
	}
	out := ServiceDiagnostics{Available: true, Units: []UnitState{unit}}
	if unit.Active == "failed" || unit.LoadState == "not-found" {
		out.Failed = []UnitState{unit}
	}
	return out
}

func ClassifyMemoryPressure(totalBytes, availableBytes uint64) string {
	if totalBytes == 0 || availableBytes > totalBytes {
		return PressureUnavailable
	}
	ratio := float64(availableBytes) / float64(totalBytes)
	if ratio < 0.10 {
		return PressureCritical
	}
	if ratio < 0.20 {
		return PressureElevated
	}
	return PressureNormal
}

func ClassifyDiskPressure(totalBytes, freeBytes uint64) string {
	if totalBytes == 0 || freeBytes > totalBytes {
		return PressureUnavailable
	}
	usedRatio := float64(totalBytes-freeBytes) / float64(totalBytes)
	if usedRatio >= 0.90 {
		return PressureCritical
	}
	if usedRatio >= 0.80 {
		return PressureElevated
	}
	return PressureNormal
}

func parseNvidiaSMI(raw string) []GPUDeviceDiagnostics {
	devices := []GPUDeviceDiagnostics{}
	for _, line := range strings.Split(strings.TrimSpace(raw), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		parts := strings.Split(line, ",")
		if len(parts) < 5 {
			continue
		}
		devices = append(devices, GPUDeviceDiagnostics{
			Name:           strings.TrimSpace(parts[0]),
			DriverVersion:  strings.TrimSpace(parts[1]),
			MemoryTotalMiB: parseFloat(parts[2]),
			MemoryFreeMiB:  parseFloat(parts[3]),
			MemoryUsedMiB:  parseFloat(parts[4]),
		})
	}
	return devices
}

func parseSystemctlShow(raw string) UnitState {
	values := parseKeyValueLines(raw)
	return UnitState{
		Name:      firstNonEmpty(values["Id"], values["ID"]),
		LoadState: firstNonEmpty(values["LoadState"], values["LOAD_STATE"]),
		Active:    firstNonEmpty(values["ActiveState"], values["ACTIVE_STATE"]),
		SubState:  firstNonEmpty(values["SubState"], values["SUB_STATE"]),
	}
}

func parseKeyValueLines(raw string) map[string]string {
	out := map[string]string{}
	for _, line := range strings.Split(raw, "\n") {
		key, value, ok := strings.Cut(strings.TrimSpace(line), "=")
		if !ok {
			continue
		}
		out[strings.TrimSpace(key)] = strings.Trim(strings.TrimSpace(value), `"`)
	}
	return out
}

func addSourceError(snapshot *Snapshot, source string, err error) {
	if snapshot == nil || err == nil {
		return
	}
	snapshot.SourceErrors = append(snapshot.SourceErrors, SourceError{Source: source, Error: boundedError(err)})
}

func sortSourceErrors(in []SourceError) []SourceError {
	sort.Slice(in, func(i, j int) bool {
		if in[i].Source == in[j].Source {
			return in[i].Error < in[j].Error
		}
		return in[i].Source < in[j].Source
	})
	return in
}

func snapshotID(snapshot Snapshot) string {
	clone := snapshot
	clone.SnapshotID = ""
	body, _ := json.Marshal(clone)
	sum := sha256.Sum256(body)
	return "hostdiag_" + hex.EncodeToString(sum[:8])
}

func hostname() string {
	name, err := os.Hostname()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(name)
}

func readFileString(path string) string {
	body, err := readHostProbeFile(path)
	if err != nil {
		return ""
	}
	return string(body)
}

func readHostProbeFile(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	if info, err := f.Stat(); err == nil && info.Size() > maxHostProbeFileBytes {
		return nil, fmt.Errorf("host probe file too large: %d bytes exceeds %d byte limit", info.Size(), maxHostProbeFileBytes)
	}
	body, err := io.ReadAll(io.LimitReader(f, maxHostProbeFileBytes+1))
	if err != nil {
		return nil, err
	}
	if len(body) > maxHostProbeFileBytes {
		return nil, fmt.Errorf("host probe file too large: exceeds %d byte limit", maxHostProbeFileBytes)
	}
	return body, nil
}

func parseFloat(value string) float64 {
	parsed, _ := strconv.ParseFloat(strings.TrimSpace(value), 64)
	return round3(parsed)
}

func round3(value float64) float64 {
	return float64(int64(value*1000+0.5)) / 1000
}

func boundedError(err error) string {
	text := strings.TrimSpace(err.Error())
	if len(text) > 256 {
		return text[:256]
	}
	return text
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (s *Service) String() string {
	return fmt.Sprintf("hostbridge(proc=%s, sys=%s, storage=%s)", s.procRoot, s.sysRoot, s.storageRoot)
}
