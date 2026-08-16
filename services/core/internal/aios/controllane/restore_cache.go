package controllane

import (
	"context"

	"forge/projectforge/services/core/internal/aios/domain"
)

// selectCompileContextRestoreCandidateCached retains the compatibility name
// while deliberately performing no caching. Restore selection affects durable
// context evidence, so the production path must derive it from the exact
// request and current governed inputs on every invocation. The legacy cache
// key omitted authority-bearing request fields and is retired rather than
// expanded into a second decision authority.
func selectCompileContextRestoreCandidateCached(ctx context.Context, engine RuleEngine, now int64, current compiledContextSnapshot, packets []domain.ContextPacket, snapshotKind string, hints compileContextResumeHints, outcomes []RestoreOutcomeEvent, _ bool) compileContextRestoreSelection {
	selection := selectCompileContextRestoreCandidateWithFeedback(ctx, engine, now, current, packets, snapshotKind, hints, outcomes)
	selection.CacheHit = false
	selection.CacheKey = ""
	return selection
}
