package librarian

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"forge/projectforge/services/core/internal/aios/domain"
)

const (
	cellIntake        = "IntakeCell"
	cellCatalog       = "CatalogCell"
	cellLinker        = "LinkerCell"
	cellContradiction = "ContradictionCell"
	cellState         = "StateCell"
	cellPattern       = "PatternCell"
	cellRecall        = "RecallCell"
	cellCleanup       = "CleanupCell"
	phase4Version     = "phase5-v1"
)

type IntakeRuntimeCell struct{}
type CatalogRuntimeCell struct{}
type LinkerRuntimeCell struct{}
type ContradictionRuntimeCell struct{}
type StateRuntimeCell struct{}
type PatternRuntimeCell struct {
	SupportThreshold int
}
type RecallRuntimeCell struct{}
type CleanupRuntimeCell struct{}

func DefaultCells() []RuntimeCell {
	return []RuntimeCell{
		IntakeRuntimeCell{},
		CatalogRuntimeCell{},
		LinkerRuntimeCell{},
		ContradictionRuntimeCell{},
		StateRuntimeCell{},
		PatternRuntimeCell{SupportThreshold: 2},
		RecallRuntimeCell{},
		CleanupRuntimeCell{},
	}
}

func (IntakeRuntimeCell) Name() string                { return cellIntake }
func (IntakeRuntimeCell) Version() string             { return phase4Version }
func (IntakeRuntimeCell) Lane() string                { return "compute" }
func (IntakeRuntimeCell) Dependencies() []string      { return nil }
func (CatalogRuntimeCell) Name() string               { return cellCatalog }
func (CatalogRuntimeCell) Version() string            { return phase4Version }
func (CatalogRuntimeCell) Lane() string               { return "compute" }
func (CatalogRuntimeCell) Dependencies() []string     { return []string{cellIntake} }
func (LinkerRuntimeCell) Name() string                { return cellLinker }
func (LinkerRuntimeCell) Version() string             { return phase4Version }
func (LinkerRuntimeCell) Lane() string                { return "compute" }
func (LinkerRuntimeCell) Dependencies() []string      { return []string{cellCatalog} }
func (ContradictionRuntimeCell) Name() string         { return cellContradiction }
func (ContradictionRuntimeCell) Version() string      { return phase4Version }
func (ContradictionRuntimeCell) Lane() string         { return "compute" }
func (ContradictionRuntimeCell) Dependencies() []string { return []string{cellCatalog, cellLinker} }
func (StateRuntimeCell) Name() string                 { return cellState }
func (StateRuntimeCell) Version() string              { return phase4Version }
func (StateRuntimeCell) Lane() string                 { return "compute" }
func (StateRuntimeCell) Dependencies() []string       { return []string{cellCatalog} }
func (c PatternRuntimeCell) Name() string             { return cellPattern }
func (PatternRuntimeCell) Version() string            { return phase4Version }
func (PatternRuntimeCell) Lane() string               { return "compute" }
func (PatternRuntimeCell) Dependencies() []string     { return []string{cellCatalog, cellLinker} }
func (RecallRuntimeCell) Name() string                { return cellRecall }
func (RecallRuntimeCell) Version() string             { return phase4Version }
func (RecallRuntimeCell) Lane() string                { return "compute" }
func (RecallRuntimeCell) Dependencies() []string      { return []string{cellState} }
func (CleanupRuntimeCell) Name() string               { return cellCleanup }
func (CleanupRuntimeCell) Version() string            { return phase4Version }
func (CleanupRuntimeCell) Lane() string               { return "compute" }
func (CleanupRuntimeCell) Dependencies() []string     { return []string{cellRecall} }

func (IntakeRuntimeCell) CanRun(_ context.Context, _ CellRunContext) (bool, string) { return true, "" }
func (CatalogRuntimeCell) CanRun(_ context.Context, in CellRunContext) (bool, string) {
	if len(in.ExistingActions) == 0 {
		return false, "no candidate actions to catalog"
	}
	return true, ""
}
func (LinkerRuntimeCell) CanRun(_ context.Context, in CellRunContext) (bool, string) {
	if len(in.ExistingActions) == 0 && len(in.ActiveNotes) == 0 {
		return false, "no note candidates for linking"
	}
	return true, ""
}
func (ContradictionRuntimeCell) CanRun(_ context.Context, in CellRunContext) (bool, string) {
	if len(in.ExistingActions) == 0 {
		return false, "no candidate actions for contradiction analysis"
	}
	return true, ""
}
func (StateRuntimeCell) CanRun(_ context.Context, _ CellRunContext) (bool, string) { return true, "" }
func (PatternRuntimeCell) CanRun(_ context.Context, in CellRunContext) (bool, string) {
	if len(in.ActiveNotes)+len(in.ExistingActions) < 2 {
		return false, "insufficient evidence volume for pattern proposals"
	}
	return true, ""
}
func (RecallRuntimeCell) CanRun(_ context.Context, _ CellRunContext) (bool, string) { return true, "" }
func (CleanupRuntimeCell) CanRun(_ context.Context, _ CellRunContext) (bool, string) { return true, "" }

