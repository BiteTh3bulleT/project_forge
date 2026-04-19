package controllane

import (
	"fmt"

	"forge/projectforge/services/core/internal/aios/domain"
)

func createTestJournalEvent(id, workspaceID, correlationID string, ts int64) domain.JournalEvent {
	return domain.JournalEvent{
		ID:        id,
		Type:      "semantic_syscall.create_note",
		Timestamp: ts,
		Source:    "forge_kernel",
		Scope: domain.ForgeScope{
			WorkspaceID: workspaceID,
			LaneID:      "control.semantic",
		},
		Payload: map[string]any{
			"action": "CREATE_NOTE",
		},
		CorrelationID: correlationID,
		Provenance: domain.Provenance{
			Actor:     "operator",
			ActorType: "user",
			Source:    "test",
			TraceID:   correlationID + ":trace",
		},
	}
}

func createTestMemoryNote(id, workspaceID, title string, ts int64) domain.MemoryNote {
	return domain.MemoryNote{
		ID:      id,
		Type:    domain.NoteFact,
		Title:   title,
		Content: "content-" + id,
		Scope: domain.ForgeScope{
			WorkspaceID: workspaceID,
			LaneID:      "memory.notes",
		},
		Confidence: 0.8,
		Status:     domain.NoteActive,
		CreatedAt:  ts,
		UpdatedAt:  ts,
		Provenance: domain.Provenance{
			Actor:     "operator",
			ActorType: "user",
			Source:    "test",
			TraceID:   "trace-" + id,
		},
	}
}

func createTestSemanticLink(id, workspaceID, sourceID, targetID string, ts int64) domain.SemanticLink {
	return domain.SemanticLink{
		ID:       id,
		Type:     domain.LinkSupports,
		SourceID: sourceID,
		TargetID: targetID,
		Scope: domain.ForgeScope{
			WorkspaceID: workspaceID,
			LaneID:      "memory.links",
		},
		Confidence: 0.7,
		Provenance: domain.Provenance{
			Actor:     "operator",
			ActorType: "user",
			Source:    "test",
			TraceID:   "trace-" + id,
		},
		CreatedAt: ts,
	}
}

func createTestStateItem(id, workspaceID, key string, value any, ts int64) domain.StateItem {
	return domain.StateItem{
		ID:  id,
		Key: key,
		Value: map[string]any{
			"value": value,
		},
		Scope: domain.ForgeScope{
			WorkspaceID: workspaceID,
			LaneID:      "control.state",
		},
		Status:      domain.StateActive,
		DerivedFrom: []string{"note-a"},
		UpdatedAt:   ts,
	}
}

func createTestStateHistory(itemID, key, workspaceID string, idx int, ts int64) StateVersionRecord {
	return StateVersionRecord{
		ID:          int64(idx),
		StateItemID: itemID,
		StateKey:    key,
		WorkspaceID: workspaceID,
		PreviousValue: map[string]any{
			"value": idx - 1,
		},
		NewValue: map[string]any{
			"value": idx,
		},
		ChangedBy:     "operator",
		DerivedFrom:   []string{"note-a"},
		SyscallID:     fmt.Sprintf("state-%d", idx),
		CorrelationID: fmt.Sprintf("corr-state-%d", idx),
		TraceID:       fmt.Sprintf("trace-state-%d", idx),
		CreatedAt:     ts,
		Metadata:      map[string]any{},
	}
}

func createTestOpenLoop(id, workspaceID string, ts int64) domain.OpenLoop {
	return domain.OpenLoop{
		ID:    id,
		Title: "loop-" + id,
		State: domain.LoopOpen,
		Scope: domain.ForgeScope{
			WorkspaceID: workspaceID,
			LaneID:      "control.loops",
		},
		Priority:     "high",
		Owner:        "operator",
		Blocker:      "",
		NextAction:   "do work",
		RelatedNotes: []string{"note-a"},
		CreatedFrom:  "syscall",
		CreatedAt:    ts,
		UpdatedAt:    ts,
	}
}

