package controllane

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"

	"forge/projectforge/services/core/internal/aios/domain"
	"forge/projectforge/services/core/internal/forgekernel"
	"forge/projectforge/services/core/internal/forgekernel/contextcompile"
)

type GovernedContextSource struct {
	EvidenceID     string   `json:"evidenceId"`
	ContentSummary string   `json:"contentSummary"`
	RawRef         string   `json:"rawRef"`
	SourceType     string   `json:"sourceType"`
	SourceRefs     []string `json:"sourceRefs"`
	ContentHash    string   `json:"contentHash"`
}

type GovernedContextBundle struct {
	Input                    contextcompile.Input             `json:"input"`
	Decision                 contextcompile.Decision          `json:"decision"`
	Candidate                contextcompile.CandidateSnapshot `json:"candidate"`
	Sources                  []GovernedContextSource          `json:"sources"`
	Scope                    domain.ForgeScope                `json:"scope"`
	Query                    string                           `json:"query"`
	CreatedAt                int64                            `json:"createdAt"`
	ProvenanceID             string                           `json:"provenanceId"`
	SyscallID                string                           `json:"syscallId"`
	CorrelationID            string                           `json:"correlationId"`
	TraceID                  string                           `json:"traceId"`
	TransactionID            string                           `json:"transactionId"`
	JournalEventID           string                           `json:"journalEventId"`
	AuditOutboxID            string                           `json:"auditOutboxId"`
	AuthorizationFingerprint string                           `json:"authorizationFingerprint"`
	CommittedBy              string                           `json:"committedBy"`
}

func prepareContextCompileAuthorityInput(req domain.SyscallRequest, store SemanticReadStore) (contextcompile.Input, error) {
	if store == nil {
		return contextcompile.Input{}, fmt.Errorf("context authority store unavailable")
	}
	scope := contextCompileScope(req.Scope)
	budget := defaultBudget()
	if raw, ok := req.Payload["budget"].(map[string]any); ok {
		budget.MaxTokens = readInt(raw, "maxTokens", budget.MaxTokens)
		budget.MaxEvents = readInt(raw, "maxEvents", budget.MaxEvents)
		budget.MaxNotes = readInt(raw, "maxNotes", budget.MaxNotes)
	}
	maxSources := budget.MaxNotes
	if maxSources < 1 {
		maxSources = 1
	}
	evidence, err := store.ListCurrentMemoryEvidence(req.Scope, maxSources)
	if err != nil {
		return contextcompile.Input{}, err
	}
	commitments := make([]contextcompile.SourceCommitment, 0, len(evidence))
	for _, source := range evidence {
		commitments = append(commitments, contextcompile.SourceCommitment{
			MemoryEvidenceRowID: source.RowID, EvidenceID: source.EvidenceID,
			RootEvidenceID: source.RootEvidenceID, Revision: source.Revision,
			CourtCaseID: source.CourtCaseID, CourtExhibitID: source.CourtExhibitID,
			CourtRulingID: source.CourtRulingID, AdmissionSyscallID: source.AdmissionSyscallID,
			SourceObjectKind: source.SourceObjectKind, SourceObjectID: source.SourceObjectID,
			SourceObjectVersion: source.SourceObjectVersion, SourceObjectHash: source.SourceObjectHash,
			Scope: contextCompileScope(source.Scope), ContentHash: source.ContentHash,
			SourceProvenanceID:          source.SourceProvenanceID,
			MaterializationProvenanceID: source.MaterializationProvenanceID,
			SyscallID:                   source.SyscallID, TransactionID: source.TransactionID,
			JournalEventID: source.JournalEventID, AuditOutboxID: source.AuditOutboxID,
			AuthorizationFingerprint: source.AuthorizationFingerprint,
			CommittedBy:              source.CommittedBy, Current: true, Admitted: true,
		})
	}
	manifest, err := contextcompile.SealSourceManifest(contextcompile.SourceManifest{
		Version: contextcompile.SourceManifestVersion, Scope: scope, Sources: commitments,
	})
	if err != nil {
		return contextcompile.Input{}, err
	}
	opts := mergeCompileContextOptions(req.Payload)
	query := strings.TrimSpace(readString(req.Payload, "query"))
	if query == "" {
		query = strings.TrimSpace(readString(req.Metadata, "query"))
	}
	candidateLimit := opts.RestoreCandidateLimit
	if candidateLimit <= 0 {
		candidateLimit = defaultRestoreCandidateLimit
	}
	candidates, err := store.ListGovernedContextCandidates(req.Scope, query, opts.SnapshotKind, candidateLimit)
	if err != nil {
		return contextcompile.Input{}, err
	}
	prior, present, err := store.FindGovernedContextHead(req.Scope)
	if err != nil {
		return contextcompile.Input{}, err
	}
	if !present {
		prior = contextcompile.PriorSnapshotHead{}
	}
	hints := readCompileContextResumeHints(req.Payload)
	preferred := []string{}
	if strings.TrimSpace(hints.PreferredSnapshotID) != "" {
		preferred = []string{strings.TrimSpace(hints.PreferredSnapshotID)}
	}
	minimum := int64(opts.RestoreMinScore * float64(contextcompile.ScoreScale))
	if minimum < 0 {
		minimum = 0
	}
	return contextcompile.Input{
		Request: contextcompile.Request{
			Scope: scope, Query: query,
			Budget:  contextcompile.Budget{MaxSources: maxSources, MaxSnapshots: candidateLimit, MaxTokens: budget.MaxTokens, MaxBytes: minInt(4<<20, budget.MaxTokens*16)},
			Options: contextcompile.Options{SnapshotKind: nonEmpty(opts.SnapshotKind, "chat"), AllowRestore: opts.RestoreMode && !hints.FreshCompileOnly, PersistSnapshot: opts.PersistSnapshot, RenderCard: opts.RenderSnapshotCard, HeaderOnly: false, MinimumRestoreScore: minimum},
			Hints:   contextcompile.Hints{PreferredSnapshotIDs: preferred}, RequestedAt: req.RequestedAt, PolicyVersion: contextcompile.PolicyVersionV1,
		},
		SourceManifest: manifest, Candidates: candidates, OutcomeHeads: []contextcompile.OutcomeProjectionHead{},
		PriorSnapshotHead: prior, Policy: contextcompile.V1Policy(),
	}, nil
}

