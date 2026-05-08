package forgeh

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
	"time"

	"forge/projectforge/services/core/internal/hostbridge"
)

type Options struct {
	Now func() time.Time
}

type Service struct {
	now func() time.Time
}

func New(opts Options) *Service {
	now := opts.Now
	if now == nil {
		now = time.Now
	}
	return &Service{now: now}
}

func (s *Service) Evaluate(snapshot hostbridge.Snapshot) ResourcePolicySnapshot {
	if s == nil {
		s = New(Options{})
	}
	out := ResourcePolicySnapshot{
		CapturedAt:       s.now().UTC(),
		SourceSnapshotID: snapshot.SnapshotID,
		RAMPressure:      ClassifyRAMPressure(snapshot.Memory.TotalBytes, snapshot.Memory.AvailableBytes),
		SwapPressure:     ClassifySwapPressure(snapshot.Memory.SwapTotalBytes, snapshot.Memory.SwapFreeBytes),
		DiskPressure:     ClassifyDiskPressure(snapshot.Disk.TotalBytes, snapshot.Disk.FreeBytes),
		VRAMPressure:     ClassifyVRAMPressure(snapshot.GPU),
		ThermalPressure:  ClassifyThermalPressure(snapshot.Thermal),
		AdvisoryOnly:     true,
	}
	out.OverallPosture = CalculateOverallPosture(out)
	out.LaneDecisions = BuildLaneDecisions(out)
	out.ModelLoadRecommendation = ModelLoadRecommendation(out)
	out.BackgroundWorkRecommendation = BackgroundWorkRecommendation(out)
	out.Warnings = BuildWarnings(snapshot, out)
	out.OperatorActions = BuildOperatorActions(out)
	out.SourceErrors = convertSourceErrors(snapshot.SourceErrors)
	out.PolicyID = policyID(out)
	return out
}

func ClassifyRAMPressure(totalBytes, availableBytes uint64) string {
	if totalBytes == 0 || availableBytes > totalBytes {
		return ResourcePressureUnavailable
	}
	availableRatio := float64(availableBytes) / float64(totalBytes)
	if availableRatio < 0.07 {
		return ResourcePressureCritical
	}
	if availableRatio < 0.15 {
		return ResourcePressureConstrained
	}
	if availableRatio < 0.30 {
		return ResourcePressureElevated
	}
	return ResourcePressureNormal
}

func ClassifySwapPressure(totalBytes, freeBytes uint64) string {
	if totalBytes == 0 {
		return ResourcePressureNormal
	}
	if freeBytes > totalBytes {
		return ResourcePressureUnavailable
	}
	usedRatio := float64(totalBytes-freeBytes) / float64(totalBytes)
	if usedRatio > 0.80 {
		return ResourcePressureCritical
	}
	if usedRatio > 0.50 {
		return ResourcePressureConstrained
	}
	if usedRatio > 0.25 {
		return ResourcePressureElevated
	}
	return ResourcePressureNormal
}

func ClassifyDiskPressure(totalBytes, freeBytes uint64) string {
	if totalBytes == 0 || freeBytes > totalBytes {
		return ResourcePressureUnavailable
	}
	freeRatio := float64(freeBytes) / float64(totalBytes)
	if freeRatio < 0.05 {
		return ResourcePressureCritical
	}
	if freeRatio < 0.10 {
		return ResourcePressureConstrained
	}
	if freeRatio < 0.20 {
		return ResourcePressureElevated
	}
	return ResourcePressureNormal
}

func ClassifyVRAMPressure(gpu hostbridge.GPUDiagnostics) string {
	if !gpu.Available || len(gpu.Devices) == 0 {
		return ResourcePressureUnavailable
	}
	worst := ResourcePressureUnavailable
	for _, device := range gpu.Devices {
		if device.MemoryTotalMiB <= 0 || device.MemoryFreeMiB < 0 || device.MemoryFreeMiB > device.MemoryTotalMiB {
			continue
		}
		freeRatio := device.MemoryFreeMiB / device.MemoryTotalMiB
		switch {
		case freeRatio < 0.10:
			worst = worsePressure(worst, ResourcePressureCritical)
		case freeRatio < 0.20:
			worst = worsePressure(worst, ResourcePressureConstrained)
		case freeRatio < 0.35:
			worst = worsePressure(worst, ResourcePressureElevated)
		default:
			worst = worsePressure(worst, ResourcePressureNormal)
		}
	}
	return worst
}

