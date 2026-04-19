package controllane

import (
	"fmt"
	"strings"

	"forge/projectforge/services/core/internal/aios/domain"
)

type SemanticActionValidator interface {
	ValidateEnvelope(req domain.SyscallRequest, def ActionDefinition) []domain.SyscallError
	ValidatePayload(req domain.SyscallRequest, def ActionDefinition, store SemanticReadStore) []domain.SyscallError
}

type DeterministicValidator struct{}

func NewDeterministicValidator() *DeterministicValidator {
	return &DeterministicValidator{}
}

func (v *DeterministicValidator) ValidateEnvelope(req domain.SyscallRequest, def ActionDefinition) []domain.SyscallError {
	issues := append([]domain.SyscallError{}, req.Validate()...)
	if !isKnownSource(req.Source) {
		issues = append(issues, errField(domain.ErrInvalidProvenance, "source", "unknown source value"))
	}
	if def.Mutating {
		if strings.TrimSpace(req.Provenance.Actor) == "" || strings.TrimSpace(req.Provenance.ActorType) == "" {
			issues = append(issues, errField(domain.ErrInvalidProvenance, "provenance", "mutating actions require provenance.actor and provenance.actorType"))
		}
	}
	if strings.TrimSpace(req.Scope.WorkspaceID) == "" {
		issues = append(issues, errField(domain.ErrInvalidScope, "scope.workspaceId", "workspace scope is required"))
	}
	return issues
}

func (v *DeterministicValidator) ValidatePayload(req domain.SyscallRequest, def ActionDefinition, store SemanticReadStore) []domain.SyscallError {
	switch def.Action {
	case domain.ActionCreateNote:
		return validateCreateNote(req)
	case domain.ActionCreateLink:
		return validateCreateLink(req, store)
	case domain.ActionUpdateState:
		return validateUpdateState(req)
	case domain.ActionOpenLoop:
		return validateOpenLoop(req, store)
	case domain.ActionCloseLoop:
		return validateCloseLoop(req, store)
	case domain.ActionMarkSuperseded:
		return validateMarkSuperseded(req, store)
	case domain.ActionRegisterContradict:
		return validateRegisterContradiction(req, store)
	case domain.ActionDeriveModel:
		return validateDeriveModel(req, store)
	case domain.ActionArchiveNote:
		return validateArchiveNote(req, store)
	case domain.ActionCompileContext:
		return validateCompileContext(req)
	default:
		return []domain.SyscallError{errField(domain.ErrUnsupportedAction, "action", "unsupported action")}
	}
}

func validateCreateNote(req domain.SyscallRequest) []domain.SyscallError {
	var issues []domain.SyscallError
	typ := readString(req.Payload, "type")
	if !isAllowedNoteType(domain.MemoryNoteType(typ)) {
		issues = append(issues, errField(domain.ErrInvalidPayload, "payload.type", "note type is invalid"))
	}
	if strings.TrimSpace(readString(req.Payload, "title")) == "" {
		issues = append(issues, errField(domain.ErrMissingRequiredField, "payload.title", "title is required"))
	}
	if strings.TrimSpace(readString(req.Payload, "content")) == "" {
		issues = append(issues, errField(domain.ErrMissingRequiredField, "payload.content", "content is required"))
	}
	conf := readFloat(req.Payload, "confidence", 0.7)
	if conf < 0 || conf > 1 {
		issues = append(issues, errField(domain.ErrInvalidPayload, "payload.confidence", "confidence must be between 0 and 1"))
	}
	status := readString(req.Payload, "status")
	if strings.TrimSpace(status) != "" && !isAllowedNoteStatus(domain.MemoryNoteStatus(status)) {
		issues = append(issues, errField(domain.ErrInvalidPayload, "payload.status", "note status is invalid"))
	}
	return issues
}

