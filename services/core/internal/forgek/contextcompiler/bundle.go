package contextcompiler

import "time"

type ContextBundle struct {
	BundleID                string         `json:"bundle_id"`
	WorkspaceID             string         `json:"workspace_id"`
	CaseID                  string         `json:"case_id,omitempty"`
	SnapshotID              string         `json:"snapshot_id,omitempty"`
	RestoreSeedID           string         `json:"restore_seed_id,omitempty"`
	LayoutID                string         `json:"layout_id"`
	LayoutVersion           string         `json:"layout_version"`
	Blocks                  []ContextBlock `json:"blocks"`
	CanonicalPromptText     string         `json:"canonical_prompt_text"`
	BundleHash              string         `json:"bundle_hash"`
	TokenInputHash          string         `json:"token_input_hash"`
	EstimatedTokenCount     int            `json:"estimated_token_count"`
	StablePrefixHash        string         `json:"stable_prefix_hash,omitempty"`
	VolatileSuffixHash      string         `json:"volatile_suffix_hash,omitempty"`
	CacheEligibilitySummary map[string]int `json:"cache_eligibility_summary"`
	SourceRefs              []string       `json:"source_refs,omitempty"`
	CreatedBy               string         `json:"created_by"`
	CreatedAt               time.Time      `json:"created_at"`
	JournalRefs             []string       `json:"journal_refs,omitempty"`
	Metadata                map[string]any `json:"metadata,omitempty"`
}

func FinalizeBundle(bundle ContextBundle) ContextBundle {
	bundle.SourceRefs = NormalizeRefs(bundle.SourceRefs)
	bundle.JournalRefs = NormalizeRefs(bundle.JournalRefs)
	bundle.Blocks = cloneBlocks(bundle.Blocks)
	parts := make([]string, 0, len(bundle.Blocks))
	stableParts := make([]string, 0)
	volatileParts := make([]string, 0)
	summary := make(map[string]int)
	totalTokens := 0
	for i := range bundle.Blocks {
		bundle.Blocks[i].LayoutPosition = i + 1
		bundle.Blocks[i] = FinalizeBlock(bundle.Blocks[i])
		block := bundle.Blocks[i]
		parts = append(parts, block.CanonicalText)
		totalTokens += block.TokenCountEstimate
		summary[string(block.CacheEligibility)]++
		if IsStableCacheEligibility(block.CacheEligibility) {
			stableParts = append(stableParts, block.CanonicalText)
		} else {
			volatileParts = append(volatileParts, block.CanonicalText)
		}
	}
	bundle.CanonicalPromptText = joinSections(parts)
	bundle.EstimatedTokenCount = totalTokens
	bundle.TokenInputHash = TokenInputHash(bundle.CanonicalPromptText)
	if len(stableParts) > 0 {
		bundle.StablePrefixHash = SHA256Text(joinSections(stableParts))
	}
	if len(volatileParts) > 0 {
		bundle.VolatileSuffixHash = SHA256Text(joinSections(volatileParts))
	}
	bundle.CacheEligibilitySummary = summary
	bundle.BundleHash = SHA256Text(SerializeBundle(bundle))
	return bundle
}

func (b ContextBundle) Clone() ContextBundle {
	b.Blocks = cloneBlocks(b.Blocks)
	b.SourceRefs = CloneStrings(b.SourceRefs)
	b.JournalRefs = CloneStrings(b.JournalRefs)
	b.Metadata = CloneMap(b.Metadata)
	if b.CacheEligibilitySummary != nil {
		summary := make(map[string]int, len(b.CacheEligibilitySummary))
		for key, value := range b.CacheEligibilitySummary {
			summary[key] = value
		}
		b.CacheEligibilitySummary = summary
	}
	return b
}

func (b ContextBundle) IsCanonicalTruth() bool { return false }
func (b ContextBundle) IsModelResponse() bool  { return false }
func (b ContextBundle) IsKVCache() bool        { return false }

func joinSections(parts []string) string {
	if len(parts) == 0 {
		return ""
	}
	out := parts[0]
	for i := 1; i < len(parts); i++ {
		out += "\n---\n" + parts[i]
	}
	return out
}