func ClassifyThermalPressure(thermal hostbridge.ThermalDiagnostics) string {
	if !thermal.Available || len(thermal.Sensors) == 0 {
		return ResourcePressureUnavailable
	}
	worst := ResourcePressureNormal
	for _, sensor := range thermal.Sensors {
		switch {
		case sensor.TemperatureC >= 90:
			worst = worsePressure(worst, ResourcePressureCritical)
		case sensor.TemperatureC >= 80:
			worst = worsePressure(worst, ResourcePressureConstrained)
		case sensor.TemperatureC >= 70:
			worst = worsePressure(worst, ResourcePressureElevated)
		default:
			worst = worsePressure(worst, ResourcePressureNormal)
		}
	}
	return worst
}

func CalculateOverallPosture(policy ResourcePolicySnapshot) string {
	pressures := []string{
		policy.RAMPressure,
		policy.SwapPressure,
		policy.DiskPressure,
		policy.VRAMPressure,
		policy.ThermalPressure,
	}
	for _, pressure := range pressures {
		if pressure == ResourcePressureCritical {
			return ResourcePostureCritical
		}
	}
	for _, pressure := range pressures {
		if pressure == ResourcePressureConstrained {
			return ResourcePostureConstrained
		}
	}
	for _, pressure := range pressures {
		if pressure == ResourcePressureElevated {
			return ResourcePostureDegraded
		}
	}
	if policy.RAMPressure == ResourcePressureUnavailable || policy.DiskPressure == ResourcePressureUnavailable {
		return ResourcePostureDegraded
	}
	return ResourcePostureNormal
}

func BuildLaneDecisions(policy ResourcePolicySnapshot) map[string]LaneDecision {
	lanes := map[string]LaneDecision{
		WorkloadLaneInteractive:      interactiveDecision(policy),
		WorkloadLaneDesktopUI:        interactiveDecision(policy),
		WorkloadLaneBackgroundIngest: backgroundIngestDecision(policy),
		WorkloadLaneEmbedding:        embeddingDecision(policy),
		WorkloadLaneModelLoad:        modelLoadDecision(policy),
		WorkloadLaneModelInference:   modelInferenceDecision(policy),
		WorkloadLaneMaintenance:      maintenanceDecision(policy),
	}
	return lanes
}

func ModelLoadRecommendation(policy ResourcePolicySnapshot) string {
	if policy.RAMPressure == ResourcePressureUnavailable {
		return ModelLoadUnavailable
	}
	if policy.RAMPressure == ResourcePressureCritical || policy.VRAMPressure == ResourcePressureCritical {
		return ModelLoadDenyNewModelLoad
	}
	if policy.RAMPressure == ResourcePressureConstrained || policy.VRAMPressure == ResourcePressureConstrained {
		return ModelLoadDeferLargeModel
	}
	if policy.VRAMPressure == ResourcePressureUnavailable {
		if policy.RAMPressure == ResourcePressureNormal || policy.RAMPressure == ResourcePressureElevated {
			return ModelLoadCPUOnlySafeMode
		}
		return ModelLoadCurrentModelOnly
	}
	if policy.RAMPressure == ResourcePressureElevated || policy.VRAMPressure == ResourcePressureElevated {
		return ModelLoadCurrentModelOnly
	}
	return ModelLoadSmallLocalOK
}

func BackgroundWorkRecommendation(policy ResourcePolicySnapshot) string {
	if policy.OverallPosture == ResourcePostureCritical {
		return BackgroundWorkDeny
	}
	if policy.OverallPosture == ResourcePostureConstrained || policy.OverallPosture == ResourcePostureDegraded {
		return BackgroundWorkDefer
	}
	return BackgroundWorkAllow
}

