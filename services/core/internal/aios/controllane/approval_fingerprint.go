package controllane

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"forge/projectforge/services/core/internal/aios/domain"
)

const (
	ApprovalFingerprintVersion              = "control_lane_approval_fingerprint.v1"
	ApprovalFingerprintActionMutation       = "mutation"
	ApprovalFingerprintActionReadOnly       = "read_only"
	ApprovalFingerprintActionValidationOnly = "validation_only"
)

type ApprovalFingerprintInput struct {
	Request           domain.SyscallRequest
	Definition        ActionDefinition
	RiskClass         string
	ApprovalRequestID string
	DecisionStatus    domain.ApprovalStatus
	CreatedAtMillis   int64
	ExpiresAtMillis   int64
}

type ApprovalFingerprint struct {
	Version               string                    `json:"version"`
	SemanticAction        domain.SemanticActionType `json:"semanticAction"`
	Capability            string                    `json:"capability"`
	TargetObjectType      string                    `json:"targetObjectType"`
	Mutating              bool                      `json:"mutating"`
	ActionClass           string                    `json:"actionClass"`
	Actor                 ApprovalFingerprintActor  `json:"actor"`
	Source                domain.ActionSource       `json:"source"`
	Workspace             string                    `json:"workspace"`
	TraceID               string                    `json:"traceId,omitempty"`
	CorrelationID         string                    `json:"correlationId,omitempty"`
	PayloadShapeHash      string                    `json:"payloadShapeHash"`
	SafeTargetIdentifiers []string                  `json:"safeTargetIdentifiers,omitempty"`
	RiskClass             string                    `json:"riskClass,omitempty"`
	ApprovalRequestID     string                    `json:"approvalRequestId,omitempty"`
	DecisionStatus        domain.ApprovalStatus     `json:"decisionStatus,omitempty"`
	CreatedAtMillis       int64                     `json:"createdAtMillis,omitempty"`
	ExpiresAtMillis       int64                     `json:"expiresAtMillis,omitempty"`
}

type ApprovalFingerprintActor struct {
	ID   string `json:"id"`
	Kind string `json:"kind"`
}

func BuildApprovalFingerprint(in ApprovalFingerprintInput) ApprovalFingerprint {
	req := in.Request
	def := in.Definition
	shape := canonicalPayloadShape(req.Payload)
	return ApprovalFingerprint{
		Version:          ApprovalFingerprintVersion,
		SemanticAction:   req.Action,
		Capability:       def.Capability,
		TargetObjectType: def.TargetObjectType,
		Mutating:         def.Mutating,
		ActionClass:      approvalFingerprintActionClass(req.Action, def),
		Actor: ApprovalFingerprintActor{
			ID:   strings.TrimSpace(req.Actor.ID),
			Kind: strings.TrimSpace(req.Actor.Kind),
		},
		Source:                req.Source,
		Workspace:             strings.TrimSpace(req.Scope.WorkspaceID),
		TraceID:               strings.TrimSpace(req.TraceID),
		CorrelationID:         strings.TrimSpace(req.CorrelationID),
		PayloadShapeHash:      sha256Hex(ApprovalFingerprintVersion + "|payload_shape|" + shape),
		SafeTargetIdentifiers: extractSafeTargetIdentifiers(req.Payload),
		RiskClass:             strings.TrimSpace(in.RiskClass),
		ApprovalRequestID:     strings.TrimSpace(in.ApprovalRequestID),
		DecisionStatus:        in.DecisionStatus,
		CreatedAtMillis:       in.CreatedAtMillis,
		ExpiresAtMillis:       in.ExpiresAtMillis,
	}
}

func approvalFingerprintActionClass(action domain.SemanticActionType, def ActionDefinition) string {
	if def.Mutating {
		return ApprovalFingerprintActionMutation
	}
	actionName := string(action)
	if strings.HasPrefix(actionName, "VALIDATE_") || strings.HasPrefix(actionName, "COMPARE_") {
		return ApprovalFingerprintActionValidationOnly
	}
	return ApprovalFingerprintActionReadOnly
}

func canonicalPayloadShape(v any) string {
	return canonicalPayloadShapeValue(reflect.ValueOf(v))
}

func canonicalPayloadShapeValue(v reflect.Value) string {
	if !v.IsValid() {
		return "null"
	}
	for v.Kind() == reflect.Pointer || v.Kind() == reflect.Interface {
		if v.IsNil() {
			return "null"
		}
		v = v.Elem()
	}

	switch v.Kind() {
	case reflect.Bool:
		return "bool"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return "int"
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return "uint"
	case reflect.Float32, reflect.Float64:
		return "float"
	case reflect.String:
		return "string"
	case reflect.Map:
		if v.IsNil() {
			return "map{}"
		}
		keys := v.MapKeys()
		parts := make([]string, 0, len(keys))
		for _, key := range keys {
			keyShape := fmt.Sprint(key.Interface())
			parts = append(parts, keyShape+":"+canonicalPayloadShapeValue(v.MapIndex(key)))
		}
		sort.Strings(parts)
		return "map{" + strings.Join(parts, ",") + "}"
	case reflect.Slice, reflect.Array:
		parts := make([]string, 0, v.Len())
		for i := 0; i < v.Len(); i++ {
			parts = append(parts, canonicalPayloadShapeValue(v.Index(i)))
		}
		sort.Strings(parts)
		return "list[" + strconv.Itoa(v.Len()) + "]{" + strings.Join(parts, ",") + "}"
	default:
		return v.Kind().String()
	}
}

func extractSafeTargetIdentifiers(payload map[string]any) []string {
	out := make([]string, 0)
	collectSafeTargetIdentifiers("", payload, &out)
	sort.Strings(out)
	return out
}

func collectSafeTargetIdentifiers(prefix string, value any, out *[]string) {
	switch typed := value.(type) {
	case map[string]any:
		keys := make([]string, 0, len(typed))
		for key := range typed {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			next := key
			if prefix != "" {
				next = prefix + "." + key
			}
			collectSafeTargetIdentifiers(next, typed[key], out)
		}
	case []any:
		for _, item := range typed {
			collectSafeTargetIdentifiers(prefix, item, out)
		}
	case string:
		key := lastPathSegment(prefix)
		if isSafeTargetIdentifierKey(key) && isSafeTargetIdentifierValue(typed) {
			*out = append(*out, key+"="+typed)
		}
	}
}

func lastPathSegment(path string) string {
	if idx := strings.LastIndex(path, "."); idx >= 0 {
		return path[idx+1:]
	}
	return path
}

func isSafeTargetIdentifierKey(key string) bool {
	switch key {
	case "id", "noteId", "objectId", "sourceId", "targetId", "loopId", "oldObjectId", "newObjectId", "workspaceId", "refId", "requestId":
		return true
	default:
		return false
	}
}

func isSafeTargetIdentifierValue(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" || len(value) > 128 {
		return false
	}
	for _, r := range value {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' {
			continue
		}
		switch r {
		case '-', '_', '.', ':', '/', '@':
			continue
		default:
			return false
		}
	}
	return true
}

func sha256Hex(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}
