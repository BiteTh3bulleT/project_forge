// Package vsaprojection defines the deterministic, model-free contract for a
// governed VSA acceleration rebuild. Projection rows are derived acceleration
// only; the manifest binds them to exact scoped evidence and algorithm identity.
package vsaprojection

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"hash/fnv"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

const (
	ManifestVersion  = "forge.vsa.projection_manifest.v1"
	AlgorithmName    = "forge.vsa.observation_projection"
	AlgorithmVersion = "1"
	DefaultDims      = 128
	DefaultSeed      = uint64(17)
)

var (
	ErrInvalidScope     = errors.New("invalid VSA projection scope")
	ErrInvalidAlgorithm = errors.New("invalid VSA projection algorithm")
	ErrInvalidSource    = errors.New("invalid VSA projection source")
	ErrInvalidLink      = errors.New("invalid VSA projection link")
	tokenPattern        = regexp.MustCompile(`[A-Za-z0-9_./:-]+`)
)

type Scope struct {
	WorkspaceID string `json:"workspaceId"`
	LaneID      string `json:"laneId"`
}

type Algorithm struct {
	Name       string `json:"name"`
	Version    string `json:"version"`
	Dimensions int    `json:"dimensions"`
	Seed       uint64 `json:"seed"`
}

func DefaultAlgorithm() Algorithm {
	return Algorithm{Name: AlgorithmName, Version: AlgorithmVersion, Dimensions: DefaultDims, Seed: DefaultSeed}
}

type Source struct {
	ID                int64    `json:"id"`
	WorkspaceID       string   `json:"workspaceId"`
	LaneID            string   `json:"laneId"`
	Type              string   `json:"type"`
	TaskType          string   `json:"taskType"`
	ProjectKey        string   `json:"projectKey"`
	SourcePath        string   `json:"sourcePath"`
	Summary           string   `json:"summary"`
	RawContent        string   `json:"rawContent"`
	Entities          []string `json:"entities"`
	Tags              []string `json:"tags"`
	RelatedFiles      []string `json:"relatedFiles"`
	Lineage           []string `json:"lineage"`
	SupportCount      int      `json:"supportCount"`
	NoiseCount        int      `json:"noiseCount"`
	SourceFingerprint string   `json:"sourceFingerprint"`
}

type Link struct {
	ID                int64  `json:"id"`
	WorkspaceID       string `json:"workspaceId"`
	LaneID            string `json:"laneId"`
	FromObservationID int64  `json:"fromObservationId"`
	ToObservationID   int64  `json:"toObservationId"`
	RelationType      string `json:"relationType"`
}

type Manifest struct {
	Version       string    `json:"version"`
	Scope         Scope     `json:"scope"`
	Algorithm     Algorithm `json:"algorithm"`
	SourceSetHash string    `json:"sourceSetHash"`
	LinkSetHash   string    `json:"linkSetHash"`
	SourceCount   int       `json:"sourceCount"`
	LinkCount     int       `json:"linkCount"`
	ManifestHash  string    `json:"manifestHash"`
}

type Pointer struct {
	ObservationID     int64     `json:"observationId"`
	Dimensions        int       `json:"dimensions"`
	Vector            []float64 `json:"vector"`
	Norm              float64   `json:"norm"`
	SourceFingerprint string    `json:"sourceFingerprint"`
	SupportCount      int       `json:"supportCount"`
	NoiseCount        int       `json:"noiseCount"`
}

type Binding struct {
	ObservationID int64     `json:"observationId"`
	Role          string    `json:"role"`
	Filler        string    `json:"filler"`
	Weight        float64   `json:"weight"`
	SupportCount  int       `json:"supportCount"`
	NoiseCount    int       `json:"noiseCount"`
	Vector        []float64 `json:"vector"`
}

type Association struct {
	FromObservationID int64   `json:"fromObservationId"`
	ToObservationID   int64   `json:"toObservationId"`
	RelationType      string  `json:"relationType"`
	Strength          float64 `json:"strength"`
	SupportCount      int     `json:"supportCount"`
	NoiseCount        int     `json:"noiseCount"`
}

