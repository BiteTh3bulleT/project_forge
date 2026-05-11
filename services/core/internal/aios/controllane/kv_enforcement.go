package controllane

import (
	"strings"
	"sync"

	"forge/projectforge/services/core/internal/aios/domain"
	"forge/projectforge/services/core/internal/kvidentity"
)

const (
	KVIdentityDecisionAccepted      = "accepted"
	KVIdentityDecisionRejected      = "rejected"
	KVIdentityDecisionMalformed     = "malformed"
	KVIdentityDecisionUnsupported   = "unsupported_live_reuse"
	KVIdentityDecisionInternalError = "internal_error"

	KVIdentityPolicyVersion    = "phase-i2-v1"
	KVIdentityValidatorVersion = "kvidentity-v1"
)

type KVIdentityEnforcementDecision struct {
	Accepted         bool                        `json:"accepted"`
	Decision         string                      `json:"decision"`
	Reason           string                      `json:"reason"`
	Source           string                      `json:"source"`
	CandidateCacheID string                      `json:"candidateCacheId,omitempty"`
	IdentityHash     string                      `json:"identityHash,omitempty"`
	FailedGates      []string                    `json:"failedGates,omitempty"`
	Warnings         []string                    `json:"warnings,omitempty"`
	AccelerationOnly bool                        `json:"accelerationOnly"`
	MemoryMutation   bool                        `json:"memoryMutation"`
	RuntimeMutation  bool                        `json:"runtimeMutation"`
	LiveKVReuse      bool                        `json:"liveKVReuse"`
	ValidatorVersion string                      `json:"validatorVersion"`
	PolicyVersion    string                      `json:"policyVersion"`
	ValidationResult kvidentity.ValidationResult `json:"validationResult"`
}

type KVIdentityEnforcementCounterSnapshot struct {
	Accepted             int `json:"accepted"`
	Rejected             int `json:"rejected"`
	Malformed            int `json:"malformed"`
	UnsupportedLiveReuse int `json:"unsupportedLiveReuse"`
	InternalError        int `json:"internalError"`
}

type KVIdentityEnforcementMetrics interface {
	Record(decision KVIdentityEnforcementDecision)
}

type KVIdentityEnforcementCounters struct {
	mu       sync.Mutex
	snapshot KVIdentityEnforcementCounterSnapshot
}

func NewKVIdentityEnforcementCounters() *KVIdentityEnforcementCounters {
	return &KVIdentityEnforcementCounters{}
}

func (c *KVIdentityEnforcementCounters) Record(decision KVIdentityEnforcementDecision) {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	switch decision.Decision {
	case KVIdentityDecisionAccepted:
		c.snapshot.Accepted++
	case KVIdentityDecisionRejected:
		c.snapshot.Rejected++
	case KVIdentityDecisionMalformed:
		c.snapshot.Malformed++
	case KVIdentityDecisionUnsupported:
		c.snapshot.UnsupportedLiveReuse++
	case KVIdentityDecisionInternalError:
		c.snapshot.InternalError++
	}
}