func validateCreateLink(req domain.SyscallRequest, store SemanticReadStore) []domain.SyscallError {
	var issues []domain.SyscallError
	linkType := readString(req.Payload, "type")
	if !isAllowedLinkType(domain.SemanticLinkType(linkType)) {
		issues = append(issues, errField(domain.ErrInvalidPayload, "payload.type", "link type is invalid"))
	}
	sourceID := strings.TrimSpace(readString(req.Payload, "sourceId"))
	targetID := strings.TrimSpace(readString(req.Payload, "targetId"))
	if sourceID == "" {
		issues = append(issues, errField(domain.ErrMissingRequiredField, "payload.sourceId", "sourceId is required"))
	}
	if targetID == "" {
		issues = append(issues, errField(domain.ErrMissingRequiredField, "payload.targetId", "targetId is required"))
	}
	if sourceID != "" && sourceID == targetID {
		issues = append(issues, errField(domain.ErrInvalidPayload, "payload.targetId", "sourceId and targetId cannot be identical"))
	}
	conf := readFloat(req.Payload, "confidence", 0.7)
	if conf < 0 || conf > 1 {
		issues = append(issues, errField(domain.ErrInvalidPayload, "payload.confidence", "confidence must be between 0 and 1"))
	}
	if sourceID != "" && store != nil && !store.ExistsObject(sourceID) {
		issues = append(issues, errField(domain.ErrNotFound, "payload.sourceId", "source object not found"))
	}
	if targetID != "" && store != nil && !store.ExistsObject(targetID) {
		issues = append(issues, errField(domain.ErrNotFound, "payload.targetId", "target object not found"))
	}
	return issues
}

func validateUpdateState(req domain.SyscallRequest) []domain.SyscallError {
	var issues []domain.SyscallError
	if strings.TrimSpace(readString(req.Payload, "key")) == "" {
		issues = append(issues, errField(domain.ErrMissingRequiredField, "payload.key", "state key is required"))
	}
	if _, ok := req.Payload["value"]; !ok {
		issues = append(issues, errField(domain.ErrMissingRequiredField, "payload.value", "state value is required"))
	}
	derivedFrom := readStringSlice(req.Payload, "derivedFrom")
	if len(derivedFrom) == 0 {
		issues = append(issues, errField(domain.ErrMissingRequiredField, "payload.derivedFrom", "derivedFrom is required"))
	}
	status := readString(req.Payload, "status")
	if strings.TrimSpace(status) != "" && !isAllowedStateStatus(domain.StateItemStatus(status)) {
		issues = append(issues, errField(domain.ErrInvalidPayload, "payload.status", "state status is invalid"))
	}
	return issues
}

func validateOpenLoop(req domain.SyscallRequest, store SemanticReadStore) []domain.SyscallError {
	var issues []domain.SyscallError
	loopID := strings.TrimSpace(readString(req.Payload, "id"))
	if loopID != "" && store != nil {
		if existing, ok := store.FindLoop(loopID); ok {
			stateRaw := readString(req.Payload, "state")
			if strings.TrimSpace(stateRaw) != "" {
				next := domain.OpenLoopState(stateRaw)
				if !isAllowedOpenLoopState(next) {
					issues = append(issues, errField(domain.ErrInvalidPayload, "payload.state", "open loop state is invalid"))
				} else if next != existing.State && !IsValidOpenLoopTransition(existing.State, next) {
					issues = append(issues, errField(domain.ErrInvalidStateTransition, "payload.state", "invalid loop transition"))
				}
			}
			priority := readString(req.Payload, "priority")
			if strings.TrimSpace(priority) != "" && !isAllowedPriority(priority) {
				issues = append(issues, errField(domain.ErrInvalidPayload, "payload.priority", "priority must be low|medium|high"))
			}
			related := readStringSlice(req.Payload, "relatedNotes")
			for _, noteID := range related {
				if store != nil && !store.ExistsObject(noteID) {
					issues = append(issues, errField(domain.ErrNotFound, "payload.relatedNotes", fmt.Sprintf("related object %q not found", noteID)))
				}
			}
			if strings.TrimSpace(readString(req.Payload, "state")) == "" &&
				strings.TrimSpace(readString(req.Payload, "title")) == "" &&
				strings.TrimSpace(readString(req.Payload, "blocker")) == "" &&
				strings.TrimSpace(readString(req.Payload, "nextAction")) == "" &&
				strings.TrimSpace(readString(req.Payload, "owner")) == "" &&
				strings.TrimSpace(readString(req.Payload, "priority")) == "" &&
				len(related) == 0 {
				issues = append(issues, errField(domain.ErrMissingRequiredField, "payload.state", "loop update requires at least one changed field"))
			}
			return issues
		}
	}
	if strings.TrimSpace(readString(req.Payload, "title")) == "" {
		issues = append(issues, errField(domain.ErrMissingRequiredField, "payload.title", "loop title is required"))
	}
	state := readString(req.Payload, "state")
	if state == "" {
		state = string(domain.LoopOpen)
	}
	loopState := domain.OpenLoopState(state)
	if !isAllowedOpenLoopState(loopState) {
		issues = append(issues, errField(domain.ErrInvalidPayload, "payload.state", "open loop state is invalid"))
	} else if loopState == domain.LoopResolved || loopState == domain.LoopArchived {
		issues = append(issues, errField(domain.ErrInvalidPayload, "payload.state", "new loops must start as open, in_progress, or blocked"))
	}
	priority := readString(req.Payload, "priority")
	if priority == "" {
		priority = "medium"
	}
	if !isAllowedPriority(priority) {
		issues = append(issues, errField(domain.ErrInvalidPayload, "payload.priority", "priority must be low|medium|high"))
	}
	related := readStringSlice(req.Payload, "relatedNotes")
	for _, noteID := range related {
		if store != nil && !store.ExistsObject(noteID) {
			issues = append(issues, errField(domain.ErrNotFound, "payload.relatedNotes", fmt.Sprintf("related object %q not found", noteID)))
		}
	}
	return issues
}

