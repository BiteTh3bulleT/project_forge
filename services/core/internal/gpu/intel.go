package gpu

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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

const intelToolOutputLimit = 1 << 20

var errIntelGPUTopOutputTooLarge = errors.New("intel_gpu_top output too large")

type boundedToolOutput struct {
	buf       bytes.Buffer
	limit     int
	truncated bool
}

func newBoundedToolOutput(limit int) *boundedToolOutput {
	return &boundedToolOutput{limit: limit}
}

func (b *boundedToolOutput) Write(p []byte) (int, error) {
	if b.limit <= 0 {
		b.truncated = true
		return len(p), nil
	}
	remaining := b.limit - b.buf.Len()
	if remaining <= 0 {
		b.truncated = true
		return len(p), nil
	}
	if len(p) > remaining {
		_, _ = b.buf.Write(p[:remaining])
		b.truncated = true
		return len(p), nil
	}
	_, _ = b.buf.Write(p)
	return len(p), nil
}

func (b *boundedToolOutput) String() string {
	out := b.buf.String()
	if b.truncated {
		out += fmt.Sprintf("\n[forge: Intel telemetry tool output truncated at %d bytes]", b.limit)
	}
	return out
}

type limitedToolReader struct {
	r         io.Reader
	remaining int64
}

func newLimitedToolReader(r io.Reader, limit int64) *limitedToolReader {
	return &limitedToolReader{r: r, remaining: limit}
}

func (r *limitedToolReader) Read(p []byte) (int, error) {
	if r.remaining <= 0 {
		return 0, errIntelGPUTopOutputTooLarge
	}
	if int64(len(p)) > r.remaining {
		p = p[:r.remaining]
	}
	n, err := r.r.Read(p)
	r.remaining -= int64(n)
	return n, err
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
	if zeErr == nil {
		zeOut, err := runTool(ctx, s.timeout, zePath)
		if err != nil {
			out.Warnings = append(out.Warnings, "Level Zero ze_info failed: "+err.Error())
		} else {
			devices := parseZEInfo(zeOut)
			if len(devices) == 0 {
				devices = out.Devices
			}
			out.Available = true
			out.State = "available"
			out.Detail = "Intel Level Zero device available"
			out.Devices = devices
		}
	} else {
		out.Warnings = append(out.Warnings, "Level Zero ze_info unavailable: "+zeErr.Error())
		out.Warnings = append(out.Warnings, "install level-zero tools or set FORGE_INTEL_LEVEL_ZERO_ZE_INFO_PATH")
	}

	if topPath, err := resolveOptionalTool(s.gpuTopPath, "intel_gpu_top"); err == nil {
		if sample, sampleErr := runIntelGPUTop(ctx, s.timeout, topPath); sampleErr == nil {
			mergeIntelGPUTop(&out, sample)
			out.Available = true
			out.State = "available"
			if out.Detail == "" {
				out.Detail = "Intel GPU telemetry available through intel_gpu_top"
			}
		} else {
			out.Warnings = append(out.Warnings, "intel_gpu_top unavailable: "+sampleErr.Error())
		}
	} else {
		out.Warnings = append(out.Warnings, "intel_gpu_top unavailable: "+err.Error())
	}
	if !out.Available {
		out.Healthy = false
		out.State = "degraded"
		out.Detail = "Intel GPU telemetry unavailable"
	}
	return out
}

func detectIntelRenderNodes() []DeviceTelemetry {
	matches, _ := filepath.Glob("/dev/dri/renderD*")
	sysMatches, _ := filepath.Glob("/sys/class/drm/renderD*")
	matches = append(matches, sysMatches...)
	out := make([]DeviceTelemetry, 0, len(matches))
	seen := map[string]struct{}{}
	for _, match := range matches {
		index := filepath.Base(match)
		if _, ok := seen[index]; ok {
			continue
		}
		seen[index] = struct{}{}
		out = append(out, DeviceTelemetry{Index: index, UUID: match})
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
	stdout := newBoundedToolOutput(intelToolOutputLimit)
	stderr := newBoundedToolOutput(intelToolOutputLimit)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	err := cmd.Run()
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
	return stdout.String(), nil
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
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(runCtx, path, "-J", "-s", "250", "-o", "-")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	stderr := newBoundedToolOutput(intelToolOutputLimit)
	cmd.Stderr = stderr
	if err := cmd.Start(); err != nil {
		return nil, err
	}

	var sample map[string]any
	decodeErr := json.NewDecoder(newLimitedToolReader(stdout, intelToolOutputLimit)).Decode(&sample)
	if cmd.Process != nil {
		_ = cmd.Process.Kill()
	}
	waitErr := cmd.Wait()
	if decodeErr != nil {
		if runCtx.Err() != nil {
			return nil, runCtx.Err()
		}
		msg := strings.TrimSpace(stderr.String())
		if msg != "" {
			return nil, fmt.Errorf("%w: %s", decodeErr, msg)
		}
		return nil, decodeErr
	}
	if len(sample) == 0 {
		return nil, fmt.Errorf("empty intel_gpu_top sample")
	}
	if waitErr != nil && runCtx.Err() != nil {
		return nil, runCtx.Err()
	}
	return sample, nil
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
