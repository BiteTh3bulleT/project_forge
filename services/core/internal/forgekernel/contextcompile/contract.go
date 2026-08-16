// Package contextcompile defines the pure production FORGE-K context compile
// decision contract. It performs no I/O and has no clock, model, gateway,
// cache, database, or simulator dependency.
package contextcompile

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	ContractVersion       = "forge.context_compile.decision.v1"
	PolicyVersionV1       = "forge.context_compile.policy.v1"
	SourceManifestVersion = "forge.memory_evidence.source_manifest.v1"
	CommittedByForgeK     = "forge_k.kernel"
	ScoreScale            = int64(1000)
)

var ErrInvalidInput = errors.New("invalid context compile input")

type ValidationError struct{ Field, Reason string }

func (e *ValidationError) Error() string {
	return fmt.Sprintf("%v: %s: %s", ErrInvalidInput, e.Field, e.Reason)
}
func (e *ValidationError) Unwrap() error { return ErrInvalidInput }
func invalid(field, reason string) error { return &ValidationError{Field: field, Reason: reason} }

type Scope struct {
	WorkspaceID   string   `json:"workspaceId"`
	LaneID        string   `json:"laneId"`
	SelectedPaths []string `json:"selectedPaths"`
}
type Budget struct {
	MaxSources   int `json:"maxSources"`
	MaxSnapshots int `json:"maxSnapshots"`
	MaxTokens    int `json:"maxTokens"`
	MaxBytes     int `json:"maxBytes"`
}
type Options struct {
	SnapshotKind        string `json:"snapshotKind"`
	AllowRestore        bool   `json:"allowRestore"`
	PersistSnapshot     bool   `json:"persistSnapshot"`
	RenderCard          bool   `json:"renderCard"`
	HeaderOnly          bool   `json:"headerOnly"`
	MinimumRestoreScore int64  `json:"minimumRestoreScore"`
}
type Hints struct {
	PreferredSnapshotIDs []string `json:"preferredSnapshotIds"`
	RequiredEvidenceIDs  []string `json:"requiredEvidenceIds"`
	RejectedEvidenceIDs  []string `json:"rejectedEvidenceIds"`
}
type Request struct {
	Scope         Scope   `json:"scope"`
	Query         string  `json:"query"`
	Budget        Budget  `json:"budget"`
	Options       Options `json:"options"`
	Hints         Hints   `json:"hints"`
	RequestedAt   int64   `json:"requestedAt"`
	PolicyVersion string  `json:"policyVersion"`
}

type SourceCommitment struct {
	MemoryEvidenceRowID         int64  `json:"memoryEvidenceRowId"`
	EvidenceID                  string `json:"evidenceId"`
	RootEvidenceID              string `json:"rootEvidenceId"`
	Revision                    int    `json:"revision"`
	CourtCaseID                 string `json:"courtCaseId"`
	CourtExhibitID              string `json:"courtExhibitId"`
	CourtRulingID               string `json:"courtRulingId"`
	AdmissionSyscallID          string `json:"admissionSyscallId"`
	SourceObjectKind            string `json:"sourceObjectKind"`
	SourceObjectID              string `json:"sourceObjectId"`
	SourceObjectVersion         string `json:"sourceObjectVersion"`
	SourceObjectHash            string `json:"sourceObjectHash"`
	Scope                       Scope  `json:"scope"`
	ContentHash                 string `json:"contentHash"`
	SourceProvenanceID          string `json:"sourceProvenanceId"`
	MaterializationProvenanceID string `json:"materializationProvenanceId"`
	SyscallID                   string `json:"syscallId"`
	TransactionID               string `json:"transactionId"`
	JournalEventID              string `json:"journalEventId"`
	AuditOutboxID               string `json:"auditOutboxId"`
	AuthorizationFingerprint    string `json:"authorizationFingerprint"`
	CommittedBy                 string `json:"committedBy"`
	Current                     bool   `json:"current"`
	Admitted                    bool   `json:"admitted"`
}
type SourceManifest struct {
	Version      string             `json:"version"`
	Scope        Scope              `json:"scope"`
	Sources      []SourceCommitment `json:"sources"`
	ManifestHash string             `json:"manifestHash"`
}

