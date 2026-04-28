package controllane

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"

	"forge/projectforge/services/core/internal/aios/domain"
)

const (
	restoreScoringCacheTTLMillis = int64(2 * 60 * 1000)
	restoreScoringCacheMaxItems  = 64
)

type restoreScoringCacheEntry struct {
	key       string
	createdAt int64
	usedAt    int64
	selection compileContextRestoreSelection
}

type restoreScoringCacheStore struct {
	mu      sync.Mutex
	enabled bool
	items   map[string]restoreScoringCacheEntry
}

var restoreScoringCache = &restoreScoringCacheStore{enabled: true, items: map[string]restoreScoringCacheEntry{}}

func resetRestoreScoringCacheForTest() {
	restoreScoringCache.mu.Lock()
	defer restoreScoringCache.mu.Unlock()
	restoreScoringCache.items = map[string]restoreScoringCacheEntry{}
	restoreScoringCache.enabled = true
}

func setRestoreScoringCacheEnabledForTest(enabled bool) {
	restoreScoringCache.mu.Lock()
	defer restoreScoringCache.mu.Unlock()
	restoreScoringCache.enabled = enabled
}

func selectCompileContextRestoreCandidateCached(ctx context.Context, engine RuleEngine, now int64, current compiledContextSnapshot, packets []domain.ContextPacket, snapshotKind string, hints compileContextResumeHints, outcomes []RestoreOutcomeEvent, disabled bool) compileContextRestoreSelection {
	key := restoreScoringCacheKey(current, packets, snapshotKind, hints, outcomes)
	if !disabled {
		if selection, ok := restoreScoringCache.get(key, now); ok {
			selection.CacheHit = true
			selection.CacheKey = key
			return selection
		}
	}
	selection := selectCompileContextRestoreCandidateWithFeedback(ctx, engine, now, current, packets, snapshotKind, hints, outcomes)
	selection.CacheHit = false
	selection.CacheKey = key
	if !disabled {
		restoreScoringCache.set(key, now, selection)
	}
	return selection
}

func (c *restoreScoringCacheStore) get(key string, now int64) (compileContextRestoreSelection, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.enabled || strings.TrimSpace(key) == "" {
		return compileContextRestoreSelection{}, false
	}
	entry, ok := c.items[key]
	if !ok {
		return compileContextRestoreSelection{}, false
	}
	if now-entry.createdAt > restoreScoringCacheTTLMillis {
		delete(c.items, key)
		return compileContextRestoreSelection{}, false
	}
	entry.usedAt = now
	c.items[key] = entry
	return entry.selection, true
}

func (c *restoreScoringCacheStore) set(key string, now int64, selection compileContextRestoreSelection) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.enabled || strings.TrimSpace(key) == "" {
		return
	}
	if c.items == nil {
		c.items = map[string]restoreScoringCacheEntry{}
	}
	if len(c.items) >= restoreScoringCacheMaxItems {
		c.evictOldestLocked()
	}
	c.items[key] = restoreScoringCacheEntry{key: key, createdAt: now, usedAt: now, selection: selection}
}

func (c *restoreScoringCacheStore) evictOldestLocked() {
	oldestKey := ""
	oldestUsed := int64(0)
	for key, entry := range c.items {
		if oldestKey == "" || entry.usedAt < oldestUsed {
			oldestKey = key
			oldestUsed = entry.usedAt
		}
	}
	if oldestKey != "" {
		delete(c.items, oldestKey)
	}
}

func restoreScoringCacheKey(current compiledContextSnapshot, packets []domain.ContextPacket, snapshotKind string, hints compileContextResumeHints, outcomes []RestoreOutcomeEvent) string {
	parts := []string{
		"ws=" + current.Header.Scope.WorkspaceID,
		"lane=" + current.Header.Scope.LaneID,
		"query=" + normalizeRestoreQuery(current.Header.Query),
		"kind=" + strings.TrimSpace(snapshotKind),
		"threshold=" + fmt.Sprintf("%.4f", hints.MinimumScore),
		"freshOnly=" + fmt.Sprintf("%t", hints.FreshCompileOnly),
		"preferred=" + strings.TrimSpace(hints.PreferredSnapshotID),
		"candidates=" + restoreCandidateFingerprint(packets),
		"outcomes=" + restoreOutcomeFingerprint(outcomes),
	}
	return strings.Join(parts, "|")
}

func restoreCandidateFingerprint(packets []domain.ContextPacket) string {
	parts := make([]string, 0, len(packets))
	for _, pkt := range packets {
		kind := ""
		if pkt.CompileOptions != nil {
			kind = pkt.CompileOptions.SnapshotKind
		}
		parts = append(parts, strings.Join([]string{
			strings.TrimSpace(pkt.ID),
			strings.TrimSpace(pkt.Scope.WorkspaceID),
			strings.TrimSpace(pkt.Scope.LaneID),
			strings.TrimSpace(pkt.Query),
			strings.TrimSpace(kind),
			fmt.Sprintf("%d", pkt.CreatedAt),
		}, ":"))
	}
	sort.Strings(parts)
	return strings.Join(parts, ",")
}

func restoreOutcomeFingerprint(outcomes []RestoreOutcomeEvent) string {
	parts := make([]string, 0, len(outcomes))
	for _, outcome := range outcomes {
		parts = append(parts, strings.Join([]string{
			strings.TrimSpace(outcome.ID),
			strings.TrimSpace(outcome.WorkspaceID),
			strings.TrimSpace(outcome.LaneID),
			strings.TrimSpace(outcome.SnapshotID),
			string(outcome.Outcome),
			fmt.Sprintf("%d", outcome.UpdatedAt),
		}, ":"))
	}
	sort.Strings(parts)
	return strings.Join(parts, ",")
}
