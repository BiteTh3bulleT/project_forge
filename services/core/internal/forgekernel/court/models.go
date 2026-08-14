// Package court contains the production-owned deterministic Courthouse
// contracts. It deliberately does not depend on the FORGE-K simulator.
package court

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"forge/projectforge/services/core/internal/aios/domain"
)

const (
	DecisionAdmitted = "admitted"
	DecisionRejected = "rejected"

	MetadataDecisionKey = "forgeKCourthouseDecision"
	PolicyVersion       = "forge-k-court-v1"
)

type Exhibit struct {
	ID              string
	CaseID          string
	Scope           domain.ForgeScope
	SourceType      string
	SourceRefs      []string
	ContentSummary  string
	RawRef          string
	ContentHash     string
	Status          string
	CurrentRulingID string
	CreatedAt       int64
	UpdatedAt       int64
	Provenance      domain.Provenance
	SyscallID       string
	CorrelationID   string
	TraceID         string
	AuditID         string
	ProposedBy      string
	CommittedBy     string
}

type Ruling struct {
	ID            string
	CaseID        string
	ExhibitID     string
	AppealID      string
	PriorRulingID string
	Scope         domain.ForgeScope
	Decision      string
	ReasonCode    string
	Reason        string
	PolicyVersion string
	PolicyRefs    []string
	InputRefs     []string
	ContentHash   string
	CreatedAt     int64
	Provenance    domain.Provenance
	SyscallID     string
	CorrelationID string
	TraceID       string
	AuditID       string
	ProposedBy    string
	CommittedBy   string
}

type Appeal struct {
	ID             string
	CaseID         string
	ExhibitID      string
	PriorRulingID  string
	NewRulingID    string
	Scope          domain.ForgeScope
	Grounds        string
	NewSourceRefs  []string
	NewContentHash string
	CreatedAt      int64
	Provenance     domain.Provenance
	SyscallID      string
	CorrelationID  string
	TraceID        string
	AuditID        string
	ProposedBy     string
	CommittedBy    string
}

// Decision is created only by the production Kernel after the ordinary
// envelope/capability/approval/payload preflight has completed. The durable
// adapter may persist it but may not invent or alter it.
type Decision struct {
	Action        domain.SemanticActionType
	CaseID        string
	ExhibitID     string
	RulingID      string
	AppealID      string
	PriorRulingID string
	Decision      string
	ReasonCode    string
	Reason        string
	PolicyVersion string
	PolicyRefs    []string
	InputRefs     []string
	ContentHash   string
}

func IsAction(action domain.SemanticActionType) bool {
	return action == domain.ActionAdmitEvidence || action == domain.ActionAppealRuling
}

// Decide applies the deliberately small deterministic v1 admission policy.
// Evidence is admissible only when it has stable source refs, a content hash,
// and policy refs. Missing policy material is a persisted rejection, not an
// invitation for a model to make the decision.
func Decide(req domain.SyscallRequest) (Decision, []domain.SyscallError) {
	if !IsAction(req.Action) {
		return Decision{}, []domain.SyscallError{{Code: domain.ErrUnsupportedAction, Field: "action", Message: "not a Courthouse action"}}
	}
	if isModelActor(req.Actor.Kind) {
		return Decision{}, []domain.SyscallError{{Code: domain.ErrUnauthorized, Field: "actor.kind", Message: "models and neural workers cannot admit evidence or rule on appeals"}}
	}
	caseID := strings.TrimSpace(readString(req.Payload, "caseId"))
	exhibitID := strings.TrimSpace(readString(req.Payload, "exhibitId"))
	if exhibitID == "" {
		exhibitID = req.ID + ":exhibit"
	}
	rulingID := strings.TrimSpace(readString(req.Payload, "rulingId"))
	if rulingID == "" {
		rulingID = req.ID + ":ruling"
	}
	d := Decision{
		Action:        req.Action,
		CaseID:        caseID,
		ExhibitID:     exhibitID,
		RulingID:      rulingID,
		PolicyVersion: PolicyVersion,
		PolicyRefs:    cleanStrings(req.Payload["policyRefs"]),
		InputRefs:     cleanStrings(req.Payload["sourceRefs"]),
		ContentHash:   strings.TrimSpace(readString(req.Payload, "contentHash")),
	}
	if req.Action == domain.ActionAppealRuling {
		d.AppealID = strings.TrimSpace(readString(req.Payload, "appealId"))
		if d.AppealID == "" {
			d.AppealID = req.ID + ":appeal"
		}
		d.PriorRulingID = strings.TrimSpace(readString(req.Payload, "priorRulingId"))
		d.InputRefs = cleanStrings(req.Payload["newSourceRefs"])
		d.ContentHash = strings.TrimSpace(readString(req.Payload, "newContentHash"))
	}

	missing := make([]string, 0, 3)
	if len(d.InputRefs) == 0 {
		missing = append(missing, "stable source refs")
	}
	if !validContentHash(d.ContentHash) {
		missing = append(missing, "valid sha256 content hash")
	}
	if len(d.PolicyRefs) == 0 {
		missing = append(missing, "policy refs")
	}
	if len(missing) == 0 {
		d.Decision = DecisionAdmitted
		d.ReasonCode = "policy_material_complete"
		d.Reason = "evidence satisfies deterministic admission policy"
	} else {
		d.Decision = DecisionRejected
		d.ReasonCode = "policy_material_incomplete"
		d.Reason = fmt.Sprintf("evidence rejected: missing %s", strings.Join(missing, ", "))
	}
	return d, nil
}

func DecisionFromMetadata(metadata map[string]any) (Decision, bool) {
	if metadata == nil {
		return Decision{}, false
	}
	d, ok := metadata[MetadataDecisionKey].(Decision)
	return d, ok && IsAction(d.Action) && d.PolicyVersion == PolicyVersion
}

func cleanStrings(raw any) []string {
	var values []string
	switch typed := raw.(type) {
	case []string:
		values = typed
	case []any:
		for _, value := range typed {
			if text, ok := value.(string); ok {
				values = append(values, text)
			}
		}
	}
	out := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func validContentHash(value string) bool {
	const prefix = "sha256:"
	if !strings.HasPrefix(value, prefix) || len(value) != len(prefix)+sha256.Size*2 {
		return false
	}
	raw := strings.TrimPrefix(value, prefix)
	decoded, err := hex.DecodeString(raw)
	return err == nil && len(decoded) == sha256.Size && raw == strings.ToLower(raw)
}

func readString(payload map[string]any, key string) string {
	if payload == nil {
		return ""
	}
	value, _ := payload[key].(string)
	return value
}

func isModelActor(kind string) bool {
	kind = strings.ToLower(strings.TrimSpace(kind))
	return strings.Contains(kind, "model") || strings.Contains(kind, "llm") || strings.Contains(kind, "neural")
}