func BuildWarnings(snapshot hostbridge.Snapshot, policy ResourcePolicySnapshot) []string {
	warnings := append([]string{}, snapshot.Warnings...)
	if policy.VRAMPressure == ResourcePressureUnavailable {
		warnings = append(warnings, "VRAM diagnostics unavailable; GPU model load policy is advisory only.")
	}
	if policy.ThermalPressure == ResourcePressureUnavailable {
		warnings = append(warnings, "Thermal diagnostics unavailable; continue without thermal policy.")
	}
	if policy.RAMPressure == ResourcePressureCritical {
		warnings = append(warnings, "RAM pressure is critical.")
	}
	if policy.DiskPressure == ResourcePressureCritical {
		warnings = append(warnings, "Disk pressure is critical.")
	}
	sort.Strings(warnings)
	return compactStrings(warnings)
}

func BuildOperatorActions(policy ResourcePolicySnapshot) []string {
	actions := []string{}
	if policy.RAMPressure == ResourcePressureElevated || policy.RAMPressure == ResourcePressureConstrained {
		actions = append(actions, "Defer background ingest until RAM pressure returns to normal.")
	}
	if policy.RAMPressure == ResourcePressureCritical {
		actions = append(actions, "Pause non-essential work and reduce RAM pressure before loading new models.")
	}
	if policy.DiskPressure == ResourcePressureElevated || policy.DiskPressure == ResourcePressureConstrained || policy.DiskPressure == ResourcePressureCritical {
		actions = append(actions, "Free disk space under the FORGE storage root before running embedding or ingest jobs.")
	}
	if policy.VRAMPressure == ResourcePressureConstrained || policy.VRAMPressure == ResourcePressureCritical {
		actions = append(actions, "Avoid loading large GPU models; VRAM is constrained.")
	}
	if policy.VRAMPressure == ResourcePressureUnavailable {
		actions = append(actions, "Use CPU-only safe mode for model work unless GPU diagnostics become available.")
	}
	if policy.ThermalPressure == ResourcePressureUnavailable {
		actions = append(actions, "Thermal diagnostics unavailable; continue without thermal policy.")
	}
	if policy.ThermalPressure == ResourcePressureConstrained || policy.ThermalPressure == ResourcePressureCritical {
		actions = append(actions, "Defer heavy background work until thermal pressure falls.")
	}
	sort.Strings(actions)
	return compactStrings(actions)
}

func interactiveDecision(policy ResourcePolicySnapshot) LaneDecision {
	switch policy.OverallPosture {
	case ResourcePostureCritical:
		return LaneDecision{Decision: PolicyDecisionDeny, Reasons: []string{"overall posture critical"}}
	case ResourcePostureConstrained:
		return LaneDecision{Decision: PolicyDecisionAllowWithWarning, Reasons: []string{"overall posture constrained"}}
	case ResourcePostureDegraded:
		return LaneDecision{Decision: PolicyDecisionAllowWithWarning, Reasons: []string{"overall posture degraded"}}
	default:
		return LaneDecision{Decision: PolicyDecisionAllow}
	}
}

func backgroundIngestDecision(policy ResourcePolicySnapshot) LaneDecision {
	switch policy.OverallPosture {
	case ResourcePostureCritical:
		return LaneDecision{Decision: PolicyDecisionDeny, Reasons: []string{"overall posture critical"}}
	case ResourcePostureConstrained, ResourcePostureDegraded:
		return LaneDecision{Decision: PolicyDecisionDefer, Reasons: []string{"background work should wait for resource recovery"}}
	default:
		return LaneDecision{Decision: PolicyDecisionAllow}
	}
}

func embeddingDecision(policy ResourcePolicySnapshot) LaneDecision {
	if policy.RAMPressure == ResourcePressureUnavailable || policy.DiskPressure == ResourcePressureUnavailable {
		return LaneDecision{Decision: PolicyDecisionUnavailable, Reasons: []string{"RAM or disk diagnostics unavailable"}}
	}
	if policy.RAMPressure == ResourcePressureCritical || policy.DiskPressure == ResourcePressureCritical {
		return LaneDecision{Decision: PolicyDecisionDeny, Reasons: []string{"RAM or disk pressure critical"}}
	}
	if policy.RAMPressure == ResourcePressureConstrained || policy.DiskPressure == ResourcePressureConstrained || policy.RAMPressure == ResourcePressureElevated || policy.DiskPressure == ResourcePressureElevated {
		return LaneDecision{Decision: PolicyDecisionDefer, Reasons: []string{"RAM or disk pressure elevated"}}
	}
	return LaneDecision{Decision: PolicyDecisionAllow}
}

