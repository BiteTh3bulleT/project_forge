package vsaprojection

import (
	"errors"
	"reflect"
	"testing"
)

func TestBuildIsDeterministicAcrossInputOrder(t *testing.T) {
	scope := Scope{WorkspaceID: "ws-1", LaneID: "memory"}
	sources := []Source{
		{ID: 2, WorkspaceID: "ws-1", LaneID: "memory", Type: "fact", SourcePath: "b.go", Summary: "beta", Tags: []string{"two", "one"}},
		{ID: 1, WorkspaceID: "ws-1", LaneID: "memory", Type: "fact", SourcePath: "a.go", Summary: "alpha", Entities: []string{"Forge"}, SupportCount: 2},
	}
	links := []Link{{ID: 9, WorkspaceID: "ws-1", LaneID: "memory", FromObservationID: 1, ToObservationID: 2, RelationType: "supports"}}
	first, err := Build(scope, DefaultAlgorithm(), sources, links)
	if err != nil {
		t.Fatalf("build first: %v", err)
	}
	second, err := Build(scope, DefaultAlgorithm(), []Source{sources[1], sources[0]}, links)
	if err != nil {
		t.Fatalf("build second: %v", err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("projection changed with input order\nfirst=%+v\nsecond=%+v", first, second)
	}
	if first.Manifest.ManifestHash == "" || first.Manifest.SourceSetHash == "" || len(first.Pointers) != 2 || len(first.Associations) != 1 {
		t.Fatalf("incomplete projection: %+v", first)
	}
}

func TestBuildRejectsLegacyUnscopedSource(t *testing.T) {
	_, err := Build(Scope{WorkspaceID: "ws-1", LaneID: "memory"}, DefaultAlgorithm(), []Source{{ID: 1, WorkspaceID: "", LaneID: "", Type: "legacy"}}, nil)
	if !errors.Is(err, ErrInvalidSource) {
		t.Fatalf("error=%v want ErrInvalidSource", err)
	}
}

func TestManifestDetectsSourceAlgorithmAndLinkDivergence(t *testing.T) {
	scope := Scope{WorkspaceID: "ws-1", LaneID: "memory"}
	sources := []Source{{ID: 1, WorkspaceID: "ws-1", LaneID: "memory", Type: "fact", Summary: "alpha"}, {ID: 2, WorkspaceID: "ws-1", LaneID: "memory", Type: "fact", Summary: "beta"}}
	links := []Link{{ID: 1, WorkspaceID: "ws-1", LaneID: "memory", FromObservationID: 1, ToObservationID: 2, RelationType: "related"}}
	base, err := Build(scope, DefaultAlgorithm(), sources, links)
	if err != nil {
		t.Fatal(err)
	}

	tamperedSources := append([]Source{}, sources...)
	tamperedSources[0].Summary = "changed"
	tampered, err := Build(scope, DefaultAlgorithm(), tamperedSources, links)
	if err != nil {
		t.Fatal(err)
	}
	if base.Manifest.ManifestHash == tampered.Manifest.ManifestHash {
		t.Fatal("source tamper preserved manifest hash")
	}

	algorithm := DefaultAlgorithm()
	algorithm.Dimensions = 64
	differentAlgorithm, err := Build(scope, algorithm, sources, links)
	if err != nil {
		t.Fatal(err)
	}
	if base.Manifest.ManifestHash == differentAlgorithm.Manifest.ManifestHash {
		t.Fatal("algorithm change preserved manifest hash")
	}

	differentLinks, err := Build(scope, DefaultAlgorithm(), sources, []Link{{ID: 1, WorkspaceID: "ws-1", LaneID: "memory", FromObservationID: 1, ToObservationID: 2, RelationType: "blocks"}})
	if err != nil {
		t.Fatal(err)
	}
	if base.Manifest.ManifestHash == differentLinks.Manifest.ManifestHash {
		t.Fatal("link change preserved manifest hash")
	}
	if err := VerifyExpectedManifest(base, tampered.Manifest.ManifestHash); err == nil {
		t.Fatal("expected manifest mismatch")
	}
}

func TestBuildRejectsDuplicateAndCrossScopeInputs(t *testing.T) {
	scope := Scope{WorkspaceID: "ws-1", LaneID: "memory"}
	source := Source{ID: 1, WorkspaceID: "ws-1", LaneID: "memory", Type: "fact"}
	if _, err := Build(scope, DefaultAlgorithm(), []Source{source, source}, nil); !errors.Is(err, ErrInvalidSource) {
		t.Fatalf("duplicate source error=%v", err)
	}
	if _, err := Build(scope, DefaultAlgorithm(), []Source{source}, []Link{{ID: 1, WorkspaceID: "ws-2", LaneID: "memory", FromObservationID: 1, ToObservationID: 2, RelationType: "related"}}); !errors.Is(err, ErrInvalidLink) {
		t.Fatalf("cross-scope link error=%v", err)
	}
}
