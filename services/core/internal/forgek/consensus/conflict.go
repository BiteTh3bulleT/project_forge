package consensus

import "sort"

type Conflict struct {
	ConflictKey string   `json:"conflict_key"`
	ClaimIDs    []string `json:"claim_ids"`
	Reason      string   `json:"reason"`
}

func DetectConflicts(claims []Claim) []Conflict {
	groups := make(map[string][]Claim)
	for _, claim := range claims {
		if claim.ClaimType == ClaimTypeUncertainty {
			continue
		}
		groups[ConflictKey(claim)] = append(groups[ConflictKey(claim)], claim)
	}
	conflicts := make([]Conflict, 0)
	for key, group := range groups {
		values := make(map[string][]string)
		for _, claim := range group {
			valueKey := CanonicalValue(claim.ValueJSON)
			values[valueKey] = append(values[valueKey], claim.ClaimID)
		}
		if len(values) < 2 {
			continue
		}
		claimIDs := make([]string, 0)
		for _, ids := range values {
			claimIDs = append(claimIDs, ids...)
		}
		claimIDs = NormalizeRefs(claimIDs)
		conflicts = append(conflicts, Conflict{ConflictKey: key, ClaimIDs: claimIDs, Reason: "same subject, predicate, scope, and temporal bucket with different values"})
	}
	sort.Slice(conflicts, func(i, j int) bool { return conflicts[i].ConflictKey < conflicts[j].ConflictKey })
	return conflicts
}

func ClaimsConflict(left Claim, right Claim) bool {
	if left.ClaimType == ClaimTypeUncertainty || right.ClaimType == ClaimTypeUncertainty {
		return false
	}
	return ConflictKey(left) == ConflictKey(right) && CanonicalValue(left.ValueJSON) != CanonicalValue(right.ValueJSON)
}
