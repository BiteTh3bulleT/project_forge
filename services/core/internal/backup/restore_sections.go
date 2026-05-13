package backup

import (
	"sort"
	"strings"
)

func knownSections(doc BundleDoc) []string {
	out := make([]string, 0, len(doc.Entities))
	for k := range doc.Entities {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func normalizeSections(sections []string) []string {
	if len(sections) == 0 {
		return nil
	}
	out := make([]string, 0, len(sections))
	seen := make(map[string]struct{}, len(sections))
	for _, sec := range sections {
		normalized := strings.ToLower(strings.TrimSpace(sec))
		if normalized == "" {
			continue
		}
		if _, exists := seen[normalized]; exists {
			continue
		}
		seen[normalized] = struct{}{}
		out = append(out, normalized)
	}
	return out
}

func orderSectionsForRestore(sections []string) []string {
	if len(sections) <= 1 {
		return sections
	}
	type orderedSection struct {
		name     string
		priority int
		index    int
	}
	ordered := make([]orderedSection, 0, len(sections))
	for i, sec := range sections {
		priority, ok := restoreSectionPriority[sec]
		if !ok {
			priority = 1000
		}
		ordered = append(ordered, orderedSection{
			name:     sec,
			priority: priority,
			index:    i,
		})
	}
	sort.SliceStable(ordered, func(i, j int) bool {
		if ordered[i].priority == ordered[j].priority {
			return ordered[i].index < ordered[j].index
		}
		return ordered[i].priority < ordered[j].priority
	})
	out := make([]string, 0, len(ordered))
	for _, item := range ordered {
		out = append(out, item.name)
	}
	return out
}

var restoreSectionPriority = map[string]int{
	"sources":                       1,
	"files":                         2,
	"chunks":                        3,
	"embedding_records":             4,
	"dossiers":                      8,
	"task_packets":                  10,
	"project_context_records":       11,
	"permission_profiles":           15,
	"approval_presets":              16,
	"dossier_profiles":              17,
	"execution_strategies":          18,
	"automation_rules":              19,
	"action_lanes":                  20,
	"jobs":                          21,
	"events":                        25,
	"dossier_sources":               26,
	"dossier_jobs":                  27,
	"dossier_packets":               28,
	"dossier_briefs":                29,
	"approval_requests":             30,
	"approval_decisions":            31,
	"job_status_history":            32,
	"job_events":                    33,
	"artifacts":                     34,
	"evaluation_records":            36,
	"provenance_records":            40,
	"gateway_invocations":           41,
	"audit_records":                 42,
	"retrieval_runs":                43,
	"retrieval_results":             44,
	"retrieval_result_selection":    45,
	"packet_retrieval_runs":         46,
	"context_evidence":              47,
	"journal_events":                50,
	"state_items":                   55,
	"state_versions":                56,
	"memory_observations":           57,
	"memory_observation_links":      58,
	"retrieval_result_observations": 59,
	"memory_notes":                  60,
	"semantic_links":                61,
	"open_loops":                    62,
	"artifact_refs":                 63,
	"derived_models":                64,
	"contradiction_records":         65,
	"supersession_records":          66,
	"context_packet_snapshots":      67,
	"dream_reports":                 68,
	"restore_outcome_events":        69,
	"semantic_idempotency_keys":     70,
	"autonomy_settings":             71,
	"memory_usefulness_events":      72,
	"packet_alignment_notes":        73,
	"memory_repair_runs":            74,
	"memory_repair_items":           75,
	"model_manifests":               80,
	"model_registry_status":         81,
	"model_runtime_loads":           82,
	"chat_threads":                  90,
	"chat_messages":                 91,
	"canvas_boards":                 92,
	"canvas_notes":                  93,
	"tool_capability_overrides":     94,
	"feature_flags":                 95,
	"alert_rules":                   96,
	"scheduled_tasks":               97,
}

var restoreExportOnlyReasons = map[string]string{
	"memory_vsa_pointers":          "restore export-only by policy: VSA pointers are derived from observation lineage and fingerprint state",
	"memory_vsa_role_bindings":     "restore export-only by policy: VSA role bindings are derived from observation lineage and role reconciliation",
	"memory_vsa_associations":      "restore export-only by policy: VSA associations are derived graph edges and must be recomputed",
	"retrieval_result_vsa_signals": "restore export-only by policy: VSA signals are derived from retrieval runs/results and must be recomputed",
	"memory_vsa_reindex_runs":      "restore export-only by policy: reindex runs are operational maintenance history tied to live memory state",
	"memory_vsa_reindex_items":     "restore export-only by policy: reindex items are operational maintenance history tied to live memory state",
}

func restoreExportOnlyReason(section string) (string, bool) {
	reason, ok := restoreExportOnlyReasons[section]
	return reason, ok
}

func restoreUnsupportedReason(section string) string {
	if reason, ok := restoreExportOnlyReason(section); ok {
		return reason
	}
	return "restore mapping not implemented"
}

func restoreSectionWarning(section string) string {
	switch section {
	case "artifacts":
		return "restore limitation: artifact rows are restored, but artifact file bytes are not imported or rollback-managed"
	default:
		return ""
	}
}

func restoreSectionNonDBSideEffect(section string) string {
	return restoreSectionWarning(section)
}

func appendUniqueString(items []string, value string) []string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return items
	}
	for _, item := range items {
		if item == trimmed {
			return items
		}
	}
	return append(items, trimmed)
}