type CandidateSnapshot struct {
	SnapshotID         string `json:"snapshotId"`
	Scope              Scope  `json:"scope"`
	QueryHash          string `json:"queryHash"`
	SnapshotKind       string `json:"snapshotKind"`
	CreatedAt          int64  `json:"createdAt"`
	SourceManifestHash string `json:"sourceManifestHash"`
	PacketHash         string `json:"packetHash"`
	SnapshotHash       string `json:"snapshotHash"`
	PolicyDigest       string `json:"policyDigest"`
	ProvenanceID       string `json:"provenanceId"`
	SyscallID          string `json:"syscallId"`
	JournalEventID     string `json:"journalEventId"`
	CommittedBy        string `json:"committedBy"`
}
type OutcomeProjectionHead struct {
	SnapshotID     string `json:"snapshotId"`
	Scope          Scope  `json:"scope"`
	Revision       int64  `json:"revision"`
	HelpfulCount   int64  `json:"helpfulCount"`
	HarmfulCount   int64  `json:"harmfulCount"`
	EventSetHash   string `json:"eventSetHash"`
	HeadHash       string `json:"headHash"`
	ProvenanceID   string `json:"provenanceId"`
	SyscallID      string `json:"syscallId"`
	JournalEventID string `json:"journalEventId"`
	CommittedBy    string `json:"committedBy"`
}
type PriorSnapshotHead struct {
	Present        bool   `json:"present"`
	Scope          Scope  `json:"scope"`
	SnapshotID     string `json:"snapshotId"`
	SnapshotHash   string `json:"snapshotHash"`
	HeadHash       string `json:"headHash"`
	Revision       int64  `json:"revision"`
	ProvenanceID   string `json:"provenanceId"`
	SyscallID      string `json:"syscallId"`
	JournalEventID string `json:"journalEventId"`
	CommittedBy    string `json:"committedBy"`
}

type Limits struct {
	MaxSelectedPaths        int   `json:"maxSelectedPaths"`
	MaxQueryBytes           int   `json:"maxQueryBytes"`
	MaxSources              int   `json:"maxSources"`
	MaxCandidates           int   `json:"maxCandidates"`
	MaxOutcomeHeads         int   `json:"maxOutcomeHeads"`
	MaxHintIDs              int   `json:"maxHintIds"`
	MaxTokens               int   `json:"maxTokens"`
	MaxBytes                int   `json:"maxBytes"`
	MaxStringBytes          int   `json:"maxStringBytes"`
	MaxOutcomeEventsPerHead int64 `json:"maxOutcomeEventsPerHead"`
}
type Weights struct {
	QueryMatch                 int64 `json:"queryMatch"`
	SourceManifestMatch        int64 `json:"sourceManifestMatch"`
	SnapshotKindMatch          int64 `json:"snapshotKindMatch"`
	PreferredSnapshot          int64 `json:"preferredSnapshot"`
	PriorHeadContinuity        int64 `json:"priorHeadContinuity"`
	RecentSnapshot             int64 `json:"recentSnapshot"`
	HeaderOnlyPenalty          int64 `json:"headerOnlyPenalty"`
	HelpfulOutcomeUnit         int64 `json:"helpfulOutcomeUnit"`
	HarmfulOutcomeUnit         int64 `json:"harmfulOutcomeUnit"`
	OutcomeAdjustmentCap       int64 `json:"outcomeAdjustmentCap"`
	MaxRecentAgeMs             int64 `json:"maxRecentAgeMs"`
	DefaultMinimumRestoreScore int64 `json:"defaultMinimumRestoreScore"`
}
type Policy struct {
	Version string  `json:"version"`
	Limits  Limits  `json:"limits"`
	Weights Weights `json:"weights"`
	Digest  string  `json:"digest"`
}
type Input struct {
	Request           Request                 `json:"request"`
	SourceManifest    SourceManifest          `json:"sourceManifest"`
	Candidates        []CandidateSnapshot     `json:"candidates"`
	OutcomeHeads      []OutcomeProjectionHead `json:"outcomeHeads"`
	PriorSnapshotHead PriorSnapshotHead       `json:"priorSnapshotHead"`
	Policy            Policy                  `json:"policy"`
}

type ScoreBreakdown struct {
	QueryMatch          int64 `json:"queryMatch"`
	SourceManifestMatch int64 `json:"sourceManifestMatch"`
	SnapshotKindMatch   int64 `json:"snapshotKindMatch"`
	PreferredSnapshot   int64 `json:"preferredSnapshot"`
	PriorHeadContinuity int64 `json:"priorHeadContinuity"`
	Recency             int64 `json:"recency"`
	HeaderOnlyPenalty   int64 `json:"headerOnlyPenalty"`
	OutcomeAdjustment   int64 `json:"outcomeAdjustment"`
}
type RestoreScore struct {
	SnapshotID      string         `json:"snapshotId"`
	SnapshotHash    string         `json:"snapshotHash"`
	Total           int64          `json:"total"`
	Breakdown       ScoreBreakdown `json:"breakdown"`
	Eligible        bool           `json:"eligible"`
	RejectionReason string         `json:"rejectionReason"`
}
type Selection struct {
	Mode         string `json:"mode"`
	SnapshotID   string `json:"snapshotId"`
	SnapshotHash string `json:"snapshotHash"`
	Score        int64  `json:"score"`
	Reason       string `json:"reason"`
	Commitment   string `json:"commitment"`
}
type Decision struct {
	Version                     string         `json:"version"`
	RequestHash                 string         `json:"requestHash"`
	PacketID                    string         `json:"packetId"`
	SnapshotID                  string         `json:"snapshotId"`
	PacketCommitment            string         `json:"packetCommitment"`
	SnapshotCommitment          string         `json:"snapshotCommitment"`
	RestoreScoreTable           []RestoreScore `json:"restoreScoreTable"`
	RestoreScoreTableCommitment string         `json:"restoreScoreTableCommitment"`
	Selection                   Selection      `json:"selection"`
	OutcomeCommitment           string         `json:"outcomeCommitment"`
	FeedbackHeadsCommitment     string         `json:"feedbackHeadsCommitment"`
	CandidateSetCommitment      string         `json:"candidateSetCommitment"`
	CardCommitment              string         `json:"cardCommitment"`
	SelectedEvidenceIDs         []string       `json:"selectedEvidenceIds"`
	SourceManifestHash          string         `json:"sourceManifestHash"`
	PriorSnapshotHead           string         `json:"priorSnapshotHead"`
	PriorSnapshotHeadCommitment string         `json:"priorSnapshotHeadCommitment"`
	PolicyDigest                string         `json:"policyDigest"`
	DecisionDigest              string         `json:"decisionDigest"`
}

