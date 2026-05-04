package contextcompiler

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

func NormalizeWhitespace(value string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
}

func SerializeBlock(block ContextBlock) string {
	lines := []string{
		fmt.Sprintf("[BLOCK: %s]", block.BlockType),
		"workspace_id: " + block.WorkspaceID,
	}
	addScalar(&lines, "case_id", block.CaseID)
	addScalar(&lines, "snapshot_id", block.SnapshotID)
	addScalar(&lines, "restore_seed_id", block.RestoreSeedID)
	addScalar(&lines, "layout_position", fmt.Sprintf("%d", block.LayoutPosition))
	addScalar(&lines, "cache_eligibility", string(block.CacheEligibility))
	addScalar(&lines, "invalidation_scope", block.InvalidationScope)
	addScalar(&lines, "policy_version", block.PolicyVersion)
	addScalar(&lines, "syscall_schema_version", block.SyscallSchemaVersion)
	addRefs(&lines, "source_object_refs", block.SourceObjectRefs)
	addRefs(&lines, "source_refs", block.SourceRefs)
	addRefs(&lines, "admitted_exhibit_refs", block.AdmittedExhibitRefs)
	addRefs(&lines, "rejected_exhibit_refs", block.RejectedExhibitRefs)
	addRefs(&lines, "ruling_refs", block.RulingRefs)
	addRefs(&lines, "contradiction_refs", block.ContradictionRefs)
	addRefs(&lines, "supersession_refs", block.SupersessionRefs)
	addRefs(&lines, "palace_route_refs", block.PalaceRouteRefs)
	addRefs(&lines, "semantic_operation_refs", block.SemanticOperationRefs)
	addRefs(&lines, "derived_object_refs", block.DerivedObjectRefs)
	if block.ContentSummary != "" {
		lines = append(lines, "summary:", NormalizeWhitespace(block.ContentSummary))
	}
	if len(block.Metadata) > 0 {
		lines = append(lines, "metadata: "+StableJSON(block.Metadata))
	}
	return strings.Join(lines, "\n")
}

func SerializeBundle(bundle ContextBundle) string {
	lines := []string{
		"[CONTEXT_BUNDLE]",
		"workspace_id: " + bundle.WorkspaceID,
		"layout_version: " + bundle.LayoutVersion,
	}
	addScalar(&lines, "case_id", bundle.CaseID)
	addScalar(&lines, "snapshot_id", bundle.SnapshotID)
	addScalar(&lines, "restore_seed_id", bundle.RestoreSeedID)
	addRefs(&lines, "source_refs", bundle.SourceRefs)
	lines = append(lines, "blocks:")
	for _, block := range bundle.Blocks {
		lines = append(lines, "- "+string(block.BlockType)+" "+block.ContentHash)
	}
	lines = append(lines, "canonical_prompt_text_hash: "+SHA256Text(bundle.CanonicalPromptText))
	return strings.Join(lines, "\n")
}

func AssembleCanonicalPromptText(blocks []ContextBlock, layout PromptLayout) string {
	ordered := SortBlocksForLayout(blocks, layout)
	parts := make([]string, 0, len(ordered))
	for _, block := range ordered {
		parts = append(parts, block.CanonicalText)
	}
	return strings.Join(parts, "\n---\n")
}

func StableJSON(value any) string {
	encoded, _ := json.Marshal(value)
	return string(encoded)
}

func addScalar(lines *[]string, key string, value string) {
	value = NormalizeWhitespace(value)
	if value != "" {
		*lines = append(*lines, key+": "+value)
	}
}

func addRefs(lines *[]string, key string, values []string) {
	refs := NormalizeRefs(values)
	if len(refs) == 0 {
		return
	}
	*lines = append(*lines, key+":")
	for _, ref := range refs {
		*lines = append(*lines, "- "+ref)
	}
}

func sortedStringKeys(values map[string]int) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
