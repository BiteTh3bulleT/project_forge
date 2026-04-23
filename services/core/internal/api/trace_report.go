package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"forge/projectforge/services/core/internal/artifacts"
	"forge/projectforge/services/core/internal/audit"
	"forge/projectforge/services/core/internal/gateway"
)

type correlationTraceReport struct {
	CorrelationID       string                          `json:"correlationId"`
	GeneratedAtMs       int64                           `json:"generatedAtMs"`
	GatewayInvocations  []gateway.InvocationRecord      `json:"gatewayInvocations"`
	AuditRecords        []audit.Record                  `json:"auditRecords"`
	ArtifactRecords     []artifacts.Artifact            `json:"artifactRecords"`
	GatewayArtifactRefs []correlationGatewayArtifactRef `json:"gatewayArtifactRefs"`
	ProvenanceRecords   []correlationProvenanceRecord   `json:"provenanceRecords"`
	JournalEvents       []correlationJournalEvent       `json:"journalEvents"`
	ArtifactRefs        []correlationArtifactRef        `json:"artifactRefs"`
	Links               correlationTraceLinks           `json:"links"`
}

type correlationGatewayArtifactRef struct {
	GatewayInvocationID int64  `json:"gatewayInvocationId"`
	ToolID              string `json:"toolId"`
	Type                string `json:"type"`
	Path                string `json:"path"`
	Summary             string `json:"summary"`
}

type correlationProvenanceRecord struct {
	ID            string          `json:"id"`
	Actor         string          `json:"actor"`
	ActorType     string          `json:"actorType"`
	Source        string          `json:"source"`
	TraceID       string          `json:"traceId"`
	WorkspaceID   string          `json:"workspaceId"`
	LaneID        string          `json:"laneId"`
	SelectedPaths json.RawMessage `json:"selectedPaths"`
	Metadata      json.RawMessage `json:"metadata"`
	CreatedAtMs   int64           `json:"createdAtMs"`
	ProposedBy    string          `json:"proposedBy"`
	CommittedBy   string          `json:"committedBy"`
	SyscallID     string          `json:"syscallId"`
	CorrelationID string          `json:"correlationId"`
	AuditID       string          `json:"auditId"`
}

type correlationJournalEvent struct {
	ID            string          `json:"id"`
	Type          string          `json:"type"`
	Source        string          `json:"source"`
	Actor         string          `json:"actor"`
	WorkspaceID   string          `json:"workspaceId"`
	LaneID        string          `json:"laneId"`
	SelectedPaths json.RawMessage `json:"selectedPaths"`
	Payload       json.RawMessage `json:"payload"`
	CorrelationID string          `json:"correlationId"`
	TraceID       string          `json:"traceId"`
	ProvenanceID  *string         `json:"provenanceId,omitempty"`
	Provenance    json.RawMessage `json:"provenance"`
	CreatedAtMs   int64           `json:"createdAtMs"`
	Metadata      json.RawMessage `json:"metadata"`
	ProposedBy    string          `json:"proposedBy"`
	CommittedBy   string          `json:"committedBy"`
	SyscallID     string          `json:"syscallId"`
	AuditID       string          `json:"auditId"`
}

type correlationArtifactRef struct {
	ID            string          `json:"id"`
	Type          string          `json:"type"`
	URI           string          `json:"uri"`
	ContentHash   string          `json:"contentHash"`
	WorkspaceID   string          `json:"workspaceId"`
	LaneID        string          `json:"laneId"`
	SelectedPaths json.RawMessage `json:"selectedPaths"`
	ProvenanceID  *string         `json:"provenanceId,omitempty"`
	Provenance    json.RawMessage `json:"provenance"`
	CreatedAtMs   int64           `json:"createdAtMs"`
	Metadata      json.RawMessage `json:"metadata"`
	ProposedBy    string          `json:"proposedBy"`
	CommittedBy   string          `json:"committedBy"`
	SyscallID     string          `json:"syscallId"`
	CorrelationID string          `json:"correlationId"`
	TraceID       string          `json:"traceId"`
	AuditID       string          `json:"auditId"`
}