func (c IntakeRuntimeCell) Run(ctx context.Context, in CellRunContext) (CellRunResult, error) {
	start := time.Now()
	out := CellRunResult{CellName: c.Name(), CellVersion: c.Version(), Confidence: 0.6}
	text := requestText(in)
	lower := strings.ToLower(text)

	if isLowValueMessage(lower) {
		out.AnalysisNotes = append(out.AnalysisNotes, "low-value input; no durable intake proposal")
	} else {
		if strings.Contains(lower, "i prefer ") {
			content := text
			title := "Preference: " + shortTitle(content, "preference")
			out.ProposedActions = append(out.ProposedActions, newCellAction(in, c.Name(), c.Version(), domain.ActionCreateNote, map[string]any{
				"id":         objectID("note", in.Event.ID, "preference", content),
				"type":       string(domain.NotePreference),
				"title":      title,
				"content":    strings.TrimSpace(content),
				"confidence": 0.85,
				"status":     string(domain.NoteActive),
			}))
		}
		if strings.Contains(lower, "remember that ") {
			content := text
			title := "Fact: " + shortTitle(content, "fact")
			out.ProposedActions = append(out.ProposedActions, newCellAction(in, c.Name(), c.Version(), domain.ActionCreateNote, map[string]any{
				"id":         objectID("note", in.Event.ID, "fact", content),
				"type":       string(domain.NoteFact),
				"title":      title,
				"content":    strings.TrimSpace(content),
				"confidence": 0.75,
				"status":     string(domain.NoteActive),
			}))
		}
		if strings.Contains(lower, "we should ") || strings.HasPrefix(lower, "should ") {
			content := text
			title := "Goal: " + shortTitle(content, "goal")
			noteType := domain.NoteGoal
			if strings.Contains(lower, "decide") || strings.Contains(lower, "use ") {
				noteType = domain.NoteDecision
			}
			out.ProposedActions = append(out.ProposedActions, newCellAction(in, c.Name(), c.Version(), domain.ActionCreateNote, map[string]any{
				"id":         objectID("note", in.Event.ID, "goal", content),
				"type":       string(noteType),
				"title":      title,
				"content":    strings.TrimSpace(content),
				"confidence": 0.72,
				"status":     string(domain.NoteActive),
			}))
		}
		if strings.Contains(lower, "architecture") && strings.Contains(lower, "use ") {
			content := text
			out.ProposedActions = append(out.ProposedActions, newCellAction(in, c.Name(), c.Version(), domain.ActionCreateNote, map[string]any{
				"id":         objectID("note", in.Event.ID, "decision", content),
				"type":       string(domain.NoteDecision),
				"title":      "Architecture decision",
				"content":    strings.TrimSpace(content),
				"confidence": 0.78,
				"status":     string(domain.NoteActive),
			}))
		}
		if strings.Contains(lower, "policy") && strings.Contains(lower, "use ") {
			content := text
			out.ProposedActions = append(out.ProposedActions, newCellAction(in, c.Name(), c.Version(), domain.ActionCreateNote, map[string]any{
				"id":         objectID("note", in.Event.ID, "policy", content),
				"type":       string(domain.NotePolicy),
				"title":      "Policy direction",
				"content":    strings.TrimSpace(content),
				"confidence": 0.74,
				"status":     string(domain.NoteActive),
			}))
		}
		if blocker := extractPhrase(lower, []string{"the blocker is ", "blocked by "}); blocker != "" {
			blockerText := strings.TrimSpace(blocker)
			out.ProposedActions = append(out.ProposedActions, newCellAction(in, c.Name(), c.Version(), domain.ActionOpenLoop, map[string]any{
				"id":         objectID("loop", in.Event.ID, "blocker", blockerText),
				"title":      "Resolve blocker: " + shortTitle(blockerText, "blocker"),
				"state":      string(domain.LoopOpen),
				"priority":   "high",
				"owner":      nonEmptyTrim(in.Actor.ID, "system"),
				"blocker":    blockerText,
				"createdFrom": in.Event.ID,
			}))
		}
	}

	if in.Semantic != nil {
		candidates, err := in.Semantic.ExtractCandidates(ctx, InferenceRequest{
			Event:         in.Event,
			Scope:         in.Scope,
			CorrelationID: in.CorrelationID,
			TraceID:       in.TraceID,
			Content:       text,
			Hints:         map[string]any{"cell": c.Name()},
		})
		if err != nil {
			out.Warnings = append(out.Warnings, "semantic inference extraction failed: "+err.Error())
		} else {
			for _, cand := range candidates {
				out.ProposedActions = append(out.ProposedActions, normalizeCandidateFromInference(in, c.Name(), c.Version(), cand))
			}
		}
	}

	out.Duration = time.Since(start)
	return out, nil
}

