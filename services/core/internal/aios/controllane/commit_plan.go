package controllane

import (
	"fmt"
	"strings"

	"forge/projectforge/services/core/internal/aios/domain"
	"forge/projectforge/services/core/internal/forgekernel/commitproof"
)

// buildPreparedCommitPlan declares every durable id the existing apply
// adapter may produce before FORGE-K seals the request. Mutating syscalls keep
// the journal id in the existing SyscallResult object-id contract; validation-
// only syscalls still append a required journal event, but expose it through
// the typed receipt rather than pretending it is a semantic mutation.
func buildPreparedCommitPlan(req domain.SyscallRequest, def ActionDefinition, read SemanticReadStore) (commitproof.PreparedPlan, error) {
	objectIDs, err := expectedCommitObjectIDs(req, read)
	if err != nil {
		return commitproof.PreparedPlan{}, err
	}
	journalID := req.ID + ":journal_event"
	mutating := len(objectIDs) > 0
	if mutating {
		objectIDs = append(objectIDs, journalID)
	}
	details := map[string]any{
		"semanticApplyObjectCount": len(objectIDs) - boolInt(mutating),
		"journalEventId":           journalID,
	}
	if req.Action == domain.ActionRebuildMemoryAcceleration {
		details["workspaceId"] = req.Scope.WorkspaceID
		details["laneId"] = req.Scope.LaneID
		details["expectedManifestHash"] = readString(req.Payload, "expectedManifestHash")
		details["expectedPriorManifestHash"] = readString(req.Payload, "expectedPriorManifestHash")
		details["algorithmName"] = readString(req.Payload, "algorithmName")
		details["algorithmVersion"] = readString(req.Payload, "algorithmVersion")
		details["dimensions"] = readInt(req.Payload, "dimensions", 0)
		details["seed"] = readInt(req.Payload, "seed", 0)
		details["requestedAtMs"] = readInt64(req.Payload, "requestedAtMs")
	}
	return commitproof.PreparedPlan{
		Action:                req.Action,
		Capability:            def.Capability,
		TargetObjectType:      def.TargetObjectType,
		Mutating:              mutating,
		JournalEventType:      "semantic_syscall." + strings.ToLower(string(req.Action)),
		ExpectedObjectIDs:     objectIDs,
		ExpectedProvenanceIDs: []string{provenanceID(req.Scope, req.Provenance)},
		Details:               details,
	}, nil
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func expectedCommitObjectIDs(req domain.SyscallRequest, read SemanticReadStore) ([]string, error) {
	switch req.Action {
	case domain.ActionCreateNote:
		return []string{nonEmpty(readString(req.Payload, "id"), req.ID+":note")}, nil
	case domain.ActionCreateLink:
		return []string{nonEmpty(readString(req.Payload, "id"), req.ID+":link")}, nil
	case domain.ActionUpdateState:
		if read != nil {
			if current, ok := read.FindStateByScopeKey(req.Scope, readString(req.Payload, "key")); ok {
				return []string{current.ID}, nil
			}
		}
		return []string{nonEmpty(readString(req.Payload, "id"), req.ID+":state")}, nil
	case domain.ActionOpenLoop:
		return []string{nonEmpty(readString(req.Payload, "id"), req.ID+":loop")}, nil
	case domain.ActionCloseLoop:
		return []string{readString(req.Payload, "loopId")}, nil
	case domain.ActionMarkSuperseded:
		return []string{req.ID + ":supersession", req.ID + ":supersedes_link"}, nil
	case domain.ActionRegisterContradict:
		return []string{req.ID + ":contradiction", req.ID + ":contradiction_link"}, nil
	case domain.ActionDeriveModel:
		return []string{nonEmpty(readString(req.Payload, "id"), req.ID+":model")}, nil
	case domain.ActionArchiveNote:
		return []string{readString(req.Payload, "noteId")}, nil
	case domain.ActionCompileContext:
		return expectedCompileContextIDs(req), nil
	case domain.ActionAdmitEvidence:
		return []string{
			nonEmpty(readString(req.Payload, "exhibitId"), req.ID+":exhibit"),
			nonEmpty(readString(req.Payload, "rulingId"), req.ID+":ruling"),
		}, nil
	case domain.ActionAppealRuling:
		return []string{
			nonEmpty(readString(req.Payload, "appealId"), req.ID+":appeal"),
			nonEmpty(readString(req.Payload, "rulingId"), req.ID+":ruling"),
			readString(req.Payload, "exhibitId"),
		}, nil
	case domain.ActionRecordRetrievalEvidence:
		evidence, decodeErr := decodeRetrievalEvidence(req.Payload)
		if decodeErr != nil {
			return nil, decodeErr
		}
		ids := make([]string, 0, 1+len(evidence.Results))
		ids = append(ids, evidence.EvidenceID)
		for _, result := range evidence.Results {
			ids = append(ids, result.EvidenceID)
		}
		return ids, nil
	case domain.ActionMaterializeAdmittedEvidence:
		return []string{req.ID + ":memory_evidence"}, nil
	case domain.ActionReviseMemoryEvidence:
		return []string{req.ID + ":memory_evidence", req.ID + ":memory_supersession"}, nil
	case domain.ActionComputeSemanticDiff:
		return semanticDiffObjectIDs(req.ID), nil
	case domain.ActionRebuildMemoryAcceleration:
		return []string{readString(req.Payload, "expectedManifestHash")}, nil
	case domain.ActionRecordRetrievalUsefulness:
		return []string{"retrieval-usefulness:" + strings.TrimSpace(req.ID)}, nil
	case domain.ActionRecordRestoreOutcomeFeedback:
		return []string{"restore-outcome-feedback:" + strings.TrimSpace(req.ID)}, nil
	case domain.ActionValidateKVIdentity,
		domain.ActionValidateRefShape,
		domain.ActionCompareRefShape,
		domain.ActionValidateSourceObject,
		domain.ActionValidateSemanticOperation,
		domain.ActionValidateAdmissionCandidate,
		domain.ActionValidateContextAttribution:
		return []string{}, nil
	default:
		return nil, fmt.Errorf("unsupported action %q has no FORGE-K commit plan", req.Action)
	}
}

func expectedCompileContextIDs(req domain.SyscallRequest) []string {
	opts := mergeCompileContextOptions(req.Payload)
	if !opts.PersistSnapshot {
		return []string{}
	}
	query := readString(req.Payload, "query")
	if strings.TrimSpace(query) == "" {
		if value, ok := req.Metadata["query"]; ok {
			query = strings.TrimSpace(fmt.Sprintf("%v", value))
		}
	}
	packetID := "ctx-" + strings.ReplaceAll(query, " ", "_") + "-" + fmt.Sprintf("%d", req.RequestedAt)
	ids := []string{packetID}
	if opts.RenderSnapshotCard {
		ids = append(ids, packetID+":snapshot_card")
	}
	ids = append(ids, NewRestoreOutcomeID(req.ID, packetID, "compiled"))
	return ids
}
