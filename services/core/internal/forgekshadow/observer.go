package forgekshadow

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"forge/projectforge/services/core/internal/forgek/shadowharness"
)

type Observer struct {
	cfg     Config
	sink    Sink
	policy  shadowharness.ShadowHarnessPolicy
	now     func() time.Time
	counter atomic.Uint64
}

func NewObserver(cfg Config) *Observer {
	cfg = cfg.normalized()
	return &Observer{
		cfg:    cfg,
		sink:   NewMemorySink(cfg.MaxReports),
		policy: shadowharness.DefaultShadowHarnessPolicy(),
		now:    time.Now,
	}
}

func NewObserverWithSink(cfg Config, sink Sink, now func() time.Time) *Observer {
	cfg = cfg.normalized()
	if sink == nil {
		sink = NewMemorySink(cfg.MaxReports)
	}
	if now == nil {
		now = time.Now
	}
	return &Observer{
		cfg:    cfg,
		sink:   sink,
		policy: shadowharness.DefaultShadowHarnessPolicy(),
		now:    now,
	}
}

func (o *Observer) Enabled() bool {
	return o != nil && o.cfg.Enabled
}

func (o *Observer) Reports() []DiagnosticReport {
	if o == nil || o.sink == nil {
		return nil
	}
	return o.sink.List()
}

func (o *Observer) ObserveBestEffort(ctx context.Context, input ObservationInput) {
	defer func() {
		_ = recover()
	}()
	_ = o.Observe(ctx, input)
}

func (o *Observer) Observe(ctx context.Context, input ObservationInput) error {
	if o == nil || !o.cfg.Enabled {
		return nil
	}
	if err := shadowharness.ValidateShadowHarnessPolicy(o.policy); err != nil {
		return fmt.Errorf("%w: %v", ErrPolicyRejected, err)
	}
	now := o.now()
	metadata, err := safeMetadata(input.Metadata)
	if err != nil {
		return err
	}
	requestID := strings.TrimSpace(input.RequestID)
	if requestID == "" {
		requestID = fmt.Sprintf("shadow-request-%d", o.counter.Add(1))
	}
	workspaceID := strings.TrimSpace(input.WorkspaceID)
	if workspaceID == "" {
		workspaceID = "workspace:unknown"
	}
	livePath := strings.TrimSpace(input.LivePath)
	if livePath == "" {
		livePath = strings.TrimSpace(input.Method + " " + input.Path)
	}
	summary := strings.TrimSpace(input.RequestSummary)
	if summary == "" {
		summary = "read-only live request metadata"
	}

	obs, err := shadowharness.NewShadowObservation(shadowharness.ShadowObservation{
		ObservationID:  fmt.Sprintf("shadow-observation-%d", o.counter.Add(1)),
		WorkspaceID:    workspaceID,
		RequestID:      requestID,
		ObservedAt:     now,
		LivePath:       livePath,
		RequestSummary: summary,
		Metadata:       metadata,
	})
	if err != nil {
		return err
	}

	report, err := shadowharness.NewShadowComparisonReport(shadowharness.ShadowComparisonReport{
		ReportID:        fmt.Sprintf("shadow-report-%d", o.counter.Add(1)),
		WorkspaceID:     workspaceID,
		RequestID:       requestID,
		GeneratedAt:     now,
		ObservationRefs: []string{obs.ObservationID},
		ConsensusShadow: shadowharness.ConsensusShadowReport{
			ReportID:               fmt.Sprintf("shadow-consensus-%s", requestID),
			RequestID:              requestID,
			DiagnosticOnlyVerified: true,
		},
		ContextShadow: shadowharness.ContextShadowReport{
			ReportID:               fmt.Sprintf("shadow-context-%s", requestID),
			RequestID:              requestID,
			DiagnosticOnlyVerified: true,
		},
		RAGShadow: shadowharness.RAGShadowReport{
			ReportID:            fmt.Sprintf("shadow-rag-%s", requestID),
			RequestID:           requestID,
			NoExecutionVerified: true,
		},
		RuntimeShadow: shadowharness.RuntimeShadowReport{
			ReportID:             fmt.Sprintf("shadow-runtime-%s", requestID),
			RequestID:            requestID,
			ProposalOnlyVerified: true,
		},
		KVShadow: shadowharness.KVShadowReport{
			ReportID:                      fmt.Sprintf("shadow-kv-%s", requestID),
			RequestID:                     requestID,
			AccelerationNotMemoryVerified: true,
		},
		LymphaticShadow: shadowharness.LymphaticShadowReport{
			ReportID:                 fmt.Sprintf("shadow-lymphatic-%s", requestID),
			RequestID:                requestID,
			NoSilentMutationVerified: true,
			ProposalsDoNotExecute:    true,
		},
		NoEffectVerified: true,
		Metadata:         metadata,
	})
	if err != nil {
		return err
	}
	if err := shadowharness.ValidateNoEffect(o.policy, report); err != nil {
		return err
	}
	return o.sink.Store(ctx, DiagnosticReport{
		Observation: obs,
		Comparison:  report,
		StoredAt:    now,
	})
}