func (c CatalogRuntimeCell) Run(_ context.Context, in CellRunContext) (CellRunResult, error) {
	start := time.Now()
	out := CellRunResult{CellName: c.Name(), CellVersion: c.Version(), Confidence: 0.7}
	for _, action := range in.ExistingActions {
		switch action.Action {
		case domain.ActionCreateNote:
			normalized := action
			normalized.Payload = cloneMap(action.Payload)
			typ := strings.ToLower(strings.TrimSpace(readPayloadString(action.Payload, "type")))
			if !isAllowedNoteType(domain.MemoryNoteType(typ)) {
				out.Warnings = append(out.Warnings, fmt.Sprintf("invalid note type %q normalized to fact", typ))
				typ = string(domain.NoteFact)
			}
			title := strings.TrimSpace(readPayloadString(action.Payload, "title"))
			content := strings.TrimSpace(readPayloadString(action.Payload, "content"))
			if title == "" {
				title = shortTitle(content, "note")
			}
			if content == "" {
				out.Warnings = append(out.Warnings, "discarded malformed note candidate with empty content")
				continue
			}
			conf := readPayloadFloat(action.Payload, "confidence", 0.7)
			if conf <= 0 {
				conf = 0.65
			}
			if conf > 1 {
				conf = 1
			}
			normalized.Payload["id"] = nonEmptyTrim(readPayloadString(action.Payload, "id"), objectID("note", in.Event.ID, typ, title+"|"+content))
			normalized.Payload["type"] = typ
			normalized.Payload["title"] = title
			normalized.Payload["content"] = content
			normalized.Payload["confidence"] = conf
			if strings.TrimSpace(readPayloadString(action.Payload, "status")) == "" {
				normalized.Payload["status"] = string(domain.NoteActive)
			}
			normalized.Metadata = mergeMetadata(normalized.Metadata, map[string]any{
				"catalogedBy": c.Name(),
			})
			normalized.Provenance = withCellProvenance(c.Name(), normalized.Provenance, in.TraceID)
			out.ProposedActions = append(out.ProposedActions, normalized)
		case domain.ActionOpenLoop:
			normalized := action
			normalized.Payload = cloneMap(action.Payload)
			normalized.Payload["title"] = strings.TrimSpace(readPayloadString(action.Payload, "title"))
			if normalized.Payload["title"] == "" {
				out.Warnings = append(out.Warnings, "discarded malformed open loop with empty title")
				continue
			}
			state := strings.TrimSpace(readPayloadString(action.Payload, "state"))
			if state == "" {
				state = string(domain.LoopOpen)
			}
			switch domain.OpenLoopState(state) {
			case domain.LoopOpen, domain.LoopInProgress, domain.LoopBlocked:
			default:
				out.Warnings = append(out.Warnings, "normalized open loop state to open")
				state = string(domain.LoopOpen)
			}
			normalized.Payload["state"] = state
			normalized.Payload["id"] = nonEmptyTrim(readPayloadString(action.Payload, "id"), objectID("loop", in.Event.ID, state, readPayloadString(action.Payload, "title")))
			normalized.Metadata = mergeMetadata(normalized.Metadata, map[string]any{
				"catalogedBy": c.Name(),
			})
			out.ProposedActions = append(out.ProposedActions, normalized)
		}
	}
	out.Duration = time.Since(start)
	return out, nil
}

func (c LinkerRuntimeCell) Run(_ context.Context, in CellRunContext) (CellRunResult, error) {
	start := time.Now()
	out := CellRunResult{CellName: c.Name(), CellVersion: c.Version(), Confidence: 0.65}
	candidateNotes := candidateNoteActions(in.ExistingActions)
	loopIDs := candidateLoopIDs(in.ExistingActions, in.ActiveLoops)
	for _, note := range candidateNotes {
		noteID := readPayloadString(note.Payload, "id")
		title := strings.ToLower(readPayloadString(note.Payload, "title"))
		content := strings.ToLower(readPayloadString(note.Payload, "content"))
		if noteID == "" {
			continue
		}
		for _, existing := range in.ActiveNotes {
			if existing.ID == noteID {
				continue
			}
			if sharedTopicScore(title+" "+content, strings.ToLower(existing.Title+" "+existing.Content)) >= 2 {
				out.ProposedActions = append(out.ProposedActions, newCellAction(in, c.Name(), c.Version(), domain.ActionCreateLink, map[string]any{
					"id":         objectID("link", in.Event.ID, "relates", noteID+"|"+existing.ID),
					"type":       string(domain.LinkRelatesTo),
					"sourceId":   noteID,
					"targetId":   existing.ID,
					"confidence": 0.66,
				}))
			}
		}
		for _, ref := range in.RecentArtifacts {
			refMatch := strings.Contains(content, strings.ToLower(ref.ID)) ||
				(ref.URI != "" && strings.Contains(content, strings.ToLower(ref.URI))) ||
				(ref.ContentHash != "" && strings.Contains(content, strings.ToLower(ref.ContentHash)))
			if !refMatch {
				continue
			}
			out.ProposedActions = append(out.ProposedActions, newCellAction(in, c.Name(), c.Version(), domain.ActionCreateLink, map[string]any{
				"id":         objectID("link", in.Event.ID, "about_artifact", noteID+"|"+ref.ID),
				"type":       string(domain.LinkAbout),
				"sourceId":   noteID,
				"targetId":   ref.ID,
				"confidence": 0.72,
			}))
		}
		if strings.Contains(content, "blocker") {
			for _, loopID := range loopIDs {
				if loopID == "" {
					continue
				}
				out.ProposedActions = append(out.ProposedActions, newCellAction(in, c.Name(), c.Version(), domain.ActionCreateLink, map[string]any{
					"id":         objectID("link", in.Event.ID, "blocks", noteID+"|"+loopID),
					"type":       string(domain.LinkBlocks),
					"sourceId":   noteID,
					"targetId":   loopID,
					"confidence": 0.7,
				}))
			}
		}
	}
	out.Duration = time.Since(start)
	return out, nil
}