func modelLoadDecision(policy ResourcePolicySnapshot) LaneDecision {
	if policy.RAMPressure == ResourcePressureUnavailable {
		return LaneDecision{Decision: PolicyDecisionUnavailable, Reasons: []string{"RAM diagnostics unavailable"}}
	}
	if policy.RAMPressure == ResourcePressureCritical || policy.VRAMPressure == ResourcePressureCritical {
		return LaneDecision{Decision: PolicyDecisionDeny, Reasons: []string{"RAM or VRAM pressure critical"}}
	}
	if policy.RAMPressure == ResourcePressureConstrained || policy.VRAMPressure == ResourcePressureConstrained {
		return LaneDecision{Decision: PolicyDecisionDefer, Reasons: []string{"RAM or VRAM constrained"}}
	}
	if policy.RAMPressure == ResourcePressureElevated || policy.VRAMPressure == ResourcePressureElevated || policy.VRAMPressure == ResourcePressureUnavailable {
		return LaneDecision{Decision: PolicyDecisionAllowWithWarning, Reasons: []string{"model load is advisory under current pressure"}}
	}
	return LaneDecision{Decision: PolicyDecisionAllow}
}

func modelInferenceDecision(policy ResourcePolicySnapshot) LaneDecision {
	if policy.OverallPosture == ResourcePostureCritical {
		return LaneDecision{Decision: PolicyDecisionDeny, Reasons: []string{"overall posture critical"}}
	}
	if policy.OverallPosture == ResourcePostureConstrained {
		return LaneDecision{Decision: PolicyDecisionDefer, Reasons: []string{"defer large or new inference work while constrained"}}
	}
	if policy.OverallPosture == ResourcePostureDegraded {
		return LaneDecision{Decision: PolicyDecisionAllowWithWarning, Reasons: []string{"prefer current or small models while degraded"}}
	}
	return LaneDecision{Decision: PolicyDecisionAllow}
}

func maintenanceDecision(policy ResourcePolicySnapshot) LaneDecision {
	switch policy.OverallPosture {
	case ResourcePostureCritical:
		return LaneDecision{Decision: PolicyDecisionDeny, Reasons: []string{"overall posture critical"}}
	case ResourcePostureConstrained:
		return LaneDecision{Decision: PolicyDecisionDefer, Reasons: []string{"overall posture constrained"}}
	default:
		return LaneDecision{Decision: PolicyDecisionAllow}
	}
}

func worsePressure(left, right string) string {
	if pressureRank(right) > pressureRank(left) {
		return right
	}
	return left
}

func pressureRank(pressure string) int {
	switch pressure {
	case ResourcePressureCritical:
		return 4
	case ResourcePressureConstrained:
		return 3
	case ResourcePressureElevated:
		return 2
	case ResourcePressureNormal:
		return 1
	default:
		return 0
	}
}

func policyID(policy ResourcePolicySnapshot) string {
	clone := policy
	clone.PolicyID = ""
	body, _ := json.Marshal(clone)
	sum := sha256.Sum256(body)
	return "forgeh_policy_" + hex.EncodeToString(sum[:8])
}

func convertSourceErrors(in []hostbridge.SourceError) []ResourcePolicyError {
	out := make([]ResourcePolicyError, 0, len(in))
	for _, item := range in {
		out = append(out, ResourcePolicyError{Source: item.Source, Error: item.Error})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Source == out[j].Source {
			return out[i].Error < out[j].Error
		}
		return out[i].Source < out[j].Source
	})
	return out
}

func compactStrings(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	out := make([]string, 0, len(in))
	seen := map[string]struct{}{}
	for _, item := range in {
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	return out
}
