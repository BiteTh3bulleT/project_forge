package memory

import (
	"hash/fnv"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

const (
	defaultVSADims = 128
	defaultVSASeed = 17
)

var tokenPattern = regexp.MustCompile(`[A-Za-z0-9_./:-]+`)

type VSAEngine struct {
	dims int
	seed uint64
}

func NewVSAEngine(dims int, seed uint64) *VSAEngine {
	if dims <= 0 {
		dims = defaultVSADims
	}
	if seed == 0 {
		seed = defaultVSASeed
	}
	return &VSAEngine{dims: dims, seed: seed}
}

func (e *VSAEngine) Dims() int { return e.dims }

func (e *VSAEngine) Seed() uint64 { return e.seed }

func (e *VSAEngine) EncodeText(text string) []float64 {
	vec := make([]float64, e.dims)
	tokens := tokenize(text)
	if len(tokens) == 0 {
		return vec
	}
	for _, tok := range tokens {
		h := stableHash(tok, e.seed)
		idx := int(h % uint64(e.dims))
		sgn := 1.0
		if (h>>7)&1 == 1 {
			sgn = -1.0
		}
		vec[idx] += sgn
	}
	normalizeInPlace(vec)
	return vec
}

func (e *VSAEngine) EncodeMapValues(values map[string]string) []float64 {
	if len(values) == 0 {
		return make([]float64, e.dims)
	}
	keys := make([]string, 0, len(values))
	for k := range values {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	acc := make([]float64, e.dims)
	for _, role := range keys {
		filler := strings.TrimSpace(values[role])
		if filler == "" {
			continue
		}
		roleVec := e.EncodeText(role)
		fillVec := e.EncodeText(filler)
		bound := e.Bind(roleVec, fillVec)
		addInPlace(acc, bound)
	}
	normalizeInPlace(acc)
	return acc
}

func (e *VSAEngine) Bind(a, b []float64) []float64 {
	if len(a) == 0 || len(b) == 0 {
		return make([]float64, e.dims)
	}
	left := resizeVec(a, e.dims)
	right := resizeVec(b, e.dims)
	out := make([]float64, e.dims)
	for i := 0; i < e.dims; i++ {
		sum := 0.0
		for j := 0; j < e.dims; j++ {
			k := i - j
			if k < 0 {
				k += e.dims
			}
			sum += left[j] * right[k]
		}
		out[i] = sum
	}
	normalizeInPlace(out)
	return out
}

func (e *VSAEngine) Unbind(bound, cue []float64) []float64 {
	if len(bound) == 0 || len(cue) == 0 {
		return make([]float64, e.dims)
	}
	return e.Bind(bound, reverseVec(resizeVec(cue, e.dims)))
}

func (e *VSAEngine) Similarity(a, b []float64) float64 {
	left := resizeVec(a, e.dims)
	right := resizeVec(b, e.dims)
	denom := vectorNorm(left) * vectorNorm(right)
	if denom == 0 {
		return 0
	}
	dot := 0.0
	for i := 0; i < e.dims; i++ {
		dot += left[i] * right[i]
	}
	return dot / denom
}

func (e *VSAEngine) ComposeObservationPointer(obs Observation) []float64 {
	acc := make([]float64, e.dims)
	addInPlace(acc, e.EncodeText(obs.Type+" "+obs.TaskType))
	addInPlace(acc, scaleVec(e.EncodeText(obs.SourcePath), 0.5))
	addInPlace(acc, scaleVec(e.EncodeText(obs.Summary), 1.2))
	addInPlace(acc, scaleVec(e.EncodeText(obs.RawContent), 0.8))

	entities := parseRawStringSlice(obs.Entities)
	tags := parseRawStringSlice(obs.Tags)
	related := parseRawStringSlice(obs.RelatedFiles)
	for _, entity := range entities {
		addInPlace(acc, scaleVec(e.Bind(e.EncodeText("entity"), e.EncodeText(entity)), 0.75))
	}
	for _, tag := range tags {
		addInPlace(acc, scaleVec(e.Bind(e.EncodeText("tag"), e.EncodeText(tag)), 0.6))
	}
	for _, file := range related {
		addInPlace(acc, scaleVec(e.Bind(e.EncodeText("related_file"), e.EncodeText(file)), 0.5))
	}
	normalizeInPlace(acc)
	return acc
}

func tokenize(raw string) []string {
	matches := tokenPattern.FindAllString(strings.ToLower(raw), -1)
	if len(matches) == 0 {
		return nil
	}
	out := make([]string, 0, len(matches))
	for _, m := range matches {
		m = strings.TrimSpace(m)
		if m == "" {
			continue
		}
		out = append(out, m)
	}
	return out
}

func stableHash(token string, seed uint64) uint64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(token))
	_, _ = h.Write([]byte("#"))
	_, _ = h.Write([]byte(strconv.FormatUint(seed, 10)))
	return h.Sum64()
}

func vectorNorm(vec []float64) float64 {
	sum := 0.0
	for _, v := range vec {
		sum += v * v
	}
	if sum <= 0 {
		return 0
	}
	return math.Sqrt(sum)
}

func normalizeInPlace(vec []float64) {
	n := vectorNorm(vec)
	if n == 0 {
		return
	}
	for i := range vec {
		vec[i] /= n
	}
}

func addInPlace(dst, src []float64) {
	if len(dst) == 0 || len(src) == 0 {
		return
	}
	limit := len(dst)
	if len(src) < limit {
		limit = len(src)
	}
	for i := 0; i < limit; i++ {
		dst[i] += src[i]
	}
}

func scaleVec(src []float64, scalar float64) []float64 {
	if len(src) == 0 {
		return nil
	}
	out := make([]float64, len(src))
	for i := range src {
		out[i] = src[i] * scalar
	}
	return out
}

func resizeVec(src []float64, dims int) []float64 {
	if dims <= 0 {
		return nil
	}
	if len(src) == dims {
		out := make([]float64, dims)
		copy(out, src)
		return out
	}
	out := make([]float64, dims)
	copy(out, src)
	return out
}

func reverseVec(src []float64) []float64 {
	if len(src) == 0 {
		return nil
	}
	out := make([]float64, len(src))
	out[0] = src[0]
	for i := 1; i < len(src); i++ {
		out[i] = src[len(src)-i]
	}
	return out
}
