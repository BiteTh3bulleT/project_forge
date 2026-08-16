package contextcompile

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func TestCompileGoldenV1(t *testing.T) {
	in := validInput(t)
	decision, err := Compile(in)
	if err != nil {
		t.Fatal(err)
	}
	if decision.Selection.Mode != "restore" || decision.Selection.SnapshotID != "snapshot-a" {
		t.Fatalf("selection=%+v", decision.Selection)
	}
	if got, want := decision.SelectedEvidenceIDs, []string{"evidence-a", "evidence-b"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("evidence=%v want=%v", got, want)
	}
	const golden = "sha256:ecc7b2d922805765b5dde316702aea4a9a9edbbaaa34f6172e026e11634d0b4e"
	if decision.DecisionDigest != golden {
		t.Fatalf("decision digest=%s; update golden only with an explicit policy/contract version review", decision.DecisionDigest)
	}
}

func TestCompilePermutationInvariantAndStableTieBreak(t *testing.T) {
	first := validInput(t)
	first.Candidates[0].CreatedAt = first.Candidates[1].CreatedAt
	first.Candidates[0].SnapshotID, first.Candidates[1].SnapshotID = "snapshot-a", "snapshot-b"
	first.Candidates[0], _ = SealCandidateSnapshot(first.Candidates[0])
	first.Candidates[1], _ = SealCandidateSnapshot(first.Candidates[1])
	first.OutcomeHeads = nil
	first.Request.Hints.PreferredSnapshotIDs = nil
	first.PriorSnapshotHead = PriorSnapshotHead{}
	a, err := Compile(first)
	if err != nil {
		t.Fatal(err)
	}
	second := first
	second.Request.Scope.SelectedPaths = reverse(second.Request.Scope.SelectedPaths)
	second.Request.Hints.PreferredSnapshotIDs = reverse(second.Request.Hints.PreferredSnapshotIDs)
	second.SourceManifest.Sources = reverseSources(second.SourceManifest.Sources)
	second.Candidates = reverseCandidates(second.Candidates)
	b, err := Compile(second)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(a, b) {
		t.Fatalf("permutation changed decision\na=%+v\nb=%+v", a, b)
	}
	if a.Selection.SnapshotID != "snapshot-a" {
		t.Fatalf("stable lexical tie break selected %s", a.Selection.SnapshotID)
	}
}

func TestCompileFailsClosedMalformedAuthorityScopeDuplicatesAndHashes(t *testing.T) {
	tests := map[string]func(*Input){
		"cross scope source": func(in *Input) {
			in.SourceManifest.Sources[0].Scope.WorkspaceID = "other"
			in.SourceManifest, _ = SealSourceManifest(in.SourceManifest)
		},
		"duplicate source": func(in *Input) {
			in.SourceManifest.Sources = append(in.SourceManifest.Sources, in.SourceManifest.Sources[0])
			in.SourceManifest, _ = SealSourceManifest(in.SourceManifest)
		},
		"manifest tamper": func(in *Input) { in.SourceManifest.Sources[0].ContentHash = hashOf('f') },
		"not current": func(in *Input) {
			in.SourceManifest.Sources[0].Current = false
			in.SourceManifest, _ = SealSourceManifest(in.SourceManifest)
		},
		"not admitted": func(in *Input) {
			in.SourceManifest.Sources[0].Admitted = false
			in.SourceManifest, _ = SealSourceManifest(in.SourceManifest)
		},
		"bad provenance": func(in *Input) {
			in.SourceManifest.Sources[0].MaterializationProvenanceID = ""
			in.SourceManifest, _ = SealSourceManifest(in.SourceManifest)
		},
		"non K commit": func(in *Input) {
			in.SourceManifest.Sources[0].CommittedBy = "legacy"
			in.SourceManifest, _ = SealSourceManifest(in.SourceManifest)
		},
		"candidate future":      func(in *Input) { in.Candidates[0].CreatedAt = in.Request.RequestedAt + 1 },
		"candidate hash tamper": func(in *Input) { in.Candidates[0].PacketHash = hashOf('e') },
		"duplicate candidate": func(in *Input) {
			in.Candidates = append(in.Candidates, in.Candidates[0])
			in.Request.Budget.MaxSnapshots++
		},
		"cross scope outcome":  func(in *Input) { in.OutcomeHeads[0].Scope.LaneID = "other" },
		"duplicate path":       func(in *Input) { in.Request.Scope.SelectedPaths = []string{"/a", "/a"} },
		"policy weight tamper": func(in *Input) { in.Policy.Weights.QueryMatch++ },
		"required absent":      func(in *Input) { in.Request.Hints.RequiredEvidenceIDs = []string{"evidence-missing"} },
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			in := validInput(t)
			mutate(&in)
			_, err := Compile(in)
			if !errors.Is(err, ErrInvalidInput) {
				t.Fatalf("err=%v", err)
			}
		})
	}
}