func (c *KVIdentityEnforcementCounters) Snapshot() KVIdentityEnforcementCounterSnapshot {
	if c == nil {
		return KVIdentityEnforcementCounterSnapshot{}
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.snapshot
}

func EnforceKVIdentity(req domain.SyscallRequest) KVIdentityEnforcementDecision {
	base := KVIdentityEnforcementDecision{
		Source:           string(req.Source),
		AccelerationOnly: true,
		MemoryMutation:   false,
		RuntimeMutation:  false,
		LiveKVReuse:      false,
		ValidatorVersion: KVIdentityValidatorVersion,
		PolicyVersion:    KVIdentityPolicyVersion,
	}
	if unsupportedLiveReuseRequested(req.Payload) {
		base.Accepted = false
		base.Decision = KVIdentityDecisionUnsupported
		base.Reason = "live KV reuse is not implemented"
		return base
	}
	if issues := validateKVIdentity(req); len(issues) > 0 {
		base.Accepted = false
		base.Decision = KVIdentityDecisionMalformed
		base.Reason = issues[0].Message
		return base
	}
	manifest := kvManifestIdentityFromPayload(req.Payload["manifest"])
	request := kvRequestIdentityFromPayload(req.Payload["request"])
	result := kvidentity.ValidateIdentity(req.ID+":kv_identity_enforcement", manifest, request, liveKVManifestHitEligible(manifest.Status), millisToTime(req.RequestedAt))
	base.ValidationResult = result
	base.CandidateCacheID = result.CandidateCacheID
	base.IdentityHash = firstNonEmpty(result.RequestIdentity.FinalTokenIDsHash, result.RequestIdentity.TokenInputHash)
	base.FailedGates = append([]string{}, result.FailedGates...)
	base.Warnings = append([]string{}, result.Warnings...)
	if result.Passed {
		base.Accepted = true
		base.Decision = KVIdentityDecisionAccepted
		base.Reason = "KV identity validation accepted"
		return base
	}
	base.Accepted = false
	base.Decision = KVIdentityDecisionRejected
	base.Reason = "KV identity validation rejected: " + strings.Join(result.FailedGates, ",")
	return base
}

func (d KVIdentityEnforcementDecision) ToStateSummary() map[string]any {
	return map[string]any{
		"kvIdentityEnforcement": d.ToAuditFields(),
		"kvIdentityValidation":  d.ValidationResult,
		"passed":                d.Accepted,
		"candidateCacheId":      d.CandidateCacheID,
		"failedGates":           append([]string{}, d.FailedGates...),
		"warnings":              append([]string{}, d.Warnings...),
		"accelerationOnly":      d.AccelerationOnly,
		"memoryMutation":        d.MemoryMutation,
		"runtimeMutation":       d.RuntimeMutation,
		"liveKVReuse":           d.LiveKVReuse,
		"forgeKActivation":      forgeKActivationSummary(string(domain.ActionValidateKVIdentity)),
		"forgeKNoEffect":        forgeKNoEffectSummary(),
	}
}

func (d KVIdentityEnforcementDecision) ToAuditFields() map[string]any {
	return map[string]any{
		"accepted":         d.Accepted,
		"decision":         d.Decision,
		"reason":           d.Reason,
		"source":           d.Source,
		"candidateCacheId": d.CandidateCacheID,
		"identityHash":     d.IdentityHash,
		"failedGates":      append([]string{}, d.FailedGates...),
		"warnings":         append([]string{}, d.Warnings...),
		"failureCount":     len(d.FailedGates),
		"warningCount":     len(d.Warnings),
		"accelerationOnly": d.AccelerationOnly,
		"memoryMutation":   d.MemoryMutation,
		"runtimeMutation":  d.RuntimeMutation,
		"liveKVReuse":      d.LiveKVReuse,
		"validatorVersion": d.ValidatorVersion,
		"policyVersion":    d.PolicyVersion,
		"forgeKActivation": forgeKActivationSummary(string(domain.ActionValidateKVIdentity)),
		"forgeKNoEffect":   forgeKNoEffectSummary(),
	}
}

func (d KVIdentityEnforcementDecision) ToSyscallError() domain.SyscallError {
	code := domain.ErrInvalidPayload
	field := "payload.request"
	if d.Decision == KVIdentityDecisionUnsupported {
		field = "payload.liveKVReuse"
	}
	return domain.SyscallError{Code: code, Field: field, Message: d.Reason}
}

func unsupportedLiveReuseRequested(payload map[string]any) bool {
	if truthyClaim(payload, "liveKVReuse") ||
		truthyClaim(payload, "live_kv_reuse") ||
		truthyClaim(payload, "enable_live_reuse") ||
		truthyClaim(payload, "backend_cache_reuse") ||
		truthyClaim(payload, "reuse_live_kv") ||
		truthyClaim(payload, "allow_live_kv_reuse") {
		return true
	}
	if request, ok := payload["request"].(map[string]any); ok {
		return truthyClaim(request, "liveKVReuse") ||
			truthyClaim(request, "live_kv_reuse") ||
			truthyClaim(request, "enable_live_reuse") ||
			truthyClaim(request, "backend_cache_reuse") ||
			truthyClaim(request, "reuse_live_kv") ||
			truthyClaim(request, "allow_live_kv_reuse")
	}
	return false
}

func truthyClaim(values map[string]any, key string) bool {
	if values == nil {
		return false
	}
	raw, ok := values[key]
	if !ok {
		return false
	}
	switch v := raw.(type) {
	case bool:
		return v
	case string:
		return strings.EqualFold(strings.TrimSpace(v), "true") ||
			strings.EqualFold(strings.TrimSpace(v), "yes") ||
			strings.TrimSpace(v) == "1"
	default:
		return false
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
