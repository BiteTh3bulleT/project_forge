package modelruntime

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type ImportModelOptions struct {
	ID            string
	DisplayName   string
	Family        string
	Backend       ModelBackendKind
	Capabilities  []ModelCapability
	License       string
	Quantization  string
	ContextLength int
	Preferred     bool
	Metadata      map[string]any
}

type ImportModelResult struct {
	Model       StoredModel
	Duplicate   bool
	ManagedPath string
	SourcePath  string
}

func (s *ModelStore) Import(ctx context.Context, inputPath string, opts ImportModelOptions) (ImportModelResult, error) {
	_ = ctx
	absInput, err := filepath.Abs(strings.TrimSpace(inputPath))
	if err != nil {
		return ImportModelResult{}, fmt.Errorf("%w: %v", ErrImportPathInvalid, err)
	}
	st, err := os.Stat(absInput)
	if err != nil {
		if os.IsNotExist(err) {
			return ImportModelResult{}, fmt.Errorf("%w: %s", ErrImportPathInvalid, absInput)
		}
		return ImportModelResult{}, fmt.Errorf("stat import path: %w", err)
	}
	if st.IsDir() {
		return s.importDirectory(absInput, opts)
	}
	return s.importFile(absInput, opts)
}

func (s *ModelStore) Verify(ctx context.Context, modelID string) (StoredModel, error) {
	_ = ctx
	rec, err := s.Load(context.Background(), modelID)
	if err != nil {
		return StoredModel{}, err
	}
	actualChecksum, err := fileSHA256(rec.ModelFilePath)
	if err != nil {
		return StoredModel{}, fmt.Errorf("verify checksum: %w", err)
	}
	if rec.Manifest.SHA256 != "" && normalizeSHA256(rec.Manifest.SHA256) != actualChecksum {
		return StoredModel{}, fmt.Errorf("%w: model %s checksum mismatch: expected=%s actual=%s", ErrChecksumMismatch, rec.Manifest.ID, normalizeSHA256(rec.Manifest.SHA256), actualChecksum)
	}
	rec.State.Normalize()
	rec.State.VerifiedAt = time.Now().UTC()
	if rec.State.Status != StatusDisabled && rec.State.Status != StatusArchived {
		rec.State.Status = StatusVerified
	}
	if err := writeModelState(filepath.Join(rec.ModelDir, ModelStateFilename), rec.State); err != nil {
		return StoredModel{}, err
	}
	rec.State.Status = stateStatusOrDefault(rec.State, rec.State.Status)
	return rec, nil
}

func (s *ModelStore) SetDisabled(ctx context.Context, modelID string, disabled bool) (StoredModel, error) {
	_ = ctx
	rec, err := s.Load(context.Background(), modelID)
	if err != nil {
		return StoredModel{}, err
	}
	if rec.State.Status == StatusArchived {
		return StoredModel{}, ErrModelArchived
	}
	rec.State.Normalize()
	if disabled {
		rec.State.Status = StatusDisabled
		rec.State.DisabledAt = time.Now().UTC()
	} else {
		rec.State.DisabledAt = time.Time{}
		rec.State.Status = stateStatusOrDefault(rec.State, StatusAvailable)
		if rec.State.Status == StatusDisabled {
			rec.State.Status = StatusAvailable
		}
	}
	if err := writeModelState(filepath.Join(rec.ModelDir, ModelStateFilename), rec.State); err != nil {
		return StoredModel{}, err
	}
	return s.Load(context.Background(), modelID)
}

func (s *ModelStore) SetPreferred(ctx context.Context, modelID string, preferred bool) (StoredModel, error) {
	_ = ctx
	records, err := s.Scan(context.Background())
	if err != nil {
		return StoredModel{}, err
	}
	var target StoredModel
	found := false
	for _, rec := range records {
		rec.State.Normalize()
		if rec.Manifest.ID == modelID {
			found = true
			target = rec
		}
		if rec.State.Preferred && rec.Manifest.ID != modelID {
			rec.State.Preferred = false
			if err := writeModelState(filepath.Join(rec.ModelDir, ModelStateFilename), rec.State); err != nil {
				return StoredModel{}, err
			}
		}
	}
	if !found {
		return StoredModel{}, fmt.Errorf("%w: %s", ErrModelNotFound, modelID)
	}
	target.State.Preferred = preferred
	if err := writeModelState(filepath.Join(target.ModelDir, ModelStateFilename), target.State); err != nil {
		return StoredModel{}, err
	}
	return s.Load(context.Background(), modelID)
}

