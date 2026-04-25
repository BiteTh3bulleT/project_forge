package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"forge/projectforge/services/core/internal/aios/controllane"
	"forge/projectforge/services/core/internal/aios/domain"
)

func TestHandleContextSnapshotInspectorListAndGet(t *testing.T) {
	srv, st := newBackupAuditHarness(t)

	packet := domain.ContextPacket{
		ID:    "snap-inspector-1",
		Query: "restore compile context",
		Scope: domain.ForgeScope{
			WorkspaceID:   "workspace-inspector",
			LaneID:        "control.semantic",
			SelectedPaths: []string{"docs", "services/core"},
		},
		CompileOptions: &domain.ContextCompileOptions{SnapshotKind: "restore"},
		RestoreSnapshot: &domain.ContextRestoreSnapshot{
			SnapshotID:   "snap-inspector-1",
			SnapshotKind: "restore",
			Evidence: map[string]any{
				"header": map[string]any{"title": "Restore snapshot"},
				"graph":  map[string]any{"nodes": []string{"root", "packet"}},
				"delta":  map[string]any{"added": []string{"packet"}},
			},
			Metadata: map[string]any{
				"fingerprint":               "fp-inspector-1",
				"parent_snapshot_id":        "snap-parent-0",
				"rendered_card_artifact_id": "artifact-render-1",
				"restore_scores_json":       map[string]any{"coverage": 0.93, "freshness": 0.81},
				"resume_hints_json":         map[string]any{"next": "review packet"},
			},
		},
		ActiveState: []domain.StateItem{
			{ID: "state-1"},
		},
		OpenLoops: []domain.OpenLoop{
			{ID: "loop-1"},
		},
		Notes: []domain.MemoryNote{
			{ID: "note-1"},
		},
		LinkedNotes: []domain.SemanticLink{
			{ID: "link-1"},
		},
		Models: []domain.AdaptivePolicyModel{
			{ID: "model-1"},
		},
		Artifacts: []domain.ArtifactRef{
			{ID: "artifact-1"},
		},
		RawEvents: []domain.JournalEvent{
			{ID: "event-1"},
		},
		Budget: domain.ContextBudget{
			MaxTokens: 2048,
			MaxEvents: 12,
			MaxNotes:  24,
		},
		InclusionReasons: map[string]string{
			"note-1": "recent blocking note",
		},
		CreatedAt: time.Now().UnixMilli(),
	}

	semanticStore := controllane.NewSQLiteSemanticStore(st.DB)
	if err := semanticStore.CreateSnapshot(
		context.Background(),
		packet,
		"syscall-inspector-1",
		"corr-inspector-1",
		"trace-inspector-1",
		map[string]any{"source": "operator_inspector_test"},
	); err != nil {
		t.Fatalf("create snapshot: %v", err)
	}

	listReq := httptest.NewRequest(
		http.MethodGet,
		"/api/context-inspector/snapshots?workspaceId=workspace-inspector&snapshotKind=restore",
		nil,
	)
	listRR := httptest.NewRecorder()
	srv.handleContextSnapshotList(listRR, listReq)
	if listRR.Code != http.StatusOK {
		t.Fatalf("list status=%d body=%s", listRR.Code, strings.TrimSpace(listRR.Body.String()))
	}

	var listResp struct {
		Snapshots []struct {
			ID               string `json:"id"`
			CorrelationID    string `json:"correlationId"`
			TraceID          string `json:"traceId"`
			SnapshotKind     string `json:"snapshotKind"`
			ParentSnapshotID string `json:"parentSnapshotId"`
			HasRestoreTrace  bool   `json:"hasRestoreTrace"`
			Counts           struct {
				State     int `json:"state"`
				OpenLoops int `json:"openLoops"`
				Notes     int `json:"notes"`
				Artifacts int `json:"artifacts"`
			} `json:"counts"`
			HasGraph         bool `json:"hasGraph"`
			HasRestoreScores bool `json:"hasRestoreScores"`
		} `json:"snapshots"`
	}
	if err := json.Unmarshal(listRR.Body.Bytes(), &listResp); err != nil {
		t.Fatalf("decode list response: %v body=%s", err, listRR.Body.String())
	}
	if len(listResp.Snapshots) != 1 {
		t.Fatalf("expected 1 snapshot, got %d", len(listResp.Snapshots))
	}
	if listResp.Snapshots[0].ID != packet.ID {
		t.Fatalf("list snapshot id=%q want %q", listResp.Snapshots[0].ID, packet.ID)
	}
	if listResp.Snapshots[0].CorrelationID != "corr-inspector-1" {
		t.Fatalf("list correlation=%q", listResp.Snapshots[0].CorrelationID)
	}
	if listResp.Snapshots[0].TraceID != "trace-inspector-1" {
		t.Fatalf("list trace=%q", listResp.Snapshots[0].TraceID)
	}
	if listResp.Snapshots[0].SnapshotKind != "restore" {
		t.Fatalf("list snapshot kind=%q", listResp.Snapshots[0].SnapshotKind)
	}
	if listResp.Snapshots[0].ParentSnapshotID != "snap-parent-0" {
		t.Fatalf("list parent snapshot=%q", listResp.Snapshots[0].ParentSnapshotID)
	}
	if listResp.Snapshots[0].Counts.State != 1 || listResp.Snapshots[0].Counts.OpenLoops != 1 || listResp.Snapshots[0].Counts.Notes != 1 || listResp.Snapshots[0].Counts.Artifacts != 1 {
		t.Fatalf("unexpected counts: %+v", listResp.Snapshots[0].Counts)
	}
	if !listResp.Snapshots[0].HasGraph || !listResp.Snapshots[0].HasRestoreScores {
		t.Fatalf("expected graph and restore scores to be present: %+v", listResp.Snapshots[0])
	}
	if listResp.Snapshots[0].HasRestoreTrace {
		t.Fatalf("expected restore trace visibility to remain false when trace is absent: %+v", listResp.Snapshots[0])
	}

	getReq := withRouteParam(httptest.NewRequest(http.MethodGet, "/api/context-inspector/snapshots/"+packet.ID, nil), "id", packet.ID)
	getRR := httptest.NewRecorder()
	srv.handleContextSnapshotGet(getRR, getReq)
	if getRR.Code != http.StatusOK {
		t.Fatalf("get status=%d body=%s", getRR.Code, strings.TrimSpace(getRR.Body.String()))
	}

	var getResp struct {
		Snapshot struct {
			Summary struct {
				ID               string         `json:"id"`
				CorrelationID    string         `json:"correlationId"`
				RenderArtifactID string         `json:"renderArtifactRefId"`
				HasRestoreTrace  bool           `json:"hasRestoreTrace"`
				RestoreTrace     map[string]any `json:"restoreTrace"`
			} `json:"summary"`
			IncludedStateIDs    []string            `json:"includedStateIds"`
			IncludedArtifactIDs []string            `json:"includedArtifactIds"`
			RestoreScores       map[string]float64  `json:"restoreScores"`
			ResumeHints         map[string]string   `json:"resumeHints"`
			Metadata            map[string]any      `json:"metadata"`
			Budget              map[string]int      `json:"budget"`
			Header              map[string]any      `json:"header"`
			Graph               map[string][]string `json:"graph"`
			Delta               map[string][]string `json:"delta"`
		} `json:"snapshot"`
	}
	if err := json.Unmarshal(getRR.Body.Bytes(), &getResp); err != nil {
		t.Fatalf("decode get response: %v body=%s", err, getRR.Body.String())
	}
	if getResp.Snapshot.Summary.ID != packet.ID {
		t.Fatalf("detail snapshot id=%q want %q", getResp.Snapshot.Summary.ID, packet.ID)
	}
	if getResp.Snapshot.Summary.CorrelationID != "corr-inspector-1" {
		t.Fatalf("detail correlation=%q", getResp.Snapshot.Summary.CorrelationID)
	}
	if getResp.Snapshot.Summary.RenderArtifactID != "artifact-render-1" {
		t.Fatalf("detail render artifact=%q", getResp.Snapshot.Summary.RenderArtifactID)
	}
	if getResp.Snapshot.Summary.HasRestoreTrace {
		t.Fatalf("restore trace should be absent in this fixture")
	}
	if len(getResp.Snapshot.Summary.RestoreTrace) != 0 {
		t.Fatalf("restore trace payload should default to empty object when absent: %+v", getResp.Snapshot.Summary.RestoreTrace)
	}
	if len(getResp.Snapshot.IncludedStateIDs) != 1 || getResp.Snapshot.IncludedStateIDs[0] != "state-1" {
		t.Fatalf("unexpected included state ids: %+v", getResp.Snapshot.IncludedStateIDs)
	}
	if len(getResp.Snapshot.IncludedArtifactIDs) != 1 || getResp.Snapshot.IncludedArtifactIDs[0] != "artifact-1" {
		t.Fatalf("unexpected included artifact ids: %+v", getResp.Snapshot.IncludedArtifactIDs)
	}
	if getResp.Snapshot.RestoreScores["coverage"] != 0.93 {
		t.Fatalf("restore coverage=%v", getResp.Snapshot.RestoreScores["coverage"])
	}
	if getResp.Snapshot.ResumeHints["next"] != "review packet" {
		t.Fatalf("resume hint next=%q", getResp.Snapshot.ResumeHints["next"])
	}
	if getResp.Snapshot.Budget["maxTokens"] != 2048 {
		t.Fatalf("budget maxTokens=%d", getResp.Snapshot.Budget["maxTokens"])
	}
	if getResp.Snapshot.Header["title"] != "Restore snapshot" {
		t.Fatalf("header title=%v", getResp.Snapshot.Header["title"])
	}
	if len(getResp.Snapshot.Graph["nodes"]) != 2 || getResp.Snapshot.Delta["added"][0] != "packet" {
		t.Fatalf("unexpected graph/delta: graph=%+v delta=%+v", getResp.Snapshot.Graph, getResp.Snapshot.Delta)
	}
	if getResp.Snapshot.Metadata["source"] != "operator_inspector_test" {
		t.Fatalf("metadata source=%v", getResp.Snapshot.Metadata["source"])
	}
}

