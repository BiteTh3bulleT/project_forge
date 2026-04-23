package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestHandleAuditTraceBuildsCorrelationReportAcrossGatewayAuditArtifactAndProvenance(t *testing.T) {
	srv, st := newBackupAuditHarness(t)

	correlationID := "corr-trace-report"
	traceID := "trace-trace-report"
	workspaceID := "workspace-trace-report"

	gatewayResult := invokeLegacyAdapterViaGateway(t, srv, map[string]any{
		"toolId":        "fs.list",
		"laneId":        "read_only.inspect",
		"correlationId": correlationID,
		"traceId":       traceID,
		"workspaceId":   workspaceID,
		"paths":         []string{"."},
		"input":         map[string]any{},
		"initiator":     "traceability-test",
	})
	if gatewayResult.InvocationID <= 0 {
		t.Fatalf("expected gateway invocation id, got %d", gatewayResult.InvocationID)
	}

	thread, err := srv.chat.CreateThread(context.Background(), "trace report thread", nil)
	if err != nil {
		t.Fatalf("create thread: %v", err)
	}

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", "trace-note.txt")
	if err != nil {
		t.Fatalf("create multipart file: %v", err)
	}
	if _, err := io.WriteString(part, "traceability report upload payload"); err != nil {
		t.Fatalf("write multipart payload: %v", err)
	}
	if err := writer.WriteField("title", "Traceability Artifact"); err != nil {
		t.Fatalf("write title field: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}

	uploadPath := fmt.Sprintf(
		"/api/chat/threads/%d/attachments?correlationId=%s&traceId=%s&workspaceId=%s",
		thread.ID, correlationID, traceID, workspaceID,
	)
	uploadReq := withRouteParam(
		httptest.NewRequest(http.MethodPost, uploadPath, &body),
		"id",
		strconv.FormatInt(thread.ID, 10),
	)
	uploadReq.Header.Set("Content-Type", writer.FormDataContentType())

	uploadRR := httptest.NewRecorder()
	srv.handleChatAttachmentUpload(uploadRR, uploadReq)
	if uploadRR.Code != http.StatusOK {
		t.Fatalf("upload status=%d body=%s", uploadRR.Code, strings.TrimSpace(uploadRR.Body.String()))
	}

	var uploadResp struct {
		Artifact struct {
			ID       int64  `json:"id"`
			FilePath string `json:"filePath"`
		} `json:"artifact"`
	}
	if err := json.Unmarshal(uploadRR.Body.Bytes(), &uploadResp); err != nil {
		t.Fatalf("decode upload response: %v body=%s", err, uploadRR.Body.String())
	}
	if uploadResp.Artifact.ID <= 0 {
		t.Fatalf("expected artifact id in upload response, got %d", uploadResp.Artifact.ID)
	}

	var artifactAuditID int64
	if err := st.DB.QueryRowContext(context.Background(), `
SELECT id
FROM audit_records
WHERE action = 'artifact.uploaded' AND correlation_id = ?
ORDER BY id DESC LIMIT 1`, correlationID).Scan(&artifactAuditID); err != nil {
		t.Fatalf("query artifact upload audit id: %v", err)
	}

	now := time.Now().UnixMilli()
	provenanceID := "prov-trace-report"
	journalID := "journal-trace-report"
	artifactRefID := "artifact-ref-trace-report"

	if _, err := st.DB.ExecContext(context.Background(), `
INSERT INTO provenance_records(
  id, actor, actor_type, source, trace_id, workspace_id, lane_id, selected_paths_json, metadata_json,
  created_at, proposed_by, committed_by, syscall_id, correlation_id, audit_id
) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		provenanceID, "api-test", "operator", "api.trace_test", traceID, workspaceID, "trace.report",
		`[]`, `{"suite":"phase5_trace_report"}`, now, "test-proposer", "forge_kernel", "syscall-trace-report",
		correlationID, strconv.FormatInt(artifactAuditID, 10),
	); err != nil {
		t.Fatalf("insert provenance record: %v", err)
	}

	if _, err := st.DB.ExecContext(context.Background(), `
INSERT INTO journal_events(
  id, type, source, actor, workspace_id, lane_id, selected_paths_json, payload_json,
  correlation_id, trace_id, provenance_id, provenance_json, created_at, metadata_json,
  proposed_by, committed_by, syscall_id, audit_id
) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		journalID, "trace.report.created", "api.trace_test", "api-test", workspaceID, "trace.report", `[]`,
		`{"artifactId":`+strconv.FormatInt(uploadResp.Artifact.ID, 10)+`}`, correlationID, traceID, provenanceID,
		`{"id":"`+provenanceID+`"}`, now+1, `{"suite":"phase5_trace_report"}`,
		"test-proposer", "forge_kernel", "syscall-trace-report", strconv.FormatInt(artifactAuditID, 10),
	); err != nil {
		t.Fatalf("insert journal event: %v", err)
	}

	if _, err := st.DB.ExecContext(context.Background(), `
INSERT INTO artifact_refs(
  id, type, uri, content_hash, workspace_id, lane_id, selected_paths_json, provenance_id, provenance_json,
  created_at, metadata_json, proposed_by, committed_by, syscall_id, correlation_id, trace_id, audit_id
) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		artifactRefID, "file", uploadResp.Artifact.FilePath, "", workspaceID, "trace.report", `[]`, provenanceID,
		`{"id":"`+provenanceID+`"}`, now+2, `{"suite":"phase5_trace_report"}`, "test-proposer", "forge_kernel",
		"syscall-trace-report", correlationID, traceID, strconv.FormatInt(artifactAuditID, 10),
	); err != nil {
		t.Fatalf("insert artifact ref: %v", err)
	}

	traceReq := withRouteParam(
		httptest.NewRequest(http.MethodGet, "/api/audit/trace/"+correlationID, nil),
		"correlationId",
		correlationID,
	)
	traceRR := httptest.NewRecorder()
	srv.handleAuditTrace(traceRR, traceReq)
	if traceRR.Code != http.StatusOK {
		t.Fatalf("trace status=%d body=%s", traceRR.Code, strings.TrimSpace(traceRR.Body.String()))
	}

	var traceResp struct {
		CorrelationID string `json:"correlationId"`
		Records       []struct {
			ID int64 `json:"id"`
		} `json:"records"`
		Report struct {
			CorrelationID      string `json:"correlationId"`
			GatewayInvocations []struct {
				ID            int64  `json:"id"`
				CorrelationID string `json:"correlationId"`
			} `json:"gatewayInvocations"`
			AuditRecords []struct {
				ID int64 `json:"id"`
			} `json:"auditRecords"`
			ArtifactRecords []struct {
				ID int64 `json:"id"`
			} `json:"artifactRecords"`
			ProvenanceRecords []struct {
				ID      string `json:"id"`
				AuditID string `json:"auditId"`
			} `json:"provenanceRecords"`
			JournalEvents []struct {
				ID           string  `json:"id"`
				ProvenanceID *string `json:"provenanceId"`
			} `json:"journalEvents"`
			ArtifactRefs []struct {
				ID           string  `json:"id"`
				ProvenanceID *string `json:"provenanceId"`
			} `json:"artifactRefs"`
			Links struct {
				AuditToGateway []struct {
					AuditRecordID       int64 `json:"auditRecordId"`
					GatewayInvocationID int64 `json:"gatewayInvocationId"`
				} `json:"auditToGateway"`
				AuditToArtifact []struct {
					AuditRecordID int64 `json:"auditRecordId"`
					ArtifactID    int64 `json:"artifactId"`
				} `json:"auditToArtifact"`
				ProvenanceToAudit []struct {
					ProvenanceID  string `json:"provenanceId"`
					AuditRecordID int64  `json:"auditRecordId"`
				} `json:"provenanceToAudit"`
				JournalToProvenance []struct {
					JournalEventID string `json:"journalEventId"`
					ProvenanceID   string `json:"provenanceId"`
				} `json:"journalToProvenance"`
				ArtifactRefToProvenance []struct {
					ArtifactRefID string `json:"artifactRefId"`
					ProvenanceID  string `json:"provenanceId"`
				} `json:"artifactRefToProvenance"`
			} `json:"links"`
		} `json:"report"`
	}
	if err := json.Unmarshal(traceRR.Body.Bytes(), &traceResp); err != nil {
		t.Fatalf("decode trace response: %v body=%s", err, traceRR.Body.String())
	}

	if traceResp.CorrelationID != correlationID {
		t.Fatalf("top-level correlation=%q want %q", traceResp.CorrelationID, correlationID)
	}
	if traceResp.Report.CorrelationID != correlationID {
		t.Fatalf("report correlation=%q want %q", traceResp.Report.CorrelationID, correlationID)
	}
	if len(traceResp.Records) != len(traceResp.Report.AuditRecords) {
		t.Fatalf("records length=%d report.auditRecords length=%d", len(traceResp.Records), len(traceResp.Report.AuditRecords))
	}
	if !hasGatewayInvocation(traceResp.Report.GatewayInvocations, gatewayResult.InvocationID, correlationID) {
		t.Fatalf("expected report to include gateway invocation %d", gatewayResult.InvocationID)
	}
	if !hasArtifactID(traceResp.Report.ArtifactRecords, uploadResp.Artifact.ID) {
		t.Fatalf("expected report artifactRecords to include artifact %d", uploadResp.Artifact.ID)
	}
	if !hasProvenanceID(traceResp.Report.ProvenanceRecords, provenanceID, strconv.FormatInt(artifactAuditID, 10)) {
		t.Fatalf("expected provenance record %q linked to audit %d", provenanceID, artifactAuditID)
	}
	if !hasJournalProvenance(traceResp.Report.JournalEvents, journalID, provenanceID) {
		t.Fatalf("expected journal event %q linked to provenance %q", journalID, provenanceID)
	}
	if !hasArtifactRefProvenance(traceResp.Report.ArtifactRefs, artifactRefID, provenanceID) {
		t.Fatalf("expected artifact ref %q linked to provenance %q", artifactRefID, provenanceID)
	}
	if !hasAuditGatewayLink(traceResp.Report.Links.AuditToGateway, gatewayResult.InvocationID) {
		t.Fatalf("expected auditToGateway link for invocation %d", gatewayResult.InvocationID)
	}
	if !hasAuditArtifactLink(traceResp.Report.Links.AuditToArtifact, uploadResp.Artifact.ID) {
		t.Fatalf("expected auditToArtifact link for artifact %d", uploadResp.Artifact.ID)
	}
	if !hasProvenanceAuditLink(traceResp.Report.Links.ProvenanceToAudit, provenanceID, artifactAuditID) {
		t.Fatalf("expected provenanceToAudit link %q -> %d", provenanceID, artifactAuditID)
	}
	if !hasJournalLink(traceResp.Report.Links.JournalToProvenance, journalID, provenanceID) {
		t.Fatalf("expected journalToProvenance link %q -> %q", journalID, provenanceID)
	}
	if !hasArtifactRefLink(traceResp.Report.Links.ArtifactRefToProvenance, artifactRefID, provenanceID) {
		t.Fatalf("expected artifactRefToProvenance link %q -> %q", artifactRefID, provenanceID)
	}
}

func TestHandleAuditTraceRequiresCorrelationID(t *testing.T) {
	srv, _ := newBackupAuditHarness(t)
	req := withRouteParam(httptest.NewRequest(http.MethodGet, "/api/audit/trace/", nil), "correlationId", " ")
	rr := httptest.NewRecorder()
	srv.handleAuditTrace(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d body=%s", rr.Code, strings.TrimSpace(rr.Body.String()))
	}
}

func hasGatewayInvocation(records []struct {
	ID            int64  `json:"id"`
	CorrelationID string `json:"correlationId"`
}, id int64, correlationID string) bool {
	for _, rec := range records {
		if rec.ID == id && rec.CorrelationID == correlationID {
			return true
		}
	}
	return false
}

func hasArtifactID(records []struct {
	ID int64 `json:"id"`
}, id int64) bool {
	for _, rec := range records {
		if rec.ID == id {
			return true
		}
	}
	return false
}

func hasProvenanceID(records []struct {
	ID      string `json:"id"`
	AuditID string `json:"auditId"`
}, id, auditID string) bool {
	for _, rec := range records {
		if rec.ID == id && rec.AuditID == auditID {
			return true
		}
	}
	return false
}

func hasJournalProvenance(records []struct {
	ID           string  `json:"id"`
	ProvenanceID *string `json:"provenanceId"`
}, id, provenanceID string) bool {
	for _, rec := range records {
		if rec.ID == id && rec.ProvenanceID != nil && *rec.ProvenanceID == provenanceID {
			return true
		}
	}
	return false
}

func hasArtifactRefProvenance(records []struct {
	ID           string  `json:"id"`
	ProvenanceID *string `json:"provenanceId"`
}, id, provenanceID string) bool {
	for _, rec := range records {
		if rec.ID == id && rec.ProvenanceID != nil && *rec.ProvenanceID == provenanceID {
			return true
		}
	}
	return false
}

func hasAuditGatewayLink(records []struct {
	AuditRecordID       int64 `json:"auditRecordId"`
	GatewayInvocationID int64 `json:"gatewayInvocationId"`
}, gatewayInvocationID int64) bool {
	for _, rec := range records {
		if rec.GatewayInvocationID == gatewayInvocationID {
			return true
		}
	}
	return false
}

func hasAuditArtifactLink(records []struct {
	AuditRecordID int64 `json:"auditRecordId"`
	ArtifactID    int64 `json:"artifactId"`
}, artifactID int64) bool {
	for _, rec := range records {
		if rec.ArtifactID == artifactID {
			return true
		}
	}
	return false
}

func hasProvenanceAuditLink(records []struct {
	ProvenanceID  string `json:"provenanceId"`
	AuditRecordID int64  `json:"auditRecordId"`
}, provenanceID string, auditRecordID int64) bool {
	for _, rec := range records {
		if rec.ProvenanceID == provenanceID && rec.AuditRecordID == auditRecordID {
			return true
		}
	}
	return false
}

func hasJournalLink(records []struct {
	JournalEventID string `json:"journalEventId"`
	ProvenanceID   string `json:"provenanceId"`
}, journalID, provenanceID string) bool {
	for _, rec := range records {
		if rec.JournalEventID == journalID && rec.ProvenanceID == provenanceID {
			return true
		}
	}
	return false
}

func hasArtifactRefLink(records []struct {
	ArtifactRefID string `json:"artifactRefId"`
	ProvenanceID  string `json:"provenanceId"`
}, artifactRefID, provenanceID string) bool {
	for _, rec := range records {
		if rec.ArtifactRefID == artifactRefID && rec.ProvenanceID == provenanceID {
			return true
		}
	}
	return false
}
