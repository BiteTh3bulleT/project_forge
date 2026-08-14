// Package journal defines the production FORGE-K journal integrity contract.
//
// This package is deliberately pure: it plans entries and verifies hash chains,
// but it does not read or write storage. Persistence adapters must atomically
// compare the supplied head and append the planned entry.
package journal

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"strings"
	"unicode/utf8"
)

const SchemaVersion = "forge-k-journal-chain-v1"

var (
	ErrInvalidInput     = errors.New("invalid journal input")
	ErrInvalidHead      = errors.New("invalid journal head")
	ErrSequenceOverflow = errors.New("journal sequence overflow")
	ErrDuplicateJSONKey = errors.New("duplicate JSON object key")
)

// Head is the independently persisted identity of the last committed entry.
// Its zero value represents an empty journal.
type Head struct {
	Sequence uint64 `json:"sequence"`
	EventID  string `json:"eventId,omitempty"`
	Hash     string `json:"hash,omitempty"`
}

// AppendInput is immutable journal content supplied to the kernel. Hash fields
// bind potentially large JSON documents without putting them in the hot chain.
// Each hash must be produced from its canonical persisted representation.
type AppendInput struct {
	EventID        string   `json:"eventId"`
	EventType      string   `json:"eventType"`
	Source         string   `json:"source"`
	Actor          string   `json:"actor,omitempty"`
	WorkspaceID    string   `json:"workspaceId"`
	LaneID         string   `json:"laneId,omitempty"`
	SelectedPaths  []string `json:"selectedPaths"`
	CorrelationID  string   `json:"correlationId,omitempty"`
	TraceID        string   `json:"traceId,omitempty"`
	ProvenanceID   string   `json:"provenanceId,omitempty"`
	ProvenanceHash string   `json:"provenanceHash"`
	PayloadHash    string   `json:"payloadHash"`
	MetadataHash   string   `json:"metadataHash"`
	ProposedBy     string   `json:"proposedBy,omitempty"`
	CommittedBy    string   `json:"committedBy"`
	SyscallID      string   `json:"syscallId"`
	AuditID        string   `json:"auditId,omitempty"`
	CreatedAt      int64    `json:"createdAt"`
}

// Entry is the append plan persisted by a durable journal adapter. Hash is not
// part of its own canonical input; all other fields, including PriorHash, are.
type Entry struct {
	SchemaVersion string `json:"schemaVersion"`
	Sequence      uint64 `json:"sequence"`
	AppendInput
	PriorHash string `json:"priorHash,omitempty"`
	Hash      string `json:"hash"`
}

type IssueCode string

const (
	IssueInvalidEntry      IssueCode = "invalid_entry"
	IssueDuplicateEventID  IssueCode = "duplicate_event_id"
	IssueDuplicateHash     IssueCode = "duplicate_hash"
	IssueSequenceMismatch  IssueCode = "sequence_mismatch"
	IssuePriorHashMismatch IssueCode = "prior_hash_mismatch"
	IssueHashMismatch      IssueCode = "hash_mismatch"
	IssueHeadDivergence    IssueCode = "head_divergence"
)

type Issue struct {
	Index    int       `json:"index"`
	EventID  string    `json:"eventId,omitempty"`
	Code     IssueCode `json:"code"`
	Field    string    `json:"field"`
	Message  string    `json:"message"`
	Expected string    `json:"expected,omitempty"`
	Actual   string    `json:"actual,omitempty"`
}

type VerificationReport struct {
	Passed     bool    `json:"passed"`
	EntryCount int     `json:"entryCount"`
	BaseHead   Head    `json:"baseHead"`
	Head       Head    `json:"head"`
	Issues     []Issue `json:"issues"`
}

// PlanAppend deterministically derives the only entry that may follow head.
// It does not reserve a sequence or perform a write.
func PlanAppend(head Head, input AppendInput) (Entry, error) {
	if err := ValidateHead(head); err != nil {
		return Entry{}, err
	}
	input.SelectedPaths = append([]string{}, input.SelectedPaths...)
	if err := validateInput(input); err != nil {
		return Entry{}, err
	}
	if head.Sequence == math.MaxUint64 {
		return Entry{}, ErrSequenceOverflow
	}
	if head.EventID != "" && head.EventID == input.EventID {
		return Entry{}, fmt.Errorf("%w: eventId repeats current head", ErrInvalidInput)
	}
	entry := Entry{
		SchemaVersion: SchemaVersion,
		Sequence:      head.Sequence + 1,
		AppendInput:   input,
		PriorHash:     head.Hash,
	}
	hash, err := HashEntry(entry)
	if err != nil {
		return Entry{}, err
	}
	entry.Hash = hash
	return entry, nil
}

