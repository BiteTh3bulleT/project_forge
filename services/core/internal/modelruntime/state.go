package modelruntime

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
)

const (
	ModelStateFilename      = "model.state.json"
	modelStateSchemaVersion = "forge.model.state/v1"
)

// ModelState persists management metadata alongside a managed model directory.
type ModelState struct {
	SchemaVersion string         `json:"schemaVersion"`
	Status        ModelStatus    `json:"status,omitempty"`
	SourcePath    string         `json:"sourcePath,omitempty"`
	ImportedAt    time.Time      `json:"importedAt,omitempty"`
	VerifiedAt    time.Time      `json:"verifiedAt,omitempty"`
	DisabledAt    time.Time      `json:"disabledAt,omitempty"`
	ArchivedAt    time.Time      `json:"archivedAt,omitempty"`
	Preferred     bool           `json:"preferred,omitempty"`
	Managed       bool           `json:"managed,omitempty"`
	Metadata      map[string]any `json:"metadata,omitempty"`
}

func (s *ModelState) Normalize() {
	s.SchemaVersion = strings.TrimSpace(s.SchemaVersion)
	if s.SchemaVersion == "" {
		s.SchemaVersion = modelStateSchemaVersion
	}
	if _, ok := validStatuses[s.Status]; !ok {
		s.Status = ""
	}
	s.SourcePath = strings.TrimSpace(s.SourcePath)
	if s.Metadata == nil {
		s.Metadata = map[string]any{}
	}
}

func defaultImportedState(sourcePath string, preferred bool, metadata map[string]any) ModelState {
	state := ModelState{
		SchemaVersion: modelStateSchemaVersion,
		Status:        StatusImported,
		SourcePath:    strings.TrimSpace(sourcePath),
		ImportedAt:    time.Now().UTC(),
		Preferred:     preferred,
		Managed:       true,
		Metadata:      cloneStateMetadata(metadata),
	}
	state.Normalize()
	return state
}

func stateStatusOrDefault(state ModelState, fallback ModelStatus) ModelStatus {
	state.Normalize()
	switch {
	case state.Status != "":
		return state.Status
	case !state.ArchivedAt.IsZero():
		return StatusArchived
	case !state.DisabledAt.IsZero():
		return StatusDisabled
	case !state.VerifiedAt.IsZero():
		return StatusVerified
	case !state.ImportedAt.IsZero():
		return StatusImported
	case fallback != "":
		return fallback
	default:
		return StatusAvailable
	}
}

func readModelState(path string) (ModelState, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return ModelState{}, nil
		}
		return ModelState{}, fmt.Errorf("read model state: %w", err)
	}
	var state ModelState
	if err := json.Unmarshal(body, &state); err != nil {
		return ModelState{}, fmt.Errorf("decode model state: %w", err)
	}
	state.Normalize()
	return state, nil
}

func writeModelState(path string, state ModelState) error {
	state.Normalize()
	body, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("encode model state: %w", err)
	}
	body = append(body, '\n')
	if err := os.WriteFile(path, body, 0o644); err != nil {
		return fmt.Errorf("write model state: %w", err)
	}
	return nil
}

func cloneStateMetadata(input map[string]any) map[string]any {
	if len(input) == 0 {
		return map[string]any{}
	}
	out := make(map[string]any, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}
