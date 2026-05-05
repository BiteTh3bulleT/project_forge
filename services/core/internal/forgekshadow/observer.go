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
	sink := Sink(NewMemorySink(cfg.MaxReports))
	if cfg.DisableSink {
		sink = disabledSink{}
	}
	return &Observer{
		cfg:    cfg,
		sink:   sink,
		policy: shadowharness.DefaultShadowHarnessPolicy(),
		now:    time.Now,
	}
}

func NewObserverWithSink(cfg Config, sink Sink, now func() time.Time) *Observer {
	cfg = cfg.normalized()
	if cfg.DisableSink {
		sink = disabledSink{}
	}
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
	return o.observe(ctx, input, nil, nil)
}

func (o *Observer) ObserveRouteEnvelopeBestEffort(ctx context.Context, input RouteEnvelopeInput) {
	defer func() {
		_ = recover()
	}()
	_ = o.ObserveRouteEnvelope(ctx, input)
}

func (o *Observer) ObserveRouteEnvelope(ctx context.Context, input RouteEnvelopeInput) error {
	if o == nil || !o.cfg.Enabled {
		return nil
	}
	now := o.now()
	observationID := fmt.Sprintf("shadow-route-envelope-%d", o.counter.Add(1))
	envelope, metadata, err := normalizeRouteEnvelopeInput(input, now, observationID)
	if err != nil {
		return err
	}
	routePath := envelope.RoutePattern
	if routePath == "" {
		routePath = envelope.Path
	}
	return o.observeAt(ctx, ObservationInput{
		WorkspaceID:    input.WorkspaceID,
		RequestID:      input.RequestID,
		LivePath:       strings.TrimSpace(envelope.Method + " " + routePath),
		Method:         envelope.Method,
		Path:           routePath,
		RequestSummary: "read-only route envelope metadata",
		Metadata:       metadata,
	}, now, observationID, &envelope, nil)
}

func (o *Observer) ObserveChatMetadataBestEffort(ctx context.Context, input ChatMetadataInput) {
	defer func() {
		_ = recover()
	}()
	_ = o.ObserveChatMetadata(ctx, input)
}

func (o *Observer) ObserveChatMetadata(ctx context.Context, input ChatMetadataInput) error {
	if o == nil || !o.cfg.Enabled || !o.cfg.ChatMetadataEnabled {
		return nil
	}
	now := o.now()
	observationID := fmt.Sprintf("shadow-chat-metadata-%d", o.counter.Add(1))
	chatMetadata, metadata, err := normalizeChatMetadataInput(input, now, observationID)
	if err != nil {
		return err
	}
	return o.observeAt(ctx, ObservationInput{
		WorkspaceID:    input.WorkspaceID,
		RequestID:      input.RequestID,
		LivePath:       "POST /api/chat/threads/{id}/messages",
		Method:         "POST",
		Path:           "/api/chat/threads/{id}/messages",
		RequestSummary: "read-only chat metadata",
		Metadata:       metadata,
	}, now, observationID, nil, &chatMetadata)
}

func (o *Observer) observe(ctx context.Context, input ObservationInput, routeEnvelope *RouteEnvelopeObservation, chatMetadata *ChatMetadataObservation) error {
	if o == nil || !o.cfg.Enabled {
		return nil
	}
	return o.observeAt(ctx, input, o.now(), "", routeEnvelope, chatMetadata)
}

func (o *Observer) observeAt(ctx context.Context, input ObservationInput, now time.Time, observationID string, routeEnvelope *RouteEnvelopeObservation, chatMetadata *ChatMetadataObservation) error {
	if o == nil || !o.cfg.Enabled {
		return nil
	}
	if err := shadowharness.ValidateShadowHarnessPolicy(o.policy); err != nil {
		return fmt.Errorf("%w: %v", ErrPolicyRejected, err)
	}
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
	if strings.TrimSpace(observationID) == "" {
		observationID = fmt.Sprintf("shadow-observation-%d", o.counter.Add(1))
	}

	obs, err := shadowharness.NewShadowObservation(shadowharness.ShadowObservation{
		ObservationID:  observationID,
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
		Observation:   obs,
		Comparison:    report,
		RouteEnvelope: routeEnvelope,
		ChatMetadata:  chatMetadata,
		StoredAt:      now,
	})
}