func V1Policy() Policy {
	p := Policy{Version: PolicyVersionV1,
		Limits:  Limits{MaxSelectedPaths: 64, MaxQueryBytes: 8192, MaxSources: 256, MaxCandidates: 128, MaxOutcomeHeads: 128, MaxHintIDs: 128, MaxTokens: 131072, MaxBytes: 4 << 20, MaxStringBytes: 4096, MaxOutcomeEventsPerHead: 1_000_000},
		Weights: Weights{QueryMatch: 4000, SourceManifestMatch: 2500, SnapshotKindMatch: 1000, PreferredSnapshot: 750, PriorHeadContinuity: 500, RecentSnapshot: 750, HeaderOnlyPenalty: -500, HelpfulOutcomeUnit: 100, HarmfulOutcomeUnit: -200, OutcomeAdjustmentCap: 1000, MaxRecentAgeMs: 86400000, DefaultMinimumRestoreScore: 5000},
	}
	p.Digest, _ = policyDigest(p)
	return p
}

func Compile(input Input) (Decision, error) {
	n, err := normalizeAndValidate(input)
	if err != nil {
		return Decision{}, err
	}
	reqHash, _ := hash(n.Request)
	scores := scoreCandidates(n)
	selection := choose(scores, n)
	selectedEvidence, err := selectEvidence(n)
	if err != nil {
		return Decision{}, err
	}
	scoreHash, _ := hash(scores)
	feedbackHeadsHash, _ := hash(n.OutcomeHeads)
	candidateSetHash, _ := hash(n.Candidates)
	priorHeadHash, _ := hash(n.PriorSnapshotHead)
	selection.Commitment, _ = hash(selection)
	packetShape := struct {
		Version, RequestHash, SourceManifestHash, SelectionCommitment string
		EvidenceIDs                                                   []string
		Budget                                                        Budget
	}{ContractVersion, reqHash, n.SourceManifest.ManifestHash, selection.Commitment, selectedEvidence, n.Request.Budget}
	packetHash, _ := hash(packetShape)
	packetID := "context-packet:" + hashSuffix(packetHash)
	prior := ""
	if n.PriorSnapshotHead.Present {
		prior = n.PriorSnapshotHead.HeadHash
	}
	snapshotShape := struct {
		Version, PacketHash, PriorHead, PolicyDigest string
		RequestedAt                                  int64
		Persist                                      bool
	}{ContractVersion, packetHash, prior, n.Policy.Digest, n.Request.RequestedAt, n.Request.Options.PersistSnapshot}
	snapshotHash, _ := hash(snapshotShape)
	snapshotID := "context-snapshot:" + hashSuffix(snapshotHash)
	cardHash, _ := hash(struct {
		Render                                 bool
		PacketHash, SnapshotHash, PolicyDigest string
	}{n.Request.Options.RenderCard, packetHash, snapshotHash, n.Policy.Digest})
	outcomeHash, _ := hash(struct {
		Mode                    string `json:"mode"`
		SelectedSnapshotID      string `json:"selectedSnapshotId"`
		SelectedSnapshotHash    string `json:"selectedSnapshotHash"`
		Score                   int64  `json:"score"`
		RequiresFreshCompile    bool   `json:"requiresFreshCompile"`
		FeedbackHeadsCommitment string `json:"feedbackHeadsCommitment"`
	}{selection.Mode, selection.SnapshotID, selection.SnapshotHash, selection.Score, selection.Mode == "fresh_compile", feedbackHeadsHash})
	d := Decision{Version: ContractVersion, RequestHash: reqHash, PacketID: packetID, SnapshotID: snapshotID, PacketCommitment: packetHash, SnapshotCommitment: snapshotHash, RestoreScoreTable: scores, RestoreScoreTableCommitment: scoreHash, Selection: selection, OutcomeCommitment: outcomeHash, FeedbackHeadsCommitment: feedbackHeadsHash, CandidateSetCommitment: candidateSetHash, CardCommitment: cardHash, SelectedEvidenceIDs: selectedEvidence, SourceManifestHash: n.SourceManifest.ManifestHash, PriorSnapshotHead: prior, PriorSnapshotHeadCommitment: priorHeadHash, PolicyDigest: n.Policy.Digest}
	d.DecisionDigest, _ = decisionDigest(d)
	return d, nil
}