type Projection struct {
	Manifest     Manifest      `json:"manifest"`
	Pointers     []Pointer     `json:"pointers"`
	Bindings     []Binding     `json:"bindings"`
	Associations []Association `json:"associations"`
}

func Build(scope Scope, algorithm Algorithm, sources []Source, links []Link) (Projection, error) {
	scope = normalizeScope(scope)
	if scope.WorkspaceID == "" || scope.LaneID == "" {
		return Projection{}, ErrInvalidScope
	}
	algorithm = normalizeAlgorithm(algorithm)
	if algorithm.Name != AlgorithmName || algorithm.Version != AlgorithmVersion || algorithm.Dimensions < 8 || algorithm.Dimensions > 4096 || algorithm.Seed == 0 {
		return Projection{}, ErrInvalidAlgorithm
	}
	normalizedSources, err := normalizeSources(scope, sources)
	if err != nil {
		return Projection{}, err
	}
	normalizedLinks, err := normalizeLinks(scope, normalizedSources, links)
	if err != nil {
		return Projection{}, err
	}

	sourceSetHash, err := hashJSON(normalizedSources)
	if err != nil {
		return Projection{}, err
	}
	linkSetHash, err := hashJSON(normalizedLinks)
	if err != nil {
		return Projection{}, err
	}
	manifest := Manifest{
		Version: ManifestVersion, Scope: scope, Algorithm: algorithm,
		SourceSetHash: sourceSetHash, LinkSetHash: linkSetHash,
		SourceCount: len(normalizedSources), LinkCount: len(normalizedLinks),
	}
	manifestHash, err := hashJSON(manifest)
	if err != nil {
		return Projection{}, err
	}
	manifest.ManifestHash = manifestHash

	engine := newEngine(algorithm.Dimensions, algorithm.Seed)
	projection := Projection{Manifest: manifest, Pointers: []Pointer{}, Bindings: []Binding{}, Associations: []Association{}}
	sourceByID := make(map[int64]Source, len(normalizedSources))
	for _, source := range normalizedSources {
		sourceByID[source.ID] = source
		vector := engine.compose(source)
		projection.Pointers = append(projection.Pointers, Pointer{
			ObservationID: source.ID, Dimensions: algorithm.Dimensions, Vector: vector,
			Norm: vectorNorm(vector), SourceFingerprint: source.SourceFingerprint,
			SupportCount: source.SupportCount, NoiseCount: source.NoiseCount,
		})
		for _, seed := range bindingSeeds(source) {
			projection.Bindings = append(projection.Bindings, Binding{
				ObservationID: source.ID, Role: seed.role, Filler: seed.filler, Weight: seed.weight,
				SupportCount: source.SupportCount, NoiseCount: source.NoiseCount,
				Vector: engine.bind(engine.encode(seed.role), engine.encode(seed.filler)),
			})
		}
	}
	for _, link := range normalizedLinks {
		from := sourceByID[link.FromObservationID]
		to := sourceByID[link.ToObservationID]
		projection.Associations = append(projection.Associations, Association{
			FromObservationID: link.FromObservationID, ToObservationID: link.ToObservationID,
			RelationType: link.RelationType, Strength: relationStrength(link.RelationType),
			SupportCount: from.SupportCount + to.SupportCount,
			NoiseCount:   from.NoiseCount + to.NoiseCount,
		})
	}
	return projection, nil
}

func VerifyExpectedManifest(projection Projection, expected string) error {
	expected = strings.TrimSpace(expected)
	if expected == "" || projection.Manifest.ManifestHash != expected {
		return fmt.Errorf("manifest identity mismatch: expected %q got %q", expected, projection.Manifest.ManifestHash)
	}
	return nil
}

func normalizeScope(scope Scope) Scope {
	return Scope{WorkspaceID: strings.TrimSpace(scope.WorkspaceID), LaneID: strings.TrimSpace(scope.LaneID)}
}

