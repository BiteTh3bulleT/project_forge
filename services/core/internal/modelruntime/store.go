package modelruntime

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const checksumsFilename = "checksums.json"

var (
	ErrModelHomeUnset   = errors.New("model home is not configured")
	ErrModelHomeMissing = errors.New("model home does not exist")
	ErrModelsDirMissing = errors.New("models directory does not exist")
	ErrManifestMissing  = errors.New("model manifest file missing")
	ErrChecksumMismatch = errors.New("model checksum mismatch")
	ErrModelFileMissing = errors.New("model file missing")
	ErrModelIDMismatch  = errors.New("model id mismatch")
	ErrModelIDInvalid   = errors.New("model id invalid")
)

// ModelStoreOptions tunes checksum behavior.
type ModelStoreOptions struct {
	StrictChecksum bool
}

// StoredModel is the resolved model record discovered on disk.
type StoredModel struct {
	Manifest      ModelManifest
	State         ModelState
	ModelDir      string
	ManifestPath  string
	ModelFilePath string
	Archived      bool
	Warnings      []string
}

// ModelStore scans and validates manifests under FORGE model home.
type ModelStore struct {
	modelHome      string
	strictChecksum bool
}

func NewModelStore(modelHome string, opts ModelStoreOptions) *ModelStore {
	return &ModelStore{
		modelHome:      strings.TrimSpace(modelHome),
		strictChecksum: opts.StrictChecksum,
	}
}

func (s *ModelStore) ModelHome() string {
	return s.modelHome
}

func (s *ModelStore) ResolveModelHome() (string, error) {
	home := strings.TrimSpace(s.modelHome)
	if home == "" {
		home = strings.TrimSpace(os.Getenv("FORGE_MODEL_HOME"))
	}
	if home == "" {
		return "", ErrModelHomeUnset
	}
	resolved, err := filepath.Abs(home)
	if err != nil {
		return "", fmt.Errorf("resolve model home: %w", err)
	}
	st, err := os.Stat(resolved)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("%w: %s", ErrModelHomeMissing, resolved)
		}
		return "", fmt.Errorf("stat model home: %w", err)
	}
	if !st.IsDir() {
		return "", fmt.Errorf("%w: %s is not a directory", ErrModelHomeMissing, resolved)
	}
	return resolved, nil
}

func (s *ModelStore) modelsRoot() (string, error) {
	home, err := s.ResolveModelHome()
	if err != nil {
		return "", err
	}
	root := filepath.Join(home, "models")
	st, err := os.Stat(root)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("%w: %s", ErrModelsDirMissing, root)
		}
		return "", fmt.Errorf("stat models dir: %w", err)
	}
	if !st.IsDir() {
		return "", fmt.Errorf("%w: %s is not a directory", ErrModelsDirMissing, root)
	}
	return root, nil
}

// Scan discovers and validates model manifests under <model-home>/models/<model-id>/model.forge.json.
func (s *ModelStore) Scan(ctx context.Context) ([]StoredModel, error) {
	_ = ctx
	activeRoot, err := s.modelsRoot()
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(activeRoot)
	if err != nil {
		return nil, fmt.Errorf("read models dir: %w", err)
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })

	out := make([]StoredModel, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		rec, err := s.readModelDir(filepath.Join(activeRoot, entry.Name()), false)
		if err != nil {
			return nil, err
		}
		out = append(out, rec)
	}

	archivedRoot, err := s.archivesRoot(false)
	if err != nil {
		return nil, err
	}
	if archivedRoot != "" {
		archivedEntries, err := os.ReadDir(archivedRoot)
		if err != nil {
			return nil, fmt.Errorf("read archives dir: %w", err)
		}
		sort.Slice(archivedEntries, func(i, j int) bool { return archivedEntries[i].Name() < archivedEntries[j].Name() })
		for _, entry := range archivedEntries {
			if !entry.IsDir() {
				continue
			}
			rec, err := s.readModelDir(filepath.Join(archivedRoot, entry.Name()), true)
			if err != nil {
				return nil, err
			}
			out = append(out, rec)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Manifest.ID < out[j].Manifest.ID })
	return out, nil
}