func (s *ModelStore) Archive(ctx context.Context, modelID string) (StoredModel, error) {
	_ = ctx
	rec, err := s.loadFromRoot(strings.TrimSpace(modelID), false)
	if err != nil {
		if archivedRec, archivedErr := s.loadFromRoot(strings.TrimSpace(modelID), true); archivedErr == nil {
			return archivedRec, nil
		}
		return StoredModel{}, err
	}
	archivedRoot, err := s.ensureNamedRoot("archives")
	if err != nil {
		return StoredModel{}, err
	}
	rec.State.Normalize()
	rec.State.Status = StatusArchived
	rec.State.ArchivedAt = time.Now().UTC()
	if err := writeModelState(filepath.Join(rec.ModelDir, ModelStateFilename), rec.State); err != nil {
		return StoredModel{}, err
	}
	destDir := filepath.Join(archivedRoot, rec.Manifest.ID)
	if err := os.RemoveAll(destDir); err != nil {
		return StoredModel{}, fmt.Errorf("prepare archive destination: %w", err)
	}
	if err := os.Rename(rec.ModelDir, destDir); err != nil {
		return StoredModel{}, fmt.Errorf("archive model directory: %w", err)
	}
	return s.readModelDir(destDir, true)
}

func (s *ModelStore) RemoveRegistration(ctx context.Context, modelID string) (string, error) {
	_ = ctx
	id := strings.TrimSpace(modelID)
	if id == "" {
		return "", fmt.Errorf("%w: empty id", ErrModelNotFound)
	}
	rec, err := s.Load(context.Background(), id)
	if err != nil {
		return "", err
	}
	removedRoot, err := s.ensureNamedRoot("removed")
	if err != nil {
		return "", err
	}
	destDir := filepath.Join(removedRoot, fmt.Sprintf("%s-%d", id, time.Now().UTC().Unix()))
	if err := os.Rename(rec.ModelDir, destDir); err != nil {
		return "", fmt.Errorf("remove registration: %w", err)
	}
	return destDir, nil
}

func (s *ModelStore) Reconcile(ctx context.Context) ([]StoredModel, error) {
	return s.Scan(ctx)
}