func TestCompileBoundsAndFreshCompile(t *testing.T) {
	in := validInput(t)
	in.Request.Options.AllowRestore = false
	d, err := Compile(in)
	if err != nil {
		t.Fatal(err)
	}
	if d.Selection.Mode != "fresh_compile" || d.Selection.SnapshotID != "" {
		t.Fatalf("selection=%+v", d.Selection)
	}
	in = validInput(t)
	in.Request.Query = strings.Repeat("x", V1Policy().Limits.MaxQueryBytes+1)
	if _, err := Compile(in); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("query bound err=%v", err)
	}
	in = validInput(t)
	in.Request.Budget.MaxSources = V1Policy().Limits.MaxSources + 1
	if _, err := Compile(in); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("source bound err=%v", err)
	}
}

func TestOutcomeProjectionUsesBoundedFixedPointIntegers(t *testing.T) {
	in := validInput(t)
	in.OutcomeHeads[0].HelpfulCount = 100
	in.OutcomeHeads[0], _ = SealOutcomeProjectionHead(in.OutcomeHeads[0])
	d, err := Compile(in)
	if err != nil {
		t.Fatal(err)
	}
	var found bool
	for _, row := range d.RestoreScoreTable {
		if row.SnapshotID == "snapshot-a" {
			found = true
			if row.Breakdown.OutcomeAdjustment != V1Policy().Weights.OutcomeAdjustmentCap {
				t.Fatalf("adjustment=%d", row.Breakdown.OutcomeAdjustment)
			}
		}
	}
	if !found {
		t.Fatal("missing score")
	}
	in = validInput(t)
	in.OutcomeHeads[0].HelpfulCount = V1Policy().Limits.MaxOutcomeEventsPerHead + 1
	in.OutcomeHeads[0], _ = SealOutcomeProjectionHead(in.OutcomeHeads[0])
	if _, err := Compile(in); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("outcome bound err=%v", err)
	}
}

func TestDecisionBindsCandidateAndPriorHeadCommitmentsBeyondScore(t *testing.T) {
	base := validInput(t)
	first, err := Compile(base)
	if err != nil {
		t.Fatal(err)
	}
	candidateChanged := validInput(t)
	candidateChanged.Candidates[1].PacketHash = hashOf('d')
	candidateChanged.Candidates[1], _ = SealCandidateSnapshot(candidateChanged.Candidates[1])
	second, err := Compile(candidateChanged)
	if err != nil {
		t.Fatal(err)
	}
	if scoreTotals(first.RestoreScoreTable) != scoreTotals(second.RestoreScoreTable) || first.CandidateSetCommitment == second.CandidateSetCommitment || first.DecisionDigest == second.DecisionDigest {
		t.Fatalf("candidate commitment was not independently bound: first=%+v second=%+v", first, second)
	}
	priorChanged := validInput(t)
	priorChanged.PriorSnapshotHead.SyscallID = "sys-prior-reloaded"
	priorChanged.PriorSnapshotHead, _ = SealPriorSnapshotHead(priorChanged.PriorSnapshotHead)
	third, err := Compile(priorChanged)
	if err != nil {
		t.Fatal(err)
	}
	if first.PriorSnapshotHeadCommitment == third.PriorSnapshotHeadCommitment || first.DecisionDigest == third.DecisionDigest {
		t.Fatal("full prior snapshot head commitment was not decision-bound")
	}
}

func FuzzCompileNeverPanicsOrAcceptsUnsealedManifest(f *testing.F) {
	f.Add("alpha query", "ws-main", "lane-main", int64(1760000000000))
	f.Add("\x00", "other", "lane", int64(-1))
	f.Fuzz(func(t *testing.T, query, workspace, lane string, requestedAt int64) {
		in := validInput(t)
		in.Request.Query = query
		in.Request.Scope.WorkspaceID = workspace
		in.Request.Scope.LaneID = lane
		in.Request.RequestedAt = requestedAt
		in.SourceManifest.ManifestHash = hashOf('0')
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("panic: %v", r)
			}
		}()
		d, err := Compile(in)
		if err == nil && d.SourceManifestHash == hashOf('0') {
			t.Fatal("accepted unsealed source manifest")
		}
	})
}

