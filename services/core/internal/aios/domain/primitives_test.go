package domain

import "testing"

func TestPrimitivesInstantiate(t *testing.T) {
	now := NowMillis()
	scope := ForgeScope{WorkspaceID: "ws-main", LaneID: "fs.read", SelectedPaths: []string{"docs"}}
	prov := Provenance{Actor: "operator", ActorType: "user", Source: "chat", TraceID: "corr-1"}

	evt := JournalEvent{
		ID:         "evt-1",
		Type:       "job.created",
		Timestamp:  now,
		Source:     "jobs.service",
		Scope:      scope,
		Payload:    map[string]any{"jobId": "job-1"},
		Provenance: prov,
	}
	if evt.ID == "" || evt.Scope.WorkspaceID == "" {
		t.Fatalf("journal event did not instantiate correctly: %+v", evt)
	}

	note := MemoryNote{
		ID:         "note-1",
		Type:       NoteFact,
		Title:      "Fact",
		Content:    "FORGE owns canonical state.",
		Scope:      scope,
		Confidence: 0.9,
		Status:     NoteActive,
		CreatedAt:  now,
		UpdatedAt:  now,
		Provenance: prov,
	}
	link := SemanticLink{
		ID:         "link-1",
		Type:       LinkSupports,
		SourceID:   "note-1",
		TargetID:   "state-1",
		Confidence: 0.8,
		Provenance: prov,
		CreatedAt:  now,
	}
	state := StateItem{
		ID:          "state-1",
		Key:         "kernel.mode",
		Value:       map[string]any{"value": "deterministic"},
		Scope:       scope,
		Status:      StateActive,
		DerivedFrom: []string{"evt-1"},
		UpdatedAt:   now,
	}
	loop := OpenLoop{
		ID:           "loop-1",
		Title:        "Wire semantic syscall gate",
		State:        LoopOpen,
		Priority:     "high",
		Owner:        "core",
		Blocker:      "",
		NextAction:   "define validator interface",
		RelatedNotes: []string{"note-1"},
		CreatedFrom:  "evt-1",
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	artifact := ArtifactRef{
		ID:          "art-1",
		Type:        "task_packet",
		URI:         "artifacts/packets/pkt-1.json",
		ContentHash: "sha256:abc123",
		CreatedAt:   now,
		Provenance:  prov,
		Metadata:    map[string]any{"packetId": 1},
	}
	model := AdaptivePolicyModel{
		ID:           "model-1",
		Type:         "routing_score",
		Expression:   map[string]any{"feature": "success_rate"},
		DerivedFrom:  []string{"evt-1", "note-1"},
		SupportCount: 3,
		Confidence:   0.7,
		Status:       ModelProvisional,
		CreatedAt:    now,
	}

	if note.ID == "" || link.ID == "" || state.ID == "" || loop.ID == "" || artifact.ID == "" || model.ID == "" {
		t.Fatalf("one or more primitives failed to instantiate")
	}
}

func TestSemanticActionAndContextPacketValidation(t *testing.T) {
	now := NowMillis()
	scope := ForgeScope{WorkspaceID: "ws-main"}
	prov := Provenance{Actor: "librarian.intake", ActorType: "service"}

	action := SemanticAction{
		ID:     "act-1",
		Action: ActionCreateNote,
		Actor: ActorIdentity{
			ID:   "operator",
			Kind: string(SourceUser),
		},
		Source:      SourceUser,
		Scope:       scope,
		Payload:     map[string]any{"type": string(NoteFact), "title": "New observation", "content": "content"},
		RequestedAt: now,
		Provenance:  prov,
	}
	if issues := action.Validate(); len(issues) > 0 {
		t.Fatalf("expected valid semantic action, got issues: %v", issues)
	}

	packet := ContextPacket{
		ID:          "ctx-1",
		Query:       "summarize active loops",
		Scope:       scope,
		ActiveState: []StateItem{},
		OpenLoops:   []OpenLoop{},
		Notes:       []MemoryNote{},
		LinkedNotes: []SemanticLink{},
		Models:      []AdaptivePolicyModel{},
		Artifacts:   []ArtifactRef{},
		RawEvents:   []JournalEvent{},
		Budget: ContextBudget{
			MaxTokens: 4000,
			MaxEvents: 50,
			MaxNotes:  50,
		},
		InclusionReasons: map[string]string{},
		CreatedAt:        now,
	}
	if issues := packet.Validate(); len(issues) > 0 {
		t.Fatalf("expected valid context packet, got issues: %v", issues)
	}
}
