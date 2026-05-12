import type { AuditTraceLookupReport } from "../../lib/api";

export function asRecord(value: unknown): Record<string, unknown> | null {
  if (value && typeof value === "object" && !Array.isArray(value))
    return value as Record<string, unknown>;
  return null;
}

export function asArray(value: unknown): unknown[] {
  return Array.isArray(value) ? value : [];
}

export function asString(value: unknown): string {
  return typeof value === "string" ? value : "";
}

export function asNumber(value: unknown): number {
  if (typeof value === "number") return value;
  if (typeof value === "string") {
    const parsed = Number.parseFloat(value);
    return Number.isFinite(parsed) ? parsed : 0;
  }
  return 0;
}

export function asStringArray(value: unknown): string[] {
  if (!Array.isArray(value)) return [];
  return value
    .filter(
      (item): item is string =>
        typeof item === "string" && item.trim().length > 0,
    )
    .map((item) => item.trim());
}

export type RestoreScoreCandidate = {
  snapshotId: string;
  createdAt: number;
  snapshotKind: string;
  queryScore: number;
  scopeScore: number;
  kindScore: number;
  lineageScore: number;
  stateOverlapScore: number;
  loopOverlapScore: number;
  artifactOverlapScore: number;
  nodeOverlapScore: number;
  edgeOverlapScore: number;
  recencyScore: number;
  fingerprintBonus: number;
  preferredHintBonus: number;
  contradictionPenalty: number;
  stalenessPenalty: number;
  headerOnlyPenalty: number;
  total: number;
  explain: string[];
  selected?: boolean;
};

export type RestoreScoreSummary = {
  hasStructured: boolean;
  decision: string;
  threshold: number;
  candidateCount: number;
  topScore: number;
  selectedIndex: number;
  selectedSnapshotId: string;
  topCandidateId: string;
  candidates: RestoreScoreCandidate[];
};

export type ResumeHintSummary = {
  hasStructured: boolean;
  nextAction: string;
  topBlockers: string[];
  dominantStateKeys: string[];
  dominantLoopIds: string[];
  recommendedEvidenceIds: string[];
  restoreConfidence: number;
  requiresFreshCompile: boolean;
  preferredSnapshotId: string;
  candidateCount: number;
};

export function parseRestoreScoreCandidate(
  value: unknown,
): RestoreScoreCandidate | null {
  const raw = asRecord(value);
  if (!raw) return null;
  const candidate: RestoreScoreCandidate = {
    snapshotId: asString(raw.snapshot_id || raw.snapshotId),
    createdAt: asNumber(raw.created_at ?? raw.createdAt),
    snapshotKind: asString(raw.snapshot_kind || raw.snapshotKind),
    queryScore: asNumber(raw.query_score ?? raw.queryScore),
    scopeScore: asNumber(raw.scope_score ?? raw.scopeScore),
    kindScore: asNumber(raw.kind_score ?? raw.kindScore),
    lineageScore: asNumber(raw.lineage_score ?? raw.lineageScore),
    stateOverlapScore: asNumber(
      raw.state_overlap_score ?? raw.stateOverlapScore,
    ),
    loopOverlapScore: asNumber(raw.loop_overlap_score ?? raw.loopOverlapScore),
    artifactOverlapScore: asNumber(
      raw.artifact_overlap_score ?? raw.artifactOverlapScore,
    ),
    nodeOverlapScore: asNumber(raw.node_overlap_score ?? raw.nodeOverlapScore),
    edgeOverlapScore: asNumber(raw.edge_overlap_score ?? raw.edgeOverlapScore),
    recencyScore: asNumber(raw.recency_score ?? raw.recencyScore),
    fingerprintBonus: asNumber(raw.fingerprint_bonus ?? raw.fingerprintBonus),
    preferredHintBonus: asNumber(
      raw.preferred_hint_bonus ?? raw.preferredHintBonus,
    ),
    contradictionPenalty: asNumber(
      raw.contradiction_penalty ?? raw.contradictionPenalty,
    ),
    stalenessPenalty: asNumber(raw.staleness_penalty ?? raw.stalenessPenalty),
    headerOnlyPenalty: asNumber(
      raw.header_only_penalty ?? raw.headerOnlyPenalty,
    ),
    total: asNumber(raw.total ?? raw.totalScore),
    explain: asStringArray(raw.explain),
    selected: Boolean(raw.selected),
  };
  if (!candidate.snapshotId) return null;
  return candidate;
}