// HashEntry returns the digest over CanonicalBytes. The stored Hash field is
// intentionally ignored, allowing an entry to be checked after retrieval.
func HashEntry(entry Entry) (string, error) {
	canonical, err := CanonicalBytes(entry)
	if err != nil {
		return "", err
	}
	return digest(canonical), nil
}

// CanonicalBytes is the versioned wire contract for chain hashing. A fixed
// struct (rather than a map) makes field order explicit and stable.
func CanonicalBytes(entry Entry) ([]byte, error) {
	entry.SelectedPaths = append([]string{}, entry.SelectedPaths...)
	if err := validateEntryShape(entry); err != nil {
		return nil, err
	}
	type canonicalEntry struct {
		SchemaVersion  string   `json:"schemaVersion"`
		Sequence       uint64   `json:"sequence"`
		EventID        string   `json:"eventId"`
		EventType      string   `json:"eventType"`
		Source         string   `json:"source"`
		Actor          string   `json:"actor"`
		WorkspaceID    string   `json:"workspaceId"`
		LaneID         string   `json:"laneId"`
		SelectedPaths  []string `json:"selectedPaths"`
		CorrelationID  string   `json:"correlationId"`
		TraceID        string   `json:"traceId"`
		ProvenanceID   string   `json:"provenanceId"`
		ProvenanceHash string   `json:"provenanceHash"`
		PayloadHash    string   `json:"payloadHash"`
		MetadataHash   string   `json:"metadataHash"`
		ProposedBy     string   `json:"proposedBy"`
		CommittedBy    string   `json:"committedBy"`
		SyscallID      string   `json:"syscallId"`
		AuditID        string   `json:"auditId"`
		CreatedAt      int64    `json:"createdAt"`
		PriorHash      string   `json:"priorHash"`
	}
	return json.Marshal(canonicalEntry{
		SchemaVersion: entry.SchemaVersion, Sequence: entry.Sequence,
		EventID: entry.EventID, EventType: entry.EventType, Source: entry.Source,
		Actor: entry.Actor, WorkspaceID: entry.WorkspaceID, LaneID: entry.LaneID,
		SelectedPaths: entry.SelectedPaths, CorrelationID: entry.CorrelationID,
		TraceID: entry.TraceID, ProvenanceID: entry.ProvenanceID,
		ProvenanceHash: entry.ProvenanceHash, PayloadHash: entry.PayloadHash,
		MetadataHash: entry.MetadataHash, ProposedBy: entry.ProposedBy,
		CommittedBy: entry.CommittedBy, SyscallID: entry.SyscallID,
		AuditID: entry.AuditID, CreatedAt: entry.CreatedAt, PriorHash: entry.PriorHash,
	})
}

// Verify checks a complete chain from genesis. expected == nil checks internal
// integrity only; a non-nil expected head also detects a consistently rehashed
// fork or truncation.
func Verify(entries []Entry, expected *Head) VerificationReport {
	return VerifyFrom(Head{}, entries, expected)
}