func createTestArtifactRef(id, workspaceID string, ts int64) domain.ArtifactRef {
	return domain.ArtifactRef{
		ID:   id,
		Type: "evidence",
		URI:  "artifacts/" + id + ".json",
		Scope: domain.ForgeScope{
			WorkspaceID: workspaceID,
			LaneID:      "io.artifacts",
		},
		ContentHash: "sha256:" + id,
		CreatedAt:   ts,
		Provenance: domain.Provenance{
			Actor:     "operator",
			ActorType: "user",
			Source:    "test",
			TraceID:   "trace-" + id,
		},
		Metadata: map[string]any{
			"kind": "artifact_ref",
		},
	}
}

func createTestDerivedModel(id, workspaceID string, ts int64) domain.AdaptivePolicyModel {
	return domain.AdaptivePolicyModel{
		ID:         id,
		Type:       "routing",
		Expression: map[string]any{"formula": "score=evidence*confidence"},
		DerivedFrom: []string{
			"note-a",
			"note-b",
		},
		SupportCount: 2,
		Confidence:   0.62,
		Status:       domain.ModelProvisional,
		Scope: domain.ForgeScope{
			WorkspaceID: workspaceID,
			LaneID:      "compute.models",
		},
		CreatedAt: ts,
	}
}

func createTestContradictionRecord(id, workspaceID, leftID, rightID string, ts int64) ContradictionRecord {
	return ContradictionRecord{
		ID:            id,
		LeftID:        leftID,
		LeftKind:      "memory_note",
		RightID:       rightID,
		RightKind:     "memory_note",
		Reason:        "conflicting claims",
		Severity:      "medium",
		Confidence:    0.75,
		WorkspaceID:   workspaceID,
		CorrelationID: "corr-" + id,
		TraceID:       "trace-" + id,
		SyscallID:     "syscall-" + id,
		ProposedBy:    "test",
		CommittedBy:   "forge_kernel",
		Metadata:      map[string]any{},
		CreatedAt:     ts,
		Provenance: domain.Provenance{
			Actor:     "operator",
			ActorType: "user",
			Source:    "test",
			TraceID:   "trace-" + id,
		},
	}
}

func createTestSupersessionRecord(id, workspaceID, oldID, newID string, ts int64) SupersessionRecord {
	return SupersessionRecord{
		ID:            id,
		OldID:         oldID,
		OldKind:       "memory_note",
		NewID:         newID,
		NewKind:       "memory_note",
		Reason:        "newer evidence",
		WorkspaceID:   workspaceID,
		CorrelationID: "corr-" + id,
		TraceID:       "trace-" + id,
		SyscallID:     "syscall-" + id,
		ProposedBy:    "test",
		CommittedBy:   "forge_kernel",
		Metadata:      map[string]any{},
		CreatedAt:     ts,
		Provenance: domain.Provenance{
			Actor:     "operator",
			ActorType: "user",
			Source:    "test",
			TraceID:   "trace-" + id,
		},
	}
}

func createTestContextPacketSnapshot(id, workspaceID string, ts int64) domain.ContextPacket {
	return domain.ContextPacket{
		ID:    id,
		Query: "snapshot query",
		Scope: domain.ForgeScope{
			WorkspaceID: workspaceID,
			LaneID:      "compute.context",
		},
		ActiveState: []domain.StateItem{
			createTestStateItem("state-a", workspaceID, "runtime.mode", "deterministic", ts),
		},
		OpenLoops: []domain.OpenLoop{
			createTestOpenLoop("loop-a", workspaceID, ts),
		},
		Notes: []domain.MemoryNote{
			createTestMemoryNote("note-a", workspaceID, "a", ts),
		},
		LinkedNotes: []domain.SemanticLink{
			createTestSemanticLink("link-a", workspaceID, "note-a", "note-b", ts),
		},
		Models: []domain.AdaptivePolicyModel{
			createTestDerivedModel("model-a", workspaceID, ts),
		},
		Artifacts: []domain.ArtifactRef{
			createTestArtifactRef("artifact-a", workspaceID, ts),
		},
		RawEvents: []domain.JournalEvent{
			createTestJournalEvent("evt-a", workspaceID, "corr-a", ts),
		},
		Budget: domain.ContextBudget{
			MaxTokens: 2000,
			MaxEvents: 50,
			MaxNotes:  25,
		},
		InclusionReasons: map[string]string{
			"mode": "test",
		},
		CreatedAt: ts,
	}
}
