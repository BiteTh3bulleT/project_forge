package semanticdiff

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"

	"forge/projectforge/services/core/internal/aios/domain"
)

const (
	MetadataDecisionKey = "forgeKSemanticDiffDecision"
	DerivedObjectClass  = "NONCANONICAL_DERIVED_EVIDENCE"
	KernelCommitter     = "forge_k.kernel"
)

type Evidence struct {
	RowID                       int64             `json:"rowId"`
	EvidenceID                  string            `json:"evidenceId"`
	Scope                       domain.ForgeScope `json:"scope"`
	Content                     string            `json:"content"`
	MaterialHash                string            `json:"materialHash"`
	EvidenceHash                string            `json:"evidenceHash"`
	CourtCaseID                 string            `json:"courtCaseId"`
	CourtExhibitID              string            `json:"courtExhibitId"`
	CourtRulingID               string            `json:"courtRulingId"`
	AdmissionSyscallID          string            `json:"admissionSyscallId"`
	SourceProvenanceID          string            `json:"sourceProvenanceId"`
	MaterializationProvenanceID string            `json:"materializationProvenanceId"`
	CreatedAt                   int64             `json:"createdAt"`
	CommittedBy                 string            `json:"committedBy"`
	Current                     bool              `json:"current"`
	Admitted                    bool              `json:"admitted"`
}

type AuthorityInput struct {
	Left  Evidence `json:"left"`
	Right Evidence `json:"right"`
}

type Decision struct {
	OperatorVersion    string   `json:"operatorVersion"`
	Left               Evidence `json:"left"`
	Right              Evidence `json:"right"`
	Tokens             []string `json:"tokens"`
	Content            string   `json:"content"`
	ContentHash        string   `json:"contentHash"`
	SourceManifestHash string   `json:"sourceManifestHash"`
	ObjectClass        string   `json:"objectClass"`
}

func IsAction(action domain.SemanticActionType) bool {
	return action == domain.ActionComputeSemanticDiff
}

func MaterialHash(content string) string {
	digest := sha256.Sum256([]byte(content))
	return "sha256:" + hex.EncodeToString(digest[:])
}

// Decide is the production FORGE-K Semantic Algebra authority for diff.v1.
// Storage may resolve immutable evidence, but only the Kernel calls Decide and
// injects the resulting typed decision into the sealed request.
func Decide(req domain.SyscallRequest, input AuthorityInput) (Decision, []domain.SyscallError) {
	issues := validate(req, input)
	if len(issues) > 0 {
		return Decision{}, issues
	}
	result, err := Compute(Input{Left: input.Left.Content, Right: input.Right.Content})
	if err != nil {
		return Decision{}, []domain.SyscallError{{Code: domain.ErrInvalidPayload, Field: "semanticDiff.input", Message: err.Error()}}
	}
	left := normalizeEvidence(input.Left)
	right := normalizeEvidence(input.Right)
	manifestHash, err := Fingerprint(struct {
		Version string   `json:"version"`
		Left    Evidence `json:"left"`
		Right   Evidence `json:"right"`
	}{Version: OperatorVersion, Left: left, Right: right})
	if err != nil {
		return Decision{}, []domain.SyscallError{{Code: domain.ErrInternal, Field: "semanticDiff.sourceManifest", Message: err.Error()}}
	}
	return Decision{
		OperatorVersion: result.OperatorVersion,
		Left:            left, Right: right,
		Tokens: append([]string(nil), result.Tokens...), Content: result.Content,
		ContentHash: result.ContentHash, SourceManifestHash: manifestHash,
		ObjectClass: DerivedObjectClass,
	}, nil
}

func VerifyDecision(req domain.SyscallRequest, decision Decision) error {
	recomputed, issues := Decide(req, AuthorityInput{Left: decision.Left, Right: decision.Right})
	if len(issues) > 0 {
		return fmt.Errorf("%s: %s", issues[0].Field, issues[0].Message)
	}
	if !reflect.DeepEqual(recomputed, decision) {
		return fmt.Errorf("semantic diff decision does not match deterministic recomputation")
	}
	return nil
}