// VerifyFrom checks an ordered continuation after base. Entries are never
// sorted: storage reorder is an integrity failure, not an ordering hint.
func VerifyFrom(base Head, entries []Entry, expected *Head) VerificationReport {
	report := VerificationReport{
		Passed: true, EntryCount: len(entries), BaseHead: base, Head: base,
		Issues: []Issue{},
	}
	if err := ValidateHead(base); err != nil {
		report.Passed = false
		report.Issues = append(report.Issues, Issue{Index: -1, Code: IssueInvalidEntry, Field: "baseHead", Message: err.Error()})
		return report
	}
	if expected != nil {
		if err := ValidateHead(*expected); err != nil {
			report.Passed = false
			report.Issues = append(report.Issues, Issue{Index: -1, Code: IssueInvalidEntry, Field: "expectedHead", Message: err.Error()})
			return report
		}
	}
	seenIDs := make(map[string]int, len(entries))
	seenHashes := make(map[string]int, len(entries))
	if base.EventID != "" {
		seenIDs[base.EventID] = -1
	}
	if base.Hash != "" {
		seenHashes[base.Hash] = -1
	}
	previous := base
	for index, entry := range entries {
		if first, ok := seenIDs[entry.EventID]; ok {
			report.add(Issue{Index: index, EventID: entry.EventID, Code: IssueDuplicateEventID, Field: "eventId", Message: fmt.Sprintf("event id already appeared at index %d", first), Actual: entry.EventID})
		} else {
			seenIDs[entry.EventID] = index
		}
		if first, ok := seenHashes[entry.Hash]; entry.Hash != "" && ok {
			report.add(Issue{Index: index, EventID: entry.EventID, Code: IssueDuplicateHash, Field: "hash", Message: fmt.Sprintf("entry hash already appeared at index %d", first), Actual: entry.Hash})
		} else if entry.Hash != "" {
			seenHashes[entry.Hash] = index
		}
		if err := validateEntryShape(entry); err != nil {
			report.add(Issue{Index: index, EventID: entry.EventID, Code: IssueInvalidEntry, Field: "entry", Message: err.Error()})
		}
		expectedSequence := previous.Sequence + 1
		if previous.Sequence == math.MaxUint64 || entry.Sequence != expectedSequence {
			report.add(Issue{Index: index, EventID: entry.EventID, Code: IssueSequenceMismatch, Field: "sequence", Message: "entry sequence is not the next committed sequence", Expected: fmt.Sprint(expectedSequence), Actual: fmt.Sprint(entry.Sequence)})
		}
		if entry.PriorHash != previous.Hash {
			report.add(Issue{Index: index, EventID: entry.EventID, Code: IssuePriorHashMismatch, Field: "priorHash", Message: "entry does not link to the preceding stored entry", Expected: previous.Hash, Actual: entry.PriorHash})
		}
		calculated, err := HashEntry(entry)
		if err == nil && entry.Hash != calculated {
			report.add(Issue{Index: index, EventID: entry.EventID, Code: IssueHashMismatch, Field: "hash", Message: "entry content does not match its stored hash", Expected: calculated, Actual: entry.Hash})
		}
		previous = Head{Sequence: entry.Sequence, EventID: entry.EventID, Hash: entry.Hash}
	}
	report.Head = previous
	if expected != nil && report.Head != *expected {
		report.add(Issue{Index: len(entries), EventID: report.Head.EventID, Code: IssueHeadDivergence, Field: "head", Message: "verified chain head differs from the independently recorded head", Expected: formatHead(*expected), Actual: formatHead(report.Head)})
	}
	return report
}

func (r *VerificationReport) add(issue Issue) {
	r.Passed = false
	r.Issues = append(r.Issues, issue)
}

// ValidateHead rejects partial heads, which could otherwise make append
// planning silently start from an ambiguous position.
func ValidateHead(head Head) error {
	if head == (Head{}) {
		return nil
	}
	if head.Sequence == 0 || !validRequired(head.EventID) || !validDigest(head.Hash) {
		return ErrInvalidHead
	}
	return nil
}

func validateEntryShape(entry Entry) error {
	if entry.SchemaVersion != SchemaVersion {
		return fmt.Errorf("%w: unsupported schemaVersion", ErrInvalidInput)
	}
	if entry.Sequence == 0 {
		return fmt.Errorf("%w: sequence must be positive", ErrInvalidInput)
	}
	if entry.Sequence == 1 && entry.PriorHash != "" {
		return fmt.Errorf("%w: genesis priorHash must be empty", ErrInvalidInput)
	}
	if entry.Sequence > 1 && !validDigest(entry.PriorHash) {
		return fmt.Errorf("%w: non-genesis priorHash must be a sha256 digest", ErrInvalidInput)
	}
	return validateInput(entry.AppendInput)
}

