package palace

import (
	"sort"
	"strings"
)

func ScoreAnchor(query RouteQuery, anchor MemoryAnchor, roomID string, stats RoomRouteStats) float64 {
	if query.WorkspaceID == "" || anchor.WorkspaceID != query.WorkspaceID {
		return 0
	}

	score := 0.0
	queryTerms := terms(query.QueryText + " " + query.RouteReason)
	labelTerms := terms(anchor.Label)
	for _, keyword := range anchor.Keywords {
		normalized := normalize(keyword)
		if normalized == "" {
			continue
		}
		if containsTerm(queryTerms, normalized) {
			score += 2
		}
	}
	for _, labelTerm := range labelTerms {
		if containsTerm(queryTerms, labelTerm) {
			score += 1
		}
	}
	for _, tag := range anchor.Tags {
		if containsNormalized(query.Tags, tag) {
			score += 1.5
		}
	}
	if roomID != "" && anchor.RoomID == roomID {
		score += 0.5
	}
	if stats.SuccessCount > 0 {
		score += float64(stats.SuccessCount) * 0.25
	}
	return score
}

func SortCandidates(candidates []CandidateObject) {
	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].RelevanceScore == candidates[j].RelevanceScore {
			if candidates[i].AnchorID == candidates[j].AnchorID {
				return candidates[i].SourceObjectID < candidates[j].SourceObjectID
			}
			return candidates[i].AnchorID < candidates[j].AnchorID
		}
		return candidates[i].RelevanceScore > candidates[j].RelevanceScore
	})
}

func terms(text string) []string {
	fields := strings.FieldsFunc(strings.ToLower(text), func(r rune) bool {
		return !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '_' || r == '-')
	})
	out := make([]string, 0, len(fields))
	for _, field := range fields {
		if field != "" {
			out = append(out, field)
		}
	}
	return out
}

func containsTerm(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func containsNormalized(values []string, want string) bool {
	normalized := normalize(want)
	for _, value := range values {
		if normalize(value) == normalized {
			return true
		}
	}
	return false
}

func normalize(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}
