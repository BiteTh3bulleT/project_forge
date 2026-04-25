package gpu

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type IntelOptions struct {
	Enabled     bool
	ZEInfoPath  string
	IntelGPUTop string
	Timeout     time.Duration
}

type IntelService struct {
	enabled    bool
	zeInfoPath string
	gpuTopPath string
	timeout    time.Duration
}

func NewIntel(opts IntelOptions) *IntelService {
	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = 1500 * time.Millisecond
	}
	return &IntelService{
		enabled:    opts.Enabled,
		zeInfoPath: strings.TrimSpace(opts.ZEInfoPath),
		gpuTopPath: strings.TrimSpace(opts.IntelGPUTop),
		timeout:    timeout,
	}
}

func (s *IntelService) Close() error { return nil }

func (s *IntelService) Snapshot(ctx context.Context) Telemetry {
	if s == nil {
		return Telemetry{Enabled: false, Healthy: true, State: "disabled", BackgroundAdmissionOK: true}
	}
	out := Telemetry{
		Enabled:               s.enabled,
		Healthy:               true,
		State:                 "disabled",
		BackgroundAdmissionOK: true,
		Devices:               detectIntelRenderNodes(),
	}
	if !s.enabled {
		out.Detail = "Intel Level Zero telemetry disabled"
		return out
	}
	zePath, zeErr := resolveOptionalTool(s.zeInfoPath, "ze_info")
	if zeErr != nil {
		out.Healthy = false
		out.State = "degraded"
		out.Detail = "Level Zero ze_info unavailable: " + zeErr.Error()
		out.Warnings = append(out.Warnings, "install level-zero tools or set FORGE_INTEL_LEVEL_ZERO_ZE_INFO_PATH")
		return out
	}
	zeOut, err := runTool(ctx, s.timeout, zePath)
	if err != nil {
		out.Healthy = false
		out.State = "degraded"
		out.Detail = "Level Zero ze_info failed: " + err.Error()
		return out
	}
	devices := parseZEInfo(zeOut)
	if len(devices) == 0 {
		devices = out.Devices
	}
	out.Available = true
	out.State = "available"
	out.Detail = "Intel Level Zero device available"
	out.Devices = devices

	if topPath, err := resolveOptionalTool(s.gpuTopPath, "intel_gpu_top"); err == nil {
		if sample, sampleErr := runIntelGPUTop(ctx, s.timeout, topPath); sampleErr == nil {
			mergeIntelGPUTop(&out, sample)
		} else {
			out.Warnings = append(out.Warnings, "intel_gpu_top unavailable: "+sampleErr.Error())
		}
	} else {
		out.Warnings = append(out.Warnings, "intel_gpu_top unavailable: "+err.Error())
	}
	return out
}

func detectIntelRenderNodes() []DeviceTelemetry {
	matches, _ := filepath.Glob("/dev/dri/renderD*")
	out := make([]DeviceTelemetry, 0, len(matches))
	for _, match := range matches {
		out = append(out, DeviceTelemetry{Index: filepath.Base(match), UUID: match})
	}
	return out
}

func resolveOptionalTool(configured, fallback string) (string, error) {
	if configured != "" {
		if strings.Contains(configured, string(os.PathSeparator)) {
			if _, err := os.Stat(configured); err != nil {
				return "", err
			}
			return configured, nil
		}
		return exec.LookPath(configured)
	}
	return exec.LookPath(fallback)
}

func runTool(ctx context.Context, timeout time.Duration, path string, args ...string) (string, error) {
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	cmd := exec.CommandContext(runCtx, path, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if runCtx.Err() != nil {
		return "", runCtx.Err()
	}
	if err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg != "" {
			return "", fmt.Errorf("%w: %s", err, msg)
		}
		return "", err
	}
	return string(out), nil
}

func parseZEInfo(raw string) []DeviceTelemetry {
	lines := strings.Split(raw, "\n")
	devices := []DeviceTelemetry{}
	current := DeviceTelemetry{}
	flush := func() {
		if current.Index != "" || current.UUID != "" {
			devices = append(devices, current)
			current = DeviceTelemetry{}
		}
	}
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		lower := strings.ToLower(trimmed)
		if strings.Contains(lower, "device") && strings.Contains(lower, "name") {
			flush()
			current.Index = strings.TrimSpace(valueAfterColon(trimmed))
			continue
		}
		if strings.Contains(lower, "uuid") {
			current.UUID = strings.TrimSpace(valueAfterColon(trimmed))
		}
	}
	flush()
	return devices
}

func valueAfterColon(raw string) string {
	if idx := strings.Index(raw, ":"); idx >= 0 {
		return strings.TrimSpace(raw[idx+1:])
	}
	return raw
}

func runIntelGPUTop(ctx context.Context, timeout time.Duration, path string) (map[string]any, error) {
	raw, err := runTool(ctx, timeout, path, "-J", "-s", "250", "-o", "-")
	if err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(strings.NewReader(raw))
	var last map[string]any
	for decoder.More() {
		var item map[string]any
		if err := decoder.Decode(&item); err != nil {
			break
		}
		last = item
	}
	if last != nil {
		return last, nil
	}
	var one map[string]any
	if err := json.Unmarshal([]byte(raw), &one); err != nil {
		return nil, err
	}
	return one, nil
}

func mergeIntelGPUTop(out *Telemetry, sample map[string]any) {
	if out == nil || sample == nil {
		return
	}
	busy := maxBusy(sample)
	if busy > 0 {
		if len(out.Devices) == 0 {
			out.Devices = []DeviceTelemetry{{Index: "intel"}}
		}
		out.Devices[0].GPUUtilization = busy
	}
}

func maxBusy(raw any) float64 {
	switch v := raw.(type) {
	case map[string]any:
		max := 0.0
		for key, value := range v {
			lower := strings.ToLower(key)
			if strings.Contains(lower, "busy") {
				if f, ok := floatValue(value); ok && f > max {
					max = f
				}
			}
			if nested := maxBusy(value); nested > max {
				max = nested
			}
		}
		return max
	case []any:
		max := 0.0
		for _, item := range v {
			if nested := maxBusy(item); nested > max {
				max = nested
			}
		}
		return max
	default:
		return 0
	}
}

func floatValue(raw any) (float64, bool) {
	switch v := raw.(type) {
	case float64:
		return v, true
	case string:
		f, err := strconv.ParseFloat(strings.TrimSpace(strings.TrimSuffix(v, "%")), 64)
		return f, err == nil
	default:
		return 0, false
	}
}