func TestHandleContextSnapshotInspectorListAndGetShowsRestoreTraceWhenAvailable(t *testing.T) {
	srv, st := newBackupAuditHarness(t)

	packet := domain.ContextPacket{
		ID:    "snap-inspector-trace-1",
		Query: "restore compile context",
		Scope: domain.ForgeScope{
			WorkspaceID:   "workspace-inspector-trace",
			LaneID:        "control.semantic",
			SelectedPaths: []string{"docs", "services/core"},
		},
		CompileOptions: &domain.ContextCompileOptions{SnapshotKind: "restore"},
		RestoreSnapshot: &domain.ContextRestoreSnapshot{
			SnapshotID:   "snap-inspector-trace-1",
			SnapshotKind: "restore",
			Evidence: map[string]any{
				"header": map[string]any{"title": "Restore snapshot"},
				"graph":  map[string]any{"nodes": []string{"root", "packet"}},
				"delta":  map[string]any{"added": []string{"packet"}},
			},
			Metadata: map[string]any{
				"restore_trace_json": map[string]any{
					"decision": "selected",
					"winner": map[string]any{
						"snapshot_id": "snap-inspector-trace-1",
					},
				},
				"restore_scores_json": map[string]any{"coverage": 0.81},
				"resume_hints_json":   map[string]any{"next_action": "resume"},
				"restore_reason_json": map[string]any{"mode": "compile_context_restore_selection"},
			},
		},
		ActiveState: []domain.StateItem{
			{ID: "state-trace-1"},
		},
		OpenLoops: []domain.OpenLoop{
			{ID: "loop-trace-1"},
		},
		Notes: []domain.MemoryNote{
			{ID: "note-trace-1"},
		},
		LinkedNotes: []domain.SemanticLink{
			{ID: "link-trace-1"},
		},
		Models: []domain.AdaptivePolicyModel{
			{ID: "model-trace-1"},
		},
		Artifacts: []domain.ArtifactRef{
			{ID: "artifact-trace-1"},
		},
		RawEvents: []domain.JournalEvent{
			{ID: "event-trace-1"},
		},
		Budget: domain.ContextBudget{
			MaxTokens: 2048,
			MaxEvents: 12,
			MaxNotes:  24,
		},
		InclusionReasons: map[string]string{
			"note-trace-1": "restore trace test",
		},
		CreatedAt: time.Now().UnixMilli(),
	}

	semanticStore := controllane.NewSQLiteSemanticStore(st.DB)
	if err := semanticStore.CreateSnapshot(
		context.Background(),
		packet,
		"syscall-inspector-trace-1",
		"corr-inspector-trace-1",
		"trace-inspector-trace-1",
		map[string]any{"source": "operator_inspector_trace_test"},
	); err != nil {
		t.Fatalf("create snapshot: %v", err)
	}

	listReq := httptest.NewRequest(
		http.MethodGet,
		"/api/context-inspector/snapshots?workspaceId=workspace-inspector-trace&snapshotKind=restore",
		nil,
	)
	listRR := httptest.NewRecorder()
	srv.handleContextSnapshotList(listRR, listReq)
	if listRR.Code != http.StatusOK {
		t.Fatalf("list status=%d body=%s", listRR.Code, strings.TrimSpace(listRR.Body.String()))
	}
	var listResp struct {
		Snapshots []struct {
			ID              string `json:"id"`
			HasRestoreTrace bool   `json:"hasRestoreTrace"`
		} `json:"snapshots"`
	}
	if err := json.Unmarshal(listRR.Body.Bytes(), &listResp); err != nil {
		t.Fatalf("decode list response: %v body=%s", err, listRR.Body.String())
	}
	if len(listResp.Snapshots) != 1 {
		t.Fatalf("expected 1 snapshot, got %d", len(listResp.Snapshots))
	}
	if !listResp.Snapshots[0].HasRestoreTrace {
		t.Fatalf("restore trace should be visible in list when metadata contains restore_trace_json")
	}

	getReq := withRouteParam(httptest.NewRequest(http.MethodGet, "/api/context-inspector/snapshots/"+packet.ID, nil), "id", packet.ID)
	getRR := httptest.NewRecorder()
	srv.handleContextSnapshotGet(getRR, getReq)
	if getRR.Code != http.StatusOK {
		t.Fatalf("get status=%d body=%s", getRR.Code, strings.TrimSpace(getRR.Body.String()))
	}
	var getResp struct {
		Snapshot struct {
			Summary struct {
				ID              string         `json:"id"`
				HasRestoreTrace bool           `json:"hasRestoreTrace"`
				RestoreTrace    map[string]any `json:"restoreTrace"`
			} `json:"summary"`
		} `json:"snapshot"`
	}
	if err := json.Unmarshal(getRR.Body.Bytes(), &getResp); err != nil {
		t.Fatalf("decode get response: %v body=%s", err, getRR.Body.String())
	}
	if !getResp.Snapshot.Summary.HasRestoreTrace {
		t.Fatalf("expected restore trace visibility in detail when provided")
	}
	if _, ok := getResp.Snapshot.Summary.RestoreTrace["decision"]; !ok {
		t.Fatalf("restore trace should include decision field: %+v", getResp.Snapshot.Summary.RestoreTrace)
	}
}

