package semanticdiff

import (
	"errors"
	"strings"
	"testing"
)

func TestComputeGoldenUnicodeDifference(t *testing.T) {
	got, err := Compute(Input{
		Left:  "ＦＯＲＧＥ café CAFÉ; alpha, beta beta １２３",
		Right: "forge CAFE\u0301 alpha 123",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.OperatorVersion != OperatorVersion || got.Content != "beta" || len(got.Tokens) != 1 || got.Tokens[0] != "beta" {
		t.Fatalf("unexpected result: %+v", got)
	}
	if got.ContentHash != "sha256:f44e64e75f3948e9f73f8dfa94721c4ce8cbb4f265c4790c702b2d41cfbf2753" {
		t.Fatalf("unexpected content hash %q", got.ContentHash)
	}
}

func TestComputeIsSetBasedSortedAndDirectional(t *testing.T) {
	forward, err := Compute(Input{Left: "Zulu alpha zulu bravo", Right: "bravo"})
	if err != nil {
		t.Fatal(err)
	}
	if forward.Content != "alpha zulu" {
		t.Fatalf("unexpected sorted diff %q", forward.Content)
	}
	reverse, err := Compute(Input{Left: "bravo", Right: "Zulu alpha zulu bravo"})
	if err != nil {
		t.Fatal(err)
	}
	if reverse.Content != "" || reverse.ContentHash == forward.ContentHash {
		t.Fatalf("direction was not preserved: forward=%+v reverse=%+v", forward, reverse)
	}
}

func TestComputeRejectsInvalidUTF8AndBounds(t *testing.T) {
	if _, err := Compute(Input{Left: string([]byte{0xff}), Right: "ok"}); !errors.Is(err, ErrInvalidUTF8) {
		t.Fatalf("expected invalid UTF-8, got %v", err)
	}
	if _, err := Compute(Input{Left: strings.Repeat("a", MaxInputBytes+1), Right: "ok"}); !errors.Is(err, ErrInputBound) {
		t.Fatalf("expected input bound, got %v", err)
	}
	if _, err := Compute(Input{Left: strings.Repeat("a", MaxTokenRunes+1), Right: "ok"}); !errors.Is(err, ErrInputBound) {
		t.Fatalf("expected token bound, got %v", err)
	}
}

func TestFingerprintHasDeterministicMapOrdering(t *testing.T) {
	a, err := Fingerprint(map[string]any{"z": 2, "a": map[string]any{"y": 1, "b": 2}})
	if err != nil {
		t.Fatal(err)
	}
	b, err := Fingerprint(map[string]any{"a": map[string]any{"b": 2, "y": 1}, "z": 2})
	if err != nil {
		t.Fatal(err)
	}
	if a != b {
		t.Fatalf("fingerprints differ: %s != %s", a, b)
	}
}