func buildGovernedContextBundle(req domain.SyscallRequest, input contextcompile.Input, decision contextcompile.Decision, store SemanticReadStore) (GovernedContextBundle, error) {
	if err := contextcompile.VerifyDecision(input, decision); err != nil {
		return GovernedContextBundle{}, err
	}
	currentInput, err := prepareContextCompileAuthorityInput(req, store)
	if err != nil {
		return GovernedContextBundle{}, err
	}
	currentDecision, err := contextcompile.Compile(currentInput)
	if err != nil || currentDecision.DecisionDigest != decision.DecisionDigest {
		return GovernedContextBundle{}, fmt.Errorf("context authority inputs changed before commit")
	}
	sources := make([]GovernedContextSource, 0, len(decision.SelectedEvidenceIDs))
	for _, id := range decision.SelectedEvidenceIDs {
		evidence, ok := store.FindMemoryEvidence(id, req.Scope)
		if !ok || store.HasMemoryEvidenceSupersession(id) {
			return GovernedContextBundle{}, fmt.Errorf("selected context evidence %q is no longer current", id)
		}
		sources = append(sources, GovernedContextSource{EvidenceID: id, ContentSummary: evidence.ContentSummary, RawRef: evidence.RawRef, SourceType: evidence.SourceType, SourceRefs: append([]string(nil), evidence.SourceRefs...), ContentHash: evidence.ContentHash})
	}
	sort.Slice(sources, func(i, j int) bool { return sources[i].EvidenceID < sources[j].EvidenceID })
	candidate, err := contextcompile.SealCandidateSnapshot(contextcompile.CandidateSnapshot{
		SnapshotID: decision.SnapshotID, Scope: input.Request.Scope,
		QueryHash: contextcompile.QueryCommitment(input.Request.Query), SnapshotKind: input.Request.Options.SnapshotKind,
		CreatedAt: req.RequestedAt, SourceManifestHash: decision.SourceManifestHash,
		PacketHash: decision.PacketCommitment, PolicyDigest: decision.PolicyDigest,
		ProvenanceID: provenanceID(req.Scope, req.Provenance), SyscallID: req.ID,
		JournalEventID: req.ID + ":journal_event", CommittedBy: forgekernel.AuthorityOwnerForgeK,
	})
	if err != nil {
		return GovernedContextBundle{}, err
	}
	bundle := GovernedContextBundle{
		Input: input, Decision: decision, Candidate: candidate, Sources: sources, Scope: req.Scope,
		Query: input.Request.Query, CreatedAt: req.RequestedAt, ProvenanceID: provenanceID(req.Scope, req.Provenance),
		SyscallID: req.ID, CorrelationID: req.CorrelationID, TraceID: req.TraceID,
		TransactionID: req.ID + ":transaction", JournalEventID: req.ID + ":journal_event",
		AuditOutboxID: req.ID + ":audit_outbox", AuthorizationFingerprint: readString(req.Metadata, "forgeKAuthorizationProof"),
		CommittedBy: forgekernel.AuthorityOwnerForgeK,
	}
	if err := validateGovernedContextBundle(bundle, store); err != nil {
		return GovernedContextBundle{}, err
	}
	return bundle, nil
}

