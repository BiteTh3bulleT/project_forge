package ephemeral

import (
	"context"
	"sync"
	"time"
)

type MemoryStore struct {
	policy   KeyPolicy
	now      func() time.Time
	mu       sync.Mutex
	cache    map[string]cacheRecord
	queues   map[string][][]byte
	locks    map[string]lockRecord
	progress map[string]progressRecord
	pubsub   map[string][][]byte
}

type cacheRecord struct {
	value     []byte
	expiresAt time.Time
}

type lockRecord struct {
	owner     string
	expiresAt time.Time
}

type progressRecord struct {
	entries   []ProgressEntry
	expiresAt time.Time
}

func NewMemoryStore(policy KeyPolicy) *MemoryStore {
	return &MemoryStore{
		policy:   policy,
		now:      time.Now,
		cache:    map[string]cacheRecord{},
		queues:   map[string][][]byte{},
		locks:    map[string]lockRecord{},
		progress: map[string]progressRecord{},
		pubsub:   map[string][][]byte{},
	}
}

func (s *MemoryStore) SetCache(_ context.Context, key string, value []byte, ttl time.Duration) error {
	if err := s.policy.RequireTTL(KeyKindCache, ttl); err != nil {
		return err
	}
	if err := validateFullyQualifiedKey(s.policy, key); err != nil {
		return err
	}
	if err := validateEphemeralValueBytes(value); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cache[key] = cacheRecord{value: cloneBytes(value), expiresAt: s.now().Add(ttl)}
	return nil
}

func (s *MemoryStore) GetCache(_ context.Context, key string) ([]byte, bool, error) {
	if err := validateFullyQualifiedKey(s.policy, key); err != nil {
		return nil, false, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	record, ok := s.cache[key]
	if !ok {
		return nil, false, nil
	}
	if !record.expiresAt.After(s.now()) {
		delete(s.cache, key)
		return nil, false, nil
	}
	return cloneBytes(record.value), true, nil
}

func (s *MemoryStore) PushQueue(_ context.Context, key string, value []byte) error {
	if err := validateFullyQualifiedKey(s.policy, key); err != nil {
		return err
	}
	if err := validateEphemeralValueBytes(value); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.queues[key] = append(s.queues[key], cloneBytes(value))
	return nil
}

func (s *MemoryStore) PopQueue(_ context.Context, key string) ([]byte, bool, error) {
	if err := validateFullyQualifiedKey(s.policy, key); err != nil {
		return nil, false, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	queue := s.queues[key]
	if len(queue) == 0 {
		return nil, false, nil
	}
	value := queue[0]
	s.queues[key] = queue[1:]
	return cloneBytes(value), true, nil
}

func (s *MemoryStore) AcquireLock(_ context.Context, key string, owner string, ttl time.Duration) (bool, error) {
	if err := s.policy.RequireTTL(KeyKindLock, ttl); err != nil {
		return false, err
	}
	if err := validateFullyQualifiedKey(s.policy, key); err != nil {
		return false, err
	}
	if owner == "" {
		return false, ErrInvalidConfig
	}
	if err := validateEphemeralValueString(owner); err != nil {
		return false, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.now()
	if record, ok := s.locks[key]; ok && record.expiresAt.After(now) {
		return record.owner == owner, nil
	}
	s.locks[key] = lockRecord{owner: owner, expiresAt: now.Add(ttl)}
	return true, nil
}

func (s *MemoryStore) ReleaseLock(_ context.Context, key string, owner string) error {
	if err := validateFullyQualifiedKey(s.policy, key); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	record, ok := s.locks[key]
	if !ok {
		return ErrLockNotHeld
	}
	if record.owner != owner {
		return ErrLockHeld
	}
	delete(s.locks, key)
	return nil
}

func (s *MemoryStore) AppendProgress(_ context.Context, key string, entry ProgressEntry, ttl time.Duration) error {
	if err := s.policy.RequireTTL(KeyKindProgress, ttl); err != nil {
		return err
	}
	if err := validateFullyQualifiedKey(s.policy, key); err != nil {
		return err
	}
	if err := validateProgressEntryValue(entry); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if entry.ID == "" {
		entry.ID = StableOpaqueSegment(key, entry.Message, s.now().UTC().Format(time.RFC3339Nano))
	}
	if entry.CreatedAt.IsZero() {
		entry.CreatedAt = s.now().UTC()
	}
	record := s.progress[key]
	record.entries = append(record.entries, entry)
	record.expiresAt = s.now().Add(ttl)
	s.progress[key] = record
	return nil
}

func (s *MemoryStore) ReadProgress(_ context.Context, key string, limit int) ([]ProgressEntry, error) {
	if err := validateFullyQualifiedKey(s.policy, key); err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	record, ok := s.progress[key]
	if !ok {
		return nil, nil
	}
	if !record.expiresAt.After(s.now()) {
		delete(s.progress, key)
		return nil, nil
	}
	entries := record.entries
	limit = normalizeProgressReadLimit(limit)
	if limit > 0 && len(entries) > limit {
		entries = entries[len(entries)-limit:]
	}
	out := make([]ProgressEntry, len(entries))
	copy(out, entries)
	return out, nil
}

func (s *MemoryStore) Publish(_ context.Context, channel string, value []byte) error {
	if err := validateFullyQualifiedKey(s.policy, channel); err != nil {
		return err
	}
	if err := validateEphemeralValueBytes(value); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pubsub[channel] = append(s.pubsub[channel], cloneBytes(value))
	return nil
}

func (s *MemoryStore) Health(context.Context) HealthStatus {
	return HealthStatus{Enabled: true, OK: true, Message: "memory ephemeral store ready"}
}

func validateFullyQualifiedKey(policy KeyPolicy, key string) error {
	if key == "" {
		return ErrUnsafeKey
	}
	parts := splitKey(key)
	if len(parts) < 2 || parts[0] != policy.Prefix {
		return ErrUnsafeKey
	}
	maxLength := policy.MaxSegmentLength
	if maxLength <= 0 {
		maxLength = 96
	}
	for _, part := range parts {
		if err := validateSegment(part, maxLength); err != nil {
			return err
		}
	}
	return nil
}

func splitKey(key string) []string {
	parts := make([]string, 0)
	start := 0
	for i := 0; i <= len(key); i++ {
		if i == len(key) || key[i] == ':' {
			parts = append(parts, key[start:i])
			start = i + 1
		}
	}
	return parts
}

func cloneBytes(value []byte) []byte {
	out := make([]byte, len(value))
	copy(out, value)
	return out
}
