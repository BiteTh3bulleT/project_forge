package controllane

import (
	"context"
	"testing"

	"forge/projectforge/services/core/internal/aios/domain"
)

func TestEnvelopeValidationMissingFields(t *testing.T) {
	v := NewDeterministicValidator()
	def, _ := NewStaticActionRegistry().Get(domain.ActionCreateNote)
	req := validBaseRequest(domain.ActionCreateNote)
	req.Action = ""
	req.Actor.ID = ""
	req.Source = ""
	req.Scope.WorkspaceID = ""
	req.Provenance.Actor = ""
	req.RequestedAt = 0
	issues := v.ValidateEnvelope(req, def)
	if len(issues) == 0 {
		t.Fatalf("expected envelope issues for missing required fields")
	}
}

func TestActionValidationMatrix(t *testing.T) {
	v := NewDeterministicValidator()
	reg := NewStaticActionRegistry()
	store := NewInMemorySemanticStore()

	seedReq := validBaseRequest(domain.ActionCreateNote)
	seedReq.Payload = map[string]any{
		"id":      "note-a",
		"type":    string(domain.NoteFact),
		"title":   "Note A",
		"content": "content",
	}
	defSeed, _ := reg.Get(seedReq.Action)
	if issues := v.ValidatePayload(seedReq, defSeed, store); len(issues) > 0 {
		t.Fatalf("seed note payload invalid: %v", issues)
	}
	store.CreateNote(domain.MemoryNote{
		ID:         "note-a",
		Type:       domain.NoteFact,
		Title:      "Note A",
		Content:    "content",
		Scope:      domain.ForgeScope{WorkspaceID: "ws-main"},
		Confidence: 0.8,
		Status:     domain.NoteActive,
		CreatedAt:  1,
		UpdatedAt:  1,
		Provenance: domain.Provenance{Actor: "seed", ActorType: "test"},
	})

	tests := []struct {
		name    string
		req     domain.SyscallRequest
		wantErr bool
	}{
		{
			name: "valid create note",
			req: func() domain.SyscallRequest {
				r := validBaseRequest(domain.ActionCreateNote)
				r.Payload = map[string]any{"type": string(domain.NoteFact), "title": "t", "content": "c"}
				return r
			}(),
		},
		{
			name: "invalid create note",
			req: func() domain.SyscallRequest {
				r := validBaseRequest(domain.ActionCreateNote)
				r.Payload = map[string]any{"type": "not-a-note-type", "title": ""}
				return r
			}(),
			wantErr: true,
		},
		{
			name: "valid create link",
			req: func() domain.SyscallRequest {
				r := validBaseRequest(domain.ActionCreateLink)
				r.Payload = map[string]any{"type": string(domain.LinkSupports), "sourceId": "note-a", "targetId": "note-a2", "confidence": 0.7}
				store.CreateNote(domain.MemoryNote{ID: "note-a2", Type: domain.NoteFact, Title: "b", Content: "c", Scope: domain.ForgeScope{WorkspaceID: "ws-main"}, Status: domain.NoteActive, CreatedAt: 1, UpdatedAt: 1, Provenance: domain.Provenance{Actor: "seed", ActorType: "test"}})
				return r
			}(),
		},
		{
			name: "invalid create link",
			req: func() domain.SyscallRequest {
				r := validBaseRequest(domain.ActionCreateLink)
				r.Payload = map[string]any{"type": string(domain.LinkSupports), "sourceId": "note-a", "targetId": "note-a", "confidence": 2.0}
				return r
			}(),
			wantErr: true,
		},
		{
			name: "valid update state",
			req: func() domain.SyscallRequest {
				r := validBaseRequest(domain.ActionUpdateState)
				r.Payload = map[string]any{"key": "k", "value": map[string]any{"v": 1}, "derivedFrom": []string{"note-a"}}
				return r
			}(),
		},
		{
			name: "invalid update state",
			req: func() domain.SyscallRequest {
				r := validBaseRequest(domain.ActionUpdateState)
				r.Payload = map[string]any{"value": "missing key"}
				return r
			}(),
			wantErr: true,
		},
		{
			name: "valid open loop",
			req: func() domain.SyscallRequest {
				r := validBaseRequest(domain.ActionOpenLoop)
				r.Payload = map[string]any{"title": "loop title", "state": string(domain.LoopOpen), "priority": "high"}
				return r
			}(),
		},
		{
			name: "invalid open loop",
			req: func() domain.SyscallRequest {
				r := validBaseRequest(domain.ActionOpenLoop)
				r.Payload = map[string]any{"title": "", "state": string(domain.LoopArchived)}
				return r
			}(),
			wantErr: true,
		},
		{
			name: "valid derive model",
			req: func() domain.SyscallRequest {
				r := validBaseRequest(domain.ActionDeriveModel)
				r.Payload = map[string]any{"type": "routing", "expression": "x+y", "derivedFrom": []string{"note-a"}, "supportCount": 1}
				return r
			}(),
		},
		{
			name: "invalid derive model",
			req: func() domain.SyscallRequest {
				r := validBaseRequest(domain.ActionDeriveModel)
				r.Payload = map[string]any{"type": "routing", "derivedFrom": []string{"note-a"}, "supportCount": 2}
				return r
			}(),
			wantErr: true,
		},
		{
			name: "valid mark superseded",
			req: func() domain.SyscallRequest {
				r := validBaseRequest(domain.ActionMarkSuperseded)
				r.Payload = map[string]any{
					"oldObjectId": "note-a",
					"newObjectId": "note-a2",
					"reason":      "new evidence",
				}
				return r
			}(),
		},
		{
			name: "invalid mark superseded",
			req: func() domain.SyscallRequest {
				r := validBaseRequest(domain.ActionMarkSuperseded)
				r.Payload = map[string]any{
					"oldObjectId": "note-a",
					"newObjectId": "note-a",
					"reason":      "",
				}
				return r
			}(),
			wantErr: true,
		},
		{
			name: "valid register contradiction",
			req: func() domain.SyscallRequest {
				r := validBaseRequest(domain.ActionRegisterContradict)
				r.Payload = map[string]any{
					"leftObjectId":  "note-a",
					"rightObjectId": "note-a2",
					"reason":        "claims mismatch",
					"severity":      "medium",
					"confidence":    0.6,
				}
				return r
			}(),
		},
		{
			name: "invalid register contradiction",
			req: func() domain.SyscallRequest {
				r := validBaseRequest(domain.ActionRegisterContradict)
				r.Payload = map[string]any{
					"leftObjectId":  "note-a",
					"rightObjectId": "note-a",
					"reason":        "",
					"confidence":    2.0,
				}
				return r
			}(),
			wantErr: true,
		},
		{
			name: "valid compile context",
			req: func() domain.SyscallRequest {
				r := validBaseRequest(domain.ActionCompileContext)
				r.Payload = map[string]any{"query": "summarize", "budget": map[string]any{"maxTokens": 10, "maxEvents": 5, "maxNotes": 5}}
				return r
			}(),
		},
		{
			name: "valid compile context with snapshot options",
			req: func() domain.SyscallRequest {
				r := validBaseRequest(domain.ActionCompileContext)
				r.Payload = map[string]any{
					"query":              "summarize",
					"budget":             map[string]any{"maxTokens": 10, "maxEvents": 5, "maxNotes": 5},
					"persistSnapshot":    true,
					"renderSnapshotCard": false,
					"snapshotKind":       "restore",
					"restoreSnapshot": map[string]any{
						"snapshotKind": "restore",
						"snapshotId":   "ctx-snap-1",
						"evidence":     map[string]any{"source": "seed"},
					},
				}
				return r
			}(),
		},
		{
			name: "invalid compile context",
			req: func() domain.SyscallRequest {
				r := validBaseRequest(domain.ActionCompileContext)
				r.Payload = map[string]any{"budget": map[string]any{"maxTokens": 0, "maxEvents": 1, "maxNotes": 1}}
				return r
			}(),
			wantErr: true,
		},
		{
			name: "invalid compile context snapshot options",
			req: func() domain.SyscallRequest {
				r := validBaseRequest(domain.ActionCompileContext)
				r.Payload = map[string]any{
					"query":              "summarize",
					"budget":             map[string]any{"maxTokens": 10, "maxEvents": 5, "maxNotes": 5},
					"persistSnapshot":    "yes",
					"renderSnapshotCard": true,
				}
				return r
			}(),
			wantErr: true,
		},
		{
			name: "invalid compile context restore snapshot",
			req: func() domain.SyscallRequest {
				r := validBaseRequest(domain.ActionCompileContext)
				r.Payload = map[string]any{
					"query":  "summarize",
					"budget": map[string]any{"maxTokens": 10, "maxEvents": 5, "maxNotes": 5},
					"restoreSnapshot": map[string]any{
						"snapshotId": "ctx-snap-1",
					},
				}
				return r
			}(),
			wantErr: true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			def, ok := reg.Get(tc.req.Action)
			if !ok {
				t.Fatalf("action %s not in registry", tc.req.Action)
			}
			issues := v.ValidatePayload(tc.req, def, store)
			if tc.wantErr && len(issues) == 0 {
				t.Fatalf("expected payload validation errors")
			}
			if !tc.wantErr && len(issues) > 0 {
				t.Fatalf("unexpected payload validation errors: %v", issues)
			}
		})
	}
}

