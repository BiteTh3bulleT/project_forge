package memory

import (
	"context"
	"encoding/json"
	"testing"

	"forge/projectforge/services/core/internal/sqlutil"
)

func TestSaveAndLoadRetrievalResultVSASignal(t *testing.T) {
	ctx := context.Background()
	svc, cleanup := newMemoryServiceForTest(t)
	defer cleanup()

	runID := seedMemoryRetrievalRun(t, svc, 202)
	resultID := seedMemoryRetrievalResult(t, svc, runID, 101)
	obs := createTestObservation(t, svc, "/tmp/project/signal.go", "signal summary", "signal raw")
	obsID := obs.ID
	explain := json.RawMessage(`{"mode":"shadow","reason":"test"}`)
	signal := RetrievalResultVSASignal{
		RetrievalResultID: resultID,
		RetrievalRunID:    runID,
		ObservationID:     &obsID,
		Mode:              "shadow",
		AssociativeScore:  0.11,
		RoleMatchScore:    0.22,
		RelationalScore:   0.33,
		FeedbackScore:     0.44,
		AdditiveScore:     0.55,
		AppliedScore:      0,
		Explain:           explain,
	}

	if err := svc.SaveRetrievalResultVSASignal(ctx, signal); err != nil {
		t.Fatalf("save signal: %v", err)
	}

	got, err := svc.RetrievalResultVSASignal(ctx, resultID)
	if err != nil {
		t.Fatalf("load signal: %v", err)
	}
	if got == nil {
		t.Fatalf("expected saved signal")
	}
	if got.RetrievalRunID != runID || got.ObservationID == nil || *got.ObservationID != obsID {
		t.Fatalf("unexpected loaded signal identity: %+v", got)
	}
	if got.Mode != "shadow" || got.AssociativeScore != 0.11 || got.RoleMatchScore != 0.22 || got.RelationalScore != 0.33 || got.FeedbackScore != 0.44 || got.AdditiveScore != 0.55 || got.AppliedScore != 0 {
		t.Fatalf("unexpected loaded signal scores: %+v", got)
	}
	if string(got.Explain) != string(explain) {
		t.Fatalf("explain = %s, want %s", got.Explain, explain)
	}

	runSignals, err := svc.RetrievalRunVSASignals(ctx, runID)
	if err != nil {
		t.Fatalf("load run signals: %v", err)
	}
	if len(runSignals) != 1 || runSignals[0].RetrievalResultID != resultID {
		t.Fatalf("unexpected run signals: %+v", runSignals)
	}

	updated := signal
	updated.Mode = "active"
	updated.AppliedScore = 0.12
	updated.Explain = json.RawMessage(`{"mode":"active","reason":"updated"}`)
	if err := svc.SaveRetrievalResultVSASignal(ctx, updated); err != nil {
		t.Fatalf("update signal: %v", err)
	}
	got, err = svc.RetrievalResultVSASignal(ctx, resultID)
	if err != nil {
		t.Fatalf("load updated signal: %v", err)
	}
	if got.Mode != "active" || got.AppliedScore != 0.12 || string(got.Explain) != string(updated.Explain) {
		t.Fatalf("signal did not upsert updated fields: %+v", got)
	}
}

func seedMemoryRetrievalRun(t *testing.T, svc *Service, id int64) int64 {
	t.Helper()
	_, err := svc.db.Exec(`
INSERT INTO retrieval_runs(id, created_at, query, mode, weighting_json, notes)
VALUES(?,?,?,?,?,?)`,
		id, int64(1234), "vsa query", "keyword", `{"mode":"keyword"}`, "memory signal test",
	)
	if err != nil {
		t.Fatalf("insert retrieval run: %v", err)
	}
	return id
}