type correlationTraceLinks struct {
	AuditToGateway          []traceAuditGatewayLink          `json:"auditToGateway"`
	AuditToArtifact         []traceAuditArtifactLink         `json:"auditToArtifact"`
	ProvenanceToAudit       []traceProvenanceAuditLink       `json:"provenanceToAudit"`
	JournalToProvenance     []traceJournalProvenanceLink     `json:"journalToProvenance"`
	ArtifactRefToProvenance []traceArtifactRefProvenanceLink `json:"artifactRefToProvenance"`
	GatewayToArtifact       []traceGatewayArtifactLink       `json:"gatewayToArtifact"`
}

type traceAuditGatewayLink struct {
	AuditRecordID       int64 `json:"auditRecordId"`
	GatewayInvocationID int64 `json:"gatewayInvocationId"`
}

type traceAuditArtifactLink struct {
	AuditRecordID int64 `json:"auditRecordId"`
	ArtifactID    int64 `json:"artifactId"`
}

type traceProvenanceAuditLink struct {
	ProvenanceID  string `json:"provenanceId"`
	AuditRecordID int64  `json:"auditRecordId"`
}

type traceJournalProvenanceLink struct {
	JournalEventID string `json:"journalEventId"`
	ProvenanceID   string `json:"provenanceId"`
}

type traceArtifactRefProvenanceLink struct {
	ArtifactRefID string `json:"artifactRefId"`
	ProvenanceID  string `json:"provenanceId"`
}

type traceGatewayArtifactLink struct {
	GatewayInvocationID int64  `json:"gatewayInvocationId"`
	ArtifactID          int64  `json:"artifactId"`
	Path                string `json:"path"`
}

func (s *Server) buildCorrelationTraceReport(ctx context.Context, correlationID string) (correlationTraceReport, error) {
	if s.auditSvc == nil {
		return correlationTraceReport{}, fmt.Errorf("audit unavailable")
	}

	auditRecords, err := s.auditSvc.Trace(ctx, correlationID)
	if err != nil {
		return correlationTraceReport{}, err
	}

	gatewayInvocations := []gateway.InvocationRecord{}
	if s.gateway != nil {
		gatewayInvocations, err = s.gateway.ListInvocationsByCorrelation(ctx, correlationID, 500)
		if err != nil {
			return correlationTraceReport{}, err
		}
	}

	provenanceRecords, err := s.listProvenanceRecordsByCorrelation(ctx, correlationID)
	if err != nil {
		return correlationTraceReport{}, err
	}
	journalEvents, err := s.listJournalEventsByCorrelation(ctx, correlationID)
	if err != nil {
		return correlationTraceReport{}, err
	}
	artifactRefs, err := s.listArtifactRefsByCorrelation(ctx, correlationID)
	if err != nil {
		return correlationTraceReport{}, err
	}

	auditArtifactIDs, auditArtifactLinks := collectAuditArtifactIDs(auditRecords)
	gatewayArtifactRefs := collectGatewayArtifactRefs(gatewayInvocations)
	artifactPaths := collectTraceArtifactPaths(gatewayArtifactRefs, artifactRefs)
	artifactRecords, err := s.listArtifactRecordsForTrace(ctx, auditArtifactIDs, artifactPaths)
	if err != nil {
		return correlationTraceReport{}, err
	}

	return correlationTraceReport{
		CorrelationID:       correlationID,
		GeneratedAtMs:       time.Now().UnixMilli(),
		GatewayInvocations:  gatewayInvocations,
		AuditRecords:        auditRecords,
		ArtifactRecords:     artifactRecords,
		GatewayArtifactRefs: gatewayArtifactRefs,
		ProvenanceRecords:   provenanceRecords,
		JournalEvents:       journalEvents,
		ArtifactRefs:        artifactRefs,
		Links: buildCorrelationTraceLinks(
			auditRecords,
			auditArtifactLinks,
			provenanceRecords,
			journalEvents,
			artifactRefs,
			gatewayArtifactRefs,
			artifactRecords,
		),
	}, nil
}

