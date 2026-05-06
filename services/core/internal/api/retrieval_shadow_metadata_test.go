package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"forge/projectforge/services/core/internal/config"
	"forge/projectforge/services/core/internal/forgekshadow"
	"forge/projectforge/services/core/internal/store"
)

func TestForgeKShadowRetrievalMetadataRouteInventoryUnchanged(t *testing.T) {
	disabled := collectServerRoutes(t, (&Server{}).Handler())
	enabled := collectServerRoutes(t, (&Server{
		cfg: config.Config{
			ForgeKShadowModeEnabled:              true,
			ForgeKShadowRetrievalMetadataEnabled: true,
		},
		forgeKShadow: forgekshadow.NewObserver(forgekshadow.Config{Enabled: true, RetrievalMetadataEnabled: true}),
	}).Handler())
	if !sameRouteSet(disabled, enabled) {
		t.Fatalf("retrieval metadata shadow changed route inventory\ndisabled=%#v\nenabled=%#v", routeKeys(disabled), routeKeys(enabled))
	}
	for _, route := range routeKeys(enabled) {
		normalized := strings.ToLower(route)
		for _, forbidden := range []string{"retrieval-shadow", "retrieval-metadata", "forgek-shadow", "shadow-diagnostic", "/api/shadow", "/forge/shadow"} {
			if strings.Contains(normalized, forbidden) {
				t.Fatalf("retrieval metadata shadow must not expose public diagnostics route: %s", route)
			}
		}
	}
}

func TestForgeKShadowRetrievalMetadataNewServerWiresConfigFlag(t *testing.T) {
	cases := []struct {
		name        string
		globalFlag  bool
		retrieval   bool
		wantReports int
	}{
		{"both disabled", false, false, 0},
		{"global enabled retrieval disabled", true, false, 0},
		{"both enabled", true, true, 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dataDir := t.TempDir()
			workspaceDir := t.TempDir()
			st, err := store.Open(dataDir)
			if err != nil {
				t.Fatalf("open store: %v", err)
			}
			t.Cleanup(func() { _ = st.Close() })
			srv := NewServer(st, config.Config{
				DataDir:                                     dataDir,
				WorkspaceDir:                                workspaceDir,
				ForgeKShadowModeEnabled:                     tc.globalFlag,
				ForgeKShadowRetrievalMetadataEnabled:        tc.retrieval,
				ForgeKShadowChatMetadataEnabled:             false,
				EnableOpenAICompatAPI:                       false,
				ModelRuntimeAllowOllamaCloudModels:          false,
				ModelPolicyRequireExplicitLoad:              true,
				ModelPolicyRequireWorkspaceScope:            true,
				ModelRuntimeDegradedOnUnavailableGPU:        true,
				SchedulingInteractivePriorityOverBackground: true,
			})
			t.Cleanup(func() { srv.ShutdownWatch() })
			seedRetrievalShadowContent(t, st, "phase12kl wiring searchable content")

			rr := postRetrievalRunForShadow(t, srv, `{"query":"phase12kl wiring","mode":"keyword","limit":5,"selectForPacket":1}`)
			if rr.Code != http.StatusOK {
				t.Fatalf("retrieval run status=%d body=%s", rr.Code, rr.Body.String())
			}
			retrievalReports := 0
			if srv.forgeKShadow != nil {
				for _, report := range srv.forgeKShadow.Reports() {
					if report.RetrievalMetadata != nil {
						retrievalReports++
					}
				}
			}
			if retrievalReports != tc.wantReports {
				t.Fatalf("retrieval metadata reports=%d, want %d", retrievalReports, tc.wantReports)
			}
		})
	}
}