func normalizeAlgorithm(algorithm Algorithm) Algorithm {
	algorithm.Name = strings.TrimSpace(algorithm.Name)
	algorithm.Version = strings.TrimSpace(algorithm.Version)
	return algorithm
}

func normalizeSources(scope Scope, sources []Source) ([]Source, error) {
	out := make([]Source, len(sources))
	seen := make(map[int64]struct{}, len(sources))
	for i, source := range sources {
		source.WorkspaceID = strings.TrimSpace(source.WorkspaceID)
		source.LaneID = strings.TrimSpace(source.LaneID)
		if source.ID <= 0 || source.WorkspaceID == "" || source.LaneID == "" || source.WorkspaceID != scope.WorkspaceID || source.LaneID != scope.LaneID {
			return nil, fmt.Errorf("%w: source %d is not bound to exact scope", ErrInvalidSource, source.ID)
		}
		if _, ok := seen[source.ID]; ok {
			return nil, fmt.Errorf("%w: duplicate source %d", ErrInvalidSource, source.ID)
		}
		seen[source.ID] = struct{}{}
		source.Type = normalizeText(source.Type)
		source.TaskType = normalizeText(source.TaskType)
		source.ProjectKey = normalizeText(source.ProjectKey)
		source.SourcePath = normalizeText(source.SourcePath)
		source.Summary = normalizeText(source.Summary)
		source.RawContent = normalizeText(source.RawContent)
		source.Entities = normalizeStrings(source.Entities)
		source.Tags = normalizeStrings(source.Tags)
		source.RelatedFiles = normalizeStrings(source.RelatedFiles)
		source.Lineage = normalizeStrings(source.Lineage)
		if source.SupportCount < 0 || source.NoiseCount < 0 {
			return nil, fmt.Errorf("%w: negative utility counts for source %d", ErrInvalidSource, source.ID)
		}
		source.SourceFingerprint = ""
		fingerprint, err := hashJSON(source)
		if err != nil {
			return nil, err
		}
		source.SourceFingerprint = fingerprint
		out[i] = source
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

func normalizeLinks(scope Scope, sources []Source, links []Link) ([]Link, error) {
	validSources := make(map[int64]struct{}, len(sources))
	for _, source := range sources {
		validSources[source.ID] = struct{}{}
	}
	out := make([]Link, len(links))
	seen := make(map[int64]struct{}, len(links))
	for i, link := range links {
		link.WorkspaceID = strings.TrimSpace(link.WorkspaceID)
		link.LaneID = strings.TrimSpace(link.LaneID)
		link.RelationType = strings.ToLower(normalizeText(link.RelationType))
		if link.ID <= 0 || link.WorkspaceID != scope.WorkspaceID || link.LaneID != scope.LaneID || link.FromObservationID <= 0 || link.ToObservationID <= 0 || link.FromObservationID == link.ToObservationID || link.RelationType == "" {
			return nil, fmt.Errorf("%w: invalid link %d", ErrInvalidLink, link.ID)
		}
		if _, ok := seen[link.ID]; ok {
			return nil, fmt.Errorf("%w: duplicate link %d", ErrInvalidLink, link.ID)
		}
		if _, ok := validSources[link.FromObservationID]; !ok {
			return nil, fmt.Errorf("%w: link %d source is outside source set", ErrInvalidLink, link.ID)
		}
		if _, ok := validSources[link.ToObservationID]; !ok {
			return nil, fmt.Errorf("%w: link %d target is outside source set", ErrInvalidLink, link.ID)
		}
		seen[link.ID] = struct{}{}
		out[i] = link
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].FromObservationID != out[j].FromObservationID {
			return out[i].FromObservationID < out[j].FromObservationID
		}
		if out[i].ToObservationID != out[j].ToObservationID {
			return out[i].ToObservationID < out[j].ToObservationID
		}
		if out[i].RelationType != out[j].RelationType {
			return out[i].RelationType < out[j].RelationType
		}
		return out[i].ID < out[j].ID
	})
	return out, nil
}