func validateInput(input AppendInput) error {
	requiredFields := []struct{ field, value string }{
		{"eventId", input.EventID}, {"eventType", input.EventType},
		{"source", input.Source}, {"workspaceId", input.WorkspaceID},
		{"committedBy", input.CommittedBy}, {"syscallId", input.SyscallID},
	}
	for _, candidate := range requiredFields {
		field, value := candidate.field, candidate.value
		if !validRequired(value) {
			return fmt.Errorf("%w: %s is required and must be whitespace-normalized", ErrInvalidInput, field)
		}
	}
	optionalFields := []struct{ field, value string }{
		{"actor", input.Actor}, {"laneId", input.LaneID},
		{"correlationId", input.CorrelationID}, {"traceId", input.TraceID},
		{"provenanceId", input.ProvenanceID}, {"proposedBy", input.ProposedBy},
		{"auditId", input.AuditID},
	}
	for _, candidate := range optionalFields {
		field, value := candidate.field, candidate.value
		if value != strings.TrimSpace(value) {
			return fmt.Errorf("%w: %s must be whitespace-normalized", ErrInvalidInput, field)
		}
	}
	if input.CreatedAt <= 0 {
		return fmt.Errorf("%w: createdAt must be positive", ErrInvalidInput)
	}
	hashFields := []struct{ field, value string }{
		{"payloadHash", input.PayloadHash}, {"provenanceHash", input.ProvenanceHash},
		{"metadataHash", input.MetadataHash},
	}
	for _, candidate := range hashFields {
		field, value := candidate.field, candidate.value
		if !validDigest(value) {
			return fmt.Errorf("%w: %s must be a lowercase sha256 digest", ErrInvalidInput, field)
		}
	}
	seen := make(map[string]struct{}, len(input.SelectedPaths))
	for _, path := range input.SelectedPaths {
		if !validRequired(path) {
			return fmt.Errorf("%w: selectedPaths must contain normalized non-empty values", ErrInvalidInput)
		}
		if _, ok := seen[path]; ok {
			return fmt.Errorf("%w: selectedPaths contains duplicate %q", ErrInvalidInput, path)
		}
		seen[path] = struct{}{}
	}
	return nil
}

func validRequired(value string) bool {
	return value != "" && value == strings.TrimSpace(value)
}

func validDigest(value string) bool {
	if len(value) != len("sha256:")+sha256.Size*2 || !strings.HasPrefix(value, "sha256:") {
		return false
	}
	raw := strings.TrimPrefix(value, "sha256:")
	decoded, err := hex.DecodeString(raw)
	return err == nil && len(decoded) == sha256.Size && raw == strings.ToLower(raw)
}

func digest(value []byte) string {
	sum := sha256.Sum256(value)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func formatHead(head Head) string {
	encoded, _ := json.Marshal(head)
	return string(encoded)
}

// CanonicalJSON removes insignificant whitespace, orders object keys through
// encoding/json's stable map encoding, preserves array order and JSON number
// lexemes, and rejects duplicate object keys and non-UTF-8 input.
func CanonicalJSON(raw []byte) ([]byte, error) {
	if !utf8.Valid(raw) {
		return nil, fmt.Errorf("%w: JSON must be UTF-8", ErrInvalidInput)
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	value, err := decodeJSONValue(decoder)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidInput, err)
	}
	if token, err := decoder.Token(); err != io.EOF {
		if err == nil {
			return nil, fmt.Errorf("%w: trailing JSON token %v", ErrInvalidInput, token)
		}
		return nil, fmt.Errorf("%w: %v", ErrInvalidInput, err)
	}
	return json.Marshal(value)
}

func HashJSON(raw []byte) (string, error) {
	canonical, err := CanonicalJSON(raw)
	if err != nil {
		return "", err
	}
	return digest(canonical), nil
}

func decodeJSONValue(decoder *json.Decoder) (any, error) {
	token, err := decoder.Token()
	if err != nil {
		return nil, err
	}
	delim, ok := token.(json.Delim)
	if !ok {
		return token, nil
	}
	switch delim {
	case '{':
		object := map[string]any{}
		for decoder.More() {
			keyToken, err := decoder.Token()
			if err != nil {
				return nil, err
			}
			key, ok := keyToken.(string)
			if !ok {
				return nil, errors.New("object key is not a string")
			}
			if _, exists := object[key]; exists {
				return nil, fmt.Errorf("%w: %q", ErrDuplicateJSONKey, key)
			}
			value, err := decodeJSONValue(decoder)
			if err != nil {
				return nil, err
			}
			object[key] = value
		}
		end, err := decoder.Token()
		if err != nil || end != json.Delim('}') {
			return nil, errors.New("unterminated JSON object")
		}
		return object, nil
	case '[':
		array := []any{}
		for decoder.More() {
			value, err := decodeJSONValue(decoder)
			if err != nil {
				return nil, err
			}
			array = append(array, value)
		}
		end, err := decoder.Token()
		if err != nil || end != json.Delim(']') {
			return nil, errors.New("unterminated JSON array")
		}
		return array, nil
	default:
		return nil, fmt.Errorf("unexpected JSON delimiter %q", delim)
	}
}