func (s *Server) listProvenanceRecordsByCorrelation(ctx context.Context, correlationID string) ([]correlationProvenanceRecord, error) {
	db, err := s.traceDB()
	if err != nil {
		return nil, err
	}
	rows, err := db.QueryContext(ctx, `
SELECT id, actor, actor_type, source, trace_id, workspace_id, lane_id, selected_paths_json,
       metadata_json, created_at, proposed_by, committed_by, syscall_id, correlation_id, audit_id
FROM provenance_records
WHERE correlation_id = ?
ORDER BY created_at ASC, id ASC`, correlationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []correlationProvenanceRecord{}
	for rows.Next() {
		var rec correlationProvenanceRecord
		var selectedPaths string
		var metadata string
		if err := rows.Scan(
			&rec.ID,
			&rec.Actor,
			&rec.ActorType,
			&rec.Source,
			&rec.TraceID,
			&rec.WorkspaceID,
			&rec.LaneID,
			&selectedPaths,
			&metadata,
			&rec.CreatedAtMs,
			&rec.ProposedBy,
			&rec.CommittedBy,
			&rec.SyscallID,
			&rec.CorrelationID,
			&rec.AuditID,
		); err != nil {
			return nil, err
		}
		rec.SelectedPaths = json.RawMessage(selectedPaths)
		rec.Metadata = json.RawMessage(metadata)
		out = append(out, rec)
	}
	return out, rows.Err()
}

func (s *Server) listJournalEventsByCorrelation(ctx context.Context, correlationID string) ([]correlationJournalEvent, error) {
	db, err := s.traceDB()
	if err != nil {
		return nil, err
	}
	rows, err := db.QueryContext(ctx, `
SELECT id, type, source, actor, workspace_id, lane_id, selected_paths_json, payload_json,
       correlation_id, trace_id, provenance_id, provenance_json, created_at, metadata_json,
       proposed_by, committed_by, syscall_id, audit_id
FROM journal_events
WHERE correlation_id = ?
ORDER BY created_at ASC, id ASC`, correlationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []correlationJournalEvent{}
	for rows.Next() {
		var rec correlationJournalEvent
		var selectedPaths string
		var payload string
		var provenanceID sql.NullString
		var provenance string
		var metadata string
		if err := rows.Scan(
			&rec.ID,
			&rec.Type,
			&rec.Source,
			&rec.Actor,
			&rec.WorkspaceID,
			&rec.LaneID,
			&selectedPaths,
			&payload,
			&rec.CorrelationID,
			&rec.TraceID,
			&provenanceID,
			&provenance,
			&rec.CreatedAtMs,
			&metadata,
			&rec.ProposedBy,
			&rec.CommittedBy,
			&rec.SyscallID,
			&rec.AuditID,
		); err != nil {
			return nil, err
		}
		if provenanceID.Valid {
			v := provenanceID.String
			rec.ProvenanceID = &v
		}
		rec.SelectedPaths = json.RawMessage(selectedPaths)
		rec.Payload = json.RawMessage(payload)
		rec.Provenance = json.RawMessage(provenance)
		rec.Metadata = json.RawMessage(metadata)
		out = append(out, rec)
	}
	return out, rows.Err()
}

func (s *Server) listArtifactRefsByCorrelation(ctx context.Context, correlationID string) ([]correlationArtifactRef, error) {
	db, err := s.traceDB()
	if err != nil {
		return nil, err
	}
	rows, err := db.QueryContext(ctx, `
SELECT id, type, uri, content_hash, workspace_id, lane_id, selected_paths_json, provenance_id,
       provenance_json, created_at, metadata_json, proposed_by, committed_by, syscall_id,
       correlation_id, trace_id, audit_id
FROM artifact_refs
WHERE correlation_id = ?
ORDER BY created_at ASC, id ASC`, correlationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []correlationArtifactRef{}
	for rows.Next() {
		var rec correlationArtifactRef
		var selectedPaths string
		var provenanceID sql.NullString
		var provenance string
		var metadata string
		if err := rows.Scan(
			&rec.ID,
			&rec.Type,
			&rec.URI,
			&rec.ContentHash,
			&rec.WorkspaceID,
			&rec.LaneID,
			&selectedPaths,
			&provenanceID,
			&provenance,
			&rec.CreatedAtMs,
			&metadata,
			&rec.ProposedBy,
			&rec.CommittedBy,
			&rec.SyscallID,
			&rec.CorrelationID,
			&rec.TraceID,
			&rec.AuditID,
		); err != nil {
			return nil, err
		}
		if provenanceID.Valid {
			v := provenanceID.String
			rec.ProvenanceID = &v
		}
		rec.SelectedPaths = json.RawMessage(selectedPaths)
		rec.Provenance = json.RawMessage(provenance)
		rec.Metadata = json.RawMessage(metadata)
		out = append(out, rec)
	}
	return out, rows.Err()
}

func (s *Server) listArtifactRecordsForTrace(ctx context.Context, ids []int64, paths []string) ([]artifacts.Artifact, error) {
	if len(ids) == 0 && len(paths) == 0 {
		return []artifacts.Artifact{}, nil
	}
	db, err := s.traceDB()
	if err != nil {
		return nil, err
	}

	clauses := make([]string, 0, 2)
	args := make([]any, 0, len(ids)+len(paths))
	if len(ids) > 0 {
		clauses = append(clauses, "id IN ("+traceQueryPlaceholders(len(ids))+")")
		for _, id := range ids {
			args = append(args, id)
		}
	}
	if len(paths) > 0 {
		clauses = append(clauses, "file_path IN ("+traceQueryPlaceholders(len(paths))+")")
		for _, path := range paths {
			args = append(args, path)
		}
	}

	rows, err := db.QueryContext(ctx, fmt.Sprintf(`
SELECT id, created_at, job_id, packet_id, type, title, file_path, mime_type, metadata_json
FROM artifacts
WHERE %s
ORDER BY id ASC`, strings.Join(clauses, " OR ")), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []artifacts.Artifact{}
	for rows.Next() {
		var rec artifacts.Artifact
		var jobID sql.NullString
		var packetID sql.NullInt64
		var metadata string
		if err := rows.Scan(
			&rec.ID,
			&rec.CreatedAtMs,
			&jobID,
			&packetID,
			&rec.Type,
			&rec.Title,
			&rec.FilePath,
			&rec.MimeType,
			&metadata,
		); err != nil {
			return nil, err
		}
		if jobID.Valid {
			v := jobID.String
			rec.JobID = &v
		}
		if packetID.Valid {
			v := packetID.Int64
			rec.PacketID = &v
		}
		rec.Metadata = json.RawMessage(metadata)
		out = append(out, rec)
	}
	return out, rows.Err()
}

func collectAuditArtifactIDs(records []audit.Record) ([]int64, []traceAuditArtifactLink) {
	ids := map[int64]struct{}{}
	links := []traceAuditArtifactLink{}

	for _, rec := range records {
		seen := map[int64]struct{}{}
		for _, id := range extractArtifactIDsFromAuditPayload(rec.Payload) {
			if id <= 0 {
				continue
			}
			if _, ok := seen[id]; ok {
				continue
			}
			seen[id] = struct{}{}
			ids[id] = struct{}{}
			links = append(links, traceAuditArtifactLink{
				AuditRecordID: rec.ID,
				ArtifactID:    id,
			})
		}
	}

	idList := make([]int64, 0, len(ids))
	for id := range ids {
		idList = append(idList, id)
	}
	sort.Slice(idList, func(i, j int) bool { return idList[i] < idList[j] })
	sort.Slice(links, func(i, j int) bool {
		if links[i].AuditRecordID == links[j].AuditRecordID {
			return links[i].ArtifactID < links[j].ArtifactID
		}
		return links[i].AuditRecordID < links[j].AuditRecordID
	})
	return idList, links
}

func collectGatewayArtifactRefs(invocations []gateway.InvocationRecord) []correlationGatewayArtifactRef {
	out := []correlationGatewayArtifactRef{}
	for _, inv := range invocations {
		if len(inv.Artifacts) == 0 {
			continue
		}
		var decoded any
		if err := json.Unmarshal(inv.Artifacts, &decoded); err != nil {
			continue
		}
		items, ok := decoded.([]any)
		if !ok {
			continue
		}
		for _, item := range items {
			m, ok := item.(map[string]any)
			if !ok {
				continue
			}
			ref := correlationGatewayArtifactRef{
				GatewayInvocationID: inv.ID,
				ToolID:              inv.ToolID,
				Type:                traceStringFromValue(m["type"]),
				Path:                traceStringFromValue(m["path"]),
				Summary:             traceStringFromValue(m["summary"]),
			}
			if ref.Type == "" && ref.Path == "" && ref.Summary == "" {
				continue
			}
			out = append(out, ref)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].GatewayInvocationID == out[j].GatewayInvocationID {
			return out[i].Path < out[j].Path
		}
		return out[i].GatewayInvocationID < out[j].GatewayInvocationID
	})
	return out
}

func collectTraceArtifactPaths(gatewayRefs []correlationGatewayArtifactRef, artifactRefs []correlationArtifactRef) []string {
	paths := map[string]struct{}{}
	for _, ref := range gatewayRefs {
		path := strings.TrimSpace(ref.Path)
		if path != "" {
			paths[path] = struct{}{}
		}
	}
	for _, ref := range artifactRefs {
		path := strings.TrimSpace(ref.URI)
		if path != "" {
			paths[path] = struct{}{}
		}
	}
	out := make([]string, 0, len(paths))
	for path := range paths {
		out = append(out, path)
	}
	sort.Strings(out)
	return out
}

func buildCorrelationTraceLinks(
	auditRecords []audit.Record,
	auditArtifactLinks []traceAuditArtifactLink,
	provenanceRecords []correlationProvenanceRecord,
	journalEvents []correlationJournalEvent,
	artifactRefs []correlationArtifactRef,
	gatewayArtifactRefs []correlationGatewayArtifactRef,
	artifactRecords []artifacts.Artifact,
) correlationTraceLinks {
	links := correlationTraceLinks{
		AuditToArtifact: auditArtifactLinks,
	}

	auditGatewaySeen := map[string]struct{}{}
	for _, rec := range auditRecords {
		if rec.GatewayInvocationID == nil {
			continue
		}
		link := traceAuditGatewayLink{
			AuditRecordID:       rec.ID,
			GatewayInvocationID: *rec.GatewayInvocationID,
		}
		key := fmt.Sprintf("%d:%d", link.AuditRecordID, link.GatewayInvocationID)
		if _, ok := auditGatewaySeen[key]; ok {
			continue
		}
		auditGatewaySeen[key] = struct{}{}
		links.AuditToGateway = append(links.AuditToGateway, link)
	}
	sort.Slice(links.AuditToGateway, func(i, j int) bool {
		if links.AuditToGateway[i].AuditRecordID == links.AuditToGateway[j].AuditRecordID {
			return links.AuditToGateway[i].GatewayInvocationID < links.AuditToGateway[j].GatewayInvocationID
		}
		return links.AuditToGateway[i].AuditRecordID < links.AuditToGateway[j].AuditRecordID
	})

	provenanceAuditSeen := map[string]struct{}{}
	for _, rec := range provenanceRecords {
		auditID, err := strconv.ParseInt(strings.TrimSpace(rec.AuditID), 10, 64)
		if err != nil || auditID <= 0 {
			continue
		}
		link := traceProvenanceAuditLink{ProvenanceID: rec.ID, AuditRecordID: auditID}
		key := rec.ID + ":" + strconv.FormatInt(auditID, 10)
		if _, ok := provenanceAuditSeen[key]; ok {
			continue
		}
		provenanceAuditSeen[key] = struct{}{}
		links.ProvenanceToAudit = append(links.ProvenanceToAudit, link)
	}
	sort.Slice(links.ProvenanceToAudit, func(i, j int) bool {
		if links.ProvenanceToAudit[i].AuditRecordID == links.ProvenanceToAudit[j].AuditRecordID {
			return links.ProvenanceToAudit[i].ProvenanceID < links.ProvenanceToAudit[j].ProvenanceID
		}
		return links.ProvenanceToAudit[i].AuditRecordID < links.ProvenanceToAudit[j].AuditRecordID
	})

	journalProvenanceSeen := map[string]struct{}{}
	for _, rec := range journalEvents {
		if rec.ProvenanceID == nil || strings.TrimSpace(*rec.ProvenanceID) == "" {
			continue
		}
		link := traceJournalProvenanceLink{
			JournalEventID: rec.ID,
			ProvenanceID:   strings.TrimSpace(*rec.ProvenanceID),
		}
		key := link.JournalEventID + ":" + link.ProvenanceID
		if _, ok := journalProvenanceSeen[key]; ok {
			continue
		}
		journalProvenanceSeen[key] = struct{}{}
		links.JournalToProvenance = append(links.JournalToProvenance, link)
	}
	sort.Slice(links.JournalToProvenance, func(i, j int) bool {
		if links.JournalToProvenance[i].ProvenanceID == links.JournalToProvenance[j].ProvenanceID {
			return links.JournalToProvenance[i].JournalEventID < links.JournalToProvenance[j].JournalEventID
		}
		return links.JournalToProvenance[i].ProvenanceID < links.JournalToProvenance[j].ProvenanceID
	})

	artifactRefProvenanceSeen := map[string]struct{}{}
	for _, rec := range artifactRefs {
		if rec.ProvenanceID == nil || strings.TrimSpace(*rec.ProvenanceID) == "" {
			continue
		}
		link := traceArtifactRefProvenanceLink{
			ArtifactRefID: rec.ID,
			ProvenanceID:  strings.TrimSpace(*rec.ProvenanceID),
		}
		key := link.ArtifactRefID + ":" + link.ProvenanceID
		if _, ok := artifactRefProvenanceSeen[key]; ok {
			continue
		}
		artifactRefProvenanceSeen[key] = struct{}{}
		links.ArtifactRefToProvenance = append(links.ArtifactRefToProvenance, link)
	}
	sort.Slice(links.ArtifactRefToProvenance, func(i, j int) bool {
		if links.ArtifactRefToProvenance[i].ProvenanceID == links.ArtifactRefToProvenance[j].ProvenanceID {
			return links.ArtifactRefToProvenance[i].ArtifactRefID < links.ArtifactRefToProvenance[j].ArtifactRefID
		}
		return links.ArtifactRefToProvenance[i].ProvenanceID < links.ArtifactRefToProvenance[j].ProvenanceID
	})

	artifactIDsByPath := map[string][]int64{}
	for _, art := range artifactRecords {
		path := strings.TrimSpace(art.FilePath)
		if path == "" {
			continue
		}
		artifactIDsByPath[path] = append(artifactIDsByPath[path], art.ID)
	}
	gatewayArtifactSeen := map[string]struct{}{}
	for _, ref := range gatewayArtifactRefs {
		path := strings.TrimSpace(ref.Path)
		if path == "" {
			continue
		}
		for _, artifactID := range artifactIDsByPath[path] {
			link := traceGatewayArtifactLink{
				GatewayInvocationID: ref.GatewayInvocationID,
				ArtifactID:          artifactID,
				Path:                path,
			}
			key := fmt.Sprintf("%d:%d:%s", link.GatewayInvocationID, link.ArtifactID, link.Path)
			if _, ok := gatewayArtifactSeen[key]; ok {
				continue
			}
			gatewayArtifactSeen[key] = struct{}{}
			links.GatewayToArtifact = append(links.GatewayToArtifact, link)
		}
	}
	sort.Slice(links.GatewayToArtifact, func(i, j int) bool {
		if links.GatewayToArtifact[i].GatewayInvocationID == links.GatewayToArtifact[j].GatewayInvocationID {
			if links.GatewayToArtifact[i].ArtifactID == links.GatewayToArtifact[j].ArtifactID {
				return links.GatewayToArtifact[i].Path < links.GatewayToArtifact[j].Path
			}
			return links.GatewayToArtifact[i].ArtifactID < links.GatewayToArtifact[j].ArtifactID
		}
		return links.GatewayToArtifact[i].GatewayInvocationID < links.GatewayToArtifact[j].GatewayInvocationID
	})

	return links
}

func extractArtifactIDsFromAuditPayload(payload json.RawMessage) []int64 {
	if len(payload) == 0 {
		return nil
	}
	var decoded any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		return nil
	}

	ids := map[int64]struct{}{}
	var walk func(value any, key string)
	walk = func(value any, key string) {
		switch typed := value.(type) {
		case map[string]any:
			for childKey, child := range typed {
				norm := normalizeTraceKey(childKey)
				if norm == "artifactid" {
					if id, ok := traceInt64(child); ok && id > 0 {
						ids[id] = struct{}{}
					}
				}
				if norm == "artifactids" {
					if id, ok := traceInt64(child); ok && id > 0 {
						ids[id] = struct{}{}
					}
				}
				walk(child, norm)
			}
		case []any:
			for _, item := range typed {
				walk(item, key)
			}
		default:
			if key == "artifactid" || key == "artifact" {
				if id, ok := traceInt64(typed); ok && id > 0 {
					ids[id] = struct{}{}
				}
			}
		}
	}
	walk(decoded, "")

	out := make([]int64, 0, len(ids))
	for id := range ids {
		out = append(out, id)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

func normalizeTraceKey(key string) string {
	key = strings.ToLower(strings.TrimSpace(key))
	key = strings.NewReplacer("_", "", "-", "", " ", "").Replace(key)
	return key
}

func traceInt64(value any) (int64, bool) {
	switch typed := value.(type) {
	case int:
		return int64(typed), true
	case int8:
		return int64(typed), true
	case int16:
		return int64(typed), true
	case int32:
		return int64(typed), true
	case int64:
		return typed, true
	case uint:
		return int64(typed), true
	case uint8:
		return int64(typed), true
	case uint16:
		return int64(typed), true
	case uint32:
		return int64(typed), true
	case uint64:
		if typed > math.MaxInt64 {
			return 0, false
		}
		return int64(typed), true
	case float64:
		if math.IsNaN(typed) || math.IsInf(typed, 0) || typed != math.Trunc(typed) {
			return 0, false
		}
		return int64(typed), true
	case json.Number:
		v, err := typed.Int64()
		if err != nil {
			return 0, false
		}
		return v, true
	case string:
		if typed == "" {
			return 0, false
		}
		v, err := strconv.ParseInt(strings.TrimSpace(typed), 10, 64)
		if err != nil {
			return 0, false
		}
		return v, true
	default:
		return 0, false
	}
}

func traceStringFromValue(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(typed)
	default:
		return strings.TrimSpace(fmt.Sprint(typed))
	}
}

func traceQueryPlaceholders(n int) string {
	if n <= 0 {
		return ""
	}
	return strings.TrimSuffix(strings.Repeat("?,", n), ",")
}

func (s *Server) traceDB() (*sql.DB, error) {
	if s == nil || s.st == nil || s.st.DB == nil {
		return nil, fmt.Errorf("store unavailable")
	}
	return s.st.DB, nil
}
