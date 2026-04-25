package gpu

import (
	"bufio"
	"context"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type Options struct {
	Enabled                 bool
	Endpoint                string
	Timeout                 time.Duration
	MemoryPressureThreshold float64
}

type Service struct {
	enabled                 bool
	endpoint                string
	client                  *http.Client
	memoryPressureThreshold float64
}

type DeviceTelemetry struct {
	Index          string  `json:"index,omitempty"`
	UUID           string  `json:"uuid,omitempty"`
	GPUUtilization float64 `json:"gpuUtilization,omitempty"`
	MemoryUsedMiB  float64 `json:"memoryUsedMiB,omitempty"`
	MemoryFreeMiB  float64 `json:"memoryFreeMiB,omitempty"`
	MemoryTotalMiB float64 `json:"memoryTotalMiB,omitempty"`
	MemoryPressure float64 `json:"memoryPressure,omitempty"`
	PowerWatts     float64 `json:"powerWatts,omitempty"`
	TemperatureC   float64 `json:"temperatureC,omitempty"`
}

type Telemetry struct {
	Enabled                 bool              `json:"enabled"`
	Available               bool              `json:"available"`
	Healthy                 bool              `json:"healthy"`
	State                   string            `json:"state"`
	Endpoint                string            `json:"endpoint,omitempty"`
	Detail                  string            `json:"detail,omitempty"`
	MemoryPressure          float64           `json:"memoryPressure,omitempty"`
	MemoryPressureThreshold float64           `json:"memoryPressureThreshold,omitempty"`
	BackgroundAdmissionOK   bool              `json:"backgroundAdmissionOk"`
	Devices                 []DeviceTelemetry `json:"devices,omitempty"`
	Warnings                []string          `json:"warnings,omitempty"`
}

func New(opts Options) *Service {
	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = 1500 * time.Millisecond
	}
	threshold := opts.MemoryPressureThreshold
	if threshold <= 0 || threshold > 1 {
		threshold = 0.90
	}
	return &Service{
		enabled:                 opts.Enabled,
		endpoint:                strings.TrimSpace(opts.Endpoint),
		client:                  &http.Client{Timeout: timeout},
		memoryPressureThreshold: threshold,
	}
}

func (s *Service) Close() error { return nil }

func (s *Service) Snapshot(ctx context.Context) Telemetry {
	if s == nil {
		return Telemetry{Enabled: false, Healthy: true, State: "disabled", BackgroundAdmissionOK: true}
	}
	out := Telemetry{
		Enabled:                 s.enabled,
		Healthy:                 true,
		State:                   "disabled",
		Endpoint:                s.endpoint,
		MemoryPressureThreshold: s.memoryPressureThreshold,
		BackgroundAdmissionOK:   true,
	}
	if !s.enabled {
		out.Detail = "NVIDIA DCGM telemetry disabled"
		return out
	}
	if s.endpoint == "" {
		out.Healthy = false
		out.State = "degraded"
		out.Detail = "NVIDIA DCGM endpoint is not configured"
		out.Warnings = append(out.Warnings, "dcgm endpoint missing")
		return out
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.endpoint, nil)
	if err != nil {
		out.Healthy = false
		out.State = "degraded"
		out.Detail = err.Error()
		return out
	}
	res, err := s.client.Do(req)
	if err != nil {
		out.Healthy = false
		out.State = "degraded"
		out.Detail = err.Error()
		out.Warnings = append(out.Warnings, "dcgm exporter unavailable")
		return out
	}
	defer res.Body.Close()
	if res.StatusCode >= 300 {
		out.Healthy = false
		out.State = "degraded"
		out.Detail = fmt.Sprintf("dcgm exporter returned %s", res.Status)
		return out
	}
	devices := parseDCGMMetrics(res.Body)
	out.Available = true
	out.State = "available"
	out.Detail = "NVIDIA DCGM exporter reachable"
	out.Devices = devices
	out.MemoryPressure = maxMemoryPressure(devices)
	if out.MemoryPressure >= s.memoryPressureThreshold {
		out.State = "pressure"
		out.BackgroundAdmissionOK = false
		out.Warnings = append(out.Warnings, "gpu memory pressure blocks background admission")
	}
	return out
}

func parseDCGMMetrics(r interface{ Read([]byte) (int, error) }) []DeviceTelemetry {
	devices := map[string]*DeviceTelemetry{}
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") || !strings.HasPrefix(line, "DCGM_FI_DEV_") {
			continue
		}
		name, labels, value, ok := parsePromLine(line)
		if !ok {
			continue
		}
		key := firstNonEmpty(labels["UUID"], labels["gpu"], labels["device"])
		if key == "" {
			key = "0"
		}
		dev := devices[key]
		if dev == nil {
			dev = &DeviceTelemetry{Index: firstNonEmpty(labels["gpu"], labels["device"]), UUID: labels["UUID"]}
			devices[key] = dev
		}
		switch name {
		case "DCGM_FI_DEV_GPU_UTIL":
			dev.GPUUtilization = value
		case "DCGM_FI_DEV_FB_USED":
			dev.MemoryUsedMiB = value
		case "DCGM_FI_DEV_FB_FREE":
			dev.MemoryFreeMiB = value
		case "DCGM_FI_DEV_FB_TOTAL":
			dev.MemoryTotalMiB = value
		case "DCGM_FI_DEV_POWER_USAGE":
			dev.PowerWatts = value
		case "DCGM_FI_DEV_GPU_TEMP":
			dev.TemperatureC = value
		}
	}
	out := make([]DeviceTelemetry, 0, len(devices))
	for _, dev := range devices {
		total := dev.MemoryTotalMiB
		if total <= 0 && dev.MemoryUsedMiB > 0 && dev.MemoryFreeMiB >= 0 {
			total = dev.MemoryUsedMiB + dev.MemoryFreeMiB
			dev.MemoryTotalMiB = total
		}
		if total > 0 {
			dev.MemoryPressure = round3(dev.MemoryUsedMiB / total)
		}
		out = append(out, *dev)
	}
	return out
}

func parsePromLine(line string) (string, map[string]string, float64, bool) {
	parts := strings.Fields(line)
	if len(parts) < 2 {
		return "", nil, 0, false
	}
	left := parts[0]
	value, err := strconv.ParseFloat(parts[len(parts)-1], 64)
	if err != nil || math.IsNaN(value) || math.IsInf(value, 0) {
		return "", nil, 0, false
	}
	name := left
	labels := map[string]string{}
	if idx := strings.Index(left, "{"); idx >= 0 {
		name = left[:idx]
		rawLabels := strings.TrimSuffix(left[idx+1:], "}")
		for _, item := range strings.Split(rawLabels, ",") {
			kv := strings.SplitN(item, "=", 2)
			if len(kv) != 2 {
				continue
			}
			labels[strings.TrimSpace(kv[0])] = strings.Trim(strings.TrimSpace(kv[1]), `"`)
		}
	}
	return name, labels, value, true
}

func maxMemoryPressure(devices []DeviceTelemetry) float64 {
	max := 0.0
	for _, dev := range devices {
		if dev.MemoryPressure > max {
			max = dev.MemoryPressure
		}
	}
	return round3(max)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func round3(v float64) float64 {
	return math.Round(v*1000) / 1000
}
