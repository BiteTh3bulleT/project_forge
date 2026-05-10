package refvalidation

const (
	GateCandidateRefs = "candidate_refs_valid"
	GateObservedRefs  = "observed_refs_valid"
)

type ComparisonRequest struct {
	ResultID      string      `json:"result_id"`
	WorkspaceID   string      `json:"workspace_id"`
	CandidateRefs []ObjectRef `json:"candidate_refs"`
	ObservedRefs  []ObjectRef `json:"observed_refs"`
}

type ComparisonResult struct {
	ResultID               string              `json:"result_id"`
	WorkspaceID            string              `json:"workspace_id"`
	Passed                 bool                `json:"passed"`
	Match                  bool                `json:"match"`
	CandidateRefs          []ObjectRef         `json:"candidate_refs"`
	ObservedRefs           []ObjectRef         `json:"observed_refs"`
	AddedRefs              []ObjectRef         `json:"added_refs"`
	RemovedRefs            []ObjectRef         `json:"removed_refs"`
	UnchangedRefs          []ObjectRef         `json:"unchanged_refs"`
	Failures               []ValidationFailure `json:"failures,omitempty"`
	Warnings               []string            `json:"warnings,omitempty"`
	MemoryMutation         bool                `json:"memory_mutation"`
	RuntimeMutation        bool                `json:"runtime_mutation"`
	LiveAuthorityMigration bool                `json:"live_authority_migration"`
}

func CompareRefShapes(req ComparisonRequest) ComparisonResult {
	candidate := ValidateRefs(ValidationRequest{
		ResultID:    req.ResultID + ":candidate_refs",
		WorkspaceID: req.WorkspaceID,
		Refs:        req.CandidateRefs,
	})
	observed := ValidateRefs(ValidationRequest{
		ResultID:    req.ResultID + ":observed_refs",
		WorkspaceID: req.WorkspaceID,
		Refs:        req.ObservedRefs,
	})
	failures := make([]ValidationFailure, 0, len(candidate.Failures)+len(observed.Failures))
	for _, failure := range candidate.Failures {
		failures = append(failures, ValidationFailure{Gate: GateCandidateRefs, Field: "candidate_refs." + failure.Field, Message: failure.Message})
	}
	for _, failure := range observed.Failures {
		failures = append(failures, ValidationFailure{Gate: GateObservedRefs, Field: "observed_refs." + failure.Field, Message: failure.Message})
	}
	passed := candidate.Passed && observed.Passed
	res := ComparisonResult{
		ResultID:               candidate.ResultID,
		WorkspaceID:            candidate.WorkspaceID,
		Passed:                 passed,
		CandidateRefs:          candidate.NormalizedRefs,
		ObservedRefs:           observed.NormalizedRefs,
		Failures:               failures,
		MemoryMutation:         false,
		RuntimeMutation:        false,
		LiveAuthorityMigration: false,
	}
	if !passed {
		return res
	}

	candidateByKey := refsByKey(candidate.NormalizedRefs)
	observedByKey := refsByKey(observed.NormalizedRefs)
	for _, ref := range observed.NormalizedRefs {
		if _, ok := candidateByKey[refKey(ref)]; ok {
			res.UnchangedRefs = append(res.UnchangedRefs, ref)
			continue
		}
		res.AddedRefs = append(res.AddedRefs, ref)
	}
	for _, ref := range candidate.NormalizedRefs {
		if _, ok := observedByKey[refKey(ref)]; !ok {
			res.RemovedRefs = append(res.RemovedRefs, ref)
		}
	}
	res.Match = len(res.AddedRefs) == 0 && len(res.RemovedRefs) == 0
	return res
}

func refsByKey(refs []ObjectRef) map[string]ObjectRef {
	out := make(map[string]ObjectRef, len(refs))
	for _, ref := range refs {
		out[refKey(ref)] = ref
	}
	return out
}

func refKey(ref ObjectRef) string {
	return ref.WorkspaceID + "\x00" + ref.RefType + "\x00" + ref.RefID
}
