package controllane

import (
	"strings"

	"forge/projectforge/services/core/internal/aios/domain"
)

func IsValidOpenLoopTransition(from, to domain.OpenLoopState) bool {
	if normalize(from) == normalize(to) {
		return false
	}
	switch from {
	case domain.LoopOpen:
		return to == domain.LoopInProgress || to == domain.LoopBlocked || to == domain.LoopResolved
	case domain.LoopInProgress:
		return to == domain.LoopBlocked || to == domain.LoopResolved
	case domain.LoopBlocked:
		return to == domain.LoopInProgress || to == domain.LoopResolved
	case domain.LoopResolved:
		return to == domain.LoopArchived
	case domain.LoopArchived:
		return false
	default:
		return false
	}
}

func IsValidNoteTransition(from, to domain.MemoryNoteStatus) bool {
	if normalize(from) == normalize(to) {
		return false
	}
	switch from {
	case domain.NoteActive:
		return to == domain.NoteArchived || to == domain.NoteSuperseded
	case domain.NoteSuperseded:
		return to == domain.NoteArchived
	case domain.NoteArchived:
		return false
	default:
		return false
	}
}

func IsValidModelTransition(from, to domain.AdaptivePolicyModelStatus) bool {
	if normalize(from) == normalize(to) {
		return false
	}
	switch from {
	case domain.ModelProvisional:
		return to == domain.ModelPromoted || to == domain.ModelDeprecated
	case domain.ModelPromoted:
		return to == domain.ModelDeprecated
	case domain.ModelDeprecated:
		return false
	default:
		return false
	}
}

func normalize[T ~string](v T) string {
	return strings.TrimSpace(strings.ToLower(string(v)))
}