func validateCloseLoop(req domain.SyscallRequest, store SemanticReadStore) []domain.SyscallError {
	var issues []domain.SyscallError
	loopID := strings.TrimSpace(readString(req.Payload, "loopId"))
	if loopID == "" {
		issues = append(issues, errField(domain.ErrMissingRequiredField, "payload.loopId", "loopId is required"))
		return issues
	}
	if strings.TrimSpace(readString(req.Payload, "reason")) == "" && strings.TrimSpace(readString(req.Payload, "outcome")) == "" {
		issues = append(issues, errField(domain.ErrMissingRequiredField, "payload.reason", "closure reason or outcome is required"))
	}
	if store == nil {
		return issues
	}
	loop, ok := store.FindLoop(loopID)
	if !ok {
		issues = append(issues, errField(domain.ErrNotFound, "payload.loopId", "loop not found"))
		return issues
	}
	if !IsValidOpenLoopTransition(loop.State, domain.LoopResolved) {
		issues = append(issues, errField(domain.ErrInvalidStateTransition, "payload.loopId", "loop cannot transition to resolved from current state"))
	}
	return issues
}

func validateMarkSuperseded(req domain.SyscallRequest, store SemanticReadStore) []domain.SyscallError {
	var issues []domain.SyscallError
	oldID := strings.TrimSpace(readString(req.Payload, "oldObjectId"))
	newID := strings.TrimSpace(readString(req.Payload, "newObjectId"))
	if oldID == "" {
		issues = append(issues, errField(domain.ErrMissingRequiredField, "payload.oldObjectId", "oldObjectId is required"))
	}
	if newID == "" {
		issues = append(issues, errField(domain.ErrMissingRequiredField, "payload.newObjectId", "newObjectId is required"))
	}
	if oldID != "" && oldID == newID {
		issues = append(issues, errField(domain.ErrInvalidPayload, "payload.newObjectId", "oldObjectId and newObjectId cannot be identical"))
	}
	if strings.TrimSpace(readString(req.Payload, "reason")) == "" {
		issues = append(issues, errField(domain.ErrMissingRequiredField, "payload.reason", "reason is required"))
	}
	if store != nil {
		if oldID != "" && !store.ExistsObject(oldID) {
			issues = append(issues, errField(domain.ErrNotFound, "payload.oldObjectId", "old object not found"))
		}
		if newID != "" && !store.ExistsObject(newID) {
			issues = append(issues, errField(domain.ErrNotFound, "payload.newObjectId", "new object not found"))
		}
		oldKind, oldScope, oldKnown := resolveKnownObject(store, oldID)
		newKind, newScope, newKnown := resolveKnownObject(store, newID)
		if oldKnown && !scopeCompatible(req.Scope, oldScope) {
			issues = append(issues, errField(domain.ErrInvalidScope, "payload.oldObjectId", "old object is outside request workspace scope"))
		}
		if newKnown && !scopeCompatible(req.Scope, newScope) {
			issues = append(issues, errField(domain.ErrInvalidScope, "payload.newObjectId", "new object is outside request workspace scope"))
		}
		if oldKnown && newKnown && oldKind != newKind {
			issues = append(issues, errField(domain.ErrInvalidPayload, "payload.newObjectId", "supersession requires compatible object kinds"))
		}
	}
	return issues
}

