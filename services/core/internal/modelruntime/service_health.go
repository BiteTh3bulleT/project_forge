package modelruntime

import (
	"context"
	"fmt"
	"strings"
)

func (s *Service) Health(ctx context.Context) (RuntimeHealth, error) {
	s.schedulerMu.Lock()
	s.updateCooldownStateLocked(s.clock().UTC())
	s.schedulerMu.Unlock()
	health := RuntimeHealth{
		Healthy:         true,
		RuntimeEnabled:  true,
		GPUAware:        s.gpuEnabled,
		DegradedReasons: nil,
		PolicyWarnings:  nil,
		ResourceLimits:  s.resourceLimitsSnapshot(),
		Backends:        map[ModelBackendKind]BackendHealth{},
		Loaded:          map[ModelBackendKind]string{},
		Scheduler:       s.SchedulerSnapshot(),
		State:           RuntimeHealthAvailable,
	}

	s.mu.RLock()
	for kind, modelID := range s.loadedBy {
		if modelID != "" {
			health.Loaded[kind] = modelID
		}
	}
	backends := make(map[ModelBackendKind]ModelBackend, len(s.backends))
	for kind, backend := range s.backends {
		backends[kind] = backend
	}
	s.mu.RUnlock()

	if !s.gpuEnabled {
		health.GPUAware = false
		health.PolicyWarnings = append(health.PolicyWarnings, "gpu acceleration disabled")
	}
	if s.gpuTelemetry != nil {
		if telemetry, err := s.gpuTelemetry(ctx); err == nil {
			health.GPUTelemetry = &telemetry
			if s.gpuEnabled && telemetry.Enabled && !telemetry.Healthy {
				health.DegradedReasons = append(health.DegradedReasons, "gpu telemetry degraded: "+telemetry.Detail)
				if health.State == RuntimeHealthAvailable {
					health.State = RuntimeHealthDegraded
				}
			}
			if s.gpuEnabled && telemetry.Enabled && !telemetry.BackgroundAdmissionOK {
				health.PolicyWarnings = append(health.PolicyWarnings, "gpu telemetry blocks background workloads under pressure")
			}
		} else {
			health.DegradedReasons = append(health.DegradedReasons, "gpu telemetry unavailable: "+err.Error())
			if health.State == RuntimeHealthAvailable {
				health.State = RuntimeHealthDegraded
			}
		}
	}
	if s.underCooldown {
		health.State = RuntimeHealthCooldown
		health.PolicyWarnings = append(health.PolicyWarnings, "background workload cooldown active")
	}

	if s.gpuEnabled && s.gpuDegradeOnUnavailable && !s.gpuCurrentlyAvailableLocked() {
		health.State = RuntimeHealthUnavailable
		health.DegradedReasons = append(health.DegradedReasons, "no healthy GPU-backed runtime backend available")
		health.Healthy = false
	}
	for kind, backend := range backends {
		h, err := backend.Health(ctx)
		h.Supervision = s.recordBackendSupervision(kind, h, err)
		health.Backends[kind] = h
		if err != nil {
			health.Healthy = false
			health.DegradedReasons = append(health.DegradedReasons, fmt.Sprintf("%s: %s", kind, err))
			continue
		}
		if !h.Healthy {
			health.Healthy = false
			health.DegradedReasons = append(health.DegradedReasons, fmt.Sprintf("%s backend unhealthy", kind))
		}
	}
	if health.State == RuntimeHealthAvailable && !health.Healthy {
		health.State = RuntimeHealthDegraded
	}
	if schedulerSnapshotHasBackpressure(health.Scheduler) && health.State == RuntimeHealthAvailable {
		health.State = RuntimeHealthOverloaded
	}

	maxQueue := s.maxQueueDepth
	if maxQueue > 0 {
		queued, running := len(health.Scheduler.Queued), len(health.Scheduler.Running)
		if queued >= maxQueue || running > s.maxConcurrentRequests {
			health.State = RuntimeHealthOverloaded
		}
	}
	if health.State == RuntimeHealthAvailable && len(health.DegradedReasons) > 0 {
		health.State = RuntimeHealthDegraded
	}

	return health, nil
}

func (s *Service) recordBackendSupervision(kind ModelBackendKind, health BackendHealth, probeErr error) BackendSupervisionSnapshot {
	s.mu.Lock()
	defer s.mu.Unlock()
	snapshot := s.backendSupervision[kind]
	now := s.clock().UTC()
	snapshot.LastProbeAt = now
	snapshot.ProbeCount++
	snapshot.SupervisionMode = "unmanaged_external_backend"
	snapshot.RestartPolicy = "operator_managed_restart"
	snapshot.RestartSupported = false
	snapshot.RestartAttempted = false
	if probeErr != nil || !health.Healthy {
		snapshot.State = "degraded"
		snapshot.LastFailureAt = now
		snapshot.ConsecutiveFailures++
		snapshot.RestartReason = "backend_health_probe_failed"
		if probeErr != nil {
			snapshot.LastError = probeErr.Error()
		} else {
			snapshot.LastError = strings.TrimSpace(health.Detail)
		}
		if snapshot.ConsecutiveFailures >= 2 {
			snapshot.RestartRecommended = true
			snapshot.RequiresOperatorAction = true
		}
	} else {
		snapshot.State = "healthy"
		snapshot.LastHealthyAt = now
		snapshot.ConsecutiveFailures = 0
		snapshot.LastError = ""
		snapshot.RestartRecommended = false
		snapshot.RequiresOperatorAction = false
		snapshot.RestartReason = ""
	}
	s.backendSupervision[kind] = snapshot
	return snapshot
}

func (s *Service) resourceLimitsSnapshot() RuntimeResourceLimits {
	return RuntimeResourceLimits{
		MaxLoadedModels:                    s.maxLoadedModels,
		MaxQueueDepth:                      s.maxQueueDepth,
		MaxConcurrentRequests:              s.maxConcurrentRequests,
		CompletedHistoryLimit:              s.completedHistoryLimit,
		MaxPromptTokens:                    s.maxPromptTokens,
		MaxOutputTokens:                    s.maxOutputTokens,
		MaxOutputBytes:                     s.maxOutputBytes,
		DefaultTimeoutMs:                   s.defaultTimeout.Milliseconds(),
		LoadTimeoutMs:                      s.loadTimeout.Milliseconds(),
		UnloadTimeoutMs:                    s.unloadTimeout.Milliseconds(),
		GPUEnabled:                         s.gpuEnabled,
		GPURequiredForInteractiveInference: s.gpuRequiredForInteractiveInference,
		GPUBackgroundJobsEnabled:           s.gpuBackgroundJobsEnabled,
		GPUMaxBackgroundJobs:               s.gpuMaxBackgroundJobs,
		GPUBackgroundIdleThresholdMs:       s.gpuBackgroundIdleThreshold.Milliseconds(),
		GPUVRAMHeadroomPercent:             int(s.gpuVRAMHeadroomFraction * 100),
		DegradeOnUnavailableGPU:            s.gpuDegradeOnUnavailable,
	}
}
