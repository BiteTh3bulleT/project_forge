package controllane

import (
	"strings"
	"testing"

	"forge/projectforge/services/core/internal/aios/domain"
)

func TestCompiledContextSnapshotDeterministicSVG(t *testing.T) {
	packet := createTestContextPacketSnapshot("ctx-render-1", "ws-main", 1760002000000)
	packet.Query = "summarize blockers"
	snapshot := buildCompiledContextSnapshot(compiledSnapshotBuildInput{
		Packet:        packet,
		SnapshotID:    packet.ID,
		SnapshotKind:  "restore",
		CorrelationID: "corr-render-1",
		TraceID:       "trace-render-1",
		SyscallID:     "syscall-render-1",
		ProposedBy:    "user",
		CommittedBy:   "forge_kernel",
	}, nil)

	got1 := renderCompiledContextSnapshotSVG(snapshot)
	got2 := renderCompiledContextSnapshotSVG(snapshot)
	if got1 != got2 {
		t.Fatalf("expected stable svg output")
	}
	for _, expected := range []string{"Constraints", "Evidence", "Hypotheses", "Loops", "Objective"} {
		if !strings.Contains(got1, expected) {
			t.Fatalf("expected svg to contain %q", expected)
		}
	}
}

func TestCompiledContextSnapshotRepeatedFingerprintDelta(t *testing.T) {
	packetA := createTestContextPacketSnapshot("ctx-repeat-a", "ws-main", 1760002001000)
	packetA.Query = "summarize blockers"
	packetB := createTestContextPacketSnapshot("ctx-repeat-b", "ws-main", 1760002002000)
	packetB.Query = packetA.Query

	first := buildCompiledContextSnapshot(compiledSnapshotBuildInput{
		Packet:        packetA,
		SnapshotID:    packetA.ID,
		SnapshotKind:  "restore",
		CorrelationID: "corr-repeat-a",
		TraceID:       "trace-repeat-a",
		SyscallID:     "syscall-repeat-a",
		ProposedBy:    "user",
		CommittedBy:   "forge_kernel",
	}, nil)
	second := buildCompiledContextSnapshot(compiledSnapshotBuildInput{
		Packet:        packetB,
		SnapshotID:    packetB.ID,
		SnapshotKind:  "restore",
		CorrelationID: "corr-repeat-b",
		TraceID:       "trace-repeat-b",
		SyscallID:     "syscall-repeat-b",
		ProposedBy:    "user",
		CommittedBy:   "forge_kernel",
	}, &first)

	if first.Header.Fingerprint == "" || second.Header.Fingerprint == "" {
		t.Fatalf("expected fingerprints")
	}
	if first.Header.Fingerprint != second.Header.Fingerprint {
		t.Fatalf("expected identical fingerprints for identical semantic input")
	}
	if second.Header.ParentSnapshotID != first.Header.SnapshotID {
		t.Fatalf("expected parent snapshot linkage, got %q", second.Header.ParentSnapshotID)
	}
	if !second.Delta.FingerprintMatched {
		t.Fatalf("expected repeated compile to flag fingerprint match")
	}
	if len(second.Delta.AddedNodeIDs)+len(second.Delta.RemovedNodeIDs)+len(second.Delta.ChangedNodeIDs)+len(second.Delta.AddedEdgeIDs)+len(second.Delta.RemovedEdgeIDs)+len(second.Delta.ChangedEdgeIDs) != 0 {
		t.Fatalf("expected empty delta for identical repeated compile: %+v", second.Delta)
	}

	restore := compiledContextSnapshotToDomain(second)
	decoded, ok := compiledContextSnapshotFromDomain(restore)
	if !ok {
		t.Fatalf("expected restore snapshot round-trip")
	}
	if decoded.Header.Fingerprint != second.Header.Fingerprint {
		t.Fatalf("expected round-trip fingerprint, got %q", decoded.Header.Fingerprint)
	}
}

func TestContextSnapshotArtifactIgnoredFromGraphFingerprint(t *testing.T) {
	packet := createTestContextPacketSnapshot("ctx-artifact-1", "ws-main", 1760002003000)
	packet.Query = "summarize blockers"
	packet.Artifacts = append(packet.Artifacts, domain.ArtifactRef{
		ID:          "artifact-card-1",
		Type:        "context_snapshot_card",
		URI:         "artifact://context_snapshot/card.svg",
		Scope:       packet.Scope,
		ContentHash: "sha1:card",
		CreatedAt:   packet.CreatedAt,
		Metadata:    map[string]any{"kind": "context_snapshot_card"},
	})

	snapshot := buildCompiledContextSnapshot(compiledSnapshotBuildInput{
		Packet:        packet,
		SnapshotID:    packet.ID,
		SnapshotKind:  "restore",
		CorrelationID: "corr-artifact-1",
		TraceID:       "trace-artifact-1",
		SyscallID:     "syscall-artifact-1",
		ProposedBy:    "user",
		CommittedBy:   "forge_kernel",
	}, nil)
	for _, node := range snapshot.Graph.Nodes {
		if strings.Contains(node.ID, "artifact:artifact-card-1") {
			t.Fatalf("snapshot card artifact should not participate in restore graph")
		}
	}
}
