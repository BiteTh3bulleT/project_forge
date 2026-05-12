import { j } from "./client";
import type {
  ContextSnapshotInspectorDetail,
  ContextSnapshotInspectorSummary,
  DreamReportCandidatesResponse,
  DreamReportDetail,
  DreamReportProposalsResponse,
  DreamReportSummary,
  DreamReportWarningsResponse,
  RestoreInspectorCandidatesResponse,
  RestoreInspectorResumeHintsResponse,
  RestoreInspectorScoreResponse,
} from "./types";

export const contextInspectorApi = {
  listSnapshots: (params?: {
    limit?: number;
    workspaceId?: string;
    laneId?: string;
    correlationId?: string;
    snapshotKind?: string;
    query?: string;
  }) => {
    const qs = new URLSearchParams();
    if (params?.limit != null) qs.set("limit", String(params.limit));
    if (params?.workspaceId) qs.set("workspaceId", params.workspaceId);
    if (params?.laneId) qs.set("laneId", params.laneId);
    if (params?.correlationId) qs.set("correlationId", params.correlationId);
    if (params?.snapshotKind) qs.set("snapshotKind", params.snapshotKind);
    if (params?.query) qs.set("query", params.query);
    const q = qs.toString();
    return j<{ snapshots: ContextSnapshotInspectorSummary[] }>(
      `/api/context-inspector/snapshots${q ? `?${q}` : ""}`,
    );
  },
  getSnapshot: (
    id: string,
    params?: { workspaceId?: string; laneId?: string },
  ) => {
    const qs = new URLSearchParams();
    if (params?.workspaceId) qs.set("workspaceId", params.workspaceId);
    if (params?.laneId) qs.set("laneId", params.laneId);
    const q = qs.toString();
    return j<{ snapshot: ContextSnapshotInspectorDetail }>(
      `/api/context-inspector/snapshots/${encodeURIComponent(id)}${q ? `?${q}` : ""}`,
    );
  },
  restoreRecent: (params: {
    workspaceId: string;
    laneId?: string;
    limit?: number;
    snapshotKind?: string;
  }) => {
    const qs = new URLSearchParams();
    qs.set("workspaceId", params.workspaceId);
    if (params.laneId) qs.set("laneId", params.laneId);
    if (params.limit != null) qs.set("limit", String(params.limit));
    if (params.snapshotKind) qs.set("snapshotKind", params.snapshotKind);
    const q = qs.toString();
    return j<{ snapshots: ContextSnapshotInspectorSummary[] }>(
      `/api/context/restore/recent?${q}`,
    );
  },
  restoreGet: (
    id: string,
    params: { workspaceId: string; laneId?: string },
  ) => {
    const qs = new URLSearchParams();
    qs.set("workspaceId", params.workspaceId);
    if (params.laneId) qs.set("laneId", params.laneId);
    return j<{
      snapshot: ContextSnapshotInspectorDetail;
      evidenceClass: string;
      nonCanonicalEvidence: boolean;
      canonicalWriteCommitted: boolean;
    }>(`/api/context/restore/${encodeURIComponent(id)}?${qs.toString()}`);
  },
  restoreCandidates: (
    id: string,
    params: { workspaceId: string; laneId?: string },
  ) => {
    const qs = new URLSearchParams();
    qs.set("workspaceId", params.workspaceId);
    if (params.laneId) qs.set("laneId", params.laneId);
    return j<RestoreInspectorCandidatesResponse>(
      `/api/context/restore/${encodeURIComponent(id)}/candidates?${qs.toString()}`,
    );
  },
  restoreScore: (
    id: string,
    params: { workspaceId: string; laneId?: string },
  ) => {
    const qs = new URLSearchParams();
    qs.set("workspaceId", params.workspaceId);
    if (params.laneId) qs.set("laneId", params.laneId);
    return j<RestoreInspectorScoreResponse>(
      `/api/context/restore/${encodeURIComponent(id)}/score?${qs.toString()}`,
    );
  },
  restoreResumeHints: (
    id: string,
    params: { workspaceId: string; laneId?: string },
  ) => {
    const qs = new URLSearchParams();
    qs.set("workspaceId", params.workspaceId);
    if (params.laneId) qs.set("laneId", params.laneId);
    return j<RestoreInspectorResumeHintsResponse>(
      `/api/context/restore/${encodeURIComponent(id)}/resume-hints?${qs.toString()}`,
    );
  },
};

export const dreamReportsApi = {
  list: (params: {
    workspaceId: string;
    laneId?: string;
    mode?: string;
    limit?: number;
  }) => {
    const qs = new URLSearchParams();
    qs.set("workspaceId", params.workspaceId);
    if (params.laneId) qs.set("laneId", params.laneId);
    if (params.mode) qs.set("mode", params.mode);
    if (params.limit != null) qs.set("limit", String(params.limit));
    return j<{ reports: DreamReportSummary[] }>(
      `/api/dream/reports?${qs.toString()}`,
    );
  },
  get: (id: string, params: { workspaceId: string; laneId?: string }) => {
    const qs = new URLSearchParams();
    qs.set("workspaceId", params.workspaceId);
    if (params.laneId) qs.set("laneId", params.laneId);
    return j<DreamReportDetail>(
      `/api/dream/reports/${encodeURIComponent(id)}?${qs.toString()}`,
    );
  },
  candidates: (
    id: string,
    params: { workspaceId: string; laneId?: string },
  ) => {
    const qs = new URLSearchParams();
    qs.set("workspaceId", params.workspaceId);
    if (params.laneId) qs.set("laneId", params.laneId);
    return j<DreamReportCandidatesResponse>(
      `/api/dream/reports/${encodeURIComponent(id)}/candidates?${qs.toString()}`,
    );
  },
  proposals: (
    id: string,
    params: { workspaceId: string; laneId?: string },
  ) => {
    const qs = new URLSearchParams();
    qs.set("workspaceId", params.workspaceId);
    if (params.laneId) qs.set("laneId", params.laneId);
    return j<DreamReportProposalsResponse>(
      `/api/dream/reports/${encodeURIComponent(id)}/proposals?${qs.toString()}`,
    );
  },
  warnings: (
    id: string,
    params: { workspaceId: string; laneId?: string },
  ) => {
    const qs = new URLSearchParams();
    qs.set("workspaceId", params.workspaceId);
    if (params.laneId) qs.set("laneId", params.laneId);
    return j<DreamReportWarningsResponse>(
      `/api/dream/reports/${encodeURIComponent(id)}/warnings?${qs.toString()}`,
    );
  },
};