func TestForgeKShadowRetrievalMetadataFlagMatrixAtRetrievalRoute(t *testing.T) {
	cases := []struct {
		name        string
		globalFlag  bool
		retrieval   bool
		wantReports int
	}{
		{"both disabled", false, false, 0},
		{"global disabled retrieval enabled", false, true, 0},
		{"global enabled retrieval disabled", true, false, 0},
		{"both enabled", true, true, 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv, st := newBackupAuditHarness(t)
			seedRetrievalShadowContent(t, st, "phase12kl matrix searchable content")
			observer := forgekshadow.NewObserver(forgekshadow.Config{Enabled: tc.globalFlag, RetrievalMetadataEnabled: tc.retrieval})
			srv.cfg.ForgeKShadowModeEnabled = tc.globalFlag
			srv.cfg.ForgeKShadowRetrievalMetadataEnabled = tc.retrieval
			srv.forgeKShadow = observer

			rr := postRetrievalRunForShadow(t, srv, `{"query":"phase12kl matrix","mode":"keyword","limit":5,"selectForPacket":1}`)
			if rr.Code != http.StatusOK {
				t.Fatalf("retrieval run status=%d body=%s", rr.Code, rr.Body.String())
			}
			retrievalReports := 0
			for _, report := range observer.Reports() {
				if report.RetrievalMetadata != nil {
					retrievalReports++
				}
			}
			if retrievalReports != tc.wantReports {
				t.Fatalf("retrieval metadata reports=%d, want %d; reports=%#v", retrievalReports, tc.wantReports, observer.Reports())
			}
		})
	}
}

func TestForgeKShadowRetrievalMetadataObservedWithoutChangingRetrievalResponseShape(t *testing.T) {
	body := `{"query":"phase12kl stable","mode":"keyword","limit":5,"selectForPacket":1,"model":"safe-embedding-model"}`

	baselineSrv, baselineStore := newBackupAuditHarness(t)
	seedRetrievalShadowContent(t, baselineStore, "phase12kl stable searchable content")
	baselineRR := postRetrievalRunForShadow(t, baselineSrv, body)
	if baselineRR.Code != http.StatusOK {
		t.Fatalf("baseline retrieval status=%d body=%s", baselineRR.Code, baselineRR.Body.String())
	}

	enabledSrv, enabledStore := newBackupAuditHarness(t)
	seedRetrievalShadowContent(t, enabledStore, "phase12kl stable searchable content")
	observer := forgekshadow.NewObserver(forgekshadow.Config{Enabled: true, RetrievalMetadataEnabled: true})
	enabledSrv.cfg.ForgeKShadowModeEnabled = true
	enabledSrv.cfg.ForgeKShadowRetrievalMetadataEnabled = true
	enabledSrv.forgeKShadow = observer
	enabledRR := postRetrievalRunForShadow(t, enabledSrv, body)
	if enabledRR.Code != http.StatusOK {
		t.Fatalf("enabled retrieval status=%d body=%s", enabledRR.Code, enabledRR.Body.String())
	}

	assertRetrievalRunResponseShape(t, baselineRR, enabledRR)
	report := findRetrievalMetadataReport(t, observer.Reports())
	if report.RetrievalMetadata.RetrievalRunID == "" || report.RetrievalMetadata.ResultCount != 1 || report.RetrievalMetadata.SelectedCount != 1 {
		t.Fatalf("unexpected retrieval metadata refs/counts: %#v", report.RetrievalMetadata)
	}
	if report.RetrievalMetadata.RetrievalStrategy != "keyword" || report.RetrievalMetadata.IndexType != "fts" {
		t.Fatalf("unexpected retrieval metadata strategy/index: %#v", report.RetrievalMetadata)
	}
	if report.Observation.WorkspaceID == "" || report.Observation.LivePath != "POST /api/retrieval/runs" {
		t.Fatalf("unexpected retrieval observation scope: %#v", report.Observation)
	}
	if report.Observation.Metadata["observation_type"] != "retrieval_metadata" || report.Observation.Metadata["touchpoint"] != "retrieval_run_created" {
		t.Fatalf("unexpected retrieval observation metadata: %#v", report.Observation.Metadata)
	}
	serializedReport := strings.ToLower(fmt.Sprint(report))
	for _, forbidden := range []string{"phase12kl stable", "searchable content", "query", "snippet", "content", "body", "embedding_vector", "memory_content", "prompt", "completion"} {
		if strings.Contains(serializedReport, forbidden) {
			t.Fatalf("retrieval metadata report leaked forbidden fragment %q in %q", forbidden, serializedReport)
		}
	}
}

