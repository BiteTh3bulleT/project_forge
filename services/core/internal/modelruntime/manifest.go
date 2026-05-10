package modelruntime

import (
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
)

const ManifestFilename = "model.forge.json"

var (
	ErrManifestInvalid         = errors.New("model manifest invalid")
	ErrManifestMissingRequired = errors.New("model manifest missing required field")
	ErrUnsupportedModelFormat  = errors.New("model manifest uses unsupported format")
	ErrUnsupportedBackend      = errors.New("model manifest uses unsupported backend")
	ErrUnknownCapability       = errors.New("model manifest contains unknown capability")
)

// ParseManifest parses and normalizes a manifest payload from JSON.
func ParseManifest(raw []byte) (ModelManifest, error) {
	var decoded manifestJSON
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return ModelManifest{}, fmt.Errorf("%w: decode json: %v", ErrManifestInvalid, err)
	}

	m, err := decoded.toManifest()
	if err != nil {
		return ModelManifest{}, err
	}
	normalizeManifest(&m)
	if err := ValidateManifest(m); err != nil {
		return ModelManifest{}, err
	}
	return m, nil
}

// ReadManifest reads and validates a manifest file path.
func ReadManifest(path string) (ModelManifest, error) {
	body, err := readRuntimeJSONFile(path, "model manifest")
	if err != nil {
		return ModelManifest{}, fmt.Errorf("%w: read %s: %v", ErrManifestInvalid, path, err)
	}
	m, err := ParseManifest(body)
	if err != nil {
		return ModelManifest{}, err
	}
	return m, nil
}

// ValidateManifest validates required fields and enum values.
func ValidateManifest(m ModelManifest) error {
	if strings.TrimSpace(m.SchemaVersion) == "" {
		return fmt.Errorf("%w: schemaVersion", ErrManifestMissingRequired)
	}
	if strings.TrimSpace(m.ID) == "" {
		return fmt.Errorf("%w: id", ErrManifestMissingRequired)
	}
	if strings.TrimSpace(m.DisplayName) == "" {
		return fmt.Errorf("%w: displayName", ErrManifestMissingRequired)
	}
	if strings.TrimSpace(m.Family) == "" {
		return fmt.Errorf("%w: family", ErrManifestMissingRequired)
	}
	if strings.TrimSpace(m.FilePath) == "" {
		return fmt.Errorf("%w: file/path", ErrManifestMissingRequired)
	}
	if _, ok := validFormats[m.Format]; !ok {
		return fmt.Errorf("%w: %q", ErrUnsupportedModelFormat, m.Format)
	}
	if m.Format == ModelFormatUnknown {
		return fmt.Errorf("%w: %q", ErrUnsupportedModelFormat, m.Format)
	}
	if _, ok := validBackends[m.Backend]; !ok {
		return fmt.Errorf("%w: %q", ErrUnsupportedBackend, m.Backend)
	}
	if len(m.Capabilities) == 0 {
		return fmt.Errorf("%w: capabilities", ErrManifestMissingRequired)
	}
	for _, cap := range m.Capabilities {
		if _, ok := validCapabilities[cap]; !ok {
			return fmt.Errorf("%w: %q", ErrUnknownCapability, cap)
		}
	}
	if m.SizeBytes < 0 {
		return fmt.Errorf("%w: sizeBytes must be >= 0", ErrManifestInvalid)
	}
	if m.ContextLength <= 0 {
		return fmt.Errorf("%w: contextLength must be > 0", ErrManifestInvalid)
	}
	if strings.TrimSpace(m.Quantization) == "" {
		return fmt.Errorf("%w: quantization", ErrManifestMissingRequired)
	}
	if strings.TrimSpace(m.License) == "" {
		return fmt.Errorf("%w: license", ErrManifestMissingRequired)
	}
	return nil
}

// ManifestModelPath returns the model file/path value from the manifest.
func ManifestModelPath(m ModelManifest) string {
	return strings.TrimSpace(m.FilePath)
}

func normalizeManifest(m *ModelManifest) {
	m.SchemaVersion = strings.TrimSpace(m.SchemaVersion)
	m.ID = strings.TrimSpace(m.ID)
	m.DisplayName = strings.TrimSpace(m.DisplayName)
	m.Family = strings.TrimSpace(m.Family)
	m.FilePath = strings.TrimSpace(m.FilePath)
	m.SHA256 = normalizeSHA256(m.SHA256)
	m.Quantization = strings.TrimSpace(m.Quantization)
	m.License = strings.TrimSpace(m.License)
	if m.Metadata == nil {
		m.Metadata = map[string]any{}
	}
	if m.Format == "" {
		m.Format = ModelFormatUnknown
	}
	m.Format = ParseModelFormat(string(m.Format))
	m.Backend = ParseModelBackendKind(string(m.Backend))
	m.Capabilities = normalizeCapabilities(m.Capabilities)
}

// ParseModelFormat normalizes known format values and converts unknown values to "unknown".
func ParseModelFormat(raw string) ModelFormat {
	s := strings.TrimSpace(strings.ToLower(raw))
	switch ModelFormat(s) {
	case ModelFormatGGUF, ModelFormatSafeTensor, ModelFormatONNX:
		return ModelFormat(s)
	default:
		return ModelFormatUnknown
	}
}

// ParseModelBackendKind normalizes known backends and preserves unknown values.
func ParseModelBackendKind(raw string) ModelBackendKind {
	s := strings.TrimSpace(strings.ToLower(raw))
	if s == "" {
		return ""
	}
	return ModelBackendKind(s)
}