func validateRegisterContradiction(req domain.SyscallRequest, store SemanticReadStore) []domain.SyscallError {
	var issues []domain.SyscallError
	leftID := strings.TrimSpace(readString(req.Payload, "leftObjectId"))
	rightID := strings.TrimSpace(readString(req.Payload, "rightObjectId"))
	if leftID == "" {
		issues = append(issues, errField(domain.ErrMissingRequiredField, "payload.leftObjectId", "leftObjectId is required"))
	}
	if rightID == "" {
		issues = append(issues, errField(domain.ErrMissingRequiredField, "payload.rightObjectId", "rightObjectId is required"))
	}
	if leftID != "" && leftID == rightID {
		issues = append(issues, errField(domain.ErrInvalidPayload, "payload.rightObjectId", "contradiction requires different object ids"))
	}
	if strings.TrimSpace(readString(req.Payload, "reason")) == "" {
		issues = append(issues, errField(domain.ErrMissingRequiredField, "payload.reason", "reason is required"))
	}
	conf := readFloat(req.Payload, "confidence", 0.7)
	if conf < 0 || conf > 1 {
		issues = append(issues, errField(domain.ErrInvalidPayload, "payload.confidence", "confidence must be between 0 and 1"))
	}
	severity := readString(req.Payload, "severity")
	if severity != "" && !isAllowedSeverity(severity) {
		issues = append(issues, errField(domain.ErrInvalidPayload, "payload.severity", "severity must be low|medium|high"))
	}
	if store != nil {
		if leftID != "" && !store.ExistsObject(leftID) {
			issues = append(issues, errField(domain.ErrNotFound, "payload.leftObjectId", "left object not found"))
		}
		if rightID != "" && !store.ExistsObject(rightID) {
			issues = append(issues, errField(domain.ErrNotFound, "payload.rightObjectId", "right object not found"))
		}
		_, leftScope, leftKnown := resolveKnownObject(store, leftID)
		_, rightScope, rightKnown := resolveKnownObject(store, rightID)
		if leftKnown && !scopeCompatible(req.Scope, leftScope) {
			issues = append(issues, errField(domain.ErrInvalidScope, "payload.leftObjectId", "left object is outside request workspace scope"))
		}
		if rightKnown && !scopeCompatible(req.Scope, rightScope) {
			issues = append(issues, errField(domain.ErrInvalidScope, "payload.rightObjectId", "right object is outside request workspace scope"))
		}
	}
	return issues
}

func validateDeriveModel(req domain.SyscallRequest, store SemanticReadStore) []domain.SyscallError {
	var issues []domain.SyscallError
	modelID := strings.TrimSpace(readString(req.Payload, "id"))
	if store != nil && modelID != "" {
		if existing, ok := store.FindModel(modelID); ok {
			nextStatus := strings.TrimSpace(readString(req.Payload, "status"))
			if nextStatus == "" {
				issues = append(issues, errField(domain.ErrMissingRequiredField, "payload.status", "status is required when updating an existing model"))
				return issues
			}
			status := domain.AdaptivePolicyModelStatus(nextStatus)
			if status != domain.ModelPromoted && status != domain.ModelDeprecated {
				issues = append(issues, errField(domain.ErrInvalidPayload, "payload.status", "existing model transitions allow promoted|deprecated only"))
				return issues
			}
			if !IsValidModelTransition(existing.Status, status) {
				issues = append(issues, errField(domain.ErrInvalidStateTransition, "payload.status", "invalid model lifecycle transition"))
			}
			conf := readFloat(req.Payload, "confidence", existing.Confidence)
			if conf < 0 || conf > 1 {
				issues = append(issues, errField(domain.ErrInvalidPayload, "payload.confidence", "confidence must be between 0 and 1"))
			}
			if derivedFrom := readStringSlice(req.Payload, "derivedFrom"); len(derivedFrom) > 0 {
				support := readInt(req.Payload, "supportCount", len(derivedFrom))
				if support != len(derivedFrom) {
					issues = append(issues, errField(domain.ErrInvalidPayload, "payload.supportCount", "supportCount must match derivedFrom length"))
				}
			}
			return issues
		}
	}
	if strings.TrimSpace(readString(req.Payload, "type")) == "" {
		issues = append(issues, errField(domain.ErrMissingRequiredField, "payload.type", "model type is required"))
	}
	if _, ok := req.Payload["expression"]; !ok {
		issues = append(issues, errField(domain.ErrMissingRequiredField, "payload.expression", "expression is required"))
	}
	derivedFrom := readStringSlice(req.Payload, "derivedFrom")
	if len(derivedFrom) == 0 {
		issues = append(issues, errField(domain.ErrMissingRequiredField, "payload.derivedFrom", "derivedFrom is required"))
	}
	support := readInt(req.Payload, "supportCount", len(derivedFrom))
	if support < 0 {
		issues = append(issues, errField(domain.ErrInvalidPayload, "payload.supportCount", "supportCount cannot be negative"))
	}
	if len(derivedFrom) > 0 && support != len(derivedFrom) {
		issues = append(issues, errField(domain.ErrInvalidPayload, "payload.supportCount", "supportCount must match derivedFrom length"))
	}
	conf := readFloat(req.Payload, "confidence", 0.7)
	if conf < 0 || conf > 1 {
		issues = append(issues, errField(domain.ErrInvalidPayload, "payload.confidence", "confidence must be between 0 and 1"))
	}
	status := readString(req.Payload, "status")
	if status != "" && domain.AdaptivePolicyModelStatus(status) != domain.ModelProvisional {
		issues = append(issues, errField(domain.ErrInvalidStateTransition, "payload.status", "derived models must start as provisional in Phase 2"))
	}
	return issues
}