func (c ContradictionRuntimeCell) Run(ctx context.Context, in CellRunContext) (CellRunResult, error) {
	start := time.Now()
	out := CellRunResult{CellName: c.Name(), CellVersion: c.Version(), Confidence: 0.62}
	text := strings.ToLower(requestText(in))
	newNotes := candidateNoteActions(in.ExistingActions)
	phrase := extractPhrase(text, []string{"instead of ", "over ", "replace ", "supersede ", "not "})
	for _, note := range newNotes {
		newNoteID := readPayloadString(note.Payload, "id")
		newType := strings.TrimSpace(readPayloadString(note.Payload, "type"))
		if newNoteID == "" {
			continue
		}
		var oldID string
		var oldKind string
		for _, existing := range in.ActiveNotes {
			if existing.ID == newNoteID {
				continue
			}
			if newType != "" && string(existing.Type) != newType {
				continue
			}
			if phrase != "" && strings.Contains(strings.ToLower(existing.Title+" "+existing.Content), phrase) {
				oldID = existing.ID
				oldKind = "memory_note"
				break
			}
		}
		if oldID == "" {
			continue
		}
		if supersessionAlreadyKnown(ctx, in, oldID, newNoteID) {
			continue
		}
		out.ProposedActions = append(out.ProposedActions, newCellAction(in, c.Name(), c.Version(), domain.ActionMarkSuperseded, map[string]any{
			"oldObjectId":   oldID,
			"oldObjectKind": nonEmptyTrim(oldKind, "memory_note"),
			"newObjectId":   newNoteID,
			"newObjectKind": "memory_note",
			"reason":        "new ingest evidence supersedes earlier preference/decision",
		}))
	}

	for _, action := range in.ExistingActions {
		if action.Action != domain.ActionUpdateState {
			continue
		}
		if in.DryRun {
			out.Warnings = append(out.Warnings, "state contradiction proposals skipped during dry-run due unresolved evidence IDs")
			continue
		}
		key := strings.TrimSpace(readPayloadString(action.Payload, "key"))
		if key == "" {
			continue
		}
		newValue := action.Payload["value"]
		for _, current := range in.CurrentState {
			if current.Key != key {
				continue
			}
			if valuesEquivalent(current.Value, newValue) {
				continue
			}
			if contradictionAlreadyKnown(ctx, in, current.ID, in.Event.ID) {
				continue
			}
			out.ProposedActions = append(out.ProposedActions, newCellAction(in, c.Name(), c.Version(), domain.ActionRegisterContradict, map[string]any{
				"leftObjectId":    current.ID,
				"leftObjectKind":  "state_item",
				"rightObjectId":   in.Event.ID,
				"rightObjectKind": "journal_event",
				"reason":          fmt.Sprintf("state key %q proposes a conflicting value", key),
				"severity":        "medium",
				"confidence":      0.6,
			}))
		}
	}
	out.Duration = time.Since(start)
	return out, nil
}

func (c StateRuntimeCell) Run(ctx context.Context, in CellRunContext) (CellRunResult, error) {
	start := time.Now()
	out := CellRunResult{CellName: c.Name(), CellVersion: c.Version(), Confidence: 0.68}
	text := strings.ToLower(requestText(in))
	candidateNotes := candidateNoteActions(in.ExistingActions)
	for _, note := range candidateNotes {
		noteID := readPayloadString(note.Payload, "id")
		noteType := strings.TrimSpace(readPayloadString(note.Payload, "type"))
		content := strings.ToLower(readPayloadString(note.Payload, "content"))
		title := strings.ToLower(readPayloadString(note.Payload, "title"))

		switch domain.MemoryNoteType(noteType) {
		case domain.NoteDecision:
			if strings.Contains(content+" "+title, "architecture") {
				payload := map[string]any{
					"id":         objectID("state", in.Event.ID, "architecture_direction", noteID),
					"key":        "architecture_direction",
					"value":      map[string]any{"value": strings.TrimSpace(readPayloadString(note.Payload, "title"))},
					"derivedFrom": compactIDs(noteID, in.Event.ID),
					"status":     string(domain.StateActive),
				}
				if stateAlreadyCurrent(ctx, in, "architecture_direction", payload["value"]) {
					continue
				}
				out.ProposedActions = append(out.ProposedActions, newCellAction(in, c.Name(), c.Version(), domain.ActionUpdateState, payload))
			}
		case domain.NotePreference, domain.NotePolicy:
			if strings.Contains(content, "snapshot") || strings.Contains(content, "structured memory") {
				payload := map[string]any{
					"id":         objectID("state", in.Event.ID, "context_policy", noteID),
					"key":        "context_policy",
					"value":      map[string]any{"value": "structured_snapshots"},
					"derivedFrom": compactIDs(noteID, in.Event.ID),
					"status":     string(domain.StateActive),
				}
				if stateAlreadyCurrent(ctx, in, "context_policy", payload["value"]) {
					continue
				}
				out.ProposedActions = append(out.ProposedActions, newCellAction(in, c.Name(), c.Version(), domain.ActionUpdateState, payload))
			}
		}
	}

	if strings.Contains(text, "with just forge") || strings.Contains(text, "just forge") {
		payload := map[string]any{
			"id":         objectID("state", in.Event.ID, "current_test_mode", "forge_only"),
			"key":        "current_test_mode",
			"value":      map[string]any{"value": "forge_only"},
			"derivedFrom": []string{in.Event.ID},
			"status":     string(domain.StateActive),
		}
		if !stateAlreadyCurrent(ctx, in, "current_test_mode", payload["value"]) {
			out.ProposedActions = append(out.ProposedActions, newCellAction(in, c.Name(), c.Version(), domain.ActionUpdateState, payload))
		}
	}

	if blocker := extractPhrase(text, []string{"the blocker is ", "blocked by "}); blocker != "" && !hasOpenLoopCandidate(in.ExistingActions) {
		out.ProposedActions = append(out.ProposedActions, newCellAction(in, c.Name(), c.Version(), domain.ActionOpenLoop, map[string]any{
			"id":          objectID("loop", in.Event.ID, "blocker", blocker),
			"title":       "Resolve blocker: " + shortTitle(blocker, "blocker"),
			"state":       string(domain.LoopOpen),
			"priority":    "high",
			"owner":       nonEmptyTrim(in.Actor.ID, "system"),
			"blocker":     strings.TrimSpace(blocker),
			"createdFrom": in.Event.ID,
		}))
	}

	if strings.Contains(text, "resolved") || strings.Contains(text, "fixed") || strings.Contains(text, "done") {
		target := extractResolvedTarget(text)
		if target != "" {
			for _, loop := range in.ActiveLoops {
				if !loopMatch(loop, target) {
					continue
				}
				out.ProposedActions = append(out.ProposedActions, newCellAction(in, c.Name(), c.Version(), domain.ActionCloseLoop, map[string]any{
					"loopId":  loop.ID,
					"reason":  "resolved from ingest event",
					"outcome": strings.TrimSpace(requestText(in)),
				}))
				break
			}
		}
	}
	out.Duration = time.Since(start)
	return out, nil
}