// Load loads a single model by id from disk.
func (s *ModelStore) Load(ctx context.Context, modelID string) (StoredModel, error) {
	_ = ctx
	id, err := safeModelIDSegment(modelID)
	if err != nil {
		return StoredModel{}, err
	}
	if rec, err := s.loadFromRoot(id, false); err == nil {
		return rec, nil
	} else if !errors.Is(err, ErrModelNotFound) {
		return StoredModel{}, err
	}
	return s.loadFromRoot(id, true)
}

func (s *ModelStore) loadFromRoot(modelID string, archived bool) (StoredModel, error) {
	id, err := safeModelIDSegment(modelID)
	if err != nil {
		return StoredModel{}, err
	}
	root, err := s.modelsRoot()
	if archived {
		root, err = s.archivesRoot(true)
	}
	if err != nil {
		return StoredModel{}, err
	}
	modelDir := filepath.Join(root, id)
	if _, err := os.Stat(modelDir); err != nil {
		if os.IsNotExist(err) {
			return StoredModel{}, fmt.Errorf("%w: %s", ErrModelNotFound, id)
		}
		return StoredModel{}, fmt.Errorf("stat model dir: %w", err)
	}
	return s.readModelDir(modelDir, archived)
}

func safeModelIDSegment(raw string) (string, error) {
	id := strings.TrimSpace(raw)
	if id == "" {
		return "", fmt.Errorf("%w: empty model id", ErrModelIDInvalid)
	}
	if id == "." || id == ".." {
		return "", fmt.Errorf("%w: model id must be a path segment", ErrModelIDInvalid)
	}
	if strings.ContainsAny(id, `/\`) {
		return "", fmt.Errorf("%w: model id must not contain path separators", ErrModelIDInvalid)
	}
	if filepath.Clean(id) != id || filepath.Base(id) != id {
		return "", fmt.Errorf("%w: model id must be a stable path segment", ErrModelIDInvalid)
	}
	return id, nil
}

func (s *ModelStore) readModelDir(modelDir string, archived bool) (StoredModel, error) {
	manifestPath := filepath.Join(modelDir, ManifestFilename)
	if _, err := os.Stat(manifestPath); err != nil {
		if os.IsNotExist(err) {
			return StoredModel{}, fmt.Errorf("%w: %s", ErrManifestMissing, manifestPath)
		}
		return StoredModel{}, fmt.Errorf("stat manifest: %w", err)
	}

	manifest, err := ReadManifest(manifestPath)
	if err != nil {
		return StoredModel{}, err
	}

	dirName := filepath.Base(modelDir)
	if manifest.ID != dirName {
		return StoredModel{}, fmt.Errorf("%w: manifest id=%q dir=%q", ErrModelIDMismatch, manifest.ID, dirName)
	}

	modelFilePath, err := safeJoinWithinBase(modelDir, ManifestModelPath(manifest))
	if err != nil {
		return StoredModel{}, err
	}
	st, err := os.Stat(modelFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return StoredModel{}, fmt.Errorf("%w: %s", ErrModelFileMissing, modelFilePath)
		}
		return StoredModel{}, fmt.Errorf("stat model file: %w", err)
	}
	if st.IsDir() {
		return StoredModel{}, fmt.Errorf("%w: %s is a directory", ErrModelFileMissing, modelFilePath)
	}

	state, err := readModelState(filepath.Join(modelDir, ModelStateFilename))
	if err != nil {
		return StoredModel{}, err
	}
	if archived {
		state.Status = StatusArchived
		if state.ArchivedAt.IsZero() {
			state.ArchivedAt = time.Now().UTC()
		}
		state.Managed = true
	}

	rec := StoredModel{
		Manifest:      manifest,
		State:         state,
		ModelDir:      modelDir,
		ManifestPath:  manifestPath,
		ModelFilePath: modelFilePath,
		Archived:      archived,
		Warnings:      nil,
	}

	if rec.Manifest.SizeBytes > 0 && st.Size() != rec.Manifest.SizeBytes {
		rec.Warnings = append(rec.Warnings,
			fmt.Sprintf("size mismatch: manifest=%d bytes actual=%d bytes", rec.Manifest.SizeBytes, st.Size()),
		)
	}

	expectedChecksum, err := resolveExpectedChecksum(rec.Manifest, rec.ModelDir, rec.ModelFilePath)
	if err != nil {
		return StoredModel{}, err
	}
	if expectedChecksum != "" {
		actual, err := fileSHA256(rec.ModelFilePath)
		if err != nil {
			return StoredModel{}, fmt.Errorf("hash model file: %w", err)
		}
		if actual != expectedChecksum {
			msg := fmt.Sprintf("model %s checksum mismatch: expected=%s actual=%s", rec.Manifest.ID, expectedChecksum, actual)
			if s.strictChecksum {
				return StoredModel{}, fmt.Errorf("%w: %s", ErrChecksumMismatch, msg)
			}
			rec.Warnings = append(rec.Warnings, msg)
		}
	}

	return rec, nil
}

func (s *ModelStore) archivesRoot(require bool) (string, error) {
	home, err := s.ResolveModelHome()
	if err != nil {
		return "", err
	}
	root := filepath.Join(home, "archives")
	st, err := os.Stat(root)
	if err != nil {
		if os.IsNotExist(err) && !require {
			return "", nil
		}
		if os.IsNotExist(err) {
			return "", fmt.Errorf("%w: %s", ErrModelsDirMissing, root)
		}
		return "", fmt.Errorf("stat archives dir: %w", err)
	}
	if !st.IsDir() {
		return "", fmt.Errorf("%w: %s is not a directory", ErrModelsDirMissing, root)
	}
	return root, nil
}

func resolveExpectedChecksum(m ModelManifest, modelDir, modelFilePath string) (string, error) {
	if m.SHA256 != "" {
		return normalizeSHA256(m.SHA256), nil
	}
	checksumsPath := filepath.Join(modelDir, checksumsFilename)
	if _, err := os.Stat(checksumsPath); err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", fmt.Errorf("stat checksums: %w", err)
	}

	checksums, err := readChecksumsFile(checksumsPath)
	if err != nil {
		return "", err
	}
	if len(checksums) == 0 {
		return "", nil
	}

	keys := checksumLookupKeys(m, modelDir, modelFilePath)
	for _, key := range keys {
		if value, ok := checksums[key]; ok {
			return value, nil
		}
	}
	return "", nil
}

func checksumLookupKeys(m ModelManifest, modelDir, modelFilePath string) []string {
	raw := ManifestModelPath(m)
	rel, _ := filepath.Rel(modelDir, modelFilePath)
	keys := []string{
		raw,
		filepath.ToSlash(raw),
		rel,
		filepath.ToSlash(rel),
		filepath.Base(modelFilePath),
	}
	out := make([]string, 0, len(keys))
	seen := map[string]struct{}{}
	for _, key := range keys {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, key)
	}
	return out
}

func readChecksumsFile(path string) (map[string]string, error) {
	body, err := readRuntimeJSONFile(path, "checksums file")
	if err != nil {
		return nil, fmt.Errorf("read checksums file: %w", err)
	}

	flat := map[string]string{}
	if err := json.Unmarshal(body, &flat); err == nil {
		return normalizeChecksumMap(flat), nil
	}

	var wrapped struct {
		Files map[string]string `json:"files"`
	}
	if err := json.Unmarshal(body, &wrapped); err != nil {
		return nil, fmt.Errorf("decode checksums file %s: %w", path, err)
	}
	return normalizeChecksumMap(wrapped.Files), nil
}

func normalizeChecksumMap(input map[string]string) map[string]string {
	if len(input) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(input))
	for key, value := range input {
		trimmedKey := strings.TrimSpace(key)
		trimmedValue := normalizeSHA256(value)
		if trimmedKey == "" || trimmedValue == "" {
			continue
		}
		out[trimmedKey] = trimmedValue
	}
	return out
}

func fileSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