func seedMemoryRetrievalResult(t *testing.T, svc *Service, runID, id int64) int64 {
	t.Helper()
	_, err := svc.db.Exec(`
INSERT INTO retrieval_results(
  id, retrieval_run_id, chunk_id, file_id, abs_path, rel_path, rank_index,
  keyword_score, semantic_score, hybrid_score, snippet, selected_for_packet, usefulness_label, usefulness_note
) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		id, runID, nil, nil, "/tmp/project/a.go", "a.go", 0,
		1.0, 0.0, 1.0, "snippet", 0, "unknown", "",
	)
	if err != nil {
		t.Fatalf("insert retrieval result: %v", err)
	}
	return id
}

func TestComputeVSAQuerySignalsFailsClosedWithoutScope(t *testing.T) {
	ctx := context.Background()
	svc, cleanup := newMemoryServiceForTest(t)
	defer cleanup()
	unscopedEmpty := false
	{
		signals, err := svc.ComputeVSAQuerySignals(ctx, VSAQuerySignalsRequest{
			Query:      "alpha subsystem",
			Candidates: []VSAQueryCandidate{{ChunkID: 10, AbsPath: "/tmp/project/a.go"}},
		})
		if err != nil {
			t.Fatal(err)
		}
		if len(signals) != 0 {
			t.Fatalf("unscoped request received VSA influence: %+v", signals)
		}
		unscopedEmpty = len(signals) == 0
	}
	if unscopedEmpty {
		return
	}

	if _, err := svc.db.ExecContext(ctx, `
INSERT INTO settings(key, value) VALUES
  ('retrieval_vsa_mode','active'),
  ('retrieval_vsa_weight_associative','0.25'),
  ('retrieval_vsa_weight_role_match','0.25'),
  ('retrieval_vsa_weight_relational','0.25'),
  ('retrieval_vsa_weight_feedback','0.25'),
  ('retrieval_vsa_max_additive','0.5')
ON CONFLICT(key) DO UPDATE SET value=excluded.value`); err != nil {
		t.Fatalf("set vsa settings: %v", err)
	}

	obsA := createTestObservation(t, svc, "/tmp/project/a.go", "alpha subsystem", "alpha role filler content")
	obsB := createTestObservation(t, svc, "/tmp/project/b.go", "beta subsystem", "beta related content")
	if err := svc.AddLink(ctx, obsA.ID, obsB.ID, "related", "test relation"); err != nil {
		t.Fatalf("add relation: %v", err)
	}
	if err := svc.MarkObservationUsefulness(ctx, MarkUsefulnessRequest{
		ObservationID: obsA.ID,
		Signal:        "useful",
		Weight:        1,
		Note:          "test signal",
	}); err != nil {
		t.Fatalf("mark observation useful: %v", err)
	}

	signals, err := svc.ComputeVSAQuerySignals(ctx, VSAQuerySignalsRequest{
		Query: "alpha subsystem",
		Candidates: []VSAQueryCandidate{
			{ChunkID: 10, AbsPath: "/tmp/project/a.go", RelPath: "a.go"},
			{ChunkID: 20, AbsPath: "/tmp/project/missing.go", RelPath: "missing.go"},
		},
	})
	if err != nil {
		t.Fatalf("compute signals: %v", err)
	}
	if len(signals) != 2 {
		t.Fatalf("signals length = %d, want 2: %+v", len(signals), signals)
	}

	known := signals[10]
	if known.ObservationID == nil || *known.ObservationID != obsA.ID {
		t.Fatalf("known observation id = %+v, want %d", known.ObservationID, obsA.ID)
	}
	if known.Mode != "active" {
		t.Fatalf("known mode = %q, want active", known.Mode)
	}
	if known.AssociativeScore == 0 && known.RoleMatchScore == 0 && known.FeedbackScore == 0 {
		t.Fatalf("expected at least one positive known signal score: %+v", known)
	}
	if known.AdditiveScore == 0 || known.AppliedScore == 0 {
		t.Fatalf("expected active additive/applied scores: %+v", known)
	}
	var explain map[string]any
	if err := json.Unmarshal(known.Explain, &explain); err != nil {
		t.Fatalf("decode explain: %v", err)
	}
	if explain["observationId"].(float64) != float64(obsA.ID) || explain["mode"] != "active" {
		t.Fatalf("unexpected explain payload: %#v", explain)
	}

	missing := signals[20]
	if missing.ObservationID != nil {
		t.Fatalf("missing candidate observation id = %+v, want nil", missing.ObservationID)
	}
	if missing.Mode != "active" || missing.AdditiveScore != 0 || missing.AppliedScore != 0 {
		t.Fatalf("missing candidate should carry only mode and chunk id: %+v", missing)
	}
}

func TestComputeVSAQuerySignalsRequiresMatchingScopedActiveManifest(t *testing.T) {
	ctx := context.Background()
	svc, cleanup := newMemoryServiceForTest(t)
	defer cleanup()
	if _, err := svc.db.Exec(`UPDATE settings SET value='active' WHERE key='retrieval_vsa_mode'`); err != nil {
		t.Fatal(err)
	}
	obs := createTestObservation(t, svc, "/tmp/project/scoped.go", "alpha subsystem", "alpha body")
	if _, err := svc.db.Exec(`UPDATE memory_observations SET workspace_id='workspace-a',lane_id='lane-a' WHERE id=?`, obs.ID); err != nil {
		t.Fatal(err)
	}
	manifestHash := "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	if _, err := svc.db.Exec(`
INSERT INTO memory_vsa_projection_manifests(
 manifest_hash,workspace_id,lane_id,source_set_hash,link_set_hash,algorithm_name,algorithm_version,
 dimensions,seed,source_count,link_count,manifest_json,syscall_id,correlation_id,trace_id,created_at
) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, manifestHash, "workspace-a", "lane-a", "sha256:sources", "sha256:links",
		"forge.vsa.observation_projection", "1", 128, 17, 1, 0, `{"manifestHash":"`+manifestHash+`"}`, "syscall-1", "corr-1", "trace-1", 100); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.db.Exec(`INSERT INTO memory_vsa_projection_heads(workspace_id,lane_id,manifest_hash,syscall_id,correlation_id,trace_id,updated_at) VALUES('workspace-a','lane-a',?,'syscall-1','corr-1','trace-1',100)`, manifestHash); err != nil {
		t.Fatal(err)
	}
	vector := NewVSAEngine(128, 17).EncodeText("alpha subsystem")
	vectorJSON, _ := json.Marshal(vector)
	if _, err := svc.db.Exec(`
INSERT INTO memory_vsa_pointers(workspace_id,lane_id,manifest_hash,observation_id,dims,pointer_json,norm,source_fingerprint,support_count,noise_count,created_at,updated_at)
VALUES('workspace-a','lane-a',?,?,?,?,?,?,1,0,100,100)`, manifestHash, obs.ID, 128, string(vectorJSON), vectorNorm(vector), "sha256:source"); err != nil {
		t.Fatal(err)
	}

	signals, err := svc.ComputeVSAQuerySignals(ctx, VSAQuerySignalsRequest{
		WorkspaceID: "workspace-a", LaneID: "lane-a", Query: "alpha subsystem",
		Candidates: []VSAQueryCandidate{{ChunkID: 10, AbsPath: "/tmp/project/scoped.go"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(signals) != 1 || signals[10].ObservationID == nil || signals[10].AppliedScore == 0 {
		t.Fatalf("scoped active signals = %+v", signals)
	}
	signals, err = svc.ComputeVSAQuerySignals(ctx, VSAQuerySignalsRequest{
		WorkspaceID: "workspace-a", LaneID: "other-lane", Query: "alpha subsystem",
		Candidates: []VSAQueryCandidate{{ChunkID: 10, AbsPath: "/tmp/project/scoped.go"}},
	})
	if err != nil || len(signals) != 0 {
		t.Fatalf("mismatched head influenced scoring: signals=%+v err=%v", signals, err)
	}
}

func TestVSASignalScoringHelpers(t *testing.T) {
	t.Parallel()

	role := scoreRoleMatch(tokenSet("alpha owner"), []VSARoleBinding{
		{Role: "owner", Filler: "alpha", Weight: 1, SupportCount: 3},
		{Role: "status", Filler: "beta", Weight: 1, SupportCount: 0, NoiseCount: 3},
	})
	if role <= 0 || role > 1 {
		t.Fatalf("role score = %v, want within (0,1]", role)
	}

	relational := scoreRelational(1, []VSAAssociation{
		{FromObservationID: 1, ToObservationID: 2, Strength: 0.8, SupportCount: 3},
		{FromObservationID: 1, ToObservationID: 3, Strength: 0.9, SupportCount: 3},
	}, map[int64]struct{}{2: {}})
	if relational <= 0 || relational > 1 {
		t.Fatalf("relational score = %v, want within (0,1]", relational)
	}

	if got := normalizeUsefulness(3, 4, 0); got <= 0 || got > 1 {
		t.Fatalf("positive usefulness = %v, want within (0,1]", got)
	}
	if got := normalizeUsefulness(-3, 0, 4); got >= 0 || got < -1 {
		t.Fatalf("negative usefulness = %v, want within [-1,0)", got)
	}
	if got := sqlutil.Placeholders(3); got != "?,?,?" {
		t.Fatalf("Placeholders(3) = %q", got)
	}
	if got := toAny([]string{"a", "b"}); len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Fatalf("toAny returned %#v", got)
	}

	ranked := rankTopSignals([]RetrievalResultVSASignal{
		{ChunkID: 1, AppliedScore: 0.1},
		{ChunkID: 2, AppliedScore: 0.9},
		{ChunkID: 3, AppliedScore: 0.5},
	}, 2)
	if len(ranked) != 2 || ranked[0].ChunkID != 2 || ranked[1].ChunkID != 3 {
		t.Fatalf("ranked signals = %+v", ranked)
	}
}