func (c PatternRuntimeCell) Run(ctx context.Context, in CellRunContext) (CellRunResult, error) {
	start := time.Now()
	threshold := c.SupportThreshold
	if threshold <= 0 {
		threshold = 2
	}
	out := CellRunResult{CellName: c.Name(), CellVersion: c.Version(), Confidence: 0.57}
	allNotes := append([]domain.MemoryNote{}, in.ActiveNotes...)
	for _, action := range candidateNoteActions(in.ExistingActions) {
		note := syntheticNoteFromAction(action, in.Scope, in.Provenance, in.Request.RequestedAt)
		if status := domain.MemoryNoteStatus(strings.TrimSpace(readPayloadString(action.Payload, "status"))); status == domain.NoteArchived || status == domain.NoteSuperseded {
			continue
		}
		allNotes = append(allNotes, note)
	}

	type evidence struct {
		id   string
		text string
	}
	policyEvidence := []evidence{}
	policyContrary := []evidence{}
	contradictedEvidence := 0
	for _, note := range allNotes {
		if in.Truth != nil {
			if rows, err := in.Truth.ListContradictionsForObject(ctx, note.ID, in.Scope, 5); err == nil && len(rows) > 0 {
				contradictedEvidence++
				continue
			}
		}
		text := strings.ToLower(note.Title + " " + note.Content)
		if strings.Contains(text, "snapshot") || strings.Contains(text, "structured memory") || strings.Contains(text, "concise project") {
			policyEvidence = append(policyEvidence, evidence{id: note.ID, text: text})
		}
		if strings.Contains(text, "transcript replay") {
			policyContrary = append(policyContrary, evidence{id: note.ID, text: text})
		}
	}
	if len(policyEvidence) >= threshold && len(policyEvidence) > len(policyContrary) && contradictedEvidence == 0 {
		ids := make([]string, 0, len(policyEvidence))
		for _, ev := range policyEvidence {
			ids = append(ids, ev.id)
		}
		sort.Strings(ids)
		if len(ids) > threshold {
			ids = ids[:threshold]
		}
		out.ProposedActions = append(out.ProposedActions, newCellAction(in, c.Name(), c.Version(), domain.ActionDeriveModel, map[string]any{
			"id":           objectID("model", in.Event.ID, "context_policy_preference", strings.Join(ids, "|")),
			"type":         "context_policy_preference",
			"expression":   map[string]any{"formula": "prefer_structured_snapshots_when_evidence_repeats"},
			"derivedFrom":  ids,
			"supportCount": len(ids),
			"confidence":   0.62,
		}))
	} else if contradictedEvidence > 0 {
		out.Warnings = append(out.Warnings, "contradicted evidence blocked pattern model proposal")
	} else if len(policyEvidence) > 0 && len(policyEvidence) < threshold {
		out.AnalysisNotes = append(out.AnalysisNotes, "pattern threshold not met for derived model proposal")
	} else if len(policyContrary) >= len(policyEvidence) && len(policyEvidence) > 0 {
		out.Warnings = append(out.Warnings, "contrary evidence blocks pattern model promotion")
	}
	out.Duration = time.Since(start)
	return out, nil
}