func normalizeAndValidate(in Input) (Input, error) {
	pd, err := policyDigest(in.Policy)
	if err != nil || in.Policy.Version != PolicyVersionV1 || in.Policy.Digest != pd {
		return Input{}, invalid("policy.digest", "policy is not the sealed production v1 policy")
	}
	if in.Policy != V1Policy() {
		return Input{}, invalid("policy", "weights or limits differ from sealed production v1")
	}
	r := &in.Request
	r.Query = normalizeQuery(r.Query)
	r.PolicyVersion = strings.TrimSpace(r.PolicyVersion)
	r.Scope, err = normalizeScope(r.Scope, in.Policy)
	if err != nil {
		return Input{}, err
	}
	if r.Query == "" || len(r.Query) > in.Policy.Limits.MaxQueryBytes {
		return Input{}, invalid("request.query", "normalized query is empty or exceeds bound")
	}
	if r.RequestedAt <= 0 || r.PolicyVersion != in.Policy.Version {
		return Input{}, invalid("request", "requestedAt and policyVersion are required")
	}
	if r.Budget.MaxSources < 1 || r.Budget.MaxSources > in.Policy.Limits.MaxSources || r.Budget.MaxSnapshots < 0 || r.Budget.MaxSnapshots > in.Policy.Limits.MaxCandidates || r.Budget.MaxTokens < 1 || r.Budget.MaxTokens > in.Policy.Limits.MaxTokens || r.Budget.MaxBytes < 1 || r.Budget.MaxBytes > in.Policy.Limits.MaxBytes {
		return Input{}, invalid("request.budget", "budget is outside sealed bounds")
	}
	r.Options.SnapshotKind = strings.TrimSpace(r.Options.SnapshotKind)
	if !validID(r.Options.SnapshotKind, in.Policy.Limits.MaxStringBytes) {
		return Input{}, invalid("request.options.snapshotKind", "invalid snapshot kind")
	}
	if r.Options.MinimumRestoreScore == 0 {
		r.Options.MinimumRestoreScore = in.Policy.Weights.DefaultMinimumRestoreScore
	}
	if r.Options.MinimumRestoreScore < 0 || r.Options.MinimumRestoreScore > 12000 {
		return Input{}, invalid("request.options.minimumRestoreScore", "score threshold is outside bound")
	}
	if r.Hints.PreferredSnapshotIDs, err = normalizeIDs(r.Hints.PreferredSnapshotIDs, in.Policy.Limits.MaxHintIDs); err != nil {
		return Input{}, invalid("request.hints.preferredSnapshotIds", err.Error())
	}
	if r.Hints.RequiredEvidenceIDs, err = normalizeIDs(r.Hints.RequiredEvidenceIDs, in.Policy.Limits.MaxHintIDs); err != nil {
		return Input{}, invalid("request.hints.requiredEvidenceIds", err.Error())
	}
	if r.Hints.RejectedEvidenceIDs, err = normalizeIDs(r.Hints.RejectedEvidenceIDs, in.Policy.Limits.MaxHintIDs); err != nil {
		return Input{}, invalid("request.hints.rejectedEvidenceIds", err.Error())
	}
	if overlaps(r.Hints.RequiredEvidenceIDs, r.Hints.RejectedEvidenceIDs) {
		return Input{}, invalid("request.hints", "required and rejected evidence overlap")
	}
	if len(r.Hints.RequiredEvidenceIDs) > r.Budget.MaxSources {
		return Input{}, invalid("request.hints.requiredEvidenceIds", "required evidence exceeds source budget")
	}
	if err = normalizeManifest(&in.SourceManifest, r.Scope, in.Policy); err != nil {
		return Input{}, err
	}
	if len(in.Candidates) > r.Budget.MaxSnapshots || len(in.Candidates) > in.Policy.Limits.MaxCandidates {
		return Input{}, invalid("candidates", "candidate count exceeds bound")
	}
	seenCandidates := map[string]struct{}{}
	seenCandidateHashes := map[string]struct{}{}
	for i := range in.Candidates {
		if err = normalizeCandidate(&in.Candidates[i], r.Scope, r.RequestedAt, in.Policy); err != nil {
			return Input{}, invalid(fmt.Sprintf("candidates[%d]", i), err.Error())
		}
		if _, ok := seenCandidates[in.Candidates[i].SnapshotID]; ok {
			return Input{}, invalid("candidates", "duplicate snapshot identity")
		}
		if _, ok := seenCandidateHashes[in.Candidates[i].SnapshotHash]; ok {
			return Input{}, invalid("candidates", "duplicate snapshot commitment")
		}
		seenCandidates[in.Candidates[i].SnapshotID] = struct{}{}
		seenCandidateHashes[in.Candidates[i].SnapshotHash] = struct{}{}
	}
	sort.Slice(in.Candidates, func(i, j int) bool { return in.Candidates[i].SnapshotID < in.Candidates[j].SnapshotID })
	if len(in.OutcomeHeads) > in.Policy.Limits.MaxOutcomeHeads {
		return Input{}, invalid("outcomeHeads", "outcome head count exceeds bound")
	}
	seenOutcomes := map[string]struct{}{}
	for i := range in.OutcomeHeads {
		h := &in.OutcomeHeads[i]
		h.SnapshotID = strings.TrimSpace(h.SnapshotID)
		h.Scope, err = normalizeScope(h.Scope, in.Policy)
		expectedHeadHash, _ := outcomeHeadDigest(*h)
		if err != nil || !sameScope(h.Scope, r.Scope) || !validID(h.SnapshotID, in.Policy.Limits.MaxStringBytes) || !validHash(h.EventSetHash) || !validHash(h.HeadHash) || h.HeadHash != expectedHeadHash || h.Revision < 1 || h.HelpfulCount < 0 || h.HarmfulCount < 0 || h.HelpfulCount > in.Policy.Limits.MaxOutcomeEventsPerHead || h.HarmfulCount > in.Policy.Limits.MaxOutcomeEventsPerHead || !validGovernance(h.ProvenanceID, h.SyscallID, h.JournalEventID, h.CommittedBy, in.Policy) {
			return Input{}, invalid(fmt.Sprintf("outcomeHeads[%d]", i), "malformed, cross-scope, or ungoverned outcome head")
		}
		if _, ok := seenCandidates[h.SnapshotID]; !ok {
			return Input{}, invalid("outcomeHeads", "head names unknown snapshot")
		}
		if _, ok := seenOutcomes[h.SnapshotID]; ok {
			return Input{}, invalid("outcomeHeads", "duplicate snapshot head")
		}
		seenOutcomes[h.SnapshotID] = struct{}{}
	}
	sort.Slice(in.OutcomeHeads, func(i, j int) bool { return in.OutcomeHeads[i].SnapshotID < in.OutcomeHeads[j].SnapshotID })
	if err = normalizePrior(&in.PriorSnapshotHead, r.Scope, in.Policy); err != nil {
		return Input{}, err
	}
	return in, nil
}