func DecisionFromMetadata(metadata map[string]any) (Decision, bool) {
	if metadata == nil {
		return Decision{}, false
	}
	value, ok := metadata[MetadataDecisionKey]
	if !ok {
		return Decision{}, false
	}
	if typed, ok := value.(Decision); ok {
		return typed, typed.OperatorVersion == OperatorVersion
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return Decision{}, false
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var decision Decision
	if err := decoder.Decode(&decision); err != nil || decision.OperatorVersion != OperatorVersion {
		return Decision{}, false
	}
	return decision, true
}

func validate(req domain.SyscallRequest, input AuthorityInput) []domain.SyscallError {
	var issues []domain.SyscallError
	if !IsAction(req.Action) {
		return []domain.SyscallError{{Code: domain.ErrUnsupportedAction, Field: "action", Message: "not a semantic diff action"}}
	}
	if strings.TrimSpace(readPayloadString(req.Payload, "operatorVersion")) != OperatorVersion {
		issues = append(issues, domain.SyscallError{Code: domain.ErrInvalidPayload, Field: "payload.operatorVersion", Message: "operatorVersion must be semantic.diff.v1"})
	}
	for _, item := range []struct {
		name     string
		expected string
		source   Evidence
	}{
		{"left", readPayloadString(req.Payload, "leftEvidenceId"), input.Left},
		{"right", readPayloadString(req.Payload, "rightEvidenceId"), input.Right},
	} {
		field := "semanticDiff." + item.name
		if strings.TrimSpace(item.expected) == "" || strings.TrimSpace(item.expected) != strings.TrimSpace(item.source.EvidenceID) {
			issues = append(issues, domain.SyscallError{Code: domain.ErrInvalidPayload, Field: field + ".evidenceId", Message: "resolved evidence must match the requested identity"})
		}
		if item.source.RowID <= 0 || strings.TrimSpace(item.source.EvidenceID) == "" {
			issues = append(issues, domain.SyscallError{Code: domain.ErrNotFound, Field: field, Message: "governed evidence is unavailable"})
		}
		if !sameScope(req.Scope, item.source.Scope) {
			issues = append(issues, domain.SyscallError{Code: domain.ErrInvalidScope, Field: field + ".scope", Message: "evidence scope must exactly match request scope"})
		}
		if !item.source.Current || !item.source.Admitted || item.source.CommittedBy != KernelCommitter {
			issues = append(issues, domain.SyscallError{Code: domain.ErrUnauthorized, Field: field + ".authority", Message: "evidence must be a current admitted FORGE-K materialization"})
		}
		if item.source.CreatedAt <= 0 || req.RequestedAt < item.source.CreatedAt {
			issues = append(issues, domain.SyscallError{Code: domain.ErrInvalidPayload, Field: field + ".createdAt", Message: "operation cannot predate source evidence"})
		}
		if item.source.MaterialHash != MaterialHash(item.source.Content) || !validHash(item.source.EvidenceHash) {
			issues = append(issues, domain.SyscallError{Code: domain.ErrInvalidPayload, Field: field + ".contentHash", Message: "source content commitment is invalid"})
		}
		if strings.TrimSpace(item.source.CourtCaseID) == "" || strings.TrimSpace(item.source.CourtExhibitID) == "" ||
			strings.TrimSpace(item.source.CourtRulingID) == "" || strings.TrimSpace(item.source.AdmissionSyscallID) == "" ||
			strings.TrimSpace(item.source.SourceProvenanceID) == "" || strings.TrimSpace(item.source.MaterializationProvenanceID) == "" {
			issues = append(issues, domain.SyscallError{Code: domain.ErrInvalidProvenance, Field: field + ".provenance", Message: "source admission and provenance commitments are required"})
		}
	}
	if strings.TrimSpace(input.Left.EvidenceID) != "" && input.Left.EvidenceID == input.Right.EvidenceID {
		issues = append(issues, domain.SyscallError{Code: domain.ErrInvalidPayload, Field: "payload.rightEvidenceId", Message: "semantic diff requires two distinct evidence objects"})
	}
	return issues
}

func normalizeEvidence(in Evidence) Evidence {
	in.EvidenceID = strings.TrimSpace(in.EvidenceID)
	in.Scope.WorkspaceID = strings.TrimSpace(in.Scope.WorkspaceID)
	in.Scope.LaneID = strings.TrimSpace(in.Scope.LaneID)
	in.Scope.SelectedPaths = cleanStrings(in.Scope.SelectedPaths)
	in.MaterialHash = strings.TrimSpace(in.MaterialHash)
	in.EvidenceHash = strings.TrimSpace(in.EvidenceHash)
	in.CourtCaseID = strings.TrimSpace(in.CourtCaseID)
	in.CourtExhibitID = strings.TrimSpace(in.CourtExhibitID)
	in.CourtRulingID = strings.TrimSpace(in.CourtRulingID)
	in.AdmissionSyscallID = strings.TrimSpace(in.AdmissionSyscallID)
	in.SourceProvenanceID = strings.TrimSpace(in.SourceProvenanceID)
	in.MaterializationProvenanceID = strings.TrimSpace(in.MaterializationProvenanceID)
	in.CommittedBy = strings.TrimSpace(in.CommittedBy)
	return in
}

func sameScope(a, b domain.ForgeScope) bool {
	return strings.TrimSpace(a.WorkspaceID) == strings.TrimSpace(b.WorkspaceID) &&
		strings.TrimSpace(a.LaneID) == strings.TrimSpace(b.LaneID) &&
		reflect.DeepEqual(cleanStrings(a.SelectedPaths), cleanStrings(b.SelectedPaths))
}

func cleanStrings(values []string) []string {
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
	sort.Strings(out)
	return out
}

func validHash(value string) bool {
	value = strings.TrimSpace(value)
	if !strings.HasPrefix(value, "sha256:") || len(value) != len("sha256:")+sha256.Size*2 {
		return false
	}
	raw := strings.TrimPrefix(value, "sha256:")
	decoded, err := hex.DecodeString(raw)
	return err == nil && len(decoded) == sha256.Size && raw == strings.ToLower(raw)
}

func readPayloadString(payload map[string]any, key string) string {
	value, _ := payload[key].(string)
	return strings.TrimSpace(value)
}