func (c RecallRuntimeCell) Run(ctx context.Context, in CellRunContext) (CellRunResult, error) {
	start := time.Now()
	out := CellRunResult{
		CellName:    c.Name(),
		CellVersion: c.Version(),
		Confidence:  0.5,
		Hints:       map[string]any{},
	}
	text := strings.ToLower(requestText(in))
	relevantState := []string{}
	stateWarnings := []string{}
	for _, item := range in.CurrentState {
		keyLower := strings.ToLower(item.Key)
		if text == "" || strings.Contains(text, keyLower) || strings.Contains(keyLower, "policy") {
			relevantState = append(relevantState, item.Key)
			if in.Truth != nil {
				if rows, err := in.Truth.ListContradictionsForObject(ctx, item.ID, in.Scope, 3); err == nil && len(rows) > 0 {
					stateWarnings = append(stateWarnings, "state "+item.Key+" has active contradiction records")
				}
			}
		}
	}
	relevantLoops := []string{}
	for _, loop := range in.ActiveLoops {
		lower := strings.ToLower(loop.Title + " " + loop.Blocker)
		if text == "" || sharedTopicScore(text, lower) > 0 {
			relevantLoops = append(relevantLoops, loop.ID)
		}
	}
	out.Hints["stateKeys"] = relevantState
	out.Hints["openLoopIds"] = relevantLoops
	if len(stateWarnings) > 0 {
		out.Hints["warnings"] = stateWarnings
	}
	out.AnalysisNotes = append(out.AnalysisNotes,
		fmt.Sprintf("prepared recall hints: %d state keys, %d open loops", len(relevantState), len(relevantLoops)))
	if shouldCompileContext(in) {
		query := strings.TrimSpace(requestText(in))
		if query == "" {
			query = "ingest recall context"
		}
		out.ProposedActions = append(out.ProposedActions, newCellAction(in, c.Name(), c.Version(), domain.ActionCompileContext, map[string]any{
			"query": query,
			"budget": map[string]any{
				"maxTokens": 4000,
				"maxEvents": 50,
				"maxNotes":  50,
			},
		}))
	}
	out.Duration = time.Since(start)
	return out, nil
}

func (c CleanupRuntimeCell) Run(ctx context.Context, in CellRunContext) (CellRunResult, error) {
	start := time.Now()
	out := CellRunResult{
		CellName:    c.Name(),
		CellVersion: c.Version(),
		Confidence:  0.4,
		Hints:       map[string]any{},
	}
	duplicates := duplicateActionCount(in.ExistingActions)
	if duplicates > 0 {
		out.Warnings = append(out.Warnings, fmt.Sprintf("detected %d duplicate candidate actions", duplicates))
		out.Hints["duplicateCandidates"] = duplicates
	}
	staleCutoff := in.Request.RequestedAt - int64((24 * time.Hour).Milliseconds())
	if in.Truth != nil {
		if stale, err := in.Truth.ListStaleLoops(ctx, in.Scope, staleCutoff, 120); err == nil {
			for _, loop := range stale {
				out.Warnings = append(out.Warnings, fmt.Sprintf("stale open loop detected: %s", loop.ID))
			}
		}
	} else {
		for _, loop := range in.ActiveLoops {
			if in.Request.RequestedAt-loop.UpdatedAt > int64((24*time.Hour).Milliseconds()) && loop.State != domain.LoopResolved && loop.State != domain.LoopArchived {
				out.Warnings = append(out.Warnings, fmt.Sprintf("stale open loop detected: %s", loop.ID))
			}
		}
	}
	archiveIDs := readStringSliceAny(in.Request.Metadata, "archiveNoteIds")
	archiveReason := readStringAny(in.Request.Metadata, "archiveReason")
	if len(archiveIDs) > 0 && strings.TrimSpace(archiveReason) != "" {
		for _, noteID := range archiveIDs {
			out.ProposedActions = append(out.ProposedActions, newCellAction(in, c.Name(), c.Version(), domain.ActionArchiveNote, map[string]any{
				"noteId": noteID,
				"reason": archiveReason,
			}))
		}
	}
	out.Duration = time.Since(start)
	return out, nil
}

func normalizeCandidateFromInference(in CellRunContext, cellName, cellVersion string, action domain.SyscallRequest) domain.SyscallRequest {
	next := action
	if strings.TrimSpace(string(next.Source)) == "" {
		next.Source = domain.SourceInternal
	}
	if strings.TrimSpace(next.Actor.ID) == "" {
		next.Actor.ID = "cell.semantic_inference"
	}
	if strings.TrimSpace(next.Actor.Kind) == "" {
		next.Actor.Kind = string(next.Source)
	}
	next.Scope = in.Scope
	next.CorrelationID = in.CorrelationID
	next.TraceID = in.TraceID
	next.RequestedAt = in.Request.RequestedAt
	next.Provenance = withCellProvenance(cellName, next.Provenance, in.TraceID)
	next.Metadata = mergeMetadata(next.Metadata, map[string]any{
		"cellName":    cellName,
		"cellVersion": cellVersion,
		"eventId":     in.Event.ID,
	})
	if strings.TrimSpace(next.ID) == "" {
		next.ID = objectID("action", in.Event.ID, string(next.Action), payloadFingerprint(next.Payload))
	}
	return next
}

func newCellAction(in CellRunContext, cellName, cellVersion string, actionType domain.SemanticActionType, payload map[string]any) domain.SyscallRequest {
	payload = cloneMap(payload)
	actionID := objectID("act", in.Event.ID, cellName, string(actionType)+"|"+payloadFingerprint(payload))
	return domain.SyscallRequest{
		ID:     actionID,
		Action: actionType,
		Actor: domain.ActorIdentity{
			ID:   "cell." + strings.ToLower(strings.TrimSuffix(strings.TrimSuffix(cellName, "Cell"), "Runtime")),
			Kind: string(domain.SourceInternal),
		},
		Source:  domain.SourceInternal,
		Scope:   in.Scope,
		Payload: payload,
		Provenance: domain.Provenance{
			Actor:     "cell." + strings.ToLower(strings.TrimSuffix(strings.TrimSuffix(cellName, "Cell"), "Runtime")),
			ActorType: "internal_cell",
			Source:    "compute." + strings.ToLower(strings.TrimSuffix(cellName, "Cell")),
			TraceID:   in.TraceID,
		},
		CorrelationID: in.CorrelationID,
		TraceID:       in.TraceID,
		RequestedAt:   in.Request.RequestedAt,
		Metadata: map[string]any{
			"cellName":    cellName,
			"cellVersion": cellVersion,
			"eventId":     in.Event.ID,
		},
	}
}