func normalizeManifest(m *SourceManifest, scope Scope, p Policy) error {
	var err error
	m.Version = strings.TrimSpace(m.Version)
	m.Scope, err = normalizeScope(m.Scope, p)
	if err != nil || m.Version != SourceManifestVersion || !sameScope(m.Scope, scope) || len(m.Sources) < 1 || len(m.Sources) > p.Limits.MaxSources {
		return invalid("sourceManifest", "invalid version, scope, or source count")
	}
	seen := map[string]struct{}{}
	seenRows := map[int64]struct{}{}
	seenCourtVersions := map[string]struct{}{}
	for i := range m.Sources {
		s := &m.Sources[i]
		s.EvidenceID = strings.TrimSpace(s.EvidenceID)
		s.RootEvidenceID = strings.TrimSpace(s.RootEvidenceID)
		s.Scope, err = normalizeScope(s.Scope, p)
		if err != nil || !sameScope(s.Scope, scope) || s.MemoryEvidenceRowID < 1 || s.Revision < 1 || !validID(s.EvidenceID, p.Limits.MaxStringBytes) || !validID(s.RootEvidenceID, p.Limits.MaxStringBytes) || !validID(s.CourtCaseID, p.Limits.MaxStringBytes) || !validID(s.CourtExhibitID, p.Limits.MaxStringBytes) || !validID(s.CourtRulingID, p.Limits.MaxStringBytes) || !validID(s.AdmissionSyscallID, p.Limits.MaxStringBytes) || s.SourceObjectKind != "court_exhibit" || s.SourceObjectID != s.CourtExhibitID || s.SourceObjectVersion != s.CourtRulingID || !validHash(s.SourceObjectHash) || s.ContentHash != s.SourceObjectHash || !validGovernance(s.SourceProvenanceID, s.SyscallID, s.JournalEventID, s.CommittedBy, p) || !validID(s.MaterializationProvenanceID, p.Limits.MaxStringBytes) || !validID(s.TransactionID, p.Limits.MaxStringBytes) || !validID(s.AuditOutboxID, p.Limits.MaxStringBytes) || !validHash(s.AuthorizationFingerprint) || !s.Current || !s.Admitted {
			return invalid(fmt.Sprintf("sourceManifest.sources[%d]", i), "malformed, cross-scope, non-current, non-admitted, or ungoverned K20H source")
		}
		if _, ok := seen[s.EvidenceID]; ok {
			return invalid("sourceManifest.sources", "duplicate evidence identity")
		}
		if _, ok := seenRows[s.MemoryEvidenceRowID]; ok {
			return invalid("sourceManifest.sources", "duplicate evidence row identity")
		}
		courtVersion := s.CourtExhibitID + "\x00" + s.CourtRulingID
		if _, ok := seenCourtVersions[courtVersion]; ok {
			return invalid("sourceManifest.sources", "duplicate Court source version")
		}
		seen[s.EvidenceID] = struct{}{}
		seenRows[s.MemoryEvidenceRowID] = struct{}{}
		seenCourtVersions[courtVersion] = struct{}{}
	}
	sort.Slice(m.Sources, func(i, j int) bool {
		if m.Sources[i].EvidenceID != m.Sources[j].EvidenceID {
			return m.Sources[i].EvidenceID < m.Sources[j].EvidenceID
		}
		return m.Sources[i].MemoryEvidenceRowID < m.Sources[j].MemoryEvidenceRowID
	})
	expected, _ := sourceManifestDigest(*m)
	if !validHash(m.ManifestHash) || m.ManifestHash != expected {
		return invalid("sourceManifest.manifestHash", "source manifest commitment mismatch")
	}
	return nil
}