func TestHandleAuditTraceLookupResolvesByTraceID(t *testing.T) {
	srv, _ := newBackupAuditHarness(t)

	result := invokeLegacyAdapterViaGateway(t, srv, map[string]any{
		"toolId":        "fs.list",
		"laneId":        "read_only.inspect",
		"correlationId": "corr-trace-lookup",
		"traceId":       "trace-lookup-1",
		"workspaceId":   "workspace-trace-lookup",
		"paths":         []string{"."},
		"input":         map[string]any{},
		"initiator":     "operator_inspector_test",
	})
	if result.InvocationID <= 0 {
		t.Fatalf("expected invocation id, got %d", result.InvocationID)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/audit/trace?traceId=trace-lookup-1", nil)
	rr := httptest.NewRecorder()
	srv.handleAuditTraceLookup(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("lookup status=%d body=%s", rr.Code, strings.TrimSpace(rr.Body.String()))
	}

	var resp struct {
		Mode           string   `json:"mode"`
		TraceID        string   `json:"traceId"`
		CorrelationIDs []string `json:"correlationIds"`
		Reports        []struct {
			CorrelationID string `json:"correlationId"`
			Report        struct {
				CorrelationID string `json:"correlationId"`
			} `json:"report"`
		} `json:"reports"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode lookup response: %v body=%s", err, rr.Body.String())
	}
	if resp.Mode != "trace" {
		t.Fatalf("mode=%q want trace", resp.Mode)
	}
	if resp.TraceID != "trace-lookup-1" {
		t.Fatalf("traceId=%q", resp.TraceID)
	}
	if len(resp.CorrelationIDs) != 1 || resp.CorrelationIDs[0] != "corr-trace-lookup" {
		t.Fatalf("unexpected correlation ids: %+v", resp.CorrelationIDs)
	}
	if len(resp.Reports) != 1 || resp.Reports[0].CorrelationID != "corr-trace-lookup" || resp.Reports[0].Report.CorrelationID != "corr-trace-lookup" {
		t.Fatalf("unexpected reports: %+v", resp.Reports)
	}
}

func TestHandleProcessHealthTraceFiltersProcessInvocations(t *testing.T) {
	srv, _ := newBackupAuditHarness(t)

	correlationID := "corr-process-health"
	traceID := "trace-process-health"
	result := invokeLegacyAdapterViaGateway(t, srv, map[string]any{
		"toolId":        "proc.run",
		"correlationId": correlationID,
		"traceId":       traceID,
		"workspaceId":   "workspace-process-health",
		"paths":         []string{"."},
		"input":         map[string]any{"command": "true"},
		"initiator":     "operator_inspector_test",
	})
	if result.InvocationID <= 0 {
		t.Fatalf("expected invocation id, got %d", result.InvocationID)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/process/health?correlationId="+correlationID, nil)
	rr := httptest.NewRecorder()
	srv.handleProcessHealthTrace(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("process/health status=%d body=%s", rr.Code, strings.TrimSpace(rr.Body.String()))
	}

	var resp struct {
		CorrelationID  string   `json:"correlationId"`
		CorrelationIDs []string `json:"correlationIds"`
		Reports        []struct {
			CorrelationID          string `json:"correlationId"`
			TotalInvocations       int    `json:"totalInvocations"`
			ProcessInvocationCount int    `json:"processInvocationCount"`
			ProcessInvocations     []struct {
				ToolID string `json:"toolId"`
			} `json:"processInvocations"`
		} `json:"reports"`
		Runtime struct {
			Available bool `json:"available"`
		} `json:"runtime"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode process/health response: %v body=%s", err, rr.Body.String())
	}
	if resp.CorrelationID != correlationID {
		t.Fatalf("correlationId=%q want %q", resp.CorrelationID, correlationID)
	}
	if len(resp.CorrelationIDs) != 1 || resp.CorrelationIDs[0] != correlationID {
		t.Fatalf("correlation ids unexpected: %+v", resp.CorrelationIDs)
	}
	if len(resp.Reports) != 1 {
		t.Fatalf("expected one report, got %d", len(resp.Reports))
	}
	if resp.Reports[0].CorrelationID != correlationID {
		t.Fatalf("report correlation=%q want %q", resp.Reports[0].CorrelationID, correlationID)
	}
	if resp.Reports[0].TotalInvocations < 1 {
		t.Fatalf("expected gateway invocations to be present, got %d", resp.Reports[0].TotalInvocations)
	}
	if resp.Reports[0].ProcessInvocationCount < 1 {
		t.Fatalf("expected process invocations to be present, got %d", resp.Reports[0].ProcessInvocationCount)
	}
	if len(resp.Reports[0].ProcessInvocations) == 0 || resp.Reports[0].ProcessInvocations[0].ToolID != "proc.run" {
		t.Fatalf("unexpected process invocations: %+v", resp.Reports[0].ProcessInvocations)
	}

	traceReq := httptest.NewRequest(http.MethodGet, "/api/process/health?traceId="+traceID, nil)
	traceRR := httptest.NewRecorder()
	srv.handleProcessHealthTrace(traceRR, traceReq)
	if traceRR.Code != http.StatusOK {
		t.Fatalf("process/health trace status=%d body=%s", traceRR.Code, strings.TrimSpace(traceRR.Body.String()))
	}
	var traceResp struct {
		TraceID        string   `json:"traceId"`
		CorrelationIDs []string `json:"correlationIds"`
	}
	if err := json.Unmarshal(traceRR.Body.Bytes(), &traceResp); err != nil {
		t.Fatalf("decode process/health trace response: %v body=%s", err, traceRR.Body.String())
	}
	if traceResp.TraceID != traceID {
		t.Fatalf("traceId=%q want %q", traceResp.TraceID, traceID)
	}
	if len(traceResp.CorrelationIDs) == 0 || traceResp.CorrelationIDs[0] != correlationID {
		t.Fatalf("trace correlation ids unexpected: %+v", traceResp.CorrelationIDs)
	}
}