func validateGovernedContextBundle(bundle GovernedContextBundle, store SemanticReadStore) error {
	if store == nil {
		return fmt.Errorf("context authority store unavailable")
	}
	if err := contextcompile.VerifyDecision(bundle.Input, bundle.Decision); err != nil {
		return err
	}
	if !exactUtilityScopeMatches(bundle.Scope, domain.ForgeScope{
		WorkspaceID: bundle.Input.Request.Scope.WorkspaceID, LaneID: bundle.Input.Request.Scope.LaneID,
		SelectedPaths: append([]string(nil), bundle.Input.Request.Scope.SelectedPaths...),
	}) || bundle.Query != bundle.Input.Request.Query || bundle.CreatedAt != bundle.Input.Request.RequestedAt {
		return fmt.Errorf("governed context bundle request binding mismatch")
	}
	if bundle.Decision.PacketID == "" || bundle.Candidate.SnapshotID != bundle.Decision.SnapshotID ||
		!reflect.DeepEqual(bundle.Candidate.Scope, bundle.Input.Request.Scope) || bundle.Candidate.SourceManifestHash != bundle.Decision.SourceManifestHash ||
		bundle.Candidate.PacketHash != bundle.Decision.PacketCommitment || bundle.Candidate.PolicyDigest != bundle.Decision.PolicyDigest {
		return fmt.Errorf("governed context candidate decision binding mismatch")
	}
	sealedCandidate, err := contextcompile.SealCandidateSnapshot(bundle.Candidate)
	if err != nil || !reflect.DeepEqual(sealedCandidate, bundle.Candidate) {
		return fmt.Errorf("governed context candidate commitment mismatch")
	}
	if bundle.SyscallID == "" || bundle.TransactionID != bundle.SyscallID+":transaction" ||
		bundle.JournalEventID != bundle.SyscallID+":journal_event" || bundle.AuditOutboxID != bundle.SyscallID+":audit_outbox" ||
		bundle.Candidate.SyscallID != bundle.SyscallID || bundle.Candidate.JournalEventID != bundle.JournalEventID ||
		bundle.ProvenanceID == "" || bundle.Candidate.ProvenanceID != bundle.ProvenanceID || bundle.AuthorizationFingerprint == "" ||
		bundle.CommittedBy != forgekernel.AuthorityOwnerForgeK || bundle.Candidate.CommittedBy != forgekernel.AuthorityOwnerForgeK {
		return fmt.Errorf("governed context commit authority binding mismatch")
	}
	wantIDs := append([]string(nil), bundle.Decision.SelectedEvidenceIDs...)
	sort.Strings(wantIDs)
	if len(bundle.Sources) != len(wantIDs) {
		return fmt.Errorf("governed context source count mismatch")
	}
	for index, source := range bundle.Sources {
		if source.EvidenceID != wantIDs[index] {
			return fmt.Errorf("governed context selected evidence mismatch")
		}
		evidence, ok := store.FindMemoryEvidence(source.EvidenceID, bundle.Scope)
		if !ok || store.HasMemoryEvidenceSupersession(source.EvidenceID) {
			return fmt.Errorf("governed context source %q is not current", source.EvidenceID)
		}
		expected := GovernedContextSource{
			EvidenceID: evidence.EvidenceID, ContentSummary: evidence.ContentSummary, RawRef: evidence.RawRef,
			SourceType: evidence.SourceType, SourceRefs: append([]string(nil), evidence.SourceRefs...), ContentHash: evidence.ContentHash,
		}
		if !reflect.DeepEqual(source, expected) {
			return fmt.Errorf("governed context source %q content binding mismatch", source.EvidenceID)
		}
	}
	return nil
}

func contextCompileScope(scope domain.ForgeScope) contextcompile.Scope {
	return contextcompile.Scope{WorkspaceID: scope.WorkspaceID, LaneID: scope.LaneID, SelectedPaths: append([]string(nil), scope.SelectedPaths...)}
}