func (s *ModelStore) importFile(inputPath string, opts ImportModelOptions) (ImportModelResult, error) {
	format := detectModelFormat(inputPath)
	if format != ModelFormatGGUF {
		return ImportModelResult{}, fmt.Errorf("%w: %s", ErrUnsupportedModelFormat, format)
	}
	checksum, err := fileSHA256(inputPath)
	if err != nil {
		return ImportModelResult{}, fmt.Errorf("hash import file: %w", err)
	}
	st, err := os.Stat(inputPath)
	if err != nil {
		return ImportModelResult{}, fmt.Errorf("stat import file: %w", err)
	}
	id := stableImportedModelID(inputPath, checksum, opts.ID)
	modelRoot, err := s.ensureNamedRoot("models")
	if err != nil {
		return ImportModelResult{}, err
	}
	destDir := filepath.Join(modelRoot, id)
	destFile := filepath.Join(destDir, filepath.Base(inputPath))
	if existing, err := s.loadFromRoot(id, false); err == nil {
		if normalizeSHA256(existing.Manifest.SHA256) == checksum {
			return ImportModelResult{Model: existing, Duplicate: true, ManagedPath: existing.ModelDir, SourcePath: inputPath}, nil
		}
		return ImportModelResult{}, fmt.Errorf("%w: %s", ErrModelAlreadyExists, id)
	}
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return ImportModelResult{}, fmt.Errorf("create model directory: %w", err)
	}
	if err := copyFile(inputPath, destFile); err != nil {
		return ImportModelResult{}, err
	}
	manifest := ModelManifest{
		SchemaVersion: "forge.model/v1",
		ID:            id,
		DisplayName:   firstNonEmptyImport(opts.DisplayName, humanizeModelName(strings.TrimSuffix(filepath.Base(inputPath), filepath.Ext(inputPath)))),
		Family:        firstNonEmptyImport(opts.Family, defaultModelFamily(inputPath)),
		Format:        format,
		Backend:       firstNonEmptyBackend(opts.Backend, BackendLlamaCpp),
		FilePath:      filepath.Base(destFile),
		SHA256:        checksum,
		SizeBytes:     st.Size(),
		Quantization:  firstNonEmptyImport(opts.Quantization, inferQuantization(inputPath)),
		ContextLength: positiveOrDefault(opts.ContextLength, 4096),
		Capabilities:  normalizeCapabilities(importCapabilitiesOrDefault(opts.Capabilities)),
		License:       firstNonEmptyImport(opts.License, "unknown"),
		Metadata:      cloneStateMetadata(opts.Metadata),
	}
	manifest.Metadata["sourcePath"] = inputPath
	manifest.Metadata["managed"] = true
	if err := writeManifest(filepath.Join(destDir, ManifestFilename), manifest); err != nil {
		return ImportModelResult{}, err
	}
	state := defaultImportedState(inputPath, opts.Preferred, opts.Metadata)
	if err := writeModelState(filepath.Join(destDir, ModelStateFilename), state); err != nil {
		return ImportModelResult{}, err
	}
	rec, err := s.readModelDir(destDir, false)
	if err != nil {
		return ImportModelResult{}, err
	}
	return ImportModelResult{Model: rec, ManagedPath: destDir, SourcePath: inputPath}, nil
}

func (s *ModelStore) importDirectory(inputPath string, opts ImportModelOptions) (ImportModelResult, error) {
	manifestPath := filepath.Join(inputPath, ManifestFilename)
	manifest, err := ReadManifest(manifestPath)
	if err != nil {
		return ImportModelResult{}, err
	}
	id := strings.TrimSpace(manifest.ID)
	if id == "" {
		return ImportModelResult{}, fmt.Errorf("%w: id", ErrManifestMissingRequired)
	}
	modelRoot, err := s.ensureNamedRoot("models")
	if err != nil {
		return ImportModelResult{}, err
	}
	destDir := filepath.Join(modelRoot, id)
	if existing, err := s.loadFromRoot(id, false); err == nil {
		if normalizeSHA256(existing.Manifest.SHA256) == normalizeSHA256(manifest.SHA256) {
			return ImportModelResult{Model: existing, Duplicate: true, ManagedPath: existing.ModelDir, SourcePath: inputPath}, nil
		}
		return ImportModelResult{}, fmt.Errorf("%w: %s", ErrModelAlreadyExists, id)
	}
	if err := copyDir(inputPath, destDir); err != nil {
		return ImportModelResult{}, err
	}
	copiedManifest, err := ReadManifest(filepath.Join(destDir, ManifestFilename))
	if err != nil {
		return ImportModelResult{}, err
	}
	if copiedManifest.Metadata == nil {
		copiedManifest.Metadata = map[string]any{}
	}
	copiedManifest.Metadata["sourcePath"] = inputPath
	copiedManifest.Metadata["managed"] = true
	if err := writeManifest(filepath.Join(destDir, ManifestFilename), copiedManifest); err != nil {
		return ImportModelResult{}, err
	}
	state := defaultImportedState(inputPath, opts.Preferred, opts.Metadata)
	if err := writeModelState(filepath.Join(destDir, ModelStateFilename), state); err != nil {
		return ImportModelResult{}, err
	}
	rec, err := s.readModelDir(destDir, false)
	if err != nil {
		return ImportModelResult{}, err
	}
	return ImportModelResult{Model: rec, ManagedPath: destDir, SourcePath: inputPath}, nil
}