func normalizeText(value string) string { return strings.TrimSpace(value) }

func normalizeStrings(values []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func hashJSON(value any) (string, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

type bindingSeed struct {
	role, filler string
	weight       float64
}

func bindingSeeds(source Source) []bindingSeed {
	seeds := []bindingSeed{}
	add := func(role, filler string, weight float64) {
		role, filler = strings.ToLower(strings.TrimSpace(role)), strings.ToLower(strings.TrimSpace(filler))
		if role != "" && filler != "" {
			seeds = append(seeds, bindingSeed{role: role, filler: filler, weight: weight})
		}
	}
	add("observation_type", source.Type, .9)
	add("task_type", source.TaskType, .8)
	add("project_key", source.ProjectKey, .7)
	add("source_path", source.SourcePath, .6)
	for _, value := range source.Entities {
		add("entity", value, 1)
	}
	for _, value := range source.Tags {
		add("tag", value, .75)
	}
	for _, value := range source.RelatedFiles {
		add("related_file", value, .65)
	}
	sort.Slice(seeds, func(i, j int) bool {
		if seeds[i].role == seeds[j].role {
			return seeds[i].filler < seeds[j].filler
		}
		return seeds[i].role < seeds[j].role
	})
	return seeds
}

func relationStrength(relation string) float64 {
	switch strings.ToLower(strings.TrimSpace(relation)) {
	case "depends_on", "blocks", "blocked_by":
		return .75
	case "duplicates", "similar", "related":
		return .6
	case "supports", "evidence":
		return .7
	default:
		return .5
	}
}

type engine struct {
	dims int
	seed uint64
}

func newEngine(dims int, seed uint64) engine { return engine{dims: dims, seed: seed} }

func (e engine) encode(text string) []float64 {
	vector := make([]float64, e.dims)
	for _, token := range tokenPattern.FindAllString(strings.ToLower(text), -1) {
		h := stableHash(token, e.seed)
		index := int(h % uint64(e.dims))
		sign := 1.0
		if (h>>7)&1 == 1 {
			sign = -1
		}
		vector[index] += sign
	}
	normalize(vector)
	return vector
}

func (e engine) bind(left, right []float64) []float64 {
	out := make([]float64, e.dims)
	for i := 0; i < e.dims; i++ {
		for j := 0; j < e.dims; j++ {
			index := i - j
			if index < 0 {
				index += e.dims
			}
			out[i] += left[j] * right[index]
		}
	}
	normalize(out)
	return out
}

func (e engine) compose(source Source) []float64 {
	out := make([]float64, e.dims)
	addScaled(out, e.encode(source.Type+" "+source.TaskType), 1)
	addScaled(out, e.encode(source.SourcePath), .5)
	addScaled(out, e.encode(source.Summary), 1.2)
	addScaled(out, e.encode(source.RawContent), .8)
	for _, value := range source.Entities {
		addScaled(out, e.bind(e.encode("entity"), e.encode(value)), .75)
	}
	for _, value := range source.Tags {
		addScaled(out, e.bind(e.encode("tag"), e.encode(value)), .6)
	}
	for _, value := range source.RelatedFiles {
		addScaled(out, e.bind(e.encode("related_file"), e.encode(value)), .5)
	}
	normalize(out)
	return out
}

func stableHash(token string, seed uint64) uint64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(strings.TrimSpace(token)))
	_, _ = h.Write([]byte("#"))
	_, _ = h.Write([]byte(strconv.FormatUint(seed, 10)))
	return h.Sum64()
}

func addScaled(dst, src []float64, scale float64) {
	for i := range dst {
		dst[i] += src[i] * scale
	}
}
func vectorNorm(vector []float64) float64 {
	sum := 0.0
	for _, value := range vector {
		sum += value * value
	}
	return math.Sqrt(sum)
}
func normalize(vector []float64) {
	norm := vectorNorm(vector)
	if norm == 0 {
		return
	}
	for i := range vector {
		vector[i] /= norm
	}
}