func objectID(parts ...string) string {
	sum := sha1.Sum([]byte(strings.Join(parts, "|")))
	return strings.ToLower(strings.TrimSpace(parts[0])) + "-" + hex.EncodeToString(sum[:8])
}

func payloadFingerprint(payload map[string]any) string {
	if len(payload) == 0 {
		return "empty"
	}
	keys := make([]string, 0, len(payload))
	for k := range payload {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, k+"="+fmt.Sprintf("%v", payload[k]))
	}
	return strings.Join(parts, ";")
}

func requestText(in CellRunContext) string {
	if strings.TrimSpace(in.Request.Content) != "" {
		return strings.TrimSpace(in.Request.Content)
	}
	if txt := readStringAny(in.Event.Payload, "content"); strings.TrimSpace(txt) != "" {
		return strings.TrimSpace(txt)
	}
	if txt := readStringAny(in.Event.Payload, "text"); strings.TrimSpace(txt) != "" {
		return strings.TrimSpace(txt)
	}
	if txt := readStringAny(in.Event.Payload, "message"); strings.TrimSpace(txt) != "" {
		return strings.TrimSpace(txt)
	}
	return ""
}

func readStringAny(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	v, ok := m[key]
	if !ok || v == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprintf("%v", v))
}

func readPayloadString(payload map[string]any, key string) string {
	return readStringAny(payload, key)
}

func readPayloadFloat(payload map[string]any, key string, fallback float64) float64 {
	v, ok := payload[key]
	if !ok || v == nil {
		return fallback
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
		return fallback
	}
}

func isLowValueMessage(text string) bool {
	text = strings.TrimSpace(text)
	if text == "" {
		return true
	}
	low := map[string]struct{}{
		"hi": {}, "hello": {}, "hey": {}, "thanks": {}, "thank you": {}, "ok": {}, "okay": {},
	}
	if _, ok := low[text]; ok {
		return true
	}
	if len(strings.Fields(text)) <= 2 && (strings.Contains(text, "hello") || strings.Contains(text, "hi ")) {
		return true
	}
	return false
}

func shortTitle(content, fallback string) string {
	content = strings.TrimSpace(content)
	if content == "" {
		return fallback
	}
	parts := strings.Fields(content)
	if len(parts) > 8 {
		parts = parts[:8]
	}
	return strings.Join(parts, " ")
}

func extractPhrase(text string, markers []string) string {
	for _, marker := range markers {
		idx := strings.Index(text, marker)
		if idx < 0 {
			continue
		}
		rest := strings.TrimSpace(text[idx+len(marker):])
		if rest == "" {
			continue
		}
		if cut := strings.IndexAny(rest, ".!?"); cut > 0 {
			rest = rest[:cut]
		}
		return strings.TrimSpace(rest)
	}
	return ""
}

func withCellProvenance(cellName string, prov domain.Provenance, traceID string) domain.Provenance {
	if strings.TrimSpace(prov.Actor) == "" {
		prov.Actor = "cell." + strings.ToLower(strings.TrimSuffix(cellName, "Cell"))
	}
	if strings.TrimSpace(prov.ActorType) == "" {
		prov.ActorType = "internal_cell"
	}
	if strings.TrimSpace(prov.Source) == "" {
		prov.Source = "compute." + strings.ToLower(strings.TrimSuffix(cellName, "Cell"))
	}
	if strings.TrimSpace(prov.TraceID) == "" {
		prov.TraceID = traceID
	}
	return prov
}

func mergeMetadata(base map[string]any, extra map[string]any) map[string]any {
	out := cloneMap(base)
	for k, v := range extra {
		out[k] = v
	}
	return out
}

func cloneMap(in map[string]any) map[string]any {
	if in == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func candidateNoteActions(actions []domain.SyscallRequest) []domain.SyscallRequest {
	out := make([]domain.SyscallRequest, 0)
	for _, action := range actions {
		if action.Action == domain.ActionCreateNote {
			out = append(out, action)
		}
	}
	return out
}

func candidateLoopIDs(actions []domain.SyscallRequest, existing []domain.OpenLoop) []string {
	out := []string{}
	for _, action := range actions {
		if action.Action == domain.ActionOpenLoop {
			id := strings.TrimSpace(readPayloadString(action.Payload, "id"))
			if id != "" {
				out = append(out, id)
			}
		}
	}
	for _, loop := range existing {
		out = append(out, loop.ID)
	}
	return uniqueStrings(out)
}

func sharedTopicScore(a, b string) int {
	ta := tokenSet(a)
	tb := tokenSet(b)
	if len(ta) == 0 || len(tb) == 0 {
		return 0
	}
	score := 0
	for token := range ta {
		if _, ok := tb[token]; ok {
			score++
		}
	}
	return score
}

func tokenSet(s string) map[string]struct{} {
	re := regexp.MustCompile(`[^a-z0-9]+`)
	s = strings.ToLower(strings.TrimSpace(s))
	s = re.ReplaceAllString(s, " ")
	parts := strings.Fields(s)
	out := map[string]struct{}{}
	for _, part := range parts {
		if len(part) < 4 {
			continue
		}
		out[part] = struct{}{}
	}
	return out
}

func valuesEquivalent(current map[string]any, proposed any) bool {
	switch x := proposed.(type) {
	case map[string]any:
		if len(current) != len(x) {
			return false
		}
		for k, v := range current {
			if fmt.Sprintf("%v", x[k]) != fmt.Sprintf("%v", v) {
				return false
			}
		}
		return true
	default:
		if len(current) == 1 {
			if v, ok := current["value"]; ok {
				return fmt.Sprintf("%v", v) == fmt.Sprintf("%v", proposed)
			}
		}
		return false
	}
}

func compactIDs(ids ...string) []string {
	out := []string{}
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id != "" {
			out = append(out, id)
		}
	}
	return uniqueStrings(out)
}