func TestCloseLoopAndArchiveValidation(t *testing.T) {
	k, store, _ := newTestKernel()
	ctx := context.Background()
	loopReq := validBaseRequest(domain.ActionOpenLoop)
	loopReq.Payload = map[string]any{"id": "loop-1", "title": "work", "state": string(domain.LoopOpen), "priority": "medium"}
	if _, err := k.Process(ctx, loopReq); err != nil {
		t.Fatalf("open loop failed: %v", err)
	}
	store.CreateNote(domain.MemoryNote{
		ID:         "note-x",
		Type:       domain.NoteFact,
		Title:      "x",
		Content:    "x",
		Scope:      domain.ForgeScope{WorkspaceID: "ws-main"},
		Confidence: 1,
		Status:     domain.NoteArchived,
		CreatedAt:  1,
		UpdatedAt:  1,
		Provenance: domain.Provenance{Actor: "seed", ActorType: "test"},
	})

	v := NewDeterministicValidator()
	reg := NewStaticActionRegistry()
	defClose, _ := reg.Get(domain.ActionCloseLoop)
	closeReq := validBaseRequest(domain.ActionCloseLoop)
	closeReq.Payload = map[string]any{"loopId": "loop-1", "reason": "done"}
	if issues := v.ValidatePayload(closeReq, defClose, store); len(issues) > 0 {
		t.Fatalf("expected close loop to validate, got %v", issues)
	}

	defArchive, _ := reg.Get(domain.ActionArchiveNote)
	archiveReq := validBaseRequest(domain.ActionArchiveNote)
	archiveReq.Payload = map[string]any{"noteId": "note-x", "reason": "archive"}
	if issues := v.ValidatePayload(archiveReq, defArchive, store); len(issues) == 0 {
		t.Fatalf("expected archived->archived invalid transition to be rejected")
	}
}
