package autonomy

import (
	"context"

	"forge/projectforge/services/core/internal/aios/domain"
	"forge/projectforge/services/core/internal/aios/truth"
)

type IngestAutonomyAdapter struct {
	Runtime   *RuleAgentRuntime
	MaxDepth  int
	NowMillis func() int64
}

func NewIngestAutonomyAdapter(runtime *RuleAgentRuntime, maxDepth int, nowMillis func() int64) *IngestAutonomyAdapter {
	if maxDepth <= 0 {
		maxDepth = 1
	}
	if nowMillis == nil {
		nowMillis = domain.NowMillis
	}
	return &IngestAutonomyAdapter{Runtime: runtime, MaxDepth: maxDepth, NowMillis: nowMillis}
}

func (a *IngestAutonomyAdapter) RunFromIngest(
	ctx context.Context,
	req domain.IngestRequest,
	result domain.IngestResult,
	truthEngine truth.TruthEngine,
	depth int,
) ([]domain.AutonomyRunSummary, error) {
	if a == nil || a.Runtime == nil {
		return nil, nil
	}
	if depth > a.MaxDepth {
		return []domain.AutonomyRunSummary{{
			IntentID:      "",
			Decision:      domain.DecisionAllowProposeOnly,
			Warnings:      []string{"autonomy depth cap reached"},
			Errors:        nil,
			CorrelationID: req.CorrelationID,
			TraceID:       req.TraceID,
		}}, nil
	}
	lifecycle, ok := truthEngine.(truth.OpenLoopLifecycleService)
	if !ok {
		return nil, nil
	}
	input := BuildRuleAgentInput(req.Scope, req.CorrelationID, req.TraceID, lifecycle, depth, "ingest", a.NowMillis())
	return a.Runtime.RunOnce(ctx, input)
}