func uniqueStrings(in []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(in))
	for _, v := range in {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}

func nonEmptyTrim(v, fallback string) string {
	if strings.TrimSpace(v) != "" {
		return strings.TrimSpace(v)
	}
	return strings.TrimSpace(fallback)
}

func hasOpenLoopCandidate(actions []domain.SyscallRequest) bool {
	for _, action := range actions {
		if action.Action == domain.ActionOpenLoop {
			return true
		}
	}
	return false
}

func extractResolvedTarget(text string) string {
	lower := strings.ToLower(strings.TrimSpace(text))
	for _, marker := range []string{" is resolved", " resolved", " fixed", " is fixed", " done"} {
		if idx := strings.Index(lower, marker); idx > 0 {
			return strings.TrimSpace(lower[:idx])
		}
	}
	return ""
}

func loopMatch(loop domain.OpenLoop, target string) bool {
	target = strings.ToLower(strings.TrimSpace(target))
	if target == "" {
		return false
	}
	loopText := strings.ToLower(loop.Title + " " + loop.Blocker + " " + loop.NextAction)
	return strings.Contains(loopText, target) || sharedTopicScore(loopText, target) >= 2
}

func syntheticNoteFromAction(action domain.SyscallRequest, scope domain.ForgeScope, prov domain.Provenance, now int64) domain.MemoryNote {
	id := strings.TrimSpace(readPayloadString(action.Payload, "id"))
	if id == "" {
		id = objectID("note", action.ID, "synthetic", payloadFingerprint(action.Payload))
	}
	noteType := domain.MemoryNoteType(strings.TrimSpace(readPayloadString(action.Payload, "type")))
	if noteType == "" {
		noteType = domain.NoteFact
	}
	conf := readPayloadFloat(action.Payload, "confidence", 0.6)
	return domain.MemoryNote{
		ID:         id,
		Type:       noteType,
		Title:      strings.TrimSpace(readPayloadString(action.Payload, "title")),
		Content:    strings.TrimSpace(readPayloadString(action.Payload, "content")),
		Scope:      scope,
		Confidence: conf,
		Status:     domain.NoteActive,
		CreatedAt:  now,
		UpdatedAt:  now,
		Provenance: prov,
	}
}

func shouldCompileContext(in CellRunContext) bool {
	if !in.FeatureFlags["enable_compile_context_action"] {
		return false
	}
	if in.Request.Metadata == nil {
		return false
	}
	v, ok := in.Request.Metadata["requestContextSnapshot"]
	if !ok || v == nil {
		return false
	}
	switch x := v.(type) {
	case bool:
		return x
	case string:
		return strings.EqualFold(strings.TrimSpace(x), "true")
	default:
		return false
	}
}

func duplicateActionCount(actions []domain.SyscallRequest) int {
	seen := map[string]struct{}{}
	dup := 0
	for _, action := range actions {
		key := string(action.Action) + "|" + payloadFingerprint(action.Payload)
		if _, ok := seen[key]; ok {
			dup++
			continue
		}
		seen[key] = struct{}{}
	}
	return dup
}

func stateAlreadyCurrent(ctx context.Context, in CellRunContext, key string, proposed any) bool {
	if in.Truth != nil {
		if current, ok, err := in.Truth.GetCurrentState(ctx, key, in.Scope); err == nil && ok {
			return valuesEquivalent(current.Value, proposed)
		}
	}
	for _, current := range in.CurrentState {
		if current.Key != key {
			continue
		}
		if valuesEquivalent(current.Value, proposed) {
			return true
		}
	}
	return false
}

func supersessionAlreadyKnown(ctx context.Context, in CellRunContext, oldID, newID string) bool {
	if in.Truth == nil {
		return false
	}
	chain, err := in.Truth.GetSupersessionChain(ctx, oldID, in.Scope, 20)
	if err != nil {
		return false
	}
	for _, record := range chain {
		if strings.TrimSpace(record.NewID) == strings.TrimSpace(newID) {
			return true
		}
	}
	return false
}

func contradictionAlreadyKnown(ctx context.Context, in CellRunContext, leftID, rightID string) bool {
	if in.Truth == nil {
		return false
	}
	rows, err := in.Truth.ListContradictionsForObject(ctx, leftID, in.Scope, 20)
	if err != nil {
		return false
	}
	for _, row := range rows {
		samePair := (row.LeftID == leftID && row.RightID == rightID) || (row.LeftID == rightID && row.RightID == leftID)
		if samePair {
			return true
		}
	}
	return false
}

func readStringSliceAny(m map[string]any, key string) []string {
	if m == nil {
		return nil
	}
	raw, ok := m[key]
	if !ok || raw == nil {
		return nil
	}
	switch v := raw.(type) {
	case []string:
		return uniqueStrings(v)
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			out = append(out, fmt.Sprintf("%v", item))
		}
		return uniqueStrings(out)
	default:
		return nil
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