func (s *ModelStore) ensureNamedRoot(name string) (string, error) {
	home, err := s.ResolveModelHome()
	if err != nil {
		return "", err
	}
	root := filepath.Join(home, name)
	if err := os.MkdirAll(root, 0o755); err != nil {
		return "", fmt.Errorf("create %s dir: %w", name, err)
	}
	return root, nil
}

func detectModelFormat(path string) ModelFormat {
	switch strings.ToLower(filepath.Ext(strings.TrimSpace(path))) {
	case ".gguf":
		return ModelFormatGGUF
	case ".onnx":
		return ModelFormatONNX
	case ".safetensors":
		return ModelFormatSafeTensors
	default:
		return ModelFormatUnknown
	}
}

func stableImportedModelID(path, checksum, requested string) string {
	if id := sanitizeModelID(requested); id != "" {
		return id
	}
	base := sanitizeModelID(strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)))
	if base == "" {
		base = "model"
	}
	if len(checksum) > 12 {
		checksum = checksum[:12]
	}
	return fmt.Sprintf("%s-%s", base, checksum)
}

func sanitizeModelID(raw string) string {
	raw = strings.ToLower(strings.TrimSpace(raw))
	if raw == "" {
		return ""
	}
	var b strings.Builder
	lastDash := false
	for _, r := range raw {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			lastDash = false
		case r == '-', r == '_', r == '.', r == ' ':
			if !lastDash {
				b.WriteRune('-')
				lastDash = true
			}
		}
	}
	return strings.Trim(b.String(), "-")
}

func humanizeModelName(raw string) string {
	parts := strings.Fields(strings.NewReplacer("-", " ", "_", " ", ".", " ").Replace(raw))
	for i, part := range parts {
		if part == "" {
			continue
		}
		parts[i] = strings.ToUpper(part[:1]) + part[1:]
	}
	return strings.Join(parts, " ")
}

func defaultModelFamily(path string) string {
	base := strings.ToLower(filepath.Base(path))
	for _, family := range []string{"llama", "mistral", "qwen", "gemma", "phi", "deepseek"} {
		if strings.Contains(base, family) {
			return family
		}
	}
	return sanitizeModelID(strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)))
}

func inferQuantization(path string) string {
	base := strings.ToLower(filepath.Base(path))
	for _, marker := range []string{"q2", "q3", "q4", "q5", "q6", "q8", "fp16", "f16"} {
		if strings.Contains(base, marker) {
			return marker
		}
	}
	return "unknown"
}

func importCapabilitiesOrDefault(input []ModelCapability) []ModelCapability {
	if len(input) > 0 {
		return input
	}
	return []ModelCapability{CapabilityChat, CapabilityCompletion}
}

func firstNonEmptyImport(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func firstNonEmptyBackend(value, fallback ModelBackendKind) ModelBackendKind {
	if strings.TrimSpace(string(value)) != "" {
		return value
	}
	return fallback
}

func writeManifest(path string, manifest ModelManifest) error {
	normalizeManifest(&manifest)
	if err := ValidateManifest(manifest); err != nil {
		return err
	}
	body, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("encode model manifest: %w", err)
	}
	body = append(body, '\n')
	if err := os.WriteFile(path, body, 0o644); err != nil {
		return fmt.Errorf("write model manifest: %w", err)
	}
	return nil
}

func copyFile(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return fmt.Errorf("create destination dir: %w", err)
	}
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("open source file: %w", err)
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("create destination file: %w", err)
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return fmt.Errorf("copy model file: %w", err)
	}
	return out.Close()
}

func copyDir(src, dst string) error {
	entries, err := os.ReadDir(src)
	if err != nil {
		return fmt.Errorf("read source dir: %w", err)
	}
	if err := os.MkdirAll(dst, 0o755); err != nil {
		return fmt.Errorf("create destination dir: %w", err)
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())
		if entry.IsDir() {
			if err := copyDir(srcPath, dstPath); err != nil {
				return err
			}
			continue
		}
		if err := copyFile(srcPath, dstPath); err != nil {
			return err
		}
	}
	return nil
}