func TestForgeKShadowRetrievalMetadataSinkFailureDoesNotChangeRetrievalResponse(t *testing.T) {
	body := `{"query":"phase12kl sink","mode":"keyword","limit":5,"selectForPacket":1}`
	baselineSrv, baselineStore := newBackupAuditHarness(t)
	seedRetrievalShadowContent(t, baselineStore, "phase12kl sink searchable content")
	baselineRR := postRetrievalRunForShadow(t, baselineSrv, body)
	if baselineRR.Code != http.StatusOK {
		t.Fatalf("baseline retrieval status=%d body=%s", baselineRR.Code, baselineRR.Body.String())
	}

	enabledSrv, enabledStore := newBackupAuditHarness(t)
	seedRetrievalShadowContent(t, enabledStore, "phase12kl sink searchable content")
	enabledSrv.cfg.ForgeKShadowModeEnabled = true
	enabledSrv.cfg.ForgeKShadowRetrievalMetadataEnabled = true
	enabledSrv.forgeKShadow = forgekshadow.NewObserverWithSink(forgekshadow.Config{Enabled: true, RetrievalMetadataEnabled: true}, failingShadowSink{}, nil)
	enabledRR := postRetrievalRunForShadow(t, enabledSrv, body)
	assertRetrievalRunResponseShape(t, baselineRR, enabledRR)
}

func TestForgeKShadowRetrievalMetadataDoesNotCaptureInvalidBodyAuthCookieOrQuery(t *testing.T) {
	srv, _ := newBackupAuditHarness(t)
	observer := forgekshadow.NewObserver(forgekshadow.Config{Enabled: true, RetrievalMetadataEnabled: true})
	srv.cfg.ForgeKShadowModeEnabled = true
	srv.cfg.ForgeKShadowRetrievalMetadataEnabled = true
	srv.forgeKShadow = observer

	req := httptest.NewRequest(http.MethodPost, "/api/retrieval/runs?token=SHOULD-NOT-APPEAR", strings.NewReader(`{"query":"INVALID-BODY-MUST-NOT-APPEAR"`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer SHOULD-NOT-APPEAR")
	req.Header.Set("Cookie", "session=SHOULD-NOT-APPEAR")
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("invalid retrieval post status=%d, want %d body=%s", rr.Code, http.StatusBadRequest, rr.Body.String())
	}
	if len(observer.Reports()) == 0 {
		t.Fatalf("expected route-envelope shadow report for invalid body to prove no-capture check is not vacuous")
	}
	for _, report := range observer.Reports() {
		if report.RetrievalMetadata != nil {
			t.Fatalf("invalid body should not create retrieval metadata report: %#v", report)
		}
		serialized := strings.ToLower(fmt.Sprint(report))
		for _, forbidden := range []string{"invalid-body-must-not-appear", "should-not-appear", "bearer", "cookie", "session", "token"} {
			if strings.Contains(serialized, forbidden) {
				t.Fatalf("shadow report leaked forbidden invalid-body/header/query fragment %q in %q", forbidden, serialized)
			}
		}
	}
}

func seedRetrievalShadowContent(t *testing.T, st *store.Store, content string) {
	t.Helper()
	now := time.Now().UnixMilli()
	sourceID := seedSource(t, st, "/repo")
	fileID := seedFile(t, st, sourceID, "retrieval-shadow.txt", "/repo/retrieval-shadow.txt", now)
	_ = seedChunk(t, st, fileID, content)
}

func postRetrievalRunForShadow(t *testing.T, srv *Server, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/retrieval/runs", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Request-ID", "request-retrieval-shadow")
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)
	return rr
}