func validInput(t testing.TB) Input {
	t.Helper()
	p := V1Policy()
	scope := Scope{WorkspaceID: "ws-main", LaneID: "lane-main", SelectedPaths: []string{"/b", "/a"}}
	sources := []SourceCommitment{source("evidence-b", 2, scope, 'b'), source("evidence-a", 1, scope, 'a')}
	manifest, err := SealSourceManifest(SourceManifest{Version: SourceManifestVersion, Scope: scope, Sources: sources})
	if err != nil {
		t.Fatal(err)
	}
	qhash, _ := hash("alpha query")
	candidates := []CandidateSnapshot{
		{SnapshotID: "snapshot-b", Scope: scope, QueryHash: qhash, SnapshotKind: "restore", CreatedAt: 1759999990000, SourceManifestHash: manifest.ManifestHash, PacketHash: hashOf('3'), PolicyDigest: p.Digest, ProvenanceID: "prov-b", SyscallID: "sys-b", JournalEventID: "journal-b", CommittedBy: CommittedByForgeK},
		{SnapshotID: "snapshot-a", Scope: scope, QueryHash: qhash, SnapshotKind: "restore", CreatedAt: 1759999995000, SourceManifestHash: manifest.ManifestHash, PacketHash: hashOf('5'), PolicyDigest: p.Digest, ProvenanceID: "prov-a", SyscallID: "sys-a", JournalEventID: "journal-a", CommittedBy: CommittedByForgeK},
	}
	for i := range candidates {
		candidates[i], err = SealCandidateSnapshot(candidates[i])
		if err != nil {
			t.Fatal(err)
		}
	}
	outcomes := []OutcomeProjectionHead{{SnapshotID: "snapshot-a", Scope: scope, Revision: 1, HelpfulCount: 2, HarmfulCount: 0, EventSetHash: hashOf('7'), ProvenanceID: "prov-outcome", SyscallID: "sys-outcome", JournalEventID: "journal-outcome", CommittedBy: CommittedByForgeK}}
	outcomes[0], err = SealOutcomeProjectionHead(outcomes[0])
	if err != nil {
		t.Fatal(err)
	}
	prior := PriorSnapshotHead{Present: true, Scope: scope, SnapshotID: "snapshot-b", SnapshotHash: candidates[0].SnapshotHash, Revision: 1, ProvenanceID: "prov-prior", SyscallID: "sys-prior", JournalEventID: "journal-prior", CommittedBy: CommittedByForgeK}
	prior, err = SealPriorSnapshotHead(prior)
	if err != nil {
		t.Fatal(err)
	}
	return Input{Request: Request{Scope: scope, Query: "  alpha   query ", Budget: Budget{MaxSources: 2, MaxSnapshots: 8, MaxTokens: 4096, MaxBytes: 65536}, Options: Options{SnapshotKind: "restore", AllowRestore: true, PersistSnapshot: true, RenderCard: true}, Hints: Hints{PreferredSnapshotIDs: []string{"snapshot-a"}, RequiredEvidenceIDs: []string{"evidence-a"}}, RequestedAt: 1760000000000, PolicyVersion: PolicyVersionV1}, SourceManifest: manifest, Candidates: candidates, OutcomeHeads: outcomes, PriorSnapshotHead: prior, Policy: p}
}

func source(id string, row int64, scope Scope, ch byte) SourceCommitment {
	return SourceCommitment{MemoryEvidenceRowID: row, EvidenceID: id, RootEvidenceID: "root", Revision: int(row), CourtCaseID: "case", CourtExhibitID: "exhibit-" + id, CourtRulingID: "ruling-" + id, AdmissionSyscallID: "admit-" + id, SourceObjectKind: "court_exhibit", SourceObjectID: "exhibit-" + id, SourceObjectVersion: "ruling-" + id, SourceObjectHash: hashOf(ch), Scope: scope, ContentHash: hashOf(ch), SourceProvenanceID: "source-prov-" + id, MaterializationProvenanceID: "materialize-prov-" + id, SyscallID: "materialize-" + id, TransactionID: "transaction-" + id, JournalEventID: "journal-" + id, AuditOutboxID: "outbox-" + id, AuthorizationFingerprint: hashOf(ch), CommittedBy: CommittedByForgeK, Current: true, Admitted: true}
}
func hashOf(ch byte) string { return "sha256:" + strings.Repeat(string(ch), 64) }
func scoreTotals(rows []RestoreScore) string {
	var b strings.Builder
	for _, row := range rows {
		b.WriteString(row.SnapshotID)
		b.WriteString(":")
		b.WriteString(fmt.Sprint(row.Total))
		b.WriteString(";")
	}
	return b.String()
}
func reverse(v []string) []string {
	out := append([]string(nil), v...)
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out
}
func reverseSources(v []SourceCommitment) []SourceCommitment {
	out := append([]SourceCommitment(nil), v...)
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out
}
func reverseCandidates(v []CandidateSnapshot) []CandidateSnapshot {
	out := append([]CandidateSnapshot(nil), v...)
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out
}
