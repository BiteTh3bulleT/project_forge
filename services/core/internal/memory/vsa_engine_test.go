package memory

import "testing"

func TestVSAEncodeDeterministic(t *testing.T) {
	engine := NewVSAEngine(64, 23)
	a := engine.EncodeText("alpha beta /tmp/foo.go")
	b := engine.EncodeText("alpha beta /tmp/foo.go")
	if len(a) != 64 || len(b) != 64 {
		t.Fatalf("unexpected vector dims: %d %d", len(a), len(b))
	}
	for i := range a {
		if a[i] != b[i] {
			t.Fatalf("encode mismatch at %d: %f vs %f", i, a[i], b[i])
		}
	}
}

func TestVSABindUnbindSimilarity(t *testing.T) {
	engine := NewVSAEngine(128, 11)
	role := engine.EncodeText("entity")
	filler := engine.EncodeText("scheduler")
	bound := engine.Bind(role, filler)
	recovered := engine.Unbind(bound, role)
	if sim := engine.Similarity(recovered, filler); sim < 0.55 {
		t.Fatalf("expected recovered similarity >= 0.55, got %.4f", sim)
	}
}

func TestVSAEncodeMapValuesStableAcrossKeyOrder(t *testing.T) {
	engine := NewVSAEngine(96, 7)
	left := map[string]string{
		"role_b": "beta",
		"role_a": "alpha",
		"role_c": "gamma",
	}
	right := map[string]string{
		"role_c": "gamma",
		"role_a": "alpha",
		"role_b": "beta",
	}
	a := engine.EncodeMapValues(left)
	b := engine.EncodeMapValues(right)
	for i := range a {
		if a[i] != b[i] {
			t.Fatalf("map encoding mismatch at %d: %f vs %f", i, a[i], b[i])
		}
	}
}