func validateArchiveNote(req domain.SyscallRequest, store SemanticReadStore) []domain.SyscallError {
	var issues []domain.SyscallError
	noteID := strings.TrimSpace(readString(req.Payload, "noteId"))
	if noteID == "" {
		issues = append(issues, errField(domain.ErrMissingRequiredField, "payload.noteId", "noteId is required"))
		return issues
	}
	if strings.TrimSpace(readString(req.Payload, "reason")) == "" {
		issues = append(issues, errField(domain.ErrMissingRequiredField, "payload.reason", "reason is required"))
	}
	if store == nil {
		return issues
	}
	note, ok := store.FindNote(noteID)
	if !ok {
		issues = append(issues, errField(domain.ErrNotFound, "payload.noteId", "note not found"))
		return issues
	}
	if !IsValidNoteTransition(note.Status, domain.NoteArchived) {
		issues = append(issues, errField(domain.ErrInvalidStateTransition, "payload.noteId", "note cannot transition to archived from current status"))
	}
	return issues
}

func validateCompileContext(req domain.SyscallRequest) []domain.SyscallError {
	var issues []domain.SyscallError
	query := strings.TrimSpace(readString(req.Payload, "query"))
	if query == "" {
		if v, ok := req.Metadata["query"]; ok {
			query = strings.TrimSpace(fmt.Sprintf("%v", v))
		}
	}
	if query == "" {
		issues = append(issues, errField(domain.ErrMissingRequiredField, "payload.query", "query or task descriptor is required"))
	}
	if budget, ok := req.Payload["budget"]; ok {
		bmap, ok := budget.(map[string]any)
		if !ok {
			issues = append(issues, errField(domain.ErrInvalidPayload, "payload.budget", "budget must be an object"))
			return issues
		}
		if readInt(bmap, "maxTokens", 0) <= 0 {
			issues = append(issues, errField(domain.ErrInvalidPayload, "payload.budget.maxTokens", "maxTokens must be positive"))
		}
		if readInt(bmap, "maxEvents", 0) <= 0 {
			issues = append(issues, errField(domain.ErrInvalidPayload, "payload.budget.maxEvents", "maxEvents must be positive"))
		}
		if readInt(bmap, "maxNotes", 0) <= 0 {
			issues = append(issues, errField(domain.ErrInvalidPayload, "payload.budget.maxNotes", "maxNotes must be positive"))
		}
	}
	return issues
}

func isKnownSource(src domain.ActionSource) bool {
	switch src {
	case domain.SourceUser, domain.SourceSystem, domain.SourceInternal, domain.SourceAdapter, domain.SourceFutureIRIS, domain.SourceTest:
		return true
	default:
		return false
	}
}

