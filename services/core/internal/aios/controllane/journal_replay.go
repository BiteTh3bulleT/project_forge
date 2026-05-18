package controllane

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
	"strings"

	"forge/projectforge/services/core/internal/aios/domain"
)

const JournalReplaySchemaVersion = "journal-replay-v1"

type JournalReplayRecord struct {
	SchemaVersion string `json:"schemaVersion"`
	EventID       string `json:"eventId"`
	EventType     string `json:"eventType"`
	Timestamp     int64  `json:"timestamp"`
	WorkspaceID   string `json:"workspaceId"`
	LaneID        string `json:"laneId,omitempty"`
	CorrelationID string `json:"correlationId,omitempty"`
	TraceID       string `json:"traceId,omitempty"`
	Source        string `json:"source"`
	PayloadHash   string `json:"payloadHash"`
	PreviousHash  string `json:"previousHash,omitempty"`
	EventHash     string `json:"eventHash"`
}

type JournalReplayReport struct {
	SchemaVersion string                `json:"schemaVersion"`
	Passed        bool                  `json:"passed"`
	EventCount    int                   `json:"eventCount"`
	HeadHash      string                `json:"headHash,omitempty"`
	Records       []JournalReplayRecord `json:"records"`
	Mismatches    []JournalReplayIssue  `json:"mismatches,omitempty"`
	Warnings      []string              `json:"warnings,omitempty"`
}

type JournalReplayIssue struct {
	EventID  string `json:"eventId"`
	Field    string `json:"field"`
	Message  string `json:"message"`
	Expected string `json:"expected,omitempty"`
	Actual   string `json:"actual,omitempty"`
}

func BuildJournalReplayReport(events []domain.JournalEvent, expectedHeadHash string) JournalReplayReport {
	ordered := sortedJournalReplayEvents(events)
	records := make([]JournalReplayRecord, 0, len(ordered))
	var previousHash string
	for _, evt := range ordered {
		record := journalReplayRecord(evt, previousHash)
		previousHash = record.EventHash
		records = append(records, record)
	}
	report := JournalReplayReport{
		SchemaVersion: JournalReplaySchemaVersion,
		Passed:        true,
		EventCount:    len(records),
		Records:       records,
	}
	if len(records) > 0 {
		report.HeadHash = records[len(records)-1].EventHash
	}
	if strings.TrimSpace(expectedHeadHash) != "" && strings.TrimSpace(expectedHeadHash) != report.HeadHash {
		report.Passed = false
		report.Mismatches = append(report.Mismatches, JournalReplayIssue{
			EventID:  report.lastEventID(),
			Field:    "headHash",
			Message:  "journal replay head hash mismatch",
			Expected: strings.TrimSpace(expectedHeadHash),
			Actual:   report.HeadHash,
		})
	}
	return report
}

func (r JournalReplayReport) lastEventID() string {
	if len(r.Records) == 0 {
		return ""
	}
	return r.Records[len(r.Records)-1].EventID
}

func sortedJournalReplayEvents(events []domain.JournalEvent) []domain.JournalEvent {
	out := append([]domain.JournalEvent{}, events...)
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Timestamp != out[j].Timestamp {
			return out[i].Timestamp < out[j].Timestamp
		}
		return out[i].ID < out[j].ID
	})
	return out
}

func journalReplayRecord(evt domain.JournalEvent, previousHash string) JournalReplayRecord {
	payloadHash := stableHash(map[string]any{
		"payload": evt.Payload,
	})
	record := JournalReplayRecord{
		SchemaVersion: JournalReplaySchemaVersion,
		EventID:       strings.TrimSpace(evt.ID),
		EventType:     strings.TrimSpace(evt.Type),
		Timestamp:     evt.Timestamp,
		WorkspaceID:   strings.TrimSpace(evt.Scope.WorkspaceID),
		LaneID:        strings.TrimSpace(evt.Scope.LaneID),
		CorrelationID: strings.TrimSpace(evt.CorrelationID),
		TraceID:       strings.TrimSpace(evt.Provenance.TraceID),
		Source:        strings.TrimSpace(evt.Source),
		PayloadHash:   payloadHash,
		PreviousHash:  previousHash,
	}
	record.EventHash = stableHash(map[string]any{
		"schemaVersion": record.SchemaVersion,
		"eventId":       record.EventID,
		"eventType":     record.EventType,
		"timestamp":     record.Timestamp,
		"workspaceId":   record.WorkspaceID,
		"laneId":        record.LaneID,
		"correlationId": record.CorrelationID,
		"traceId":       record.TraceID,
		"source":        record.Source,
		"payloadHash":   record.PayloadHash,
		"previousHash":  record.PreviousHash,
	})
	return record
}

func stableHash(value any) string {
	data, _ := json.Marshal(value)
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}
