package ephemeral

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"regexp"
	"sort"
	"strings"
	"time"
)

type Role string

const (
	RoleCache                 Role = "cache"
	RoleQueue                 Role = "queue"
	RoleLock                  Role = "lock"
	RolePubSub                Role = "pubsub"
	RoleProgressStream        Role = "progress_stream"
	RoleRateLimitWindow       Role = "rate_limit_window"
	RoleEphemeralCoordination Role = "ephemeral_coordination"

	RoleCanonicalTruth      Role = "canonical_truth"
	RoleDurableMemory       Role = "durable_memory"
	RoleEvidenceAdmission   Role = "evidence_admission"
	RoleProvenanceAuthority Role = "provenance_authority"
	RoleSoleJobRecord       Role = "sole_job_record"
	RoleCanonicalAudit      Role = "canonical_audit"
	RoleCanonicalSettings   Role = "canonical_settings"
	RoleVectorTruth         Role = "vector_truth"
)

var allowedRoles = map[Role]bool{
	RoleCache:                 true,
	RoleQueue:                 true,
	RoleLock:                  true,
	RolePubSub:                true,
	RoleProgressStream:        true,
	RoleRateLimitWindow:       true,
	RoleEphemeralCoordination: true,
}

var forbiddenRoles = map[Role]bool{
	RoleCanonicalTruth:      true,
	RoleDurableMemory:       true,
	RoleEvidenceAdmission:   true,
	RoleProvenanceAuthority: true,
	RoleSoleJobRecord:       true,
	RoleCanonicalAudit:      true,
	RoleCanonicalSettings:   true,
	RoleVectorTruth:         true,
}

func AllowedRoles() []Role {
	return sortedRoles(allowedRoles)
}

func ForbiddenRoles() []Role {
	return sortedRoles(forbiddenRoles)
}

func ValidateRole(role Role) error {
	if forbiddenRoles[role] {
		return ErrForbiddenRole
	}
	if !allowedRoles[role] {
		return ErrInvalidRole
	}
	return nil
}

func CanBeCanonical(role Role) bool {
	return false
}

func RedisLossRecoverable(role Role) bool {
	return allowedRoles[role]
}

func sortedRoles(values map[Role]bool) []Role {
	roles := make([]Role, 0, len(values))
	for role := range values {
		roles = append(roles, role)
	}
	sort.Slice(roles, func(i, j int) bool { return roles[i] < roles[j] })
	return roles
}

type KeyKind string

const (
	KeyKindCache    KeyKind = "cache"
	KeyKindQueue    KeyKind = "queue"
	KeyKindLock     KeyKind = "lock"
	KeyKindProgress KeyKind = "progress"
	KeyKindPubSub   KeyKind = "pubsub"
)

type KeyPolicy struct {
	Prefix           string
	MaxSegmentLength int
}

func DefaultKeyPolicy(prefix string) KeyPolicy {
	prefix = strings.TrimSpace(prefix)
	if prefix == "" {
		prefix = "forge"
	}
	return KeyPolicy{Prefix: prefix, MaxSegmentLength: 96}
}

var safeSegmentPattern = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)

var forbiddenKeyTerms = []string{
	"api_key",
	"auth",
	"bearer",
	"chunk",
	"content",
	"cookie",
	"password",
	"prompt",
	"query",
	"raw",
	"secret",
	"token",
}

func (p KeyPolicy) Key(kind KeyKind, segments ...string) (string, error) {
	if err := p.validatePrefix(); err != nil {
		return "", err
	}
	if err := validateSegment(string(kind), p.MaxSegmentLength); err != nil {
		return "", err
	}
	parts := []string{p.Prefix, string(kind)}
	for _, segment := range segments {
		normalized, err := NormalizeKeySegment(segment, p.MaxSegmentLength)
		if err != nil {
			return "", err
		}
		parts = append(parts, normalized)
	}
	return strings.Join(parts, ":"), nil
}

func (p KeyPolicy) RequireTTL(kind KeyKind, ttl time.Duration) error {
	switch kind {
	case KeyKindCache, KeyKindProgress, KeyKindLock:
		if ttl <= 0 {
			return ErrTTLRequired
		}
	}
	return nil
}

func (p KeyPolicy) validatePrefix() error {
	maxLength := p.MaxSegmentLength
	if maxLength <= 0 {
		maxLength = 96
	}
	return validateSegment(p.Prefix, maxLength)
}

func NormalizeKeySegment(segment string, maxLength int) (string, error) {
	segment = strings.TrimSpace(segment)
	if err := validateSegment(segment, maxLength); err != nil {
		return "", err
	}
	return segment, nil
}

func validateSegment(segment string, maxLength int) error {
	if maxLength <= 0 {
		maxLength = 96
	}
	if segment == "" || len(segment) > maxLength || !safeSegmentPattern.MatchString(segment) {
		return ErrUnsafeKey
	}
	lower := strings.ToLower(segment)
	for _, forbidden := range forbiddenKeyTerms {
		if strings.Contains(lower, forbidden) {
			return ErrUnsafeKey
		}
	}
	return nil
}

func StableOpaqueSegment(parts ...string) string {
	hash := sha256.New()
	for _, part := range parts {
		hash.Write([]byte(strings.TrimSpace(part)))
		hash.Write([]byte{0})
	}
	return hex.EncodeToString(hash.Sum(nil))[:24]
}

type HealthStatus struct {
	Enabled bool
	OK      bool
	Message string
}

type ProgressEntry struct {
	ID        string
	Message   string
	CreatedAt time.Time
}

type Cache interface {
	SetCache(ctx context.Context, key string, value []byte, ttl time.Duration) error
	GetCache(ctx context.Context, key string) ([]byte, bool, error)
}

type Queue interface {
	PushQueue(ctx context.Context, key string, value []byte) error
	PopQueue(ctx context.Context, key string) ([]byte, bool, error)
}

type Lock interface {
	AcquireLock(ctx context.Context, key string, owner string, ttl time.Duration) (bool, error)
	ReleaseLock(ctx context.Context, key string, owner string) error
}

type ProgressStream interface {
	AppendProgress(ctx context.Context, key string, entry ProgressEntry, ttl time.Duration) error
	ReadProgress(ctx context.Context, key string, limit int) ([]ProgressEntry, error)
}

type PubSub interface {
	Publish(ctx context.Context, channel string, value []byte) error
}

type Health interface {
	Health(ctx context.Context) HealthStatus
}

type EphemeralStore interface {
	Cache
	Queue
	Lock
	ProgressStream
	PubSub
	Health
}