func normalizeCandidate(c *CandidateSnapshot, scope Scope, requestedAt int64, p Policy) error {
	var err error
	c.SnapshotID = strings.TrimSpace(c.SnapshotID)
	c.SnapshotKind = strings.TrimSpace(c.SnapshotKind)
	c.Scope, err = normalizeScope(c.Scope, p)
	expectedSnapshotHash, _ := candidateSnapshotDigest(*c)
	if err != nil || !sameScope(c.Scope, scope) || !validID(c.SnapshotID, p.Limits.MaxStringBytes) || !validID(c.SnapshotKind, p.Limits.MaxStringBytes) || c.CreatedAt <= 0 || c.CreatedAt > requestedAt || !validHash(c.QueryHash) || !validHash(c.SourceManifestHash) || !validHash(c.PacketHash) || !validHash(c.SnapshotHash) || c.SnapshotHash != expectedSnapshotHash || !validHash(c.PolicyDigest) || c.PolicyDigest != p.Digest || !validGovernance(c.ProvenanceID, c.SyscallID, c.JournalEventID, c.CommittedBy, p) {
		return errors.New("malformed, future, cross-scope, or ungoverned candidate")
	}
	return nil
}
func normalizePrior(h *PriorSnapshotHead, scope Scope, p Policy) error {
	if !h.Present {
		*h = PriorSnapshotHead{}
		return nil
	}
	var err error
	h.Scope, err = normalizeScope(h.Scope, p)
	expectedHeadHash, _ := priorSnapshotHeadDigest(*h)
	if err != nil || !sameScope(h.Scope, scope) || !validID(h.SnapshotID, p.Limits.MaxStringBytes) || !validHash(h.SnapshotHash) || !validHash(h.HeadHash) || h.HeadHash != expectedHeadHash || h.Revision < 1 || !validGovernance(h.ProvenanceID, h.SyscallID, h.JournalEventID, h.CommittedBy, p) {
		return invalid("priorSnapshotHead", "malformed, cross-scope, or ungoverned prior head")
	}
	return nil
}

