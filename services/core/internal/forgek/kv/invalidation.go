package kv

import "time"

func InvalidateManifest(manifest KVCacheManifest, reason string, now time.Time, journalRef string) (KVCacheManifest, error) {
	if manifest.Status == StatusEvicted {
		return KVCacheManifest{}, ErrInvalidStateTransition
	}
	manifest.Status = StatusInvalidated
	manifest.InvalidatedAt = &now
	manifest.InvalidationReason = NormalizeWhitespace(reason)
	manifest.JournalRefs = appendUnique(manifest.JournalRefs, journalRef)
	return manifest, nil
}

func EvictManifest(manifest KVCacheManifest, reason string, now time.Time, journalRef string) KVCacheManifest {
	manifest.Status = StatusEvicted
	manifest.InvalidatedAt = &now
	manifest.InvalidationReason = NormalizeWhitespace(reason)
	manifest.JournalRefs = appendUnique(manifest.JournalRefs, journalRef)
	return manifest
}