func normalizeCapabilities(input []ModelCapability) []ModelCapability {
	if len(input) == 0 {
		return nil
	}
	seen := make(map[ModelCapability]struct{}, len(input))
	out := make([]ModelCapability, 0, len(input))
	for _, cap := range input {
		value := ModelCapability(strings.TrimSpace(strings.ToLower(string(cap))))
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

func normalizeSHA256(raw string) string {
	s := strings.TrimSpace(strings.ToLower(raw))
	s = strings.TrimPrefix(s, "sha256:")
	return s
}

func safeJoinWithinBase(base, entry string) (string, error) {
	if strings.TrimSpace(entry) == "" {
		return "", fmt.Errorf("%w: file/path", ErrManifestMissingRequired)
	}
	candidate := strings.TrimSpace(entry)
	if !filepath.IsAbs(candidate) {
		candidate = filepath.Join(base, candidate)
	}
	absBase, err := filepath.Abs(base)
	if err != nil {
		return "", fmt.Errorf("resolve base: %w", err)
	}
	absCandidate, err := filepath.Abs(candidate)
	if err != nil {
		return "", fmt.Errorf("resolve candidate: %w", err)
	}
	rel, err := filepath.Rel(absBase, absCandidate)
	if err != nil {
		return "", fmt.Errorf("rel path: %w", err)
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("%w: model file path escapes model directory", ErrManifestInvalid)
	}
	return absCandidate, nil
}

type manifestJSON struct {
	SchemaVersion  string          `json:"schemaVersion"`
	ID             string          `json:"id"`
	DisplayName    string          `json:"displayName"`
	Family         string          `json:"family"`
	Format         string          `json:"format"`
	Backend        string          `json:"backend"`
	FilePath       string          `json:"filePath"`
	File           string          `json:"file"`
	Path           string          `json:"path"`
	SHA256         string          `json:"sha256"`
	SizeBytes      int64           `json:"sizeBytes"`
	Quantization   string          `json:"quantization"`
	ContextLength  int             `json:"contextLength"`
	Capabilities   []string        `json:"capabilities"`
	DefaultRuntime runtimeJSON     `json:"defaultRuntime"`
	License        json.RawMessage `json:"license"`
	Metadata       map[string]any  `json:"metadata"`
}

type runtimeJSON struct {
	MaxPromptTokens int            `json:"maxPromptTokens"`
	MaxOutputTokens int            `json:"maxOutputTokens"`
	MaxTokens       int            `json:"maxTokens"`
	TimeoutMs       int            `json:"timeoutMs"`
	Temperature     float64        `json:"temperature"`
	TopP            float64        `json:"topP"`
	Metadata        map[string]any `json:"metadata"`
}

func (m manifestJSON) toManifest() (ModelManifest, error) {
	licenseName, licenseMeta, err := parseLicenseField(m.License)
	if err != nil {
		return ModelManifest{}, err
	}

	modelPath := strings.TrimSpace(m.FilePath)
	if modelPath == "" {
		modelPath = strings.TrimSpace(m.File)
	}
	if modelPath == "" {
		modelPath = strings.TrimSpace(m.Path)
	}

	capabilities := make([]ModelCapability, 0, len(m.Capabilities))
	for _, cap := range m.Capabilities {
		capabilities = append(capabilities, ModelCapability(cap))
	}

	metadata := m.Metadata
	if metadata == nil {
		metadata = map[string]any{}
	}
	for key, value := range licenseMeta {
		if _, exists := metadata[key]; !exists {
			metadata[key] = value
		}
	}

	defaultRuntime := ModelRuntimeDefaults{
		MaxPromptTokens: m.DefaultRuntime.MaxPromptTokens,
		MaxOutputTokens: m.DefaultRuntime.MaxOutputTokens,
		TimeoutMs:       m.DefaultRuntime.TimeoutMs,
		Temperature:     m.DefaultRuntime.Temperature,
		TopP:            m.DefaultRuntime.TopP,
		Metadata:        m.DefaultRuntime.Metadata,
	}
	if defaultRuntime.MaxOutputTokens <= 0 && m.DefaultRuntime.MaxTokens > 0 {
		defaultRuntime.MaxOutputTokens = m.DefaultRuntime.MaxTokens
	}

	return ModelManifest{
		SchemaVersion:  m.SchemaVersion,
		ID:             m.ID,
		DisplayName:    m.DisplayName,
		Family:         m.Family,
		Format:         ParseModelFormat(m.Format),
		Backend:        ParseModelBackendKind(m.Backend),
		FilePath:       modelPath,
		SHA256:         m.SHA256,
		SizeBytes:      m.SizeBytes,
		Quantization:   m.Quantization,
		ContextLength:  m.ContextLength,
		Capabilities:   capabilities,
		DefaultRuntime: defaultRuntime,
		License:        licenseName,
		Metadata:       metadata,
	}, nil
}

func parseLicenseField(raw json.RawMessage) (string, map[string]any, error) {
	if len(raw) == 0 {
		return "", nil, nil
	}
	var asString string
	if err := json.Unmarshal(raw, &asString); err == nil {
		return strings.TrimSpace(asString), nil, nil
	}
	var asObject struct {
		Name string `json:"name"`
		URL  string `json:"url"`
	}
	if err := json.Unmarshal(raw, &asObject); err != nil {
		return "", nil, fmt.Errorf("%w: decode license: %v", ErrManifestInvalid, err)
	}
	meta := map[string]any{}
	if strings.TrimSpace(asObject.URL) != "" {
		meta["licenseUrl"] = strings.TrimSpace(asObject.URL)
	}
	return strings.TrimSpace(asObject.Name), meta, nil
}
