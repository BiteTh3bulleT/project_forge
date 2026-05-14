import { j } from "./client";
import type { RetrievalResultVSASignal, RetrievalRun } from "@forge/shared";

export const retrievalApi = {
  createRun: (body: Record<string, unknown>) =>
    j<{ run: RetrievalRun }>("/api/retrieval/runs", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body),
    }),
  listRuns: (params?: {
    limit?: number;
    dossierId?: number;
    jobId?: string;
  }) => {
    const qs = new URLSearchParams();
    if (params?.limit != null) qs.set("limit", String(params.limit));
    if (params?.dossierId != null)
      qs.set("dossierId", String(params.dossierId));
    if (params?.jobId) qs.set("jobId", params.jobId);
    const q = qs.toString();
    return j<{ runs: RetrievalRun[] }>(
      `/api/retrieval/runs${q ? `?${q}` : ""}`,
    );
  },
  getRun: (id: number) =>
    j<{ run: RetrievalRun }>(`/api/retrieval/runs/${id}`),
  getRunVSASignals: (runId: number) =>
    j<{ signals: RetrievalResultVSASignal[] }>(
      `/api/retrieval/runs/${encodeURIComponent(String(runId))}/vsa-signals`,
    ),
  markUsefulness: (id: number, body: Record<string, unknown>) =>
    j<{ ok: boolean; resultId: number }>(
      `/api/retrieval/results/${id}/usefulness`,
      {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      },
    ),
};