func scoreCandidates(in Input) []RestoreScore {
	out := make([]RestoreScore, 0, len(in.Candidates))
	queryHash, _ := hash(in.Request.Query)
	pref := set(in.Request.Hints.PreferredSnapshotIDs)
	outcomes := map[string]OutcomeProjectionHead{}
	for _, h := range in.OutcomeHeads {
		outcomes[h.SnapshotID] = h
	}
	for _, c := range in.Candidates {
		b := ScoreBreakdown{}
		if c.QueryHash == queryHash {
			b.QueryMatch = in.Policy.Weights.QueryMatch
		}
		if c.SourceManifestHash == in.SourceManifest.ManifestHash {
			b.SourceManifestMatch = in.Policy.Weights.SourceManifestMatch
		}
		if c.SnapshotKind == in.Request.Options.SnapshotKind {
			b.SnapshotKindMatch = in.Policy.Weights.SnapshotKindMatch
		}
		if _, ok := pref[c.SnapshotID]; ok {
			b.PreferredSnapshot = in.Policy.Weights.PreferredSnapshot
		}
		if in.PriorSnapshotHead.Present && c.SnapshotID == in.PriorSnapshotHead.SnapshotID && c.SnapshotHash == in.PriorSnapshotHead.SnapshotHash {
			b.PriorHeadContinuity = in.Policy.Weights.PriorHeadContinuity
		}
		age := in.Request.RequestedAt - c.CreatedAt
		if age >= 0 && age <= in.Policy.Weights.MaxRecentAgeMs {
			b.Recency = in.Policy.Weights.RecentSnapshot * (in.Policy.Weights.MaxRecentAgeMs - age) / in.Policy.Weights.MaxRecentAgeMs
		}
		if in.Request.Options.HeaderOnly {
			b.HeaderOnlyPenalty = in.Policy.Weights.HeaderOnlyPenalty
		}
		if h, ok := outcomes[c.SnapshotID]; ok {
			b.OutcomeAdjustment = clamp(h.HelpfulCount*in.Policy.Weights.HelpfulOutcomeUnit+h.HarmfulCount*in.Policy.Weights.HarmfulOutcomeUnit, -in.Policy.Weights.OutcomeAdjustmentCap, in.Policy.Weights.OutcomeAdjustmentCap)
		}
		total := b.QueryMatch + b.SourceManifestMatch + b.SnapshotKindMatch + b.PreferredSnapshot + b.PriorHeadContinuity + b.Recency + b.HeaderOnlyPenalty + b.OutcomeAdjustment
		eligible := in.Request.Options.AllowRestore && c.QueryHash == queryHash && c.SourceManifestHash == in.SourceManifest.ManifestHash && total >= in.Request.Options.MinimumRestoreScore
		reason := ""
		if !eligible {
			reason = "below_threshold_or_identity_gate"
		}
		out = append(out, RestoreScore{SnapshotID: c.SnapshotID, SnapshotHash: c.SnapshotHash, Total: total, Breakdown: b, Eligible: eligible, RejectionReason: reason})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Total != out[j].Total {
			return out[i].Total > out[j].Total
		}
		ci, cj := candidateByID(in, out[i].SnapshotID), candidateByID(in, out[j].SnapshotID)
		if ci.CreatedAt != cj.CreatedAt {
			return ci.CreatedAt > cj.CreatedAt
		}
		if out[i].SnapshotID != out[j].SnapshotID {
			return out[i].SnapshotID < out[j].SnapshotID
		}
		return out[i].SnapshotHash < out[j].SnapshotHash
	})
	return out
}
func choose(scores []RestoreScore, in Input) Selection {
	for _, s := range scores {
		if s.Eligible {
			return Selection{Mode: "restore", SnapshotID: s.SnapshotID, SnapshotHash: s.SnapshotHash, Score: s.Total, Reason: "highest_eligible_fixed_point_score"}
		}
	}
	return Selection{Mode: "fresh_compile", Reason: "no_eligible_restore_candidate"}
}
func selectEvidence(in Input) ([]string, error) {
	required, rejected := set(in.Request.Hints.RequiredEvidenceIDs), set(in.Request.Hints.RejectedEvidenceIDs)
	available := map[string]struct{}{}
	for _, s := range in.SourceManifest.Sources {
		available[s.EvidenceID] = struct{}{}
	}
	for id := range required {
		if _, ok := available[id]; !ok {
			return nil, invalid("request.hints.requiredEvidenceIds", "required evidence absent from current admitted source manifest")
		}
	}
	out := append([]string(nil), in.Request.Hints.RequiredEvidenceIDs...)
	chosen := set(out)
	for _, s := range in.SourceManifest.Sources {
		if len(out) >= in.Request.Budget.MaxSources {
			break
		}
		if _, ok := chosen[s.EvidenceID]; ok {
			continue
		}
		if _, ok := rejected[s.EvidenceID]; ok {
			continue
		}
		out = append(out, s.EvidenceID)
	}
	sort.Strings(out)
	return out, nil
}