func findRetrievalMetadataReport(t *testing.T, reports []forgekshadow.DiagnosticReport) forgekshadow.DiagnosticReport {
	t.Helper()
	for _, report := range reports {
		if report.RetrievalMetadata != nil {
			return report
		}
	}
	t.Fatalf("expected retrieval metadata report, got %#v", reports)
	return forgekshadow.DiagnosticReport{}
}

func assertRetrievalRunResponseShape(t *testing.T, baselineRR, enabledRR *httptest.ResponseRecorder) {
	t.Helper()
	if baselineRR.Code != enabledRR.Code {
		t.Fatalf("retrieval metadata changed status baseline=%d enabled=%d", baselineRR.Code, enabledRR.Code)
	}
	if baselineRR.Header().Get("Content-Type") != enabledRR.Header().Get("Content-Type") {
		t.Fatalf("retrieval metadata changed content type baseline=%q enabled=%q", baselineRR.Header().Get("Content-Type"), enabledRR.Header().Get("Content-Type"))
	}
	baseline := decodeRetrievalRunResponse(t, baselineRR)
	enabled := decodeRetrievalRunResponse(t, enabledRR)
	if baseline.ResultCount != enabled.ResultCount || baseline.SelectedCount != enabled.SelectedCount {
		t.Fatalf("retrieval metadata changed result counts baseline=%#v enabled=%#v", baseline, enabled)
	}
	if baseline.Mode != enabled.Mode {
		t.Fatalf("retrieval metadata changed mode baseline=%q enabled=%q", baseline.Mode, enabled.Mode)
	}
	if len(baseline.Results) != len(enabled.Results) {
		t.Fatalf("retrieval metadata changed result shape baseline=%#v enabled=%#v", baseline.Results, enabled.Results)
	}
	for i := range baseline.Results {
		if baseline.Results[i] != enabled.Results[i] {
			t.Fatalf("retrieval metadata changed result %d baseline=%#v enabled=%#v", i, baseline.Results[i], enabled.Results[i])
		}
	}
}

type retrievalRunResponseShape struct {
	Mode          string
	ResultCount   int
	SelectedCount int
	Results       []retrievalResultResponseShape
}

type retrievalResultResponseShape struct {
	RelPath           string
	RankIndex         int
	SelectedForPacket bool
	KeywordScore      float64
	SemanticScore     float64
	HybridScore       float64
}

func decodeRetrievalRunResponse(t *testing.T, rr *httptest.ResponseRecorder) retrievalRunResponseShape {
	t.Helper()
	var payload struct {
		Run struct {
			Mode    string `json:"mode"`
			Results []struct {
				RelPath           string  `json:"relPath"`
				RankIndex         int     `json:"rankIndex"`
				SelectedForPacket bool    `json:"selectedForPacket"`
				KeywordScore      float64 `json:"keywordScore"`
				SemanticScore     float64 `json:"semanticScore"`
				HybridScore       float64 `json:"hybridScore"`
			} `json:"results"`
		} `json:"run"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode retrieval response: %v", err)
	}
	out := retrievalRunResponseShape{Mode: payload.Run.Mode, ResultCount: len(payload.Run.Results)}
	for _, result := range payload.Run.Results {
		if result.SelectedForPacket {
			out.SelectedCount++
		}
		out.Results = append(out.Results, retrievalResultResponseShape{
			RelPath:           result.RelPath,
			RankIndex:         result.RankIndex,
			SelectedForPacket: result.SelectedForPacket,
			KeywordScore:      result.KeywordScore,
			SemanticScore:     result.SemanticScore,
			HybridScore:       result.HybridScore,
		})
	}
	return out
}