func governedContextScopeKey(scope domain.ForgeScope) string {
	canonical := contextCompileScope(scope)
	canonical.SelectedPaths = normalizedSelectedPaths(canonical.SelectedPaths)
	raw, _ := json.Marshal(canonical)
	sum := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func listCurrentMemoryEvidenceState(state *memoryState, scope domain.ForgeScope, limit int) []MemoryEvidence {
	out := []MemoryEvidence{}
	for _, evidence := range state.memoryEvidence {
		if !exactUtilityScopeMatches(evidence.Scope, scope) {
			continue
		}
		superseded := false
		for _, edge := range state.memoryEvidenceSupersession {
			if edge.SupersededEvidenceID == evidence.EvidenceID {
				superseded = true
				break
			}
		}
		if superseded {
			continue
		}
		exhibit, exhibitOK := state.courtExhibits[evidence.CourtExhibitID]
		ruling, rulingOK := state.courtRulings[evidence.CourtRulingID]
		if !exhibitOK || !rulingOK || exhibit.CurrentRulingID != ruling.ID || exhibit.Status != "admitted" || ruling.Decision != "admitted" {
			continue
		}
		out = append(out, cloneIntegrityValue(evidence))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].EvidenceID < out[j].EvidenceID })
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out
}

func (s *InMemorySemanticStore) ListCurrentMemoryEvidence(scope domain.ForgeScope, limit int) ([]MemoryEvidence, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return listCurrentMemoryEvidenceState(&s.state, scope, limit), nil
}
func (s *TransactionalSemanticStore) ListCurrentMemoryEvidence(scope domain.ForgeScope, limit int) ([]MemoryEvidence, error) {
	return listCurrentMemoryEvidenceState(s.state, scope, limit), nil
}

func listGovernedCandidatesState(state *memoryState, scope domain.ForgeScope, query, kind string, limit int) []contextcompile.CandidateSnapshot {
	out := []contextcompile.CandidateSnapshot{}
	for _, bundle := range state.governedContextBundles {
		if exactUtilityScopeMatches(bundle.Scope, scope) && bundle.Query == strings.Join(strings.Fields(query), " ") && bundle.Candidate.SnapshotKind == kind {
			out = append(out, bundle.Candidate)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].SnapshotID < out[j].SnapshotID })
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return cloneIntegrityValue(out)
}

func (s *InMemorySemanticStore) ListGovernedContextCandidates(scope domain.ForgeScope, query, kind string, limit int) ([]contextcompile.CandidateSnapshot, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return listGovernedCandidatesState(&s.state, scope, query, kind, limit), nil
}
func (s *TransactionalSemanticStore) ListGovernedContextCandidates(scope domain.ForgeScope, query, kind string, limit int) ([]contextcompile.CandidateSnapshot, error) {
	return listGovernedCandidatesState(s.state, scope, query, kind, limit), nil
}

func (s *InMemorySemanticStore) FindGovernedContextHead(scope domain.ForgeScope) (contextcompile.PriorSnapshotHead, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	h, ok := s.state.governedContextHeads[governedContextScopeKey(scope)]
	return cloneIntegrityValue(h), ok, nil
}
func (s *TransactionalSemanticStore) FindGovernedContextHead(scope domain.ForgeScope) (contextcompile.PriorSnapshotHead, bool, error) {
	h, ok := s.state.governedContextHeads[governedContextScopeKey(scope)]
	return cloneIntegrityValue(h), ok, nil
}

func createGovernedContextBundleState(state *memoryState, bundle GovernedContextBundle) error {
	read := &TransactionalSemanticStore{state: state}
	if err := validateGovernedContextBundle(bundle, read); err != nil {
		return err
	}
	if _, exists := state.governedContextBundles[bundle.Decision.PacketID]; exists {
		return fmt.Errorf("governed context packet already exists")
	}
	key := governedContextScopeKey(bundle.Scope)
	current, present := state.governedContextHeads[key]
	expected := bundle.Input.PriorSnapshotHead
	if present != expected.Present || (present && current.HeadHash != expected.HeadHash) {
		return fmt.Errorf("governed context head changed")
	}
	revision := int64(1)
	if present {
		revision = current.Revision + 1
	}
	head, err := contextcompile.SealPriorSnapshotHead(contextcompile.PriorSnapshotHead{Present: true, Scope: contextCompileScope(bundle.Scope), SnapshotID: bundle.Candidate.SnapshotID, SnapshotHash: bundle.Candidate.SnapshotHash, Revision: revision, ProvenanceID: bundle.ProvenanceID, SyscallID: bundle.SyscallID, JournalEventID: bundle.JournalEventID, CommittedBy: bundle.CommittedBy})
	if err != nil {
		return err
	}
	state.governedContextBundles[bundle.Decision.PacketID] = cloneIntegrityValue(bundle)
	state.governedContextHeads[key] = head
	return nil
}

func (s *InMemorySemanticStore) CreateGovernedContextBundle(bundle GovernedContextBundle) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return createGovernedContextBundleState(&s.state, bundle)
}
func (s *TransactionalSemanticStore) CreateGovernedContextBundle(bundle GovernedContextBundle) error {
	return createGovernedContextBundleState(s.state, bundle)
}