func normalizeScope(s Scope, p Policy) (Scope, error) {
	s.WorkspaceID = strings.TrimSpace(s.WorkspaceID)
	s.LaneID = strings.TrimSpace(s.LaneID)
	if !validID(s.WorkspaceID, p.Limits.MaxStringBytes) || !validID(s.LaneID, p.Limits.MaxStringBytes) {
		return Scope{}, invalid("scope", "workspace and lane are required")
	}
	paths, err := normalizePaths(s.SelectedPaths, p.Limits.MaxSelectedPaths, p.Limits.MaxStringBytes)
	if err != nil {
		return Scope{}, invalid("scope.selectedPaths", err.Error())
	}
	s.SelectedPaths = paths
	return s, nil
}
func normalizePaths(v []string, max, maxBytes int) ([]string, error) {
	if len(v) == 0 || len(v) > max {
		return nil, errors.New("path count outside bound")
	}
	out := make([]string, len(v))
	for i, x := range v {
		x = strings.TrimSpace(x)
		if x == "" || len(x) > maxBytes || !utf8.ValidString(x) || strings.ContainsRune(x, 0) {
			return nil, errors.New("invalid path")
		}
		out[i] = x
	}
	sort.Strings(out)
	for i := 1; i < len(out); i++ {
		if out[i] == out[i-1] {
			return nil, errors.New("duplicate path")
		}
	}
	return out, nil
}
func normalizeIDs(v []string, max int) ([]string, error) {
	if len(v) > max {
		return nil, errors.New("count exceeds bound")
	}
	out := make([]string, len(v))
	for i, x := range v {
		x = strings.TrimSpace(x)
		if !validID(x, 4096) {
			return nil, errors.New("invalid identity")
		}
		out[i] = x
	}
	sort.Strings(out)
	for i := 1; i < len(out); i++ {
		if out[i] == out[i-1] {
			return nil, errors.New("duplicate identity")
		}
	}
	return out, nil
}
func normalizeQuery(q string) string { return strings.Join(strings.Fields(strings.TrimSpace(q)), " ") }
func validID(v string, max int) bool {
	v = strings.TrimSpace(v)
	if v == "" || len(v) > max || !utf8.ValidString(v) {
		return false
	}
	for _, r := range v {
		if unicode.IsControl(r) || unicode.IsSpace(r) {
			return false
		}
	}
	return true
}
func validHash(v string) bool {
	if len(v) != 71 || !strings.HasPrefix(v, "sha256:") {
		return false
	}
	_, err := hex.DecodeString(v[7:])
	return err == nil
}
func validGovernance(prov, sys, journal, committed string, p Policy) bool {
	return validID(prov, p.Limits.MaxStringBytes) && validID(sys, p.Limits.MaxStringBytes) && validID(journal, p.Limits.MaxStringBytes) && committed == CommittedByForgeK
}
func sameScope(a, b Scope) bool {
	return a.WorkspaceID == b.WorkspaceID && a.LaneID == b.LaneID && equalStrings(a.SelectedPaths, b.SelectedPaths)
}
func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
func set(v []string) map[string]struct{} {
	m := make(map[string]struct{}, len(v))
	for _, x := range v {
		m[x] = struct{}{}
	}
	return m
}
func overlaps(a, b []string) bool {
	m := set(a)
	for _, x := range b {
		if _, ok := m[x]; ok {
			return true
		}
	}
	return false
}
func clamp(v, min, max int64) int64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}
func candidateByID(in Input, id string) CandidateSnapshot {
	for _, c := range in.Candidates {
		if c.SnapshotID == id {
			return c
		}
	}
	return CandidateSnapshot{}
}
func hash(v any) (string, error) {
	raw, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}
func hashSuffix(v string) string {
	if len(v) > 23 {
		return v[7:23]
	}
	return v
}
func policyDigest(p Policy) (string, error)                 { p.Digest = ""; return hash(p) }
func sourceManifestDigest(m SourceManifest) (string, error) { m.ManifestHash = ""; return hash(m) }
func decisionDigest(d Decision) (string, error)             { d.DecisionDigest = ""; return hash(d) }
func candidateSnapshotDigest(c CandidateSnapshot) (string, error) {
	c.SnapshotHash = ""
	return hash(c)
}
func outcomeHeadDigest(h OutcomeProjectionHead) (string, error)   { h.HeadHash = ""; return hash(h) }
func priorSnapshotHeadDigest(h PriorSnapshotHead) (string, error) { h.HeadHash = ""; return hash(h) }

// SealSourceManifest computes the commitment after applying the same canonical
// source ordering used by Compile. It is intended for bounded adapters/tests;
// Compile still independently verifies the returned hash.
func SealSourceManifest(m SourceManifest) (SourceManifest, error) {
	p := V1Policy()
	var err error
	m.Scope, err = normalizeScope(m.Scope, p)
	if err != nil {
		return SourceManifest{}, err
	}
	for i := range m.Sources {
		m.Sources[i].Scope, err = normalizeScope(m.Sources[i].Scope, p)
		if err != nil {
			return SourceManifest{}, err
		}
	}
	sort.Slice(m.Sources, func(i, j int) bool {
		if m.Sources[i].EvidenceID != m.Sources[j].EvidenceID {
			return m.Sources[i].EvidenceID < m.Sources[j].EvidenceID
		}
		return m.Sources[i].MemoryEvidenceRowID < m.Sources[j].MemoryEvidenceRowID
	})
	m.ManifestHash, err = sourceManifestDigest(m)
	return m, err
}

func SealCandidateSnapshot(c CandidateSnapshot) (CandidateSnapshot, error) {
	p := V1Policy()
	var err error
	c.Scope, err = normalizeScope(c.Scope, p)
	if err != nil {
		return CandidateSnapshot{}, err
	}
	c.SnapshotHash, err = candidateSnapshotDigest(c)
	return c, err
}

func SealOutcomeProjectionHead(h OutcomeProjectionHead) (OutcomeProjectionHead, error) {
	p := V1Policy()
	var err error
	h.Scope, err = normalizeScope(h.Scope, p)
	if err != nil {
		return OutcomeProjectionHead{}, err
	}
	h.HeadHash, err = outcomeHeadDigest(h)
	return h, err
}

func SealPriorSnapshotHead(h PriorSnapshotHead) (PriorSnapshotHead, error) {
	if !h.Present {
		return PriorSnapshotHead{}, nil
	}
	p := V1Policy()
	var err error
	h.Scope, err = normalizeScope(h.Scope, p)
	if err != nil {
		return PriorSnapshotHead{}, err
	}
	h.HeadHash, err = priorSnapshotHeadDigest(h)
	return h, err
}
