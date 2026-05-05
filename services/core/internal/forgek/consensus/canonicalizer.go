package consensus

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

func NewClaim(input ClaimInput) (Claim, error) {
	claim := Claim{
		ClaimID:      trim(input.ClaimID),
		RequestID:    trim(input.RequestID),
		ClaimType:    ClaimType(trim(string(input.ClaimType))),
		Subject:      trim(input.Subject),
		Predicate:    trim(input.Predicate),
		ValueJSON:    normalizeValue(input.ValueJSON),
		Scope:        trim(input.Scope),
		Temporal:     trim(input.Temporal),
		EvidenceRefs: NormalizeRefs(input.EvidenceRefs),
		Confidence:   clamp01(input.Confidence),
		AgentID:      trim(input.AgentID),
		AgentRunID:   trim(input.AgentRunID),
		RiskFlags:    NormalizeRefs(input.RiskFlags),
		Status:       StatusProposed,
		CreatedAt:    input.CreatedAt,
		Metadata:     CloneMap(input.Metadata),
	}
	if err := ValidateClaim(claim); err != nil {
		return Claim{}, err
	}
	claim.ClaimKey = ClaimKey(claim)
	return claim, nil
}

func ValidateClaim(claim Claim) error {
	if claim.ClaimID == "" || claim.RequestID == "" || claim.Subject == "" || claim.Predicate == "" ||
		claim.AgentID == "" || !ValidClaimType(claim.ClaimType) || claim.Confidence < 0 || claim.Confidence > 1 ||
		containsSecretMetadata(claim.Metadata) {
		return ErrInvalidClaim
	}
	if claim.Status != "" && !ValidClaimStatus(claim.Status) {
		return ErrInvalidClaim
	}
	if requiresEvidenceByType(claim.ClaimType) && len(claim.EvidenceRefs) == 0 {
		return ErrInvalidClaim
	}
	return nil
}

func ClaimKey(claim Claim) string {
	payload := map[string]any{
		"claim_type": claim.ClaimType,
		"subject":    strings.ToLower(trim(claim.Subject)),
		"predicate":  strings.ToLower(trim(claim.Predicate)),
		"value":      normalizeValue(claim.ValueJSON),
		"scope":      strings.ToLower(trim(claim.Scope)),
		"temporal":   strings.ToLower(trim(claim.Temporal)),
	}
	return StableHash(payload)
}

func ConflictKey(claim Claim) string {
	return StableHash(map[string]any{
		"claim_type": conflictClaimType(claim.ClaimType),
		"subject":    strings.ToLower(trim(claim.Subject)),
		"predicate":  strings.ToLower(trim(claim.Predicate)),
		"scope":      strings.ToLower(trim(claim.Scope)),
		"temporal":   strings.ToLower(trim(claim.Temporal)),
	})
}

func CanonicalValue(value any) string {
	encoded, _ := json.Marshal(normalizeValue(value))
	return string(encoded)
}

func StableJSON(value any) string {
	encoded, _ := json.Marshal(value)
	return string(encoded)
}

func StableHash(value any) string {
	encoded, _ := json.Marshal(value)
	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:])
}

func normalizeValue(value any) any {
	switch typed := value.(type) {
	case nil:
		return nil
	case string:
		trimmed := strings.TrimSpace(typed)
		if parsed, ok := parseNumericString(trimmed); ok {
			return parsed
		}
		if strings.EqualFold(trimmed, "true") {
			return true
		}
		if strings.EqualFold(trimmed, "false") {
			return false
		}
		return trimmed
	case json.Number:
		if parsed, ok := parseNumericString(string(typed)); ok {
			return parsed
		}
		return string(typed)
	case int:
		return float64(typed)
	case int64:
		return float64(typed)
	case float32:
		return float64(typed)
	case float64:
		return typed
	case bool:
		return typed
	case []any:
		out := make([]any, len(typed))
		for i, item := range typed {
			out[i] = normalizeValue(item)
		}
		return out
	case map[string]any:
		out := make(map[string]any, len(typed))
		for key, item := range typed {
			out[strings.TrimSpace(key)] = normalizeValue(item)
		}
		return out
	default:
		encoded, err := json.Marshal(typed)
		if err != nil {
			return fmt.Sprint(typed)
		}
		var decoded any
		if err := json.Unmarshal(encoded, &decoded); err != nil {
			return fmt.Sprint(typed)
		}
		return normalizeValue(decoded)
	}
}

func parseNumericString(value string) (float64, bool) {
	if value == "" {
		return 0, false
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, false
	}
	return parsed, true
}

func NormalizeRefs(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = trim(value)
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func CloneStrings(values []string) []string {
	return append([]string(nil), values...)
}

func CloneMap(values map[string]any) map[string]any {
	if values == nil {
		return nil
	}
	out := make(map[string]any, len(values))
	for key, value := range values {
		out[key] = cloneAny(value)
	}
	return out
}

func cloneAny(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		return CloneMap(typed)
	case []any:
		out := make([]any, len(typed))
		for i, item := range typed {
			out[i] = cloneAny(item)
		}
		return out
	case []string:
		return CloneStrings(typed)
	default:
		return typed
	}
}

func trim(value string) string {
	return strings.TrimSpace(value)
}

func clamp01(value float64) float64 {
	if value < 0 {
		return 0
	}
	if value > 1 {
		return 1
	}
	return value
}

func requiresEvidenceByType(claimType ClaimType) bool {
	switch claimType {
	case ClaimTypeFact, ClaimTypeRecommendation, ClaimTypeActionProposal, ClaimTypeMemoryUpdateProposal:
		return true
	default:
		return false
	}
}

func conflictClaimType(claimType ClaimType) ClaimType {
	switch claimType {
	case ClaimTypeActionProposal, ClaimTypeMemoryUpdateProposal:
		return claimType
	default:
		return ClaimTypeFact
	}
}

func containsSecretMetadata(metadata map[string]any) bool {
	for key := range metadata {
		lower := strings.ToLower(key)
		if strings.Contains(lower, "secret") || strings.Contains(lower, "token") || strings.Contains(lower, "password") {
			return true
		}
	}
	return false
}