func isAllowedNoteType(t domain.MemoryNoteType) bool {
	switch t {
	case domain.NoteFact, domain.NotePreference, domain.NoteGoal, domain.NoteDecision, domain.NoteProcedure,
		domain.NoteEpisode, domain.NoteOpenLoop, domain.NoteArtifact, domain.NotePolicy, domain.NoteSystem:
		return true
	default:
		return false
	}
}

func isAllowedLinkType(t domain.SemanticLinkType) bool {
	switch t {
	case domain.LinkRelatesTo, domain.LinkSupports, domain.LinkContradicts, domain.LinkSupersedes,
		domain.LinkDependsOn, domain.LinkCauses, domain.LinkAbout, domain.LinkDerivedFrom, domain.LinkBlocks, domain.LinkResolves:
		return true
	default:
		return false
	}
}

func isAllowedNoteStatus(s domain.MemoryNoteStatus) bool {
	switch s {
	case domain.NoteActive, domain.NoteSuperseded, domain.NoteArchived:
		return true
	default:
		return false
	}
}

func isAllowedStateStatus(s domain.StateItemStatus) bool {
	switch s {
	case domain.StateActive, domain.StateSuperseded, domain.StateArchived:
		return true
	default:
		return false
	}
}

func isAllowedOpenLoopState(s domain.OpenLoopState) bool {
	switch s {
	case domain.LoopOpen, domain.LoopInProgress, domain.LoopBlocked, domain.LoopResolved, domain.LoopArchived:
		return true
	default:
		return false
	}
}

func isAllowedPriority(v string) bool {
	switch strings.TrimSpace(strings.ToLower(v)) {
	case "low", "medium", "high":
		return true
	default:
		return false
	}
}

func isAllowedSeverity(v string) bool {
	switch strings.TrimSpace(strings.ToLower(v)) {
	case "low", "medium", "high":
		return true
	default:
		return false
	}
}

func errField(code domain.SyscallErrorCode, field, message string) domain.SyscallError {
	return domain.SyscallError{Code: code, Field: field, Message: message}
}

func resolveKnownObject(store SemanticReadStore, id string) (kind string, scope domain.ForgeScope, ok bool) {
	if strings.TrimSpace(id) == "" || store == nil {
		return "", domain.ForgeScope{}, false
	}
	if note, found := store.FindNote(id); found {
		return "memory_note", note.Scope, true
	}
	if loop, found := store.FindLoop(id); found {
		return "open_loop", loop.Scope, true
	}
	if model, found := store.FindModel(id); found {
		return "derived_model", model.Scope, true
	}
	return "", domain.ForgeScope{}, false
}

func scopeCompatible(expected, actual domain.ForgeScope) bool {
	if strings.TrimSpace(expected.WorkspaceID) == "" {
		return false
	}
	if strings.TrimSpace(actual.WorkspaceID) != strings.TrimSpace(expected.WorkspaceID) {
		return false
	}
	expectedLane := strings.TrimSpace(expected.LaneID)
	actualLane := strings.TrimSpace(actual.LaneID)
	if expectedLane == "" || actualLane == "" {
		return true
	}
	return expectedLane == actualLane
}

func readString(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	v, ok := m[key]
	if !ok || v == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprintf("%v", v))
}

func readStringSlice(m map[string]any, key string) []string {
	if m == nil {
		return nil
	}
	v, ok := m[key]
	if !ok || v == nil {
		return nil
	}
	switch x := v.(type) {
	case []string:
		out := make([]string, 0, len(x))
		for _, item := range x {
			item = strings.TrimSpace(item)
			if item != "" {
				out = append(out, item)
			}
		}
		return out
	case []any:
		out := make([]string, 0, len(x))
		for _, item := range x {
			s := strings.TrimSpace(fmt.Sprintf("%v", item))
			if s != "" {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
}

func readFloat(m map[string]any, key string, def float64) float64 {
	if m == nil {
		return def
	}
	v, ok := m[key]
	if !ok || v == nil {
		return def
	}
	switch x := v.(type) {
	case float64:
		return x
	case float32:
		return float64(x)
	case int:
		return float64(x)
	case int64:
		return float64(x)
	default:
		return def
	}
}

func readInt(m map[string]any, key string, def int) int {
	if m == nil {
		return def
	}
	v, ok := m[key]
	if !ok || v == nil {
		return def
	}
	switch x := v.(type) {
	case int:
		return x
	case int64:
		return int(x)
	case float64:
		return int(x)
	case float32:
		return int(x)
	default:
		return def
	}
}