export function parseRestoreScoreSummary(raw: unknown): RestoreScoreSummary {
  const record = asRecord(raw);
  if (!record) {
    return {
      hasStructured: false,
      decision: "",
      threshold: 0,
      candidateCount: 0,
      topScore: 0,
      selectedIndex: -1,
      selectedSnapshotId: "",
      topCandidateId: "",
      candidates: [],
    };
  }
  const scoreRows =
    asArray(record.scores).length > 0
      ? asArray(record.scores)
      : asArray(record.score_breakdown ?? record.scoreBreakdown);
  const candidates = scoreRows
    .map(parseRestoreScoreCandidate)
    .filter((item): item is RestoreScoreCandidate => item != null);
  return {
    hasStructured: candidates.length > 0,
    decision: asString(record.decision),
    threshold: asNumber(record.threshold),
    candidateCount:
      candidates.length ||
      asNumber(record.candidate_count ?? record.candidateCount),
    topScore: asNumber(record.top_score ?? record.topScore),
    selectedIndex: Number(record.selected_index ?? record.selectedIndex ?? -1),
    selectedSnapshotId: asString(
      record.selected_snapshot_id ?? record.selectedSnapshotId,
    ),
    topCandidateId: asString(record.top_candidate_id ?? record.topCandidateId),
    candidates,
  };
}

export function parseResumeHintSummary(raw: unknown): ResumeHintSummary {
  const record = asRecord(raw);
  if (!record) {
    return {
      hasStructured: false,
      nextAction: "",
      topBlockers: [],
      dominantStateKeys: [],
      dominantLoopIds: [],
      recommendedEvidenceIds: [],
      restoreConfidence: 0,
      requiresFreshCompile: false,
      preferredSnapshotId: "",
      candidateCount: 0,
    };
  }
  const candidateCount = asNumber(
    record.candidate_count ?? record.candidateCount,
  );
  return {
    hasStructured:
      candidateCount > 0 ||
      asString(record.next_action || record.nextAction).length > 0,
    nextAction: asString(
      record.next_action || record.nextAction || "fresh_compile",
    ),
    topBlockers: asStringArray(record.top_blockers || record.topBlockers),
    dominantStateKeys: asStringArray(
      record.dominant_state_keys || record.dominantStateKeys,
    ),
    dominantLoopIds: asStringArray(
      record.dominant_loop_ids || record.dominantLoopIds,
    ),
    recommendedEvidenceIds: asStringArray(
      record.recommended_evidence_ids || record.recommendedEvidenceIds,
    ),
    restoreConfidence: asNumber(
      record.restore_confidence ?? record.restoreConfidence,
    ),
    requiresFreshCompile: Boolean(
      record.requires_fresh_compile || record.requiresFreshCompile,
    ),
    preferredSnapshotId: asString(
      record.preferred_snapshot_id || record.preferredSnapshotId,
    ),
    candidateCount: Math.max(candidateCount, 0),
  };
}

export function parseInspectorReportSummary(
  report: AuditTraceLookupReport | null,
) {
  const raw = asRecord(report?.report);
  return {
    gatewayInvocations: asArray(raw?.gatewayInvocations),
    auditRecords: asArray(raw?.auditRecords),
    artifactRecords: asArray(raw?.artifactRecords),
    provenanceRecords: asArray(raw?.provenanceRecords),
    journalEvents: asArray(raw?.journalEvents),
    artifactRefs: asArray(raw?.artifactRefs),
    links: asRecord(raw?.links) ?? {},
  };
}
