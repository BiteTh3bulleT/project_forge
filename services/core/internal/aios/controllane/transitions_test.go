package controllane

import (
	"testing"

	"forge/projectforge/services/core/internal/aios/domain"
)

func TestOpenLoopTransitions(t *testing.T) {
	valid := [][2]domain.OpenLoopState{
		{domain.LoopOpen, domain.LoopInProgress},
		{domain.LoopOpen, domain.LoopBlocked},
		{domain.LoopOpen, domain.LoopResolved},
		{domain.LoopInProgress, domain.LoopBlocked},
		{domain.LoopInProgress, domain.LoopResolved},
		{domain.LoopBlocked, domain.LoopInProgress},
		{domain.LoopBlocked, domain.LoopResolved},
		{domain.LoopResolved, domain.LoopArchived},
	}
	for _, tc := range valid {
		if !IsValidOpenLoopTransition(tc[0], tc[1]) {
			t.Fatalf("expected transition %s -> %s to be valid", tc[0], tc[1])
		}
	}
	if IsValidOpenLoopTransition(domain.LoopArchived, domain.LoopOpen) {
		t.Fatalf("archived loop should be terminal")
	}
}

func TestNoteTransitions(t *testing.T) {
	if !IsValidNoteTransition(domain.NoteActive, domain.NoteArchived) {
		t.Fatalf("active -> archived should be valid")
	}
	if !IsValidNoteTransition(domain.NoteActive, domain.NoteSuperseded) {
		t.Fatalf("active -> superseded should be valid")
	}
	if IsValidNoteTransition(domain.NoteArchived, domain.NoteActive) {
		t.Fatalf("archived note should be terminal")
	}
}

func TestModelTransitions(t *testing.T) {
	if !IsValidModelTransition(domain.ModelProvisional, domain.ModelPromoted) {
		t.Fatalf("provisional -> promoted should be valid")
	}
	if !IsValidModelTransition(domain.ModelProvisional, domain.ModelDeprecated) {
		t.Fatalf("provisional -> deprecated should be valid")
	}
	if IsValidModelTransition(domain.ModelDeprecated, domain.ModelPromoted) {
		t.Fatalf("deprecated model should be terminal")
	}
}
